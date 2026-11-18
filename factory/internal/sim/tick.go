package sim

// ActiveFunc reports whether buildings of an owner currently run.
type ActiveFunc func(owner uint8) bool

// Update advances the simulation by one tick.
func (w *World) Update(active ActiveFunc) {
	w.Tick++
	w.retry = w.retry[:0]
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b == nil {
			continue
		}
		if active != nil && b.Kind != KCore && !active(b.Owner) {
			continue
		}
		switch b.Kind {
		case KBelt:
			w.stepBelt(b)
		case KMiner:
			w.stepMiner(b)
		case KFurnace, KAssembler:
			w.stepCrafter(b)
		case KChest:
			w.stepChest(b)
		}
	}
	// Belts whose front item was blocked get a second chance now that
	// downstream belts have moved, which keeps lines fully compressed.
	for _, b := range w.retry {
		if len(b.Belt) > 0 && b.Belt[0].P >= 1 && w.beltOut(b, b.Belt[0].Item, 0) {
			b.Belt = b.Belt[1:]
		}
	}
}

func (w *World) stepBelt(b *Building) {
	if len(b.Belt) == 0 {
		return
	}
	mv := float32(BeltSpeed * DT)
	if np := b.Belt[0].P + mv; np >= 1 {
		if w.beltOut(b, b.Belt[0].Item, np-1) {
			b.Belt = b.Belt[1:]
		}
	}
	limit := float32(1)
	for i := range b.Belt {
		np := b.Belt[i].P + mv
		if np > limit {
			np = limit
		}
		if np > b.Belt[i].P {
			b.Belt[i].P = np
		}
		limit = b.Belt[i].P - BeltSpacing
	}
	if len(b.Belt) > 0 && b.Belt[0].P >= 1 {
		w.retry = append(w.retry, b)
	}
}

// beltOut moves an item off the end of belt b.
func (w *World) beltOut(b *Building, it Item, overflow float32) bool {
	return w.deliver(b.Pos.Step(b.Dir), b.Dir, it, overflow)
}

// deliver hands an item to whatever is at tile t, travelling in direction d.
func (w *World) deliver(t Pos, d Dir, it Item, overflow float32) bool {
	tb := w.At(t)
	if tb == nil {
		return false
	}
	if tb.Kind == KBelt {
		return w.beltInsert(tb, d, it, overflow)
	}
	return w.accept(tb, it)
}

// beltInsert puts an item on belt tb that arrives travelling in direction d.
func (w *World) beltInsert(tb *Building, d Dir, it Item, overflow float32) bool {
	if d == tb.Dir.Opp() {
		return false
	}
	pos := float32(0.5)
	if d == tb.Dir || (tb.Curve == -1 && d == tb.Dir.CW()) || (tb.Curve == 1 && d == tb.Dir.CCW()) {
		pos = overflow
		if pos > 0.2 {
			pos = 0.2
		}
		n := len(tb.Belt)
		if n > 0 && tb.Belt[n-1].P-pos < BeltSpacing {
			return false
		}
		tb.Belt = append(tb.Belt, BeltItem{it, pos})
		return true
	}
	// side-load in the middle
	idx := len(tb.Belt)
	for i, bi := range tb.Belt {
		dd := bi.P - pos
		if dd < 0 {
			dd = -dd
		}
		if dd < BeltSpacing {
			return false
		}
		if bi.P < pos && idx == len(tb.Belt) {
			idx = i
		}
	}
	tb.Belt = append(tb.Belt, BeltItem{})
	copy(tb.Belt[idx+1:], tb.Belt[idx:])
	tb.Belt[idx] = BeltItem{it, pos}
	return true
}

func inCap(need int) int { return need*2 + 2 }

// accept offers an item to a non-belt building.
func (w *World) accept(b *Building, it Item) bool {
	switch b.Kind {
	case KCore:
		w.Stock[it]++
		return true
	case KChest:
		if b.StoreCount() >= ChestCap {
			return false
		}
		if n := len(b.Store); n > 0 && b.Store[n-1].Item == it {
			b.Store[n-1].N++
		} else {
			b.Store = append(b.Store, Stack{it, 1})
		}
		return true
	case KFurnace, KAssembler:
		if b.Recipe < 0 || (b.AutoRecipe && b.idle() && !recipeUses(b.Recipe, it)) {
			r := w.autoRecipe(b.Kind, it)
			if r < 0 {
				return false
			}
			w.SetRecipe(b.ID, r)
			b.AutoRecipe = true
		}
		for i, in := range Recipes[b.Recipe].In {
			if in.Item == it {
				if b.InBuf[i].N >= inCap(in.N) {
					return false
				}
				b.InBuf[i].N++
				return true
			}
		}
	}
	return false
}

func recipeUses(r int, it Item) bool {
	for _, in := range Recipes[r].In {
		if in.Item == it {
			return true
		}
	}
	return false
}

func (b *Building) idle() bool {
	if b.Working || b.OutN > 0 {
		return false
	}
	for _, s := range b.InBuf {
		if s.N > 0 {
			return false
		}
	}
	return true
}

func (w *World) autoRecipe(k Kind, it Item) int {
	for _, r := range RecipesFor(k) {
		if recipeUses(r, it) {
			return r
		}
	}
	return -1
}

func (w *World) stepMiner(b *Building) {
	if b.MinerItem == ItemNone || b.MinerTiles == 0 {
		return
	}
	if b.OutN < OutCap {
		b.Working = true
		b.Progress += float32(DT / (MinerTime * 4 / float64(b.MinerTiles)))
		if b.Progress >= 1 {
			b.Progress -= 1
			b.OutItem = b.MinerItem
			b.OutN++
		}
	} else {
		b.Working = false
	}
	w.pushOut(b)
}

func (w *World) stepCrafter(b *Building) {
	if b.Recipe >= 0 {
		r := Recipes[b.Recipe]
		if !b.Working && b.OutN+r.Out.N <= OutCap && (b.OutN == 0 || b.OutItem == r.Out.Item) {
			ok := true
			for i, in := range r.In {
				if b.InBuf[i].N < in.N {
					ok = false
					break
				}
			}
			if ok {
				for i, in := range r.In {
					b.InBuf[i].N -= in.N
				}
				b.Working = true
				b.Progress = 0
			}
		}
		if b.Working {
			b.Progress += float32(DT / r.Time)
			if b.Progress >= 1 {
				b.Working = false
				b.Progress = 0
				b.OutItem = r.Out.Item
				b.OutN += r.Out.N
			}
		}
	}
	w.pushOut(b)
}

func (w *World) stepChest(b *Building) {
	if len(b.Store) == 0 {
		return
	}
	it := b.Store[0].Item
	if w.pushItem(b, it) {
		b.Store[0].N--
		if b.Store[0].N == 0 {
			b.Store = b.Store[1:]
		}
	}
}

func (w *World) pushOut(b *Building) {
	if b.OutN > 0 && w.pushItem(b, b.OutItem) {
		b.OutN--
	}
}

// pushItem tries to output one item through the building's front face.
func (w *World) pushItem(b *Building, it Item) bool {
	fronts := b.FrontTiles()
	for i := range fronts {
		k := (b.OutRR + i) % len(fronts)
		t := fronts[k]
		tb := w.At(t)
		if tb == nil {
			continue
		}
		if tb.Kind == KBelt && tb.Dir == b.Dir.Opp() {
			continue
		}
		if w.deliver(t, b.Dir, it, 0) {
			b.OutRR = k + 1
			return true
		}
	}
	return false
}
