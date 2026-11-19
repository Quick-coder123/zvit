// Package world is the simulation: terraced terrain, flowing water, nature,
// buildings and settlers. It has no graphics so it can be tested headless.
package world

import (
	"math"
	"sort"
)

const (
	Size     = 96 // tiles per side
	MaxLevel = 8  // highest terrain level
	TPS      = 60
	DT       = 1.0 / TPS
)

// Prop is a natural object on a tile.
type Prop uint8

const (
	PropNone Prop = iota
	PropTree
	PropBush
	PropRock
)

// World is the whole game state. Exported fields are saved.
type World struct {
	Seed   int64
	Tick   uint64
	Height []uint8   // terrain level per tile
	Water  []float32 // water depth per tile, in levels
	Props  []Prop

	Planted map[int]uint64 // tile -> tick a sapling was planted
	Picked  map[int]uint64 // tile -> tick a bush was picked

	Buildings   map[int32]*Building
	NextID      int32
	Settlers    []*Settler
	NextSettler int32

	Drought    bool
	SeasonLeft float64
	Cycle      int
	Day        float64 // in-game days for flavour (1 day = 2 minutes)
	Deaths     int
	Peaceful   bool

	Notices []string // UI messages (not saved)

	owner      []int32 // top-layer building id per tile
	support    []int32 // platform/bridge id per tile
	pathEpoch  uint32  // bumped whenever walkability changes
	flux       []float32
	springs    []int
	sinks      []int
	moist      []bool
	order      []int32
	dirty      bool
	fullWarned uint64

	pathG    []float32
	pathFrom []int32
	pathSeen []uint32
	pathGen  uint32
}

func Idx(x, y int) int { return y*Size + x }

func In(x, y int) bool { return x >= 0 && y >= 0 && x < Size && y < Size }

func XY(i int) (int, int) { return i % Size, i / Size }

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

// generate builds the terrain, springs and natural vegetation from the seed.
func (w *World) generate() {
	n := Size * Size
	w.Height = make([]uint8, n)
	w.Props = make([]Prop, n)
	w.springs, w.sinks = nil, nil
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			d := math.Abs(fx - w.RiverX(fy))
			h := 2.0
			switch {
			case d < 2.4:
				h = 0
			case d < 3.6:
				h = 1
			default:
				rise := (d - 3.6) / 5.5
				h = 2 + math.Floor(rise+w.noise(fx, fy, 8, 11)*1.3-0.3)
				if d > 9 {
					h += math.Floor(w.noise(fx, fy, 14, 23)*2.2 - 0.6)
				}
			}
			h = math.Max(0, math.Min(MaxLevel, h))
			i := Idx(x, y)
			w.Height[i] = uint8(h)
			if h == 0 && y < 2 {
				w.springs = append(w.springs, i)
			}
			if h == 0 && y >= Size-2 {
				w.sinks = append(w.sinks, i)
			}
			if h >= 2 {
				forest := w.noise(fx, fy, 10, 31)
				r := w.hash(x, y, 41)
				switch {
				case forest > 0.6 && r < 0.65:
					w.Props[i] = PropTree
				case forest > 0.42 && r < 0.06:
					w.Props[i] = PropBush
				case r > 0.99:
					w.Props[i] = PropRock
				}
			}
		}
	}
}

func (w *World) initCaches() {
	n := Size * Size
	w.owner = make([]int32, n)
	w.support = make([]int32, n)
	w.flux = make([]float32, 4*n)
	w.moist = make([]bool, n)
	if w.Planted == nil {
		w.Planted = map[int]uint64{}
	}
	if w.Picked == nil {
		w.Picked = map[int]uint64{}
	}
	if w.Buildings == nil {
		w.Buildings = map[int32]*Building{}
	}
	for _, b := range w.Buildings {
		w.mark(b, b.ID)
	}
	w.dirty = true
	w.updateMoisture()
}

