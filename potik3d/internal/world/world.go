// Package world holds the map: terraced terrain, flowing water and what
// stands on each tile. It has no graphics so it can be tested headless.
package world

import "math"

const (
	Size     = 96 // tiles per side
	MaxLevel = 8  // highest terrain level
)

// Prop is a natural object on a tile.
type Prop uint8

const (
	PropNone Prop = iota
	PropTree
	PropBush
	PropRock
)

// Kind is a placeholder building type for the prototype.
type Kind uint8

const (
	KNone Kind = iota
	KHall
	KHouse
	KLumber
	KPump
	KDam
)

// Building occupies a square of tiles.
type Building struct {
	Kind Kind
	X, Y int
	Size int
}

type World struct {
	Seed   int64
	Height []uint8   // terrain level per tile
	Water  []float32 // water depth per tile, in levels
	Props  []Prop
	Owner  []int16 // building index+1 per tile, 0 if free
	Builds []*Building
	Tick   uint64

	flux    []float32
	springs []int
	sinks   []int
}

func Idx(x, y int) int { return y*Size + x }

func In(x, y int) bool { return x >= 0 && y >= 0 && x < Size && y < Size }

// --- noise ------------------------------------------------------------------

func splitmix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

func (w *World) hash(a, b int, salt uint64) float64 {
	h := splitmix(uint64(w.Seed) ^ salt)
	h = splitmix(h ^ uint64(uint32(a)))
	h = splitmix(h ^ uint64(uint32(b))<<1)
	return float64(h>>11) / float64(1<<53)
}

func (w *World) noise(x, y, scale float64, salt uint64) float64 {
	fx, fy := x/scale, y/scale
	ix, iy := math.Floor(fx), math.Floor(fy)
	tx, ty := fx-ix, fy-iy
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	at := func(a, b float64) float64 { return w.hash(int(a), int(b), salt) }
	return (at(ix, iy)*(1-tx)+at(ix+1, iy)*tx)*(1-ty) + (at(ix, iy+1)*(1-tx)+at(ix+1, iy+1)*tx)*ty
}

// RiverX is the river's centre line at row y.
func (w *World) RiverX(y float64) float64 {
	return Size*0.42 + 6*math.Sin(y/14+float64(w.Seed%7)) + 2.5*math.Sin(y/5.3)
}

// New generates a valley with a river running north to south.
func New(seed int64) *World {
	n := Size * Size
	w := &World{Seed: seed, Height: make([]uint8, n), Water: make([]float32, n),
		Props: make([]Prop, n), Owner: make([]int16, n), flux: make([]float32, 4*n)}
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := math.Abs(fx - w.RiverX(fy))
			// valley: low near the river, rising in terraces towards the edges
			h := 2.0
			switch {
			case d < 2.4:
				h = 0
			case d < 3.6:
				h = 1
			default:
				// valley walls climb in terraces; hills further out
				rise := (d - 3.6) / 5.5
				h = 2 + math.Floor(rise+w.noise(fx, fy, 8, 11)*1.3-0.3)
				h += math.Floor(w.noise(fx, fy, 14, 23)*2.2-0.6) * boolf(d > 9)
			}
			if h > MaxLevel {
				h = MaxLevel
			}
			if h < 0 {
				h = 0
			}
			i := Idx(x, y)
			w.Height[i] = uint8(h)
			if h == 0 && y < 2 {
				w.springs = append(w.springs, i)
			}
			if h == 0 && y >= Size-2 {
				w.sinks = append(w.sinks, i)
			}
			// vegetation
			if h >= 2 {
				forest := w.noise(fx, fy, 10, 31)
				r := w.hash(x, y, 41)
				switch {
				case forest > 0.6 && r < 0.65:
					w.Props[i] = PropTree
				case forest > 0.45 && r < 0.05:
					w.Props[i] = PropBush
				case r > 0.985:
					w.Props[i] = PropRock
				}
			}
		}
	}
	for k := 0; k < 1500; k++ {
		w.StepWater()
	}
	return w
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// Place puts a building on flat, dry, free ground.
func (w *World) Place(k Kind, x, y, size int) bool {
	if !In(x, y) || !In(x+size-1, y+size-1) {
		return false
	}
	h := w.Height[Idx(x, y)]
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			i := Idx(x+dx, y+dy)
			if w.Owner[i] != 0 || w.Height[i] != h || (w.Water[i] > 0.2 && k != KDam) {
				return false
			}
		}
	}
	w.Builds = append(w.Builds, &Building{Kind: k, X: x, Y: y, Size: size})
	id := int16(len(w.Builds))
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			i := Idx(x+dx, y+dy)
			w.Owner[i] = id
			w.Props[i] = PropNone
		}
	}
	return true
}

