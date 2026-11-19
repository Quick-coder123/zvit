package sim

import (
	"math"
	"sort"
)

// TaskKind is what a settler is doing besides working in a building.
type TaskKind uint8

const (
	TaskNone  TaskKind = iota
	TaskHaul           // carry items From -> To
	TaskBuild          // work on a construction site
)

// Settler is one inhabitant. Fields are exported for gob saves.
type Settler struct {
	ID     int32
	X, Y   float64
	Home   int32 // housing building
	Job    int32 // workplace (a station makes them a carrier)
	Inside bool  // inside the workplace and working

	Task  TaskKind
	From  int32
	To    int32
	Item  Item
	N     int
	Phase uint8 // haul: 0 = going to pick up, 1 = going to drop off
	Carry Stack

	MealIn   float32 // seconds until the next meal
	Starving float32 // seconds spent without food
	Seed     float32 // per-settler variation for idle wandering
}

const (
	WalkSpeed     = 2.2
	CarrierSpeed  = 3.2
	CarryFree     = 3
	CarryStation  = 6
	StationRadius = 12.0
	MealInterval  = 90.0  // a settler eats one food this often
	StarveDeath   = 120.0 // seconds of starving before death
	BirthTime     = 75.0  // seconds for a home with two adults and food
	WandererTime  = 60.0  // a newcomer arrives when the town is (almost) empty
	allocEvery    = 15    // ticks between job/haul planning passes
)

func (s *Settler) idle() bool { return s.Task == TaskNone && s.Carry.N == 0 }

// Starved reports whether the settler is currently going hungry.
func (s *Settler) Starved() bool { return s.Starving > 0 }

func (w *World) spawnSettler(at *Building) *Settler {
	w.NextSettler++
	x, y := at.center()
	s := &Settler{
		ID: w.NextSettler, X: x, Y: y + float64(at.Size())/2 + 0.4,
		MealIn: float32(MealInterval*0.5 + float64(w.NextSettler%7)*6),
		Seed:   float32(unit(hash3(w.Seed, int(w.NextSettler), 0, 77)) * 6.28),
	}
	w.Settlers = append(w.Settlers, s)
	return s
}

// station returns the cargo station a settler works for, or nil.
func (w *World) station(s *Settler) *Building {
	if b := w.Buildings[s.Job]; b != nil && b.Kind == KStation {
		return b
	}
	return nil
}

// Capacity is how many items a settler carries at once.
func (w *World) Capacity(s *Settler) int {
	if w.station(s) != nil {
		if w.Researched["belt2"] {
			return CarryStation * 2
		}
		return CarryStation
	}
	return CarryFree
}

func (w *World) speed(s *Settler) float64 {
	v := WalkSpeed
	if w.station(s) != nil {
		v = CarrierSpeed
	}
	if w.Researched["belt1"] {
		v *= 1.25
	}
	if s.Starved() {
		v *= 0.6
	}
	return v
}

// walkTo moves a settler toward a building and reports arrival.
func (w *World) walkTo(s *Settler, b *Building) bool {
	d, nx, ny := distToBuilding(s.X, s.Y, b)
	if d <= 0.3 {
		return true
	}
	step := w.speed(s) * DT
	if step >= d-0.25 {
		step = d - 0.25
	}
	s.X += (nx - s.X) / d * step
	s.Y += (ny - s.Y) / d * step
	return false
}

func (w *World) walkToPoint(s *Settler, x, y, speed float64) {
	dx, dy := x-s.X, y-s.Y
	d := math.Hypot(dx, dy)
	if d < 0.05 {
		return
	}
	step := math.Min(d, speed*DT)
	s.X += dx / d * step
	s.Y += dy / d * step
}

// Population and housing.

func (w *World) Population() int { return len(w.Settlers) }

func (w *World) Housing() int {
	n := 0
	for _, b := range w.Buildings {
		if !b.Site {
			n += Builds[b.Kind].Housing
		}
	}
	return n
}

// FreeWorkers counts settlers without a workplace.
func (w *World) FreeWorkers() int {
	n := 0
	for _, s := range w.Settlers {
		if s.Job == 0 {
			n++
		}
	}
	return n
}

func (w *World) updateSettlers() {
	for _, s := range w.Settlers {
		w.stepSettler(s)
	}
	// starvation
	k := 0
	for _, s := range w.Settlers {
		if s.Starving >= StarveDeath {
			w.Deaths++
			w.notice("Житель помер від голоду. Потрібна їжа — став ферми!")
			continue
		}
		w.Settlers[k] = s
		k++
	}
	for i := k; i < len(w.Settlers); i++ {
		w.Settlers[i] = nil
	}
	w.Settlers = w.Settlers[:k]

	w.updateBirths()
	if w.Tick%allocEvery == 0 {
		w.allocate()
	}
}