// New creates a world with a town hall by the river and six settlers.
func New(seed int64) *World {
	w := &World{Seed: seed}
	w.generate()
	w.Water = make([]float32, Size*Size)
	w.initCaches()
	for k := 0; k < 1500; k++ {
		w.StepWater()
	}
	w.updateMoisture()
	w.SeasonLeft = w.WetLength(0)
	w.Peaceful = true

	// town hall on flat dry ground east of the river, near the map centre
	hx, hy := w.findFlat(int(w.RiverX(Size/2))+8, Size/2, 2)
	hall := w.add(&Building{Kind: KHall, X: hx, Y: hy, Recipe: -1})
	hall.Store = []Stack{{Log, 40}, {Plank, 10}, {Berries, 30}, {Water, 30}}
	for i := 0; i < 6; i++ {
		s := w.spawnSettler(hall)
		s.X += float64(i%3) - 1
	}
	return w
}

// findFlat finds a free flat dry square of the given size near (x, y).
func (w *World) findFlat(x, y, size int) (int, int) {
	for r := 0; r < Size; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if w.CanPlaceAt(x+dx, y+dy, size, false) {
					return x + dx, y + dy
				}
			}
		}
	}
	return x, y
}

// CanPlaceAt reports whether a square is flat, dry, free and above the river.
func (w *World) CanPlaceAt(x, y, size int, inWater bool) bool {
	if !In(x, y) || !In(x+size-1, y+size-1) {
		return false
	}
	h := w.Height[Idx(x, y)]
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			i := Idx(x+dx, y+dy)
			if w.owner[i] != 0 || w.support[i] != 0 || w.Height[i] != h {
				return false
			}
			if !inWater && (w.Water[i] > 0.15 || h == 0) {
				return false
			}
		}
	}
	return true
}

// Support returns the platform or bridge under a tile, or nil.
func (w *World) Support(i int) *Building {
	if id := w.support[i]; id != 0 {
		return w.Buildings[id]
	}
	return nil
}

// SupportAt returns the platform or bridge on a tile, or nil.
func (w *World) SupportAt(x, y int) *Building {
	if !In(x, y) {
		return nil
	}
	return w.Support(Idx(x, y))
}

// supportLevels is how many levels a support currently adds.
func supportLevels(b *Building) int {
	switch {
	case b == nil:
		return 0
	case b.Kind == KPlatform && b.Site:
		return max(0, b.Levels-1)
	case b.Site:
		return 0
	}
	return b.Levels
}

// Surface is the level of the walkable surface of a tile: the ground plus
// any finished platform or bridge on it.
func (w *World) Surface(i int) int {
	return int(w.Height[i]) + supportLevels(w.Support(i))
}

// occupied reports tiles field workers must leave alone.
func (w *World) occupied(i int) bool { return w.owner[i] != 0 || w.support[i] != 0 }

// Owner returns the top-layer building on a tile, or nil.
func (w *World) Owner(i int) *Building {
	if id := w.owner[i]; id != 0 {
		return w.Buildings[id]
	}
	return nil
}

// BuildingAt returns the building on a tile, or nil.
func (w *World) BuildingAt(x, y int) *Building {
	if !In(x, y) {
		return nil
	}
	return w.Owner(Idx(x, y))
}

func (w *World) mark(b *Building, id int32) {
	layer := w.owner
	if b.Kind.IsSupport() {
		layer = w.support
	}
	w.pathEpoch++
	s := b.Size()
	for dy := 0; dy < s; dy++ {
		for dx := 0; dx < s; dx++ {
			if In(b.X+dx, b.Y+dy) {
				layer[Idx(b.X+dx, b.Y+dy)] = id
			}
		}
	}
}

func (w *World) add(b *Building) *Building {
	w.NextID++
	b.ID = w.NextID
	w.Buildings[b.ID] = b
	w.mark(b, b.ID)
	for dy := 0; dy < b.Size(); dy++ {
		for dx := 0; dx < b.Size(); dx++ {
			i := Idx(b.X+dx, b.Y+dy)
			w.Props[i] = PropNone
			delete(w.Planted, i)
		}
	}
	w.dirty = true
	return b
}

