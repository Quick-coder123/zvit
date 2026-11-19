package world

import (
	"errors"
	"math"
)

// Building is any placed structure. Exported fields are saved.
type Building struct {
	ID       int32
	Kind     Kind
	X, Y     int
	Dir      int
	Site     bool    // under construction
	Need     []Stack // materials still to bring to the site
	Work     float32 // construction progress once materials are in (0..1)
	Paused   bool
	Progress float32 // production progress
	Working  bool
	Levels   int  // platforms: levels raised (1..3); bridges: deck height above ground
	Recipe   int  // reserved for crafters (-1 none)
	InN      int  // input buffer (sawmill logs)
	Out      Item // produced item waiting for pickup
	OutN     int
	Store    []Stack // storage contents

	staff, inside int
	pend, res     map[Item]int
	builder       int32
}

func (b *Building) Size() int      { return Builds[b.Kind].Size }
func (b *Building) Def() *BuildDef { return &Builds[b.Kind] }
func (b *Building) Staff() int     { return b.staff }
func (b *Building) Inside() int    { return b.inside }

// Running reports whether a building is finished, switched on and staffed.
func (b *Building) Running() bool {
	return !b.Site && !b.Paused && b.inside >= Builds[b.Kind].Workers
}

// Center returns the middle of the footprint in tile coordinates.
func (b *Building) Center() (float64, float64) {
	h := float64(b.Size()) / 2
	return float64(b.X) + h, float64(b.Y) + h
}

func (b *Building) Covers(x, y int) bool {
	s := b.Size()
	return x >= b.X && y >= b.Y && x < b.X+s && y < b.Y+s
}

// --- storage ------------------------------------------------------------------

func (b *Building) Has(it Item) int {
	for _, s := range b.Store {
		if s.Item == it {
			return s.N
		}
	}
	return 0
}

func (b *Building) Used() int {
	n := 0
	for _, s := range b.Store {
		n += s.N
	}
	return n
}

func (b *Building) put(it Item, n int) {
	for i := range b.Store {
		if b.Store[i].Item == it {
			b.Store[i].N += n
			return
		}
	}
	b.Store = append(b.Store, Stack{it, n})
}

func (b *Building) take(it Item, n int) int {
	for i := range b.Store {
		if b.Store[i].Item == it {
			n = min(n, b.Store[i].N)
			b.Store[i].N -= n
			if b.Store[i].N == 0 {
				b.Store = append(b.Store[:i], b.Store[i+1:]...)
			}
			return n
		}
	}
	return 0
}

// Total counts an item across finished storage buildings.
func (w *World) Total(it Item) int {
	n := 0
	for _, b := range w.Buildings {
		if b.Def().Cap > 0 && !b.Site {
			n += b.Has(it)
		}
	}
	return n
}

// Available counts an item in storage and waiting at producers.
func (w *World) Available(it Item) int {
	n := w.Total(it)
	for _, b := range w.Buildings {
		if b.Out == it {
			n += b.OutN
		}
	}
	return n
}

// Space returns used and total storage space.
func (w *World) Space() (used, total int) {
	for _, b := range w.Buildings {
		if c := b.Def().Cap; c > 0 && !b.Site {
			used += b.Used()
			total += c
		}
	}
	return
}

// consume takes one unit of the first available item from storage or a
// producer's output (eating, drinking).
func (w *World) consume(items ...Item) Item {
	for _, it := range items {
		for _, id := range w.Order() {
			b := w.Buildings[id]
			if b.Def().Cap > 0 && !b.Site && b.take(it, 1) == 1 {
				return it
			}
		}
	}
	for _, it := range items {
		for _, id := range w.Order() {
			b := w.Buildings[id]
			if b.Out == it && b.OutN > b.res[it] {
				b.OutN--
				return it
			}
		}
	}
	return ItemNone
}

// --- placing ---------------------------------------------------------------------

var (
	ErrBlocked    = errors.New("тут не можна будувати: потрібна рівна суха вільна земля")
	ErrNoWater    = errors.New("насос треба ставити біля води")
	ErrDry        = errors.New("поле потребує вологого ґрунту біля води")
	ErrHall       = errors.New("ратушу не можна знести")
	ErrTooHigh    = errors.New("помост вже найвищий (3 рівні)")
	ErrGroundOnly = errors.New("це ставиться лише на землю, не на поміст")
	ErrStairs     = errors.New("сходи мають вести вгору: поставте біля уступу на 1–2 рівні")
	ErrNoBank     = errors.New("міст має спиратися на берег або інший міст")
	ErrTopFirst   = errors.New("спочатку знесіть те, що стоїть зверху")
)