func (w *World) stepSettler(s *Settler) {
	// eating
	s.MealIn -= DT
	if s.MealIn <= 0 {
		if w.TakeStock(Food, 1) {
			s.MealIn = MealInterval
			s.Starving = 0
		} else {
			s.MealIn = 0
			s.Starving += DT
		}
	}

	switch s.Task {
	case TaskHaul:
		w.stepHaul(s)
		return
	case TaskBuild:
		site := w.Buildings[s.To]
		if site == nil || !site.Site || len(site.Need) > 0 {
			s.Task = TaskNone
			return
		}
		if w.walkTo(s, site) {
			rate := 1.0
			if s.Starved() {
				rate = 0.5
			}
			site.BuildWork += float32(DT * rate / BuildTime(site.Kind))
			if site.BuildWork >= 1 {
				site.Site, site.BuildWork = false, 0
				s.Task = TaskNone
			}
		}
		return
	}

	if s.Job != 0 {
		jb := w.Buildings[s.Job]
		if jb == nil {
			s.Job, s.Inside = 0, false
			return
		}
		if s.Inside {
			s.X, s.Y = jb.center()
			return
		}
		if w.walkTo(s, jb) && jb.Kind != KStation {
			s.Inside = true
		}
		return
	}

	// idle: drift around home
	home := w.Buildings[s.Home]
	if home == nil {
		home = w.Core()
	}
	if home == nil {
		return
	}
	hx, hy := home.center()
	t := float64(w.Tick)/TPS*0.25 + float64(s.Seed)
	r := float64(home.Size())/2 + 1.2
	w.walkToPoint(s, hx+math.Cos(t)*r, hy+math.Sin(t*1.3)*r*0.6, WalkSpeed*0.35)
}

func (w *World) stepHaul(s *Settler) {
	if s.Phase == 0 {
		from := w.Buildings[s.From]
		if from == nil {
			s.Task = TaskNone
			return
		}
		if !w.walkTo(s, from) {
			return
		}
		got := w.pickup(from, s.Item, s.N)
		if got == 0 {
			s.Task = TaskNone
			return
		}
		s.Carry = Stack{s.Item, got}
		s.N = got
		s.Phase = 1
		return
	}
	to := w.Buildings[s.To]
	if to == nil {
		w.redirectToStorage(s)
		return
	}
	if !w.walkTo(s, to) {
		return
	}
	left := w.deliver(to, s.Carry)
	if left > 0 {
		s.Carry.N = left
		s.N = left
		w.redirectToStorage(s)
		return
	}
	s.Carry = Stack{}
	s.Task = TaskNone
}

// redirectToStorage sends whatever a settler carries to the nearest storage.
func (w *World) redirectToStorage(s *Settler) {
	st := w.nearestStorage(s.X, s.Y, s.Carry.N, 0)
	if st == nil || s.Carry.N == 0 {
		s.Carry, s.Task = Stack{}, TaskNone
		return
	}
	s.Task, s.Phase, s.To, s.Item, s.N = TaskHaul, 1, st.ID, s.Carry.Item, s.Carry.N
}

// pickup takes up to n items from a building's output or storage.
func (w *World) pickup(b *Building, it Item, n int) int {
	switch {
	case b.Kind == KCore:
		n = min(n, w.Stock[it])
		w.Stock[it] -= n
		return n
	case b.Kind == KChest:
		return b.storeTake(it, n)
	case b.OutItem == it:
		n = min(n, b.OutN)
		b.OutN -= n
		return n
	}
	return 0
}

// deliver hands a carried stack to a building; returns what it refused.
func (w *World) deliver(b *Building, st Stack) int {
	n := st.N
	switch {
	case b.Site:
		for i := range b.Need {
			if b.Need[i].Item == st.Item {
				d := min(n, b.Need[i].N)
				b.Need[i].N -= d
				n -= d
				if b.Need[i].N == 0 {
					b.Need = append(b.Need[:i], b.Need[i+1:]...)
				}
				break
			}
		}
		return n
	case b.Kind == KCore:
		w.Stock[st.Item] += n
		return 0
	case b.Kind == KChest:
		d := min(n, w.ChestCapacity()-b.StoreCount())
		if d > 0 {
			b.storeAdd(st.Item, d)
		}
		return n - d
	case b.Kind.Base() == KTurret && st.Item == Ammo:
		b.Ammo += int32(n * AmmoShots)
		return 0
	case b.Kind == KFlamer && st.Item == Coal:
		b.Ammo += int32(n * CoalFuel)
		return 0
	case b.Kind.IsCrafter() && b.Recipe >= 0:
		for i, in := range Recipes[b.Recipe].In {
			if in.Item == st.Item {
				b.InBuf[i].N += n
				return 0
			}
		}
	}
	return n
}

