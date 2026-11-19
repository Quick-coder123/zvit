package world

import (
	"math"
	"sort"
)

// TaskKind is what a settler does besides working inside a building.
type TaskKind uint8

const (
	TaskNone TaskKind = iota
	TaskHaul
	TaskBuild
)

// Settler is one inhabitant. Exported fields are saved.
type Settler struct {
	ID     int32
	X, Y   float64 // position in tile units
	Z      float64 // standing level (smoothed)
	Home   int32
	Job    int32
	Inside bool

	Task  TaskKind
	From  int32
	To    int32
	Item  Item
	N     int
	Phase uint8 // haul: 0 to pick-up, 1 to drop-off
	Carry Stack

	Field  uint8 // outdoor work: 0 none, 1 out, 2 working, 3 back
	FX, FY int
	FT     float32

	MealIn, DrinkIn float32
	Hunger, Thirst  float32 // seconds without food / water
	LastFood        Item
	Seed            float32

	path    []int
	pathKey int
	noPath  float32
}

const (
	WalkSpeed     = 2.2
	CarryN        = 3
	MealInterval  = 100.0
	DrinkInterval = 70.0
	StarveDeath   = 120.0
	ThirstDeath   = 150.0
	BirthTime     = 70.0
	allocEvery    = 15
)

// Suffering reports a hungry or thirsty settler.
func (s *Settler) Suffering() bool { return s.Hunger > 0 || s.Thirst > 0 }

func (s *Settler) idle() bool { return s.Task == TaskNone && s.Carry.N == 0 && s.Field == 0 }

func (s *Settler) tile() int { return Idx(clampi(int(s.X), 0, Size-1), clampi(int(s.Y), 0, Size-1)) }

func clampi(v, a, b int) int { return max(a, min(b, v)) }

func (w *World) spawnSettler(at *Building) *Settler {
	w.NextSettler++
	cx, cy := at.Center()
	s := &Settler{ID: w.NextSettler, X: cx, Y: float64(at.Y+at.Size()) + 0.5,
		MealIn:  float32(MealInterval*0.4 + float64(w.NextSettler%7)*5),
		DrinkIn: float32(DrinkInterval*0.4 + float64(w.NextSettler%5)*5),
		Seed:    float32(w.hash(int(w.NextSettler), 0, 77) * 6.28)}
	if !In(int(s.X), int(s.Y)) || w.Owner(s.tile()) != nil {
		s.X, s.Y = cx, cy
	}
	s.Z = float64(w.WalkHeight(s.tile()))
	w.Settlers = append(w.Settlers, s)
	return s
}

// --- movement -----------------------------------------------------------------

func (w *World) speed(s *Settler) float64 {
	v := WalkSpeed
	if s.Suffering() {
		v *= 0.6
	}
	if w.Water[s.tile()] > 0.25 {
		v *= 0.5
	}
	return v
}

// walk moves a settler towards a goal and reports arrival. key identifies
// the goal so the path is reused between ticks.
func (w *World) walk(s *Settler, key int, isGoal func(int) bool, tx, ty float64, goal int32) bool {
	cur := s.tile()
	if isGoal(cur) && len(s.path) == 0 {
		s.pathKey = key
		return true
	}
	if s.pathKey != key || s.path == nil {
		s.path = w.FindPath(cur, isGoal, tx, ty, goal)
		s.pathKey = key
		if s.path == nil {
			s.noPath += 1
			s.pathKey = 0
			return false
		}
		s.noPath = 0
	}
	if len(s.path) == 0 {
		return true
	}
	next := s.path[0]
	nx, ny := XY(next)
	px, py := float64(nx)+0.5, float64(ny)+0.5
	dx, dy := px-s.X, py-s.Y
	d := math.Hypot(dx, dy)
	step := w.speed(s) * DT
	if d <= step {
		s.X, s.Y = px, py
		s.path = s.path[1:]
		if len(s.path) == 0 {
			return isGoal(next)
		}
	} else {
		s.X += dx / d * step
		s.Y += dy / d * step
	}
	return false
}

func (w *World) walkToBuilding(s *Settler, b *Building) bool {
	cx, cy := b.Center()
	return w.walk(s, int(b.ID), func(i int) bool { x, y := XY(i); return b.Covers(x, y) }, cx, cy, b.ID)
}

func (w *World) walkToTile(s *Settler, t int) bool {
	x, y := XY(t)
	return w.walk(s, -(t + 1), func(i int) bool { return i == t }, float64(x)+0.5, float64(y)+0.5, 0)
}

// --- life -----------------------------------------------------------------------