// Placement is what placing a building at a spot would do.
type Placement struct {
	Dir     int  // stairs: direction they climb (0 +x, 1 -x, 2 +y, 3 -y)
	Levels  int  // platform/bridge: levels above the ground
	Upgrade bool // adds a level to an existing platform
	Base    int  // level the building stands on (supports: ground level)
}

// PlanPlacement validates a placement and works out its automatic choices.
// Place uses the same result, so a preview always matches the build.
func (w *World) PlanPlacement(k Kind, x, y int) (Placement, error) {
	if k <= KNone || k >= KindCount {
		return Placement{}, ErrBlocked
	}
	d := &Builds[k]
	sz := d.Size
	if d.Hidden || !In(x, y) || !In(x+sz-1, y+sz-1) {
		return Placement{}, ErrBlocked
	}
	i := Idx(x, y)
	switch k {
	case KPlatform:
		if w.owner[i] != 0 {
			return Placement{}, ErrBlocked
		}
		if sp := w.Support(i); sp != nil {
			if sp.Kind != KPlatform || sp.Site {
				return Placement{}, ErrBlocked
			}
			if sp.Levels >= MaxStack {
				return Placement{}, ErrTooHigh
			}
			return Placement{Levels: sp.Levels + 1, Upgrade: true, Base: int(w.Height[i])}, nil
		}
		if w.Height[i] == 0 || w.Water[i] > 0.15 {
			return Placement{}, ErrBlocked
		}
		return Placement{Levels: 1, Base: int(w.Height[i])}, nil
	case KBridge:
		if w.owner[i] != 0 || w.support[i] != 0 {
			return Placement{}, ErrBlocked
		}
		target := -1
		for _, dd := range dirs4 {
			nx, ny := x+dd[0], y+dd[1]
			if !In(nx, ny) {
				continue
			}
			j := Idx(nx, ny)
			h := -1
			if sb := w.Support(j); sb != nil && sb.Kind == KBridge {
				h = int(w.Height[j]) + sb.Levels // planned deck, even while building
			} else if w.passable(j, 0) {
				h = w.WalkHeight(j)
			}
			target = max(target, h)
		}
		if target < 0 {
			return Placement{}, ErrNoBank
		}
		lv := max(1, min(MaxStack, target-int(w.Height[i])))
		return Placement{Levels: lv, Base: int(w.Height[i])}, nil
	}

	// top-layer buildings: on level ground or on finished platforms
	base := w.Surface(i)
	elevated := w.support[i] != 0
	for dy := 0; dy < sz; dy++ {
		for dx := 0; dx < sz; dx++ {
			j := Idx(x+dx, y+dy)
			if w.owner[j] != 0 || w.Surface(j) != base {
				return Placement{}, ErrBlocked
			}
			sp := w.Support(j)
			if (sp != nil) != elevated || (sp != nil && (sp.Kind != KPlatform || sp.Site)) {
				return Placement{}, ErrBlocked
			}
			if elevated {
				if float64(w.Height[j])+float64(w.Water[j]) >= float64(base)-0.15 {
					return Placement{}, ErrBlocked
				}
			} else if !d.WaterOK && (w.Water[j] > 0.15 || w.Height[j] == 0) {
				return Placement{}, ErrBlocked
			}
		}
	}
	p := Placement{Base: base}
	switch k {
	case KFarm, KPump, KDam:
		if elevated {
			return Placement{}, ErrGroundOnly
		}
	}
	switch k {
	case KPump:
		if !w.nearWater(x, y, sz) {
			return Placement{}, ErrNoWater
		}
	case KFarm:
		if !w.moist[i] {
			return Placement{}, ErrDry
		}
	case KStairs:
		p.Dir = -1
		for _, rise := range []int{2, 1} {
			for dir, dd := range dirs4 {
				nx, ny := x+dd[0], y+dd[1]
				if p.Dir < 0 && In(nx, ny) && w.passable(Idx(nx, ny), 0) && w.WalkHeight(Idx(nx, ny))-base == rise {
					p.Dir, p.Levels = dir, rise
				}
			}
		}
		if p.Dir < 0 {
			return Placement{}, ErrStairs
		}
	}
	return p, nil
}

