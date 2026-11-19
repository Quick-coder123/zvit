package sim

import "math"

// Vegetation on a tile.
type Veg uint8

const (
	VegNone Veg = iota
	VegTree
	VegBush
)

const (
	TreeGrowTime  = 240.0 // seconds for a planted sapling to become a tree
	BushRegrow    = 90.0  // seconds for picked berries to grow back
	FieldRadius   = 6     // reach of lumberjacks, foresters and gatherers
	ChopTime      = 4.0
	PlantTime     = 3.0
	PickTime      = 3.0
	WoodPerTree   = 2
	FoodPerBush   = 2
	vegChunkTiles = chunkSize * chunkSize
)

// value noise in [0,1)
func (w *World) noise(x, y, scale float64, salt uint64) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-ix, fy-iy
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	at := func(a, b float64) float64 { return unit(hash3(w.Seed, int(a), int(b), salt)) }
	v00, v10 := at(ix, iy), at(ix+1, iy)
	v01, v11 := at(ix, iy+1), at(ix+1, iy+1)
	return (v00*(1-tx)+v10*tx)*(1-ty) + (v01*(1-tx)+v11*tx)*ty
}

// startForests guarantee wood and berries near the town hall.
var startForests = []struct {
	x, y, r float64
	veg     Veg
}{
	{11, 8, 4.5, VegTree},
	{-8, 4, 2.5, VegBush},
}

func (w *World) computeVeg(p Pos) Veg {
	if w.OreAt(p) != ItemNone || w.genHeight(p) == 0 {
		return VegNone
	}
	fx, fy := float64(p.X)+0.5, float64(p.Y)+0.5
	d := math.Hypot(fx, fy)
	h := unit(hash3(w.Seed, p.X, p.Y, 4711))
	for _, f := range startForests {
		if math.Hypot(fx-f.x, fy-f.y) < f.r {
			if f.veg == VegTree && h < 0.7 || f.veg == VegBush && h < 0.45 {
				return f.veg
			}
		}
	}
	if d < 6 {
		return VegNone // keep the town square clear
	}
	forest := w.noise(fx, fy, 11, 811)
	if forest > 0.64 && h < 0.6 {
		return VegTree
	}
	if forest > 0.5 && forest < 0.64 && h < 0.04 {
		return VegBush
	}
	return VegNone
}

func (w *World) naturalVeg(p Pos) Veg {
	cp := Pos{floorDiv(p.X, chunkSize), floorDiv(p.Y, chunkSize)}
	c := w.vegChunks[cp]
	if c == nil {
		c = new([vegChunkTiles]Veg)
		for y := 0; y < chunkSize; y++ {
			for x := 0; x < chunkSize; x++ {
				c[y*chunkSize+x] = w.computeVeg(Pos{cp.X*chunkSize + x, cp.Y*chunkSize + y})
			}
		}
		w.vegChunks[cp] = c
	}
	return c[floorMod(p.Y, chunkSize)*chunkSize+floorMod(p.X, chunkSize)]
}

// VegAt reports what grows on a tile and how grown it is (trees: 0..1;
// bushes: 1 when berries are ripe).
func (w *World) VegAt(p Pos) (Veg, float64) {
	if t, ok := w.Planted[p]; ok {
		return VegTree, math.Min(1, float64(w.Tick-t)/TPS/TreeGrowTime)
	}
	if w.Cleared[p] {
		return VegNone, 0
	}
	v := w.naturalVeg(p)
	switch v {
	case VegTree:
		return v, 1
	case VegBush:
		if !w.MoistAt(p) {
			return v, 0 // no berries on dry soil
		}
		if t, ok := w.Picked[p]; ok {
			return v, math.Min(1, float64(w.Tick-t)/TPS/BushRegrow)
		}
		return v, 1
	}
	return VegNone, 0
}

// clearVeg removes whatever grows on a tile (building over it, chopping).
func (w *World) clearVeg(p Pos) {
	delete(w.Planted, p)
	if w.naturalVeg(p) != VegNone {
		w.Cleared[p] = true
	}
}