// BuildingAt returns the building on a tile, or nil.
func (w *World) BuildingAt(x, y int) *Building {
	if !In(x, y) {
		return nil
	}
	if id := w.Owner[Idx(x, y)]; id > 0 {
		return w.Builds[id-1]
	}
	return nil
}

// --- water --------------------------------------------------------------------

const (
	waterDT    = 0.1
	gravity    = 9.8
	friction   = 0.985
	tilt       = 0.01
	sourceRate = 0.9
	evaporate  = 0.0003
)

func (w *World) wallAt(i int) float32 {
	if id := w.Owner[i]; id > 0 && w.Builds[id-1].Kind == KDam {
		return 1
	}
	return 0
}

// StepWater advances the virtual-pipes water simulation by 0.1 s.
func (w *World) StepWater() {
	n := Size * Size
	water, flux := w.Water, w.flux
	level := func(i int) float32 {
		return float32(w.Height[i]) + w.wallAt(i) + water[i] + float32(Size-1-i/Size)*tilt
	}
	for i := 0; i < n; i++ {
		f := flux[4*i : 4*i+4]
		if water[i] < 1e-4 {
			f[0], f[1], f[2], f[3] = 0, 0, 0, 0
			continue
		}
		x, y := i%Size, i/Size
		li := level(i)
		nbs := [4]int{-1, -1, -1, -1}
		if x < Size-1 {
			nbs[0] = i + 1
		}
		if x > 0 {
			nbs[1] = i - 1
		}
		if y < Size-1 {
			nbs[2] = i + Size
		}
		if y > 0 {
			nbs[3] = i - Size
		}
		sum := float32(0)
		for d, j := range nbs {
			if j < 0 {
				f[d] = 0
				continue
			}
			v := f[d]*friction + waterDT*gravity*(li-level(j))
			if v < 0 {
				v = 0
			}
			f[d] = v
			sum += v
		}
		if sum*waterDT > water[i] {
			k := water[i] / (sum * waterDT)
			for d := range f {
				f[d] *= k
			}
		}
	}
	for i := 0; i < n; i++ {
		x, y := i%Size, i/Size
		in := float32(0)
		if x > 0 {
			in += flux[4*(i-1)]
		}
		if x < Size-1 {
			in += flux[4*(i+1)+1]
		}
		if y > 0 {
			in += flux[4*(i-Size)+2]
		}
		if y < Size-1 {
			in += flux[4*(i+Size)+3]
		}
		f := flux[4*i : 4*i+4]
		water[i] += waterDT * (in - f[0] - f[1] - f[2] - f[3])
		if water[i] > 0 {
			water[i] -= evaporate * waterDT
			if water[i] < 1e-4 {
				water[i] = 0
			}
		}
	}
	for _, i := range w.springs {
		water[i] += sourceRate * waterDT
	}
	for _, i := range w.sinks {
		water[i] *= 0.5
	}
}

// Update advances the world by one 60 Hz tick.
func (w *World) Update() {
	w.Tick++
	if w.Tick%6 == 0 {
		w.StepWater()
	}
}