// Order returns building ids in a stable order.
func (w *World) Order() []int32 {
	if w.dirty {
		w.order = w.order[:0]
		for id := range w.Buildings {
			w.order = append(w.order, id)
		}
		sort.Slice(w.order, func(a, b int) bool { return w.order[a] < w.order[b] })
		w.dirty = false
	}
	return w.order
}

func (w *World) notice(s string) {
	if len(w.Notices) < 20 {
		w.Notices = append(w.Notices, s)
	}
}

// Update advances the world by one tick.
func (w *World) Update() {
	w.Tick++
	w.Day += DT / 120
	w.updateSeasons()
	if w.Tick%6 == 0 {
		w.StepWater()
	}
	if w.Tick%(2*TPS) == 0 {
		w.updateMoisture()
	}
	for _, id := range w.Order() {
		if b := w.Buildings[id]; b != nil {
			w.stepBuilding(b)
		}
	}
	w.updateSettlers()
}

// --- water --------------------------------------------------------------------

const (
	waterDT     = 0.1
	gravity     = 9.8
	friction    = 0.985
	tilt        = 0.01
	sourceRate  = 0.9
	evapWet     = 0.0003
	evapDrought = 0.004
	MoistRange  = 3
)

func (w *World) wallAt(i int) float32 {
	if b := w.Owner(i); b != nil && b.Kind == KDam && !b.Site {
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
	evap := float32(evapWet)
	if w.Drought {
		evap = evapDrought
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
			water[i] -= evap * waterDT
			if water[i] < 1e-4 {
				water[i] = 0
			}
		}
	}
	if !w.Drought {
		for _, i := range w.springs {
			water[i] += sourceRate * waterDT
		}
	}
	for _, i := range w.sinks {
		water[i] *= 0.5
	}
}

// updateMoisture marks soil within MoistRange tiles of water.
func (w *World) updateMoisture() {
	n := Size * Size
	dist := make([]int8, n)
	queue := make([]int, 0, 2048)
	for i := 0; i < n; i++ {
		dist[i] = -1
		if w.Water[i] > 0.1 {
			dist[i] = 0
			queue = append(queue, i)
		}
	}
	for q := 0; q < len(queue); q++ {
		i := queue[q]
		if dist[i] >= MoistRange {
			continue
		}
		x, y := XY(i)
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if In(nx, ny) {
				j := Idx(nx, ny)
				if dist[j] < 0 {
					dist[j] = dist[i] + 1
					queue = append(queue, j)
				}
			}
		}
	}
	for i := 0; i < n; i++ {
		w.moist[i] = dist[i] >= 0
	}
}

// Moist reports whether a tile's soil is wet enough for crops.
func (w *World) Moist(i int) bool { return w.moist[i] }

// --- seasons ---------------------------------------------------------------------

func (w *World) WetLength(cycle int) float64 {
	if cycle == 0 {
		return 480
	}
	return 360 + w.hash(cycle, 0, 515)*120
}

func DroughtLength(cycle int) float64 { return math.Min(300, 150+30*float64(cycle)) }

func (w *World) updateSeasons() {
	prev := w.SeasonLeft
	w.SeasonLeft -= DT
	if !w.Drought && prev > 60 && w.SeasonLeft <= 60 {
		w.notice("Посуха за хвилину! Запасіть воду й їжу.")
	}
	if w.SeasonLeft > 0 {
		return
	}
	if w.Drought {
		w.Drought = false
		w.Cycle++
		w.SeasonLeft = w.WetLength(w.Cycle)
		w.notice("Посуха скінчилась — річка знову наповнюється.")
	} else {
		w.Drought = true
		w.SeasonLeft = DroughtLength(w.Cycle)
		w.notice("Почалась посуха! Річка міліє, поля без вологи не родять.")
	}
}