// storageFree is how many more items a storage building accepts.
func (w *World) storageFree(b *Building) int {
	if b.Kind == KCore {
		return math.MaxInt32
	}
	n := w.ChestCapacity() - b.StoreCount()
	for _, p := range b.pend {
		n -= p
	}
	return n
}

// nearestStorage finds the closest storage with room for n items.
func (w *World) nearestStorage(x, y float64, n int, exclude int32) *Building {
	var best *Building
	bd := math.MaxFloat64
	for _, b := range w.Buildings {
		if !b.Kind.IsStorage() || b.Site || b.ID == exclude || w.storageFree(b) < n {
			continue
		}
		bx, by := b.center()
		if d := math.Hypot(bx-x, by-y); d < bd {
			best, bd = b, d
		}
	}
	return best
}

func (w *World) updateBirths() {
	pop, cap := len(w.Settlers), w.Housing()
	food := w.TotalStock(Food)
	starving := false
	for _, s := range w.Settlers {
		if s.Starved() {
			starving = true
			break
		}
	}
	if pop < 2 && pop < cap {
		w.WanderTimer += DT
		if w.WanderTimer >= WandererTime {
			w.WanderTimer = 0
			if c := w.Core(); c != nil {
				w.spawnSettler(c)
				w.notice("До поселення прийшов новий житель")
			}
		}
	}
	if pop >= cap || food < 5 || starving {
		return
	}
	residents := map[int32]int{}
	for _, s := range w.Settlers {
		residents[s.Home]++
	}
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Site || Builds[b.Kind].Housing == 0 || residents[b.ID] < 2 {
			continue
		}
		b.Progress += float32(DT / BirthTime)
		if b.Progress >= 1 {
			b.Progress = 0
			s := w.spawnSettler(b)
			s.Home = b.ID
			w.notice("Народився новий житель!")
			return
		}
	}
}

// --- planning: jobs, construction and hauling --------------------------------------

type demand struct {
	to   *Building
	item Item
	n    int
	prio int
}

func dist(a, b *Building) float64 {
	ax, ay := a.center()
	bx, by := b.center()
	return math.Hypot(ax-bx, ay-by)
}

func distTo(s *Settler, b *Building) float64 {
	x, y := b.center()
	return math.Hypot(s.X-x, s.Y-y)
}

// rebuildBookkeeping recomputes staffing and reservations from the settlers.
func (w *World) rebuildBookkeeping() {
	for _, b := range w.Buildings {
		b.staff, b.inside, b.builder, b.hungry = 0, 0, 0, false
		b.pend, b.res = nil, nil
	}
	add := func(m *map[Item]int, it Item, n int) {
		if *m == nil {
			*m = map[Item]int{}
		}
		(*m)[it] += n
	}
	for _, s := range w.Settlers {
		if jb := w.Buildings[s.Job]; jb != nil {
			jb.staff++
			if s.Inside {
				jb.inside++
				if s.Starved() {
					jb.hungry = true
				}
			}
		}
		switch s.Task {
		case TaskHaul:
			if to := w.Buildings[s.To]; to != nil {
				add(&to.pend, s.Item, s.N)
			}
			if s.Phase == 0 {
				if from := w.Buildings[s.From]; from != nil {
					add(&from.res, s.Item, s.N)
				}
			}
		case TaskBuild:
			if site := w.Buildings[s.To]; site != nil {
				site.builder = s.ID
			}
		}
	}
}

func jobPriority(w *World, b *Building) int {
	switch b.Kind.Base() {
	case KFarm:
		return 0
	case KTurret, KFlamer:
		if len(w.Enemies) > 0 {
			return 0
		}
		return 3
	case KStation:
		return 2
	}
	return 3
}

