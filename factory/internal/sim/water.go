package sim

import "math"

// Terrain heights and water. Water is simulated on a square region around
// the town hall; outside it the land is dry.
const (
	WRegion   = 128         // tiles per side of the water region
	wOff      = WRegion / 2 // region covers tiles -wOff .. wOff-1
	waterStep = 6           // ticks between water updates (10 per second)
	waterDT   = float32(waterStep) * DT

	gravity      = 9.8
	friction     = 0.985  // flux kept from the previous step
	tilt         = 0.012  // the land slopes gently south so rivers flow
	sourceRate   = 0.9    // depth added per second to each spring tile
	evapWet      = 0.0002 // depth evaporating per second
	evapDrought  = 0.004
	contDecay    = 0.004 // contamination fading per second
	DirtyWater   = 0.3   // pumps and soil reject water dirtier than this
	MoistRange   = 4     // tiles from water that count as moist soil
	PumpTime     = 3.0   // seconds per water item
	PurifyTime   = 5.0   // a purifier is slower but takes any water
	PumpDraw     = 0.08  // depth taken from the water per item
	FloodDepth   = 0.3   // buildings under this much water stop working
	BuildMaxWet  = 0.2   // most buildings can't be placed in deeper water
	PumpMinDepth = 0.15
)

// riverX is the centre line of the river at row y.
func riverX(y float64) float64 {
	return -15 + 3*math.Sin(y/13) + 1.5*math.Sin(y/5.1)
}

// genHeight is the natural terrain height of a tile (0 = river bed).
func (w *World) genHeight(p Pos) uint8 {
	fx, fy := float64(p.X)+0.5, float64(p.Y)+0.5
	d := math.Abs(fx - riverX(fy))
	switch {
	case d < 2.3:
		return 0
	case d < 3.7:
		return 1
	}
	if math.Hypot(fx, fy) > 12 {
		n := w.noise(fx, fy, 14, 333)
		switch {
		case n > 0.74:
			return 4
		case n > 0.64:
			return 3
		}
	}
	return 2
}

func regionIndex(p Pos) (int, bool) {
	x, y := p.X+wOff, p.Y+wOff
	if x < 0 || y < 0 || x >= WRegion || y >= WRegion {
		return 0, false
	}
	return y*WRegion + x, true
}

// HeightAt returns the terrain height of a tile.
func (w *World) HeightAt(p Pos) int {
	if i, ok := regionIndex(p); ok && w.height != nil {
		return int(w.height[i])
	}
	return int(w.genHeight(p))
}

// WaterAt returns the water depth on a tile.
func (w *World) WaterAt(p Pos) float64 {
	if i, ok := regionIndex(p); ok && len(w.Water) == WRegion*WRegion {
		return float64(w.Water[i])
	}
	return 0
}

// MoistAt reports whether the soil on a tile is wet enough for crops.
func (w *World) MoistAt(p Pos) bool {
	if i, ok := regionIndex(p); ok && w.moist != nil {
		return w.moist[i]
	}
	return false
}

// initWater prepares terrain, springs and outflows; a world without saved
// water gets a river that has been flowing for a while.
func (w *World) initWater() {
	n := WRegion * WRegion
	w.height = make([]uint8, n)
	w.wall = make([]float32, n)
	w.flux = make([]float32, 4*n)
	w.moist = make([]bool, n)
	w.springs, w.sinks = w.springs[:0], w.sinks[:0]
	for y := 0; y < WRegion; y++ {
		for x := 0; x < WRegion; x++ {
			p := Pos{x - wOff, y - wOff}
			i := y*WRegion + x
			w.height[i] = w.genHeight(p)
			if w.height[i] == 0 && y < 2 {
				w.springs = append(w.springs, i)
			}
			if w.height[i] == 0 && y >= WRegion-2 {
				w.sinks = append(w.sinks, i)
			}
		}
	}
	w.wallsDirty = true
	if len(w.Water) != n {
		w.Water = make([]float32, n)
		for k := 0; k < 2000; k++ {
			w.stepWater(false)
		}
	}
	w.updateMoisture()
}

// waterBlock is how much a building raises the water level it holds back.
func waterBlock(k Kind) float32 {
	switch k {
	case KDam:
		return 1
	case KLevee, KGate:
		return 3
	}
	return 0
}

func (w *World) rebuildWalls() {
	for i := range w.wall {
		w.wall[i] = 0
	}
	for _, b := range w.Buildings {
		if blk := waterBlock(b.Kind); blk > 0 && !b.Site && !(b.Kind == KGate && b.GateOpen) {
			if i, ok := regionIndex(b.Pos); ok {
				w.wall[i] = blk
			}
		}
	}
	w.wallsDirty = false
}