func (w *World) updateSettlers() {
	for _, s := range w.Settlers {
		w.stepSettler(s)
	}
	k := 0
	for _, s := range w.Settlers {
		switch {
		case s.Hunger >= StarveDeath:
			w.Deaths++
			w.notice("Житель помер від голоду. Потрібна їжа!")
		case s.Thirst >= ThirstDeath:
			w.Deaths++
			w.notice("Житель помер від спраги. Потрібні насоси й запас води!")
		default:
			w.Settlers[k] = s
			k++
			continue
		}
		if s.Carry.N > 0 {
			// what they carried is lost
		}
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
	s.MealIn -= DT
	if s.MealIn <= 0 {
		if it := w.consume(foodOrder(s)...); it != ItemNone {
			s.MealIn, s.Hunger, s.LastFood = MealInterval, 0, it
		} else {
			s.MealIn = 0
			s.Hunger += DT
		}
	}
	s.DrinkIn -= DT
	if s.DrinkIn <= 0 {
		if w.consume(Water) != ItemNone {
			s.DrinkIn, s.Thirst = DrinkInterval, 0
		} else {
			s.DrinkIn = 0
			s.Thirst += DT
		}
	}
	// smooth standing height
	s.Z += (float64(w.WalkHeight(s.tile())) - s.Z) * 0.15

	switch s.Task {
	case TaskHaul:
		w.stepHaul(s)
		return
	case TaskBuild:
		site := w.Buildings[s.To]
		if site == nil || !site.Site || len(site.Need) > 0 || s.noPath > 3 {
			s.Task = TaskNone
			return
		}
		if w.walkToBuilding(s, site) {
			rate := 1.0
			if s.Suffering() {
				rate = 0.5
			}
			site.Work += float32(DT * rate / site.Kind.BuildTime())
			if site.Work >= 1 {
				site.Site, site.Work = false, 0
				s.Task = TaskNone
				if site.Kind == KDam {
					w.notice("Гребля збудована — вода підніметься.")
				}
			}
		}
		return
	}

	if s.Job != 0 {
		jb := w.Buildings[s.Job]
		if jb == nil {
			s.Job, s.Inside, s.Field = 0, false, 0
			return
		}
		if jb.Kind.IsField() && w.stepField(s, jb) {
			return
		}
		if s.Inside {
			s.X, s.Y = jb.Center()
			return
		}
		if w.walkToBuilding(s, jb) {
			s.Inside = true
		} else if s.noPath > 3 {
			s.Job, s.noPath = 0, 0 // can't reach the workplace
		}
		return
	}

	// idle: stroll around home
	home := w.Buildings[s.Home]
	if home == nil {
		home = w.hall()
	}
	if home == nil {
		return
	}
	hx, hy := home.Center()
	t := float64(w.Tick)/TPS*0.2 + float64(s.Seed)
	r := float64(home.Size())/2 + 1.3
	tx, ty := hx+math.Cos(t)*r, hy+math.Sin(t*1.3)*r*0.7
	if In(int(tx), int(ty)) && w.passable(Idx(int(tx), int(ty)), 0) {
		dx, dy := tx-s.X, ty-s.Y
		if d := math.Hypot(dx, dy); d > 0.05 {
			step := math.Min(d, WalkSpeed*0.3*DT)
			nx, ny := s.X+dx/d*step, s.Y+dy/d*step
			ni := Idx(clampi(int(nx), 0, Size-1), clampi(int(ny), 0, Size-1))
			if w.passable(ni, 0) && absi(w.WalkHeight(ni)-w.WalkHeight(s.tile())) <= 1 {
				s.X, s.Y = nx, ny
			}
		}
	}
}

func absi(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func foodOrder(s *Settler) []Item {
	if s.LastFood == Berries {
		return []Item{Crop, Berries}
	}
	return []Item{Berries, Crop}
}

func (w *World) hall() *Building {
	for _, id := range w.Order() {
		if b := w.Buildings[id]; b.Kind == KHall {
			return b
		}
	}
	return nil
}

func (w *World) updateBirths() {
	pop, cap := len(w.Settlers), w.Housing()
	if pop >= cap || w.Available(Water) < 5 || w.Available(Berries)+w.Available(Crop) < 5 {
		return
	}
	for _, s := range w.Settlers {
		if s.Suffering() {
			return
		}
	}
	res := map[int32]int{}
	for _, s := range w.Settlers {
		res[s.Home]++
	}
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Site || b.Def().Housing == 0 || res[b.ID] < 2 {
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

// Roles counts settlers by what they do.
func (w *World) Roles() (free, working, suffering int) {
	for _, s := range w.Settlers {
		if s.Job == 0 {
			free++
		} else {
			working++
		}
		if s.Suffering() {
			suffering++
		}
	}
	return
}

// --- hauling ---------------------------------------------------------------------

func (w *World) stepHaul(s *Settler) {
	if s.noPath > 3 {
		s.Task, s.noPath = TaskNone, 0
		if s.Carry.N > 0 {
			w.redirect(s)
		}
		return
	}
	if s.Phase == 0 {
		from := w.Buildings[s.From]
		if from == nil {
			s.Task = TaskNone
			return
		}
		if !w.walkToBuilding(s, from) {
			return
		}
		got := w.pickup(from, s.Item, s.N)
		if got == 0 {
			s.Task = TaskNone
			return
		}
		s.Carry, s.N, s.Phase = Stack{s.Item, got}, got, 1
		return
	}
	to := w.Buildings[s.To]
	if to == nil {
		w.redirect(s)
		return
	}
	if !w.walkToBuilding(s, to) {
		return
	}
	if left := w.deliver(to, s.Carry); left > 0 {
		s.Carry.N, s.N = left, left
		w.redirect(s)
		return
	}
	s.Carry, s.Task = Stack{}, TaskNone
}

// redirect sends a carried load to the nearest storage with room, or keeps
// it in hand until there is room.
func (w *World) redirect(s *Settler) {
	if s.Carry.N == 0 {
		s.Task = TaskNone
		return
	}
	st := w.nearestStorage(s.X, s.Y, s.Carry.Item, 1)
	if st == nil {
		s.Task = TaskNone
		if w.Tick >= w.fullWarned {
			w.fullWarned = w.Tick + 60*TPS
			w.notice("Склади заповнені — будуйте нові склади й баки для води.")
		}
		return
	}
	s.Task, s.Phase, s.To, s.Item, s.N = TaskHaul, 1, st.ID, s.Carry.Item, s.Carry.N
}

func (w *World) pickup(b *Building, it Item, n int) int {
	switch {
	case b.Kind.IsStorage():
		return b.take(it, n)
	case b.Out == it:
		n = min(n, b.OutN)
		b.OutN -= n
		return n
	}
	return 0
}

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
	case b.Kind.IsStorage() && b.Kind.Accepts(st.Item):
		d := max(0, min(n, b.Def().Cap-b.Used()))
		b.put(st.Item, d)
		return n - d
	case b.Kind == KSawmill && st.Item == Log:
		b.InN += n
		return 0
	}
	return n
}

// --- planning -----------------------------------------------------------------------

func dist(ax, ay, bx, by float64) float64 { return math.Hypot(ax-bx, ay-by) }

func (w *World) bookkeeping() {
	for _, b := range w.Buildings {
		b.staff, b.inside, b.builder, b.pend, b.res = 0, 0, 0, nil, nil
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
			if s.Inside || s.Field != 0 {
				jb.inside++
			}
		}
		switch s.Task {
		case TaskHaul:
			if to := w.Buildings[s.To]; to != nil {
				add(&to.pend, s.Item, s.N)
			}
			if from := w.Buildings[s.From]; from != nil && s.Phase == 0 {
				add(&from.res, s.Item, s.N)
			}
		case TaskBuild:
			if b := w.Buildings[s.To]; b != nil {
				b.builder = s.ID
			}
		}
	}
}

func jobPriority(k Kind) int {
	switch k {
	case KPump:
		return 0
	case KGatherer, KFarm:
		return 1
	case KLumber:
		return 2
	}
	return 3
}

func (w *World) allocate() {
	w.bookkeeping()

	// release workers from paused, unfinished, gone or overstaffed buildings
	for _, s := range w.Settlers {
		if s.Job == 0 {
			continue
		}
		jb := w.Buildings[s.Job]
		if jb == nil || jb.Paused || jb.Site || jb.staff > jb.Def().Workers {
			if jb != nil {
				jb.staff--
			}
			s.Job, s.Inside = 0, false
			if s.Field != 0 {
				s.Field = 0
			}
		}
	}

	// homes
	res := map[int32]int{}
	for _, s := range w.Settlers {
		if hb := w.Buildings[s.Home]; hb != nil && !hb.Site && hb.Def().Housing > 0 {
			res[s.Home]++
		} else {
			s.Home = 0
		}
	}
	for _, s := range w.Settlers {
		if s.Home != 0 {
			continue
		}
		for _, id := range w.Order() {
			if b := w.Buildings[id]; !b.Site && res[id] < b.Def().Housing {
				s.Home = id
				res[id]++
				break
			}
		}
	}

	// settlers holding goods with nowhere to put them try again
	for _, s := range w.Settlers {
		if s.Task == TaskNone && s.Carry.N > 0 && s.Field == 0 {
			w.redirect(s)
		}
	}

	var idle []*Settler
	free := 0
	for _, s := range w.Settlers {
		if s.Job == 0 {
			free++
			if s.idle() {
				idle = append(idle, s)
			}
		}
	}
	take := func(x, y float64) *Settler {
		bi, bd := -1, math.MaxFloat64
		for i, s := range idle {
			if d := dist(s.X, s.Y, x, y); d < bd {
				bi, bd = i, d
			}
		}
		if bi < 0 {
			return nil
		}
		s := idle[bi]
		idle = append(idle[:bi], idle[bi+1:]...)
		return s
	}

	// jobs, keeping a few hands free for building and hauling
	keep := len(w.Settlers) / 6
	if len(w.Settlers) >= 3 {
		keep = max(keep, 1)
	}
	var jobs []*Building
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Def().Workers > 0 && !b.Site && !b.Paused && b.staff < b.Def().Workers {
			jobs = append(jobs, b)
		}
	}
	sort.SliceStable(jobs, func(a, b int) bool { return jobPriority(jobs[a].Kind) < jobPriority(jobs[b].Kind) })
	for _, b := range jobs {
		for b.staff < b.Def().Workers && free > keep {
			cx, cy := b.Center()
			s := take(cx, cy)
			if s == nil {
				break
			}
			s.Job, s.Inside = b.ID, false
			b.staff++
			free--
		}
	}

	// builders
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Site && len(b.Need) == 0 && b.builder == 0 {
			cx, cy := b.Center()
			s := take(cx, cy)
			if s == nil {
				break
			}
			s.Task, s.To, s.noPath = TaskBuild, b.ID, 0
			b.builder = s.ID
		}
	}
	if len(idle) == 0 {
		return
	}

	avail := func(b *Building, it Item) int {
		var n int
		switch {
		case b.Site:
			return 0
		case b.Kind.IsStorage():
			n = b.Has(it)
		case b.Out == it:
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
	assign := func(s *Settler, from, to *Building, it Item, n int) {
		s.Task, s.Phase, s.From, s.To, s.Item, s.N, s.noPath = TaskHaul, 0, from.ID, to.ID, it, n, 0
		reserve(&from.res, it, n)
		reserve(&to.pend, it, n)
	}
	source := func(it Item, near *Building) *Building {
		var best *Building
		bd := math.MaxFloat64
		cx, cy := near.Center()
		for _, id := range w.Order() {
			b := w.Buildings[id]
			if b == near || avail(b, it) <= 0 {
				continue
			}
			bx, by := b.Center()
			d := dist(bx, by, cx, cy)
			if !b.Kind.IsStorage() {
				d -= 4 // prefer producers
			}
			if d < bd {
				best, bd = b, d
			}
		}
		return best
	}

	// demands: construction sites first, then sawmills
	type demand struct {
		to   *Building
		item Item
		n    int
	}
	var ds []demand
	for _, id := range w.Order() {
		b := w.Buildings[id]
		switch {
		case b.Site:
			for _, n := range b.Need {
				if left := n.N - b.pend[n.Item]; left > 0 {
					ds = append(ds, demand{b, n.Item, left})
				}
			}
		}
	}
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if b.Kind == KSawmill && !b.Site && !b.Paused {
			if want := 4 - b.InN - b.pend[Log]; want > 0 {
				ds = append(ds, demand{b, Log, want})
			}
		}
	}
	for _, d := range ds {
		for d.n > 0 && len(idle) > 0 {
			from := source(d.item, d.to)
			if from == nil {
				break
			}
			fx, fy := from.Center()
			s := take(fx, fy)
			if s == nil {
				break
			}
			n := min(d.n, avail(from, d.item), CarryN)
			assign(s, from, d.to, d.item, n)
			d.n -= n
		}
	}

	// producers' goods go to the nearest storage
	for _, id := range w.Order() {
		if len(idle) == 0 {
			return
		}
		b := w.Buildings[id]
		if b.Kind.IsStorage() || b.Site || b.OutN == 0 {
			continue
		}
		n := avail(b, b.Out)
		if n <= 0 || (n < CarryN && b.OutN < OutCap-1) {
			continue
		}
		cx, cy := b.Center()
		st := w.nearestStorage(cx, cy, b.Out, 1)
		if st == nil {
			if w.Tick >= w.fullWarned {
				w.fullWarned = w.Tick + 60*TPS
				w.notice("Склади заповнені — будуйте нові склади й баки для води.")
			}
			continue
		}
		s := take(cx, cy)
		if s == nil {
			return
		}
		assign(s, b, st, b.Out, min(n, CarryN))
	}
}
