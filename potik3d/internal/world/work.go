package world

import "math"

// TreeGrowth returns how grown the tree on a tile is (0..1).
func (w *World) TreeGrowth(i int) float64 {
	if t, ok := w.Planted[i]; ok {
		return math.Min(1, float64(w.Tick-t)/TPS/TreeGrow)
	}
	return 1
}

// BushRipe reports a berry bush ready to pick.
func (w *World) BushRipe(i int) bool {
	if w.Props[i] != PropBush || !w.moist[i] {
		return false
	}
	if t, ok := w.Picked[i]; ok {
		return float64(w.Tick-t)/TPS >= BushRegrow
	}
	return true
}

func (w *World) stepBuilding(b *Building) {
	if !b.Running() {
		b.Working = false
		return
	}
	rate := float32(DT)
	switch b.Kind {
	case KFarm:
		if b.OutN >= OutCap || !w.moist[Idx(b.X, b.Y)] {
			b.Working = false
			return
		}
		b.Working = true
		b.Progress += rate / FarmTime
		if b.Progress >= 1 {
			b.Progress, b.Out = 0, Crop
			b.OutN++
		}
	case KPump:
		src := w.pumpSource(b)
		if b.OutN >= OutCap || src < 0 {
			b.Working = false
			return
		}
		b.Working = true
		b.Progress += rate / PumpTime
		if b.Progress >= 1 {
			b.Progress, b.Out = 0, Water
			b.OutN++
			w.Water[src] = float32(math.Max(0, float64(w.Water[src])-0.06))
		}
	case KSawmill:
		if b.OutN >= OutCap || (b.InN == 0 && !b.Working) {
			b.Working = false
			return
		}
		if !b.Working {
			b.InN--
			b.Working, b.Progress = true, 0
		}
		b.Progress += rate / SawTime
		if b.Progress >= 1 {
			b.Working, b.Out = false, Plank
			b.OutN++
		}
	}
}

// pumpSource finds the deepest water next to a pump (-1 if none).
func (w *World) pumpSource(b *Building) int {
	best, bd := -1, float32(0.15)
	for dy := -1; dy <= b.Size(); dy++ {
		for dx := -1; dx <= b.Size(); dx++ {
			x, y := b.X+dx, b.Y+dy
			if In(x, y) && w.Water[Idx(x, y)] >= bd {
				best, bd = Idx(x, y), w.Water[Idx(x, y)]
			}
		}
	}
	return best
}

// fieldTarget finds the nearest tile an outdoor worker can work on.
func (w *World) fieldTarget(b *Building, s *Settler) int {
	claimed := map[int]bool{}
	for _, o := range w.Settlers {
		if o != s && o.Field != 0 {
			claimed[Idx(o.FX, o.FY)] = true
		}
	}
	r := b.Def().Radius
	cx, cy := b.Center()
	best, bd := -1, math.MaxFloat64
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := b.X+dx, b.Y+dy
			if !In(x, y) {
				continue
			}
			i := Idx(x, y)
			d := dist(float64(x)+0.5, float64(y)+0.5, cx, cy)
			if d > float64(r)+0.5 || d >= bd || claimed[i] || w.occupied(i) {
				continue
			}
			ok := false
			switch b.Kind {
			case KLumber:
				ok = w.Props[i] == PropTree && w.TreeGrowth(i) >= 1
			case KGatherer:
				ok = w.BushRipe(i)
			case KForester:
				ok = w.Props[i] == PropNone && w.moist[i] && w.Water[i] < 0.05 && w.Height[i] >= 1 && (x+y)%2 == 0
			}
			if ok {
				best, bd = i, d
			}
		}
	}
	return best
}

// stepField runs an outdoor worker; returns true while they are outside.
func (w *World) stepField(s *Settler, b *Building) bool {
	switch s.Field {
	case 0:
		if !s.Inside || !b.Running() || (b.Kind != KForester && b.OutN >= OutCap) {
			return false
		}
		if s.FT > 0 {
			s.FT -= DT
			return false
		}
		t := w.fieldTarget(b, s)
		if t < 0 {
			s.FT = 2
			return false
		}
		s.FX, s.FY = XY(t)
		s.Field, s.Inside = 1, false
		return true
	case 1:
		t := Idx(s.FX, s.FY)
		if w.walkToTile(s, t) {
			s.Field = 2
			switch b.Kind {
			case KLumber:
				s.FT = ChopTime
			case KForester:
				s.FT = PlantTime
			default:
				s.FT = PickTime
			}
		} else if s.noPath > 3 {
			s.Field, s.noPath, s.FT = 3, 0, 0
		}
		return true
	case 2:
		rate := float32(DT)
		if s.Suffering() {
			rate *= 0.5
		}
		s.FT -= rate
		if s.FT > 0 {
			return true
		}
		i := Idx(s.FX, s.FY)
		switch {
		case b.Kind == KLumber && w.Props[i] == PropTree && w.TreeGrowth(i) >= 1:
			w.Props[i] = PropNone
			delete(w.Planted, i)
			s.Carry = Stack{Log, WoodPerTree}
		case b.Kind == KGatherer && w.BushRipe(i):
			w.Picked[i] = w.Tick
			s.Carry = Stack{Berries, BerriesPerBush}
		case b.Kind == KForester && w.Props[i] == PropNone && !w.occupied(i):
			w.Props[i] = PropTree
			w.Planted[i] = w.Tick
		}
		s.Field = 3
		return true
	case 3:
		if !w.walkToBuilding(s, b) {
			if s.noPath > 3 {
				s.Field, s.noPath = 0, 0
			}
			return true
		}
		if s.Carry.N > 0 {
			b.Out = s.Carry.Item
			b.OutN += s.Carry.N
			s.Carry = Stack{}
		}
		s.Field, s.Inside, s.FT = 0, true, 0.5
		return false
	}
	return false
}