// stepWater advances the water simulation by one water step, using the
// virtual-pipes method: each cell keeps an outflow towards its four
// neighbours that is accelerated by the difference in water level, which
// lets rivers actually run downhill.
func (w *World) stepWater(drought bool) {
	if w.wallsDirty {
		w.rebuildWalls()
	}
	n := WRegion * WRegion
	water, height, wall, flux := w.Water, w.height, w.wall, w.flux
	dt := waterDT
	level := func(i int) float32 {
		y := i / WRegion
		return float32(height[i]) + wall[i] + water[i] + float32(WRegion-1-y)*tilt
	}
	// 1. update outflows (R, L, D, U)
	for i := 0; i < n; i++ {
		f := flux[4*i : 4*i+4]
		if water[i] < 1e-4 {
			f[0], f[1], f[2], f[3] = 0, 0, 0, 0
			continue
		}
		x, y := i%WRegion, i/WRegion
		li := level(i)
		nbs := [4]int{-1, -1, -1, -1}
		if x < WRegion-1 {
			nbs[0] = i + 1
		}
		if x > 0 {
			nbs[1] = i - 1
		}
		if y < WRegion-1 {
			nbs[2] = i + WRegion
		}
		if y > 0 {
			nbs[3] = i - WRegion
		}
		sum := float32(0)
		for d, j := range nbs {
			if j < 0 {
				f[d] = 0
				continue
			}
			v := f[d]*friction + dt*gravity*(li-level(j))
			if v < 0 {
				v = 0
			}
			f[d] = v
			sum += v
		}
		if sum*dt > water[i] {
			k := water[i] / (sum * dt)
			for d := range f {
				f[d] *= k
			}
		}
	}
	// 2. move water (and the contamination it carries)
	if len(w.Cont) != n {
		w.Cont = make([]float32, n)
	}
	cont := w.Cont
	if len(w.massBuf) != n {
		w.massBuf = make([]float32, n)
	}
	mass := w.massBuf
	for i := 0; i < n; i++ {
		mass[i] = water[i] * cont[i]
	}
	for i := 0; i < n; i++ {
		x, y := i%WRegion, i/WRegion
		in := float32(0)
		if x > 0 {
			in += flux[4*(i-1)+0]
		}
		if x < WRegion-1 {
			in += flux[4*(i+1)+1]
		}
		if y > 0 {
			in += flux[4*(i-WRegion)+2]
		}
		if y < WRegion-1 {
			in += flux[4*(i+WRegion)+3]
		}
		f := flux[4*i : 4*i+4]
		out := f[0] + f[1] + f[2] + f[3]
		if in == 0 && out == 0 && water[i] == 0 {
			continue
		}
		// contaminant mass moves with the water
		min := float32(0)
		if x > 0 {
			min += flux[4*(i-1)+0] * cont[i-1]
		}
		if x < WRegion-1 {
			min += flux[4*(i+1)+1] * cont[i+1]
		}
		if y > 0 {
			min += flux[4*(i-WRegion)+2] * cont[i-WRegion]
		}
		if y < WRegion-1 {
			min += flux[4*(i+WRegion)+3] * cont[i+WRegion]
		}
		mass[i] += dt * (min - out*cont[i])
		water[i] += dt * (in - out)
	}
	for i := 0; i < n; i++ {
		if water[i] > 1e-4 {
			c := mass[i] / water[i] * (1 - contDecay*dt)
			if c < 0.002 {
				c = 0
			} else if c > 1 {
				c = 1
			}
			cont[i] = c
		} else {
			cont[i] = 0
		}
	}
	evap := float32(evapWet)
	if drought {
		evap = evapDrought
	}
	evap *= dt
	for i := range water {
		if v := water[i]; v > 0 {
			v -= evap
			if v < 1e-4 {
				v = 0
			}
			water[i] = v
		}
	}
	if !drought {
		for _, i := range w.springs {
			add := sourceRate * dt
			c := float32(0)
			if w.Polluted {
				c = 1
			}
			cont[i] = (cont[i]*water[i] + c*add) / (water[i] + add)
			water[i] += add
		}
	}
	for _, i := range w.sinks {
		water[i] *= 0.5
	}
}

// updateMoisture marks soil within MoistRange tiles of water.
func (w *World) updateMoisture() {
	n := WRegion * WRegion
	dist := make([]int8, n)
	queue := make([]int, 0, 4096)
	for i := 0; i < n; i++ {
		dist[i] = -1
		if w.Water[i] > 0.05 && (len(w.Cont) != n || w.Cont[i] < DirtyWater) {
			dist[i] = 0
			queue = append(queue, i)
		}
	}
	for q := 0; q < len(queue); q++ {
		i := queue[q]
		if dist[i] >= MoistRange {
			continue
		}
		x, y := i%WRegion, i/WRegion
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= WRegion || ny >= WRegion {
				continue
			}
			j := ny*WRegion + nx
			if dist[j] < 0 {
				dist[j] = dist[i] + 1
				queue = append(queue, j)
			}
		}
	}
	for i := 0; i < n; i++ {
		w.moist[i] = dist[i] >= 0
	}
}