// fieldTarget finds the nearest tile a field worker of building b can work
// on, skipping tiles other workers already claimed.
func (w *World) fieldTarget(b *Building, s *Settler) (Pos, bool) {
	cx, cy := b.center()
	claimed := map[Pos]bool{}
	for _, o := range w.Settlers {
		if o != s && o.FieldPhase != 0 {
			claimed[Pos{o.FieldX, o.FieldY}] = true
		}
	}
	best, bd, found := Pos{}, math.MaxFloat64, false
	for dy := -FieldRadius; dy <= FieldRadius; dy++ {
		for dx := -FieldRadius; dx <= FieldRadius; dx++ {
			p := Pos{b.Pos.X + dx, b.Pos.Y + dy}
			if claimed[p] || w.At(p) != nil {
				continue
			}
			v, g := w.VegAt(p)
			ok := false
			switch b.Kind {
			case KLumber:
				ok = v == VegTree && g >= 1
			case KGatherer:
				ok = v == VegBush && g >= 1
			case KForester:
				ok = v == VegNone && w.OreAt(p) == ItemNone && (p.X+p.Y)%2 == 0 && w.MoistAt(p) && w.WaterAt(p) < 0.05
			}
			if !ok {
				continue
			}
			d := math.Hypot(float64(p.X)+0.5-cx, float64(p.Y)+0.5-cy)
			if d <= FieldRadius+0.5 && d < bd {
				best, bd, found = p, d, true
			}
		}
	}
	return best, found
}

// stepFieldWorker runs a lumberjack, forester or gatherer working outdoors.
// Returns true while the settler is busy outside.
func (w *World) stepFieldWorker(s *Settler, b *Building) bool {
	switch s.FieldPhase {
	case 0: // inside: look for work when there is room for the produce
		if !s.Inside || !b.Running() || (b.Kind != KForester && b.OutN >= OutCap) {
			return false
		}
		if s.FieldT > 0 { // short rest between trips
			s.FieldT -= DT
			return false
		}
		p, ok := w.fieldTarget(b, s)
		if !ok {
			s.FieldT = 2
			return false
		}
		s.FieldX, s.FieldY, s.FieldPhase, s.Inside = p.X, p.Y, 1, false
		return true
	case 1: // walking out
		tx, ty := float64(s.FieldX)+0.5, float64(s.FieldY)+0.5
		w.walkToPoint(s, tx, ty, w.speed(s))
		if math.Hypot(tx-s.X, ty-s.Y) < 0.45 {
			s.FieldPhase = 2
			switch b.Kind {
			case KLumber:
				s.FieldT = ChopTime
			case KForester:
				s.FieldT = PlantTime
			default:
				s.FieldT = PickTime
			}
		}
		return true
	case 2: // working on the tile
		rate := 1.0
		if s.Starved() {
			rate = 0.5
		}
		s.FieldT -= float32(DT * rate)
		if s.FieldT > 0 {
			return true
		}
		p := Pos{s.FieldX, s.FieldY}
		v, g := w.VegAt(p)
		switch {
		case b.Kind == KLumber && v == VegTree && g >= 1:
			w.clearVeg(p)
			s.Carry = Stack{Wood, WoodPerTree}
		case b.Kind == KGatherer && v == VegBush && g >= 1:
			w.Picked[p] = w.Tick
			s.Carry = Stack{Food, FoodPerBush}
		case b.Kind == KForester && v == VegNone && w.At(p) == nil:
			delete(w.Cleared, p)
			w.Planted[p] = w.Tick
		}
		s.FieldPhase = 3
		return true
	case 3: // walking back
		if !w.walkTo(s, b) {
			return true
		}
		if s.Carry.N > 0 {
			b.OutItem = s.Carry.Item
			b.OutN += s.Carry.N
			s.Carry = Stack{}
		}
		s.FieldPhase, s.Inside, s.FieldT = 0, true, 0.5
		return false
	}
	return false
}

// IsFieldKind reports buildings whose workers go out to the land.
func (k Kind) IsFieldKind() bool { return k == KLumber || k == KForester || k == KGatherer }
