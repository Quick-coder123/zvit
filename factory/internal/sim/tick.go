package sim

// ActiveFunc reports whether buildings of an owner currently run.
type ActiveFunc func(owner uint8) bool

// Update advances the simulation by one tick.
func (w *World) Update(active ActiveFunc) {
	w.Tick++
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b == nil {
			continue
		}
		if active != nil && b.Kind != KCore && !active(b.Owner) {
			continue
		}
		if b.Kind == KCore {
			w.stepCoreGun(b)
			continue
		}
		if !b.Running() {
			b.Working = false
			continue
		}
		switch b.Kind.Base() {
		case KMiner:
			w.stepMiner(b)
		case KFurnace, KAssembler, KSawmill:
			w.stepCrafter(b)
		case KFarm:
			w.stepFarm(b)
		case KTurret:
			w.stepTurret(b)
		case KFlamer:
			w.stepFlamer(b)
		case KLab:
			w.stepLab(b)
		case KPump:
			w.stepPump(b)
		}
	}
	w.updateWater()
	w.updateSettlers()
	w.updateCombat()
}

// workRate is how fast a running building works, slowed by hungry workers.
func (w *World) workRate(b *Building) float64 {
	r := SpeedOf(b.Kind)
	if b.hungry {
		r *= 0.5
	}
	return r
}

func inCap(need int) int { return need*3 + 2 }

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
		if w.RecipeUnlocked(r) && recipeUses(r, it) {
			return r
		}
	}
	return -1
}

func (w *World) stepMiner(b *Building) {
	if b.MinerItem == ItemNone || b.OutN >= OutCap {
		b.Working = false
		return
	}
	b.Working = true
	rate := MinerRate(b) * w.MineMul()
	if b.hungry {
		rate *= 0.5
	}
	b.Progress += float32(DT * rate)
	if b.Progress >= 1 {
		b.Progress -= 1
		b.OutItem = b.MinerItem
		b.OutN++
	}
}

func (w *World) stepFarm(b *Building) {
	if b.OutN >= OutCap || !w.MoistAt(b.Pos) {
		b.Working = false
		return
	}
	b.Working = true
	b.Progress += float32(DT * w.workRate(b) / FarmTime)
	if b.Progress >= 1 {
		b.Progress -= 1
		b.OutItem = Food
		b.OutN++
	}
}

func (w *World) stepCrafter(b *Building) {
	if b.Recipe < 0 {
		b.Working = false
		return
	}
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
		b.Progress += float32(DT * w.CraftMul() * w.workRate(b) / r.Time)
		if b.Progress >= 1 {
			b.Working = false
			b.Progress = 0
			b.OutItem = r.Out.Item
			b.OutN += r.Out.N
		}
	}
}