// Seasons.

// WetLength and DroughtLength are the season lengths for a cycle.
func (w *World) WetLength(cycle int) float64 {
	if cycle == 0 {
		return 600
	}
	return 360 + unit(hash3(w.Seed, cycle, 0, 515))*120
}

func DroughtLength(cycle int) float64 { return math.Min(300, 120+30*float64(cycle)) }

// PollutedLength is how long a season of muddy water lasts.
func PollutedLength(cycle int) float64 { return math.Min(240, 90+20*float64(cycle)) }

// nextPolluted decides whether the coming hazard is muddy water rather
// than a drought (from the third cycle on, about a third of the time).
func (w *World) nextPolluted() bool {
	return w.Cycle >= 2 && unit(hash3(w.Seed, w.Cycle, 1, 919)) < 0.35
}

// ContAt returns how dirty the water on a tile is (0 clean .. 1 muddy).
func (w *World) ContAt(p Pos) float64 {
	if i, ok := regionIndex(p); ok && len(w.Cont) == WRegion*WRegion {
		return float64(w.Cont[i])
	}
	return 0
}

func (w *World) updateSeasons() {
	if w.SeasonLeft <= 0 && !w.Drought && w.Cycle == 0 && w.Tick < 2 {
		w.SeasonLeft = w.WetLength(0)
	}
	prev := w.SeasonLeft
	w.SeasonLeft -= DT
	if !w.Drought && !w.Polluted && prev > 60 && w.SeasonLeft <= 60 {
		if w.nextPolluted() {
			w.notice("За хвилину річка стане каламутною! Закрийте шлюзи, запасіть чисту воду.")
		} else {
			w.notice("Посуха за хвилину! Запасіть воду й їжу.")
		}
	}
	if w.SeasonLeft > 0 {
		return
	}
	switch {
	case w.Drought || w.Polluted:
		if w.Polluted {
			w.notice("Каламутна вода минула — річка поступово очиститься.")
		} else {
			w.notice("Посуха скінчилась — річка знову наповнюється.")
		}
		w.Drought, w.Polluted = false, false
		w.Cycle++
		w.SeasonLeft = w.WetLength(w.Cycle)
	case w.nextPolluted():
		w.Polluted = true
		w.SeasonLeft = PollutedLength(w.Cycle)
		w.notice("Річка каламутна! Насоси не беруть брудну воду, вона не зволожує ґрунт.")
	default:
		w.Drought = true
		w.SeasonLeft = DroughtLength(w.Cycle)
		w.notice("Почалась посуха! Річка міліє, поля без вологи не родять.")
	}
}

// updateWater runs seasons, flow and moisture.
func (w *World) updateWater() {
	w.updateSeasons()
	if w.Tick%waterStep == 0 {
		w.stepWater(w.Drought)
	}
	if w.Tick%(2*TPS) == 0 {
		w.updateMoisture()
	}
}

// pumpSource finds the deepest water a pump at b can draw from.
func (w *World) pumpSource(b *Building) (int, bool) {
	best, bd := 0, float32(PumpMinDepth)
	found := false
	for dy := -1; dy <= b.Size(); dy++ {
		for dx := -1; dx <= b.Size(); dx++ {
			i, ok := regionIndex(b.Pos.Add(dx, dy))
			if ok && b.Kind == KPump && len(w.Cont) == len(w.Water) && w.Cont[i] >= DirtyWater {
				continue
			}
			if ok && w.Water[i] >= bd {
				best, bd, found = i, w.Water[i], true
			}
		}
	}
	return best, found
}

func (w *World) stepPump(b *Building) {
	if b.OutN >= OutCap {
		b.Working = false
		return
	}
	i, ok := w.pumpSource(b)
	if !ok {
		b.Working = false
		return
	}
	b.Working = true
	t := PumpTime
	if b.Kind == KPurifier {
		t = PurifyTime
	}
	b.Progress += float32(DT * w.workRate(b) / t)
	if b.Progress >= 1 {
		b.Progress = 0
		w.Water[i] = float32(math.Max(0, float64(w.Water[i])-PumpDraw))
		b.OutItem = Water
		b.OutN++
	}
}

// Flooded reports whether water covers a building that can't stand in it.
func (w *World) Flooded(b *Building) bool {
	if b.Kind == KPump || b.Kind == KPurifier || waterBlock(b.Kind) > 0 {
		return false
	}
	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			if w.WaterAt(b.Pos.Add(x, y)) > FloodDepth {
				return true
			}
		}
	}
	return false
}

// nearWater reports whether a tile next to (or under) a footprint holds
// enough water for a pump.
func (w *World) nearWater(p Pos, size int) bool {
	for dy := -1; dy <= size; dy++ {
		for dx := -1; dx <= size; dx++ {
			if w.WaterAt(p.Add(dx, dy)) >= 0.25 {
				return true
			}
		}
	}
	return false
}