// CheckPlace validates a placement.
func (w *World) CheckPlace(k Kind, x, y int) error {
	_, err := w.PlanPlacement(k, x, y)
	return err
}

func (w *World) nearWater(x, y, size int) bool {
	for dy := -1; dy <= size; dy++ {
		for dx := -1; dx <= size; dx++ {
			if In(x+dx, y+dy) && w.Water[Idx(x+dx, y+dy)] >= 0.25 {
				return true
			}
		}
	}
	return false
}

// Place puts down a construction site (or starts raising a platform).
func (w *World) Place(k Kind, x, y int) (*Building, error) {
	p, err := w.PlanPlacement(k, x, y)
	if err != nil {
		return nil, err
	}
	if p.Upgrade {
		sp := w.Support(Idx(x, y))
		sp.Levels, sp.Site, sp.Work = p.Levels, true, 0
		sp.Need = append([]Stack(nil), Builds[k].Cost...)
		w.pathEpoch++
		return sp, nil
	}
	b := &Building{Kind: k, X: x, Y: y, Site: true, Recipe: -1, Dir: p.Dir, Levels: p.Levels}
	b.Need = append([]Stack(nil), Builds[k].Cost...)
	return w.add(b), nil
}

// Remove demolishes a building; materials and contents go to the nearest
// storage (or are lost if there is no room).
func (w *World) Remove(id int32) error {
	b := w.Buildings[id]
	if b == nil {
		return nil
	}
	if b.Kind == KHall {
		return ErrHall
	}
	if b.Kind.IsSupport() {
		for dy := 0; dy < b.Size(); dy++ {
			for dx := 0; dx < b.Size(); dx++ {
				if w.owner[Idx(b.X+dx, b.Y+dy)] != 0 {
					return ErrTopFirst
				}
			}
		}
	}
	refund := map[Item]int{}
	for _, c := range Builds[b.Kind].Cost {
		left := 0
		for _, n := range b.Need {
			if n.Item == c.Item {
				left = n.N
			}
		}
		refund[c.Item] += c.N - left
		if b.Kind == KPlatform && b.Levels > 1 {
			refund[c.Item] += c.N * (b.Levels - 1) // finished lower levels
		}
	}
	for _, s := range b.Store {
		refund[s.Item] += s.N
	}
	refund[b.Out] += b.OutN
	if b.Kind == KSawmill {
		refund[Log] += b.InN
	}
	w.mark(b, 0)
	delete(w.Buildings, id)
	w.dirty = true
	for _, s := range w.Settlers {
		if s.Job == id {
			s.Job, s.Inside, s.Field = 0, false, 0
		}
		if s.Home == id {
			s.Home = 0
		}
	}
	cx, cy := b.Center()
	for it, n := range refund {
		for n > 0 && it != ItemNone {
			st := w.nearestStorage(cx, cy, it, 1)
			if st == nil {
				break
			}
			d := min(n, st.Def().Cap-st.Used())
			st.put(it, d)
			n -= d
		}
	}
	return nil
}

// SetPaused switches a building off, freeing its workers.
func (w *World) SetPaused(id int32, p bool) {
	b := w.Buildings[id]
	if b == nil || b.Def().Workers == 0 {
		return
	}
	b.Paused = p
	if p {
		for _, s := range w.Settlers {
			if s.Job == id {
				s.Job, s.Inside, s.Field = 0, false, 0
			}
		}
	}
}

// storageFree is how much more of an item a storage accepts.
func (w *World) storageFree(b *Building, it Item) int {
	if !b.Kind.IsStorage() || b.Site || !b.Kind.Accepts(it) {
		return 0
	}
	n := b.Def().Cap - b.Used()
	for _, p := range b.pend {
		n -= p
	}
	return n
}

// nearestStorage finds the closest storage with room for n of an item.
func (w *World) nearestStorage(x, y float64, it Item, n int) *Building {
	var best *Building
	bd := math.MaxFloat64
	for _, id := range w.Order() {
		b := w.Buildings[id]
		if w.storageFree(b, it) < n {
			continue
		}
		bx, by := b.Center()
		if d := math.Hypot(bx-x, by-y); d < bd {
			best, bd = b, d
		}
	}
	return best
}

// Housing returns total housing capacity of finished homes.
func (w *World) Housing() int {
	n := 0
	for _, b := range w.Buildings {
		if !b.Site {
			n += b.Def().Housing
		}
	}
	return n
}