// allocate assigns jobs, builders and hauling tasks.
func (w *World) allocate() {
	w.rebuildBookkeeping()

	// release workers from paused, unfinished or overstaffed buildings
	for _, s := range w.Settlers {
		if s.Job == 0 {
			continue
		}
		jb := w.Buildings[s.Job]
		if jb == nil || jb.Paused || jb.Site || jb.staff > Builds[jb.Kind].Workers {
			if jb != nil {
				jb.staff--
				if s.Inside {
					jb.inside--
				}
			}
			s.Job, s.Inside = 0, false
		}
	}

	// homes
	residents := map[int32]int{}
	for _, s := range w.Settlers {
		if hb := w.Buildings[s.Home]; hb != nil && !hb.Site && Builds[hb.Kind].Housing > 0 {
			residents[s.Home]++
		} else {
			s.Home = 0
		}
	}
	for _, s := range w.Settlers {
		if s.Home != 0 {
			continue
		}
		for _, id := range w.Order() {
			b := w.Buildings[id]
			if !b.Site && residents[id] < Builds[b.Kind].Housing {
				s.Home = id
				residents[id]++
				break
			}
		}
	}

	freeIdle := func() []*Settler {
		var out []*Settler
		for _, s := range w.Settlers {
			if s.Job == 0 && s.idle() {
				out = append(out, s)
			}
		}
		return out
	}
	take := func(list []*Settler, near *Building) ([]*Settler, *Settler) {
		bi, bd := -1, math.MaxFloat64
		for i, s := range list {
			if d := distTo(s, near); d < bd {
				bi, bd = i, d
			}
		}
		if bi < 0 {
			return list, nil
		}
		s := list[bi]
		return append(list[:bi], list[bi+1:]...), s
	}

	// furnaces without a recipe pick one from what is in stock
	for _, b := range w.Buildings {
		if b.Kind.Base() == KFurnace && b.Recipe < 0 && b.AutoRecipe && !b.Site {
			for _, ore := range []Item{IronOre, CopperOre, Stone, Sand} {
				if w.TotalStock(ore) > 0 {
					if r := w.autoRecipe(b.Kind, ore); r >= 0 {
						w.SetRecipe(b.ID, r)
						b.AutoRecipe = true
						break
					}
				}
			}
		}
	}

	// jobs
	var jobs []*Building
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if Builds[b.Kind].Workers > 0 && !b.Site && !b.Paused && b.staff < Builds[b.Kind].Workers {
			jobs = append(jobs, b)
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool { return jobPriority(w, jobs[i]) < jobPriority(w, jobs[j]) })
	idle := freeIdle()
	for _, b := range jobs {
		for b.staff < Builds[b.Kind].Workers {
			var s *Settler
			idle, s = take(idle, b)
			if s == nil {
				break
			}
			s.Job, s.Inside = b.ID, false
			b.staff++
		}
	}

	// builders
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Site && len(b.Need) == 0 && b.builder == 0 {
			var s *Settler
			idle, s = take(idle, b)
			if s == nil {
				break
			}
			s.Task, s.To = TaskBuild, b.ID
			b.builder = s.ID
		}
	}

	// haulers: idle free settlers and idle carriers
	var carriers []*Settler
	for _, s := range w.Settlers {
		if st := w.station(s); st != nil && s.idle() && !st.Paused && !st.Site {
			carriers = append(carriers, s)
		}
	}
	if len(idle) == 0 && len(carriers) == 0 {
		return
	}

	// demands
	var ds []demand
	pend := func(b *Building, it Item) int { return b.pend[it] }
	for _, id := range w.Order() {
		b := w.Buildings[id]
		switch {
		case b.Site:
			for _, n := range b.Need {
				if left := n.N - pend(b, n.Item); left > 0 {
					ds = append(ds, demand{b, n.Item, left, 1})
				}
			}
		case b.Paused:
		case b.Kind.IsCrafter() && b.Recipe >= 0:
			for i, in := range Recipes[b.Recipe].In {
				if want := inCap(in.N) - b.InBuf[i].N - pend(b, in.Item); want > 0 {
					ds = append(ds, demand{b, in.Item, want, 2})
				}
			}
		case b.Kind.Base() == KTurret:
			cap := TurretAmmoCap * b.Size()
			have := int(b.Ammo) + pend(b, Ammo)*AmmoShots
			if have < cap*6/10 {
				p := 2
				if len(w.Enemies) > 0 {
					p = 0
				}
				ds = append(ds, demand{b, Ammo, (cap - have) / AmmoShots, p})
			}
		case b.Kind == KFlamer:
			have := int(b.Ammo) + pend(b, Coal)*CoalFuel
			if have < FlamerFuelCap*6/10 {
				p := 2
				if len(w.Enemies) > 0 {
					p = 0
				}
				ds = append(ds, demand{b, Coal, (FlamerFuelCap - have) / CoalFuel, p})
			}
		}
	}
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].prio < ds[j].prio })

	avail := func(b *Building, it Item) int {
		var n int
		switch {
		case b.Site:
			return 0
		case b.Kind == KCore:
			n = w.Stock[it]
		case b.Kind == KChest:
			n = b.storeHas(it)
		case b.OutItem == it:
			n = b.OutN
		}
		return n - b.res[it]
	}
	reserve := func(m *map[Item]int, it Item, n int) {
		if *m == nil {
			*m = map[Item]int{}
		}
		(*m)[it] += n
	}
	// pickHauler chooses the nearest suitable hauler for a trip from -> to.
	pickHauler := func(from, to *Building) *Settler {
		best, bd, bi, isCarrier := (*Settler)(nil), math.MaxFloat64, -1, false
		for i, s := range carriers {
			st := w.station(s)
			if dist(st, from) > StationRadius || dist(st, to) > StationRadius {
				continue
			}
			if d := distTo(s, from); d < bd {
				best, bd, bi, isCarrier = s, d, i, true
			}
		}
		if best == nil {
			for i, s := range idle {
				if d := distTo(s, from); d < bd {
					best, bd, bi = s, d, i
				}
			}
		}
		if best == nil {
			return nil
		}
		if isCarrier {
			carriers = append(carriers[:bi], carriers[bi+1:]...)
		} else {
			idle = append(idle[:bi], idle[bi+1:]...)
		}
		return best
	}
	assign := func(s *Settler, from, to *Building, it Item, n int) {
		s.Task, s.Phase, s.From, s.To, s.Item, s.N = TaskHaul, 0, from.ID, to.ID, it, n
		reserve(&from.res, it, n)
		reserve(&to.pend, it, n)
	}

	// supply sources per item: producers first, then storage
	sources := func(it Item, near *Building) []*Building {
		var out []*Building
		for _, b := range w.Buildings {
			if b != near && avail(b, it) > 0 {
				out = append(out, b)
			}
		}
		sort.Slice(out, func(i, j int) bool {
			pi, pj := out[i].Kind.IsStorage(), out[j].Kind.IsStorage()
			if pi != pj {
				return !pi
			}
			return dist(out[i], near) < dist(out[j], near)
		})
		return out
	}

	for _, d := range ds {
		for d.n > 0 && (len(idle) > 0 || len(carriers) > 0) {
			src := sources(d.item, d.to)
			if len(src) == 0 {
				break
			}
			from := src[0]
			s := pickHauler(from, d.to)
			if s == nil {
				break
			}
			n := min(d.n, avail(from, d.item), w.Capacity(s))
			assign(s, from, d.to, d.item, n)
			d.n -= n
		}
	}

	// whatever producers made goes to the nearest storage
	for _, id := range w.Order() {
		if len(idle) == 0 && len(carriers) == 0 {
			return
		}
		b := w.Buildings[id]
		if b.Kind.IsStorage() || b.Site || b.OutN == 0 {
			continue
		}
		n := avail(b, b.OutItem)
		if n <= 0 || (n < CarryFree && b.OutN < OutCap-1) {
			continue // wait for a fuller load unless the building is about to stall
		}
		x, y := b.center()
		st := w.nearestStorage(x, y, 1, 0)
		if st == nil {
			continue
		}
		s := pickHauler(b, st)
		if s == nil {
			continue
		}
		assign(s, b, st, b.OutItem, min(n, w.Capacity(s)))
	}
}

// Residents counts settlers living in a building.
func (w *World) Residents(id int32) int {
	n := 0
	for _, s := range w.Settlers {
		if s.Home == id {
			n++
		}
	}
	return n
}

// FoodMinutes estimates how long the stored food lasts.
func (w *World) FoodMinutes() float64 {
	if len(w.Settlers) == 0 {
		return math.Inf(1)
	}
	return float64(w.TotalStock(Food)) * MealInterval / float64(len(w.Settlers)) / 60
}

// Counts of settlers by role for the UI.
func (w *World) RoleCounts() (free, workers, carriers, starving int) {
	for _, s := range w.Settlers {
		switch {
		case s.Job == 0:
			free++
		case w.station(s) != nil:
			carriers++
		default:
			workers++
		}
		if s.Starved() {
			starving++
		}
	}
	return
}
