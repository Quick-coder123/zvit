package sim

import "math"

const chunkSize = 32

type patch struct {
	cx, cy float64
	r      float64
	item   Item
	phase  float64
}

func splitmix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

func hash3(seed int64, a, b int, salt uint64) uint64 {
	h := splitmix(uint64(seed) ^ salt)
	h = splitmix(h ^ uint64(uint32(a)))
	h = splitmix(h ^ (uint64(uint32(b)) << 1))
	return h
}

func unit(h uint64) float64 { return float64(h>>11) / float64(1<<53) }

// startPatches guarantee a playable start near the core.
var startPatches = []patch{
	{cx: 13, cy: -5, r: 6.5, item: IronOre, phase: 0.3},
	{cx: -13, cy: -6, r: 5.5, item: CopperOre, phase: 1.7},
	{cx: 3, cy: 13, r: 4.5, item: Coal, phase: 2.2},
	{cx: -10, cy: 10, r: 4.5, item: Stone, phase: 0.9},
	{cx: 22, cy: 16, r: 4, item: Sand, phase: 4.1},
}

func (w *World) chunkPatches(cx, cy int) []patch {
	if cx == 0 && cy == 0 {
		return startPatches
	}
	var out []patch
	dist := math.Hypot(float64(cx), float64(cy))
	if dist < 1.5 {
		// keep the area around the start readable
		return nil
	}
	n := 1
	if unit(hash3(w.Seed, cx, cy, 7)) < 0.35 {
		n = 2
	}
	for i := 0; i < n; i++ {
		h := hash3(w.Seed, cx, cy, uint64(100+i))
		if unit(h) > 0.7 {
			continue
		}
		// weighted pick, rarer stuff further away
		roll := unit(splitmix(h))
		var it Item
		switch {
		case roll < 0.30:
			it = IronOre
		case roll < 0.52:
			it = CopperOre
		case roll < 0.70:
			it = Coal
		case roll < 0.85:
			it = Stone
		default:
			it = Sand
		}
		r := 4 + unit(splitmix(h+1))*5 + math.Min(dist, 6)*0.6
		px := float64(cx*chunkSize) + 6 + unit(splitmix(h+2))*float64(chunkSize-12)
		py := float64(cy*chunkSize) + 6 + unit(splitmix(h+3))*float64(chunkSize-12)
		out = append(out, patch{cx: px, cy: py, r: r, item: it, phase: unit(splitmix(h+4)) * 6.28})
	}
	return out
}

func (w *World) computeOre(p Pos) Item {
	ccx, ccy := floorDiv(p.X, chunkSize), floorDiv(p.Y, chunkSize)
	fx, fy := float64(p.X)+0.5, float64(p.Y)+0.5
	best := ItemNone
	bestV := 0.0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			for _, pt := range w.chunkPatches(ccx+dx, ccy+dy) {
				ddx, ddy := fx-pt.cx, fy-pt.cy
				d := math.Hypot(ddx, ddy)
				if d > pt.r*1.4 {
					continue
				}
				a := math.Atan2(ddy, ddx)
				wob := 1 + 0.18*math.Sin(3*a+pt.phase) + 0.1*math.Sin(5*a-pt.phase*2)
				edge := pt.r * wob
				v := 1 - d/edge
				// ragged edge
				v += (unit(hash3(w.Seed, p.X, p.Y, 55)) - 0.5) * 0.25
				if v > 0 && v > bestV {
					bestV, best = v, pt.item
				}
			}
		}
	}
	return best
}

// OreAt returns the resource under a tile (ItemNone if empty).
func (w *World) OreAt(p Pos) Item {
	cp := Pos{floorDiv(p.X, chunkSize), floorDiv(p.Y, chunkSize)}
	c := w.chunks[cp]
	if c == nil {
		c = new([chunkSize * chunkSize]Item)
		for y := 0; y < chunkSize; y++ {
			for x := 0; x < chunkSize; x++ {
				c[y*chunkSize+x] = w.computeOre(Pos{cp.X*chunkSize + x, cp.Y*chunkSize + y})
			}
		}
		w.chunks[cp] = c
	}
	return c[floorMod(p.Y, chunkSize)*chunkSize+floorMod(p.X, chunkSize)]
}
