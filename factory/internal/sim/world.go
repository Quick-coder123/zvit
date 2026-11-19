package sim

import (
	"errors"
	"sort"
)

// BeltItem is an item riding a belt. P runs 0 (entry) .. 1 (exit).
type BeltItem struct {
	Item Item
	P    float32
}

// Building is any placed structure. Fields are exported for gob saves.
type Building struct {
	ID    int32
	Kind  Kind
	Pos   Pos // top-left tile
	Dir   Dir
	Owner uint8

	// Belt: items sorted by P descending (front first).
	Belt  []BeltItem
	Curve int8 // 0 straight, -1 fed from left, +1 fed from right

	// Machines
	Recipe     int // -1 none
	AutoRecipe bool
	Progress   float32
	Working    bool
	InBuf      []Stack
	OutItem    Item
	OutN       int
	OutRR      int
	MinerItem  Item
	MinerTiles int

	// Chest
	Store []Stack

	// Logistics
	Filter Item      // sorter
	Exit   bool      // tunnel: this end is the exit
	Buf    []BufItem // routers (splitter, merger, sorter)

	// Combat
	Dmg       float32 // damage taken (0 = full health)
	LastHit   uint64  // tick of the last damage
	Ammo      int32   // turret shots or flamer fuel
	Aim       float32 // turret barrel angle
	Cool      float32 // weapon cooldown
	FreeBuilt bool    // built with a starter token; refunds the token
}

// BufItem is an item waiting inside a router, remembering where it came from.
type BufItem struct {
	Item Item
	From Dir // travel direction it arrived with
}

func (b *Building) Size() int { return Builds[b.Kind].Size }

// Covers reports whether the building occupies tile p.
func (b *Building) Covers(p Pos) bool {
	s := b.Size()
	return p.X >= b.Pos.X && p.Y >= b.Pos.Y && p.X < b.Pos.X+s && p.Y < b.Pos.Y+s
}

// FrontTiles are the tiles directly in front of the output face.
func (b *Building) FrontTiles() []Pos {
	s := b.Size()
	out := make([]Pos, 0, s)
	for i := 0; i < s; i++ {
		var p Pos
		switch b.Dir {
		case East:
			p = Pos{b.Pos.X + s, b.Pos.Y + i}
		case West:
			p = Pos{b.Pos.X - 1, b.Pos.Y + i}
		case South:
			p = Pos{b.Pos.X + i, b.Pos.Y + s}
		case North:
			p = Pos{b.Pos.X + i, b.Pos.Y - 1}
		}
		out = append(out, p)
	}
	return out
}

// StoreCount is the total number of items held in a chest.
func (b *Building) StoreCount() int {
	n := 0
	for _, s := range b.Store {
		n += s.N
	}
	return n
}

type World struct {
	Seed      int64
	Tick      uint64
	NextID    int32
	Buildings map[int32]*Building
	Stock     [ItemCount]int

	Free       [KindCount]int  // starter buildings that cost nothing
	Researched map[string]bool // finished research
	Enemies    []*Enemy
	Wave       int     // waves spawned so far
	NextWave   float64 // seconds until the next wave
	WaveAngle  float64 // direction the next wave comes from (radians)
	Peaceful   bool    // no waves
	Ruins      []Spec  // destroyed buildings that can be rebuilt
	Kills      int

	Shots   []Shot   // transient weapon effects (not saved)
	Notices []string // messages for the UI (not saved)

	grid   map[Pos]int32
	chunks map[Pos]*[chunkSize * chunkSize]Item
	order  []int32
	dirty  bool
	retry  []*Building
}

// NewWorld creates a fresh world with a core at the origin and a starter stock.
func NewWorld(seed int64) *World {
	w := &World{Seed: seed, Buildings: map[int32]*Building{}, Researched: map[string]bool{}}
	w.initCaches()
	w.add(&Building{Kind: KCore, Pos: Pos{-1, -1}, Recipe: -1})
	w.Free[KMiner] = 1
	w.Free[KBelt] = 10
	w.NextWave = FirstWaveDelay
	w.WaveAngle = w.rollWaveAngle()
	return w
}

func (w *World) initCaches() {
	w.grid = map[Pos]int32{}
	w.chunks = map[Pos]*[chunkSize * chunkSize]Item{}
	w.dirty = true
	for _, b := range w.Buildings {
		w.mark(b, b.ID)
	}
}

func (w *World) mark(b *Building, id int32) {
	s := b.Size()
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			p := b.Pos.Add(x, y)
			if id == 0 {
				delete(w.grid, p)
			} else {
				w.grid[p] = id
			}
		}
	}
}

func (w *World) add(b *Building) *Building {
	w.NextID++
	b.ID = w.NextID
	w.Buildings[b.ID] = b
	w.mark(b, b.ID)
	w.dirty = true
	return b
}

// At returns the building covering tile p, or nil.
func (w *World) At(p Pos) *Building {
	if id, ok := w.grid[p]; ok {
		return w.Buildings[id]
	}
	return nil
}

// Order returns building IDs in a stable order.
func (w *World) Order() []int32 {
	if w.dirty {
		w.order = w.order[:0]
		for id := range w.Buildings {
			w.order = append(w.order, id)
		}
		sort.Slice(w.order, func(i, j int) bool { return w.order[i] < w.order[j] })
		w.dirty = false
	}
	return w.order
}

var (
	ErrOccupied = errors.New("місце зайняте")
	ErrNoOre    = errors.New("бур треба ставити на поклад")
	ErrNoFunds  = errors.New("бракує ресурсів у ядрі")
	ErrCore     = errors.New("ядро не можна знести")
	ErrLocked   = errors.New("спершу відкрий це в дослідженнях (T)")
)

// CanAfford checks starter tokens and the core stock for a building cost.
func (w *World) CanAfford(k Kind) bool {
	return w.Free[k] > 0 || w.affordable(Builds[k].Cost)
}

// CheckPlace validates a placement without changing anything.
func (w *World) CheckPlace(k Kind, p Pos, d Dir) error {
	s := Builds[k].Size
	if k == KBelt {
		if b := w.At(p); b != nil {
			if b.Kind == KBelt {
				return nil // re-orienting an existing belt is free
			}
			return ErrOccupied
		}
	} else {
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				if w.At(p.Add(x, y)) != nil {
					return ErrOccupied
				}
			}
		}
	}
	if !w.Unlocked(k) {
		return ErrLocked
	}
	if k.Base() == KMiner {
		if it, _ := w.minerOre(p, s); it == ItemNone {
			return ErrNoOre
		}
	}
	if !w.CanAfford(k) {
		return ErrNoFunds
	}
	return nil
}

func (w *World) minerOre(p Pos, size int) (Item, int) {
	var cnt [ItemCount]int
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cnt[w.OreAt(p.Add(x, y))]++
		}
	}
	best, bn := ItemNone, 0
	for it := Item(1); it < ItemCount; it++ {
		if cnt[it] > bn {
			best, bn = it, cnt[it]
		}
	}
	return best, bn
}

// Spec describes a building to place, including copied settings.
type Spec struct {
	Kind    Kind
	Pos     Pos
	Dir     Dir
	Recipe  int  // -1: none / automatic
	Filter  Item // sorter filter
	Exit    bool // tunnel end; only used when ExitSet
	ExitSet bool
}

// Place builds a structure, paying its cost from the core stock.
func (w *World) Place(k Kind, p Pos, d Dir, owner uint8) (*Building, error) {
	return w.PlaceSpec(Spec{Kind: k, Pos: p, Dir: d, Recipe: -1}, owner)
}

// PlaceSpec builds a structure from a full spec (used by copy/paste).
func (w *World) PlaceSpec(sp Spec, owner uint8) (*Building, error) {
	k, p, d := sp.Kind, sp.Pos, sp.Dir
	if k == KBelt {
		if b := w.At(p); b != nil && b.Kind == KBelt {
			if b.Dir != d {
				b.Dir = d
				w.refreshAround(p)
			}
			return b, nil
		}
	}
	if err := w.CheckPlace(k, p, d); err != nil {
		return nil, err
	}
	b := &Building{Kind: k, Pos: p, Dir: d, Owner: owner, Recipe: -1}
	if w.Free[k] > 0 {
		w.Free[k]--
		b.FreeBuilt = true
	} else {
		for _, c := range Builds[k].Cost {
			w.Stock[c.Item] -= c.N
		}
	}
	switch k {
	case KMiner, KMinerBig:
		b.MinerItem, b.MinerTiles = w.minerOre(p, b.Size())
	case KFurnace, KFurnaceBig:
		b.AutoRecipe = true
	case KTunnel:
		if sp.ExitSet {
			b.Exit = sp.Exit
		} else {
			b.Exit = w.TunnelWouldExit(p, d)
		}
	case KSorter:
		b.Filter = sp.Filter
	}
	w.add(b)
	if sp.Recipe >= 0 && sp.Recipe < len(Recipes) && Recipes[sp.Recipe].Machine == k.Base() && w.RecipeUnlocked(sp.Recipe) {
		w.SetRecipe(b.ID, sp.Recipe)
	}
	w.refreshAround(p)
	return b, nil
}

// SpecOf captures a building's settings for copying.
func SpecOf(b *Building) Spec {
	return Spec{Kind: b.Kind, Pos: b.Pos, Dir: b.Dir, Recipe: b.Recipe, Filter: b.Filter, Exit: b.Exit, ExitSet: b.Kind == KTunnel}
}

// TunnelWouldExit decides whether a new tunnel at p becomes the exit of an
// unpaired entrance behind it.
func (w *World) TunnelWouldExit(p Pos, d Dir) bool {
	back := d.Opp()
	q := p
	for k := 1; k <= TunnelMax+1; k++ {
		q = q.Step(back)
		tb := w.At(q)
		if tb != nil && tb.Kind == KTunnel && tb.Dir == d {
			if tb.Exit {
				return false
			}
			ex, _ := w.TunnelExit(tb)
			return ex == nil
		}
	}
	return false
}

// TunnelExit finds the exit paired with entrance b and the distance to it.
func (w *World) TunnelExit(b *Building) (*Building, int) {
	if b.Kind != KTunnel || b.Exit {
		return nil, 0
	}
	q := b.Pos
	for k := 1; k <= TunnelMax+1; k++ {
		q = q.Step(b.Dir)
		tb := w.At(q)
		if tb != nil && tb.Kind == KTunnel && tb.Dir == b.Dir {
			if tb.Exit {
				return tb, k
			}
			return nil, 0
		}
	}
	return nil, 0
}

// SetFilter sets a sorter's filter item.
func (w *World) SetFilter(id int32, it Item) {
	if b := w.Buildings[id]; b != nil && b.Kind == KSorter {
		b.Filter = it
	}
}

// Remove deletes a building and refunds its cost and contents to the core.
func (w *World) Remove(id int32) error {
	b := w.Buildings[id]
	if b == nil {
		return nil
	}
	if b.Kind == KCore {
		return ErrCore
	}
	if b.FreeBuilt {
		w.Free[b.Kind]++
	} else {
		for _, c := range Builds[b.Kind].Cost {
			w.Stock[c.Item] += c.N
		}
	}
	w.Stock[Ammo] += int(b.Ammo) / AmmoShots * boolInt(b.Kind.Base() == KTurret)
	w.Stock[Coal] += int(b.Ammo) / CoalFuel * boolInt(b.Kind == KFlamer)
	for _, it := range b.Belt {
		w.Stock[it.Item]++
	}
	for _, s := range b.InBuf {
		w.Stock[s.Item] += s.N
	}
	for _, s := range b.Store {
		w.Stock[s.Item] += s.N
	}
	for _, bi := range b.Buf {
		w.Stock[bi.Item]++
	}
	w.Stock[b.OutItem] += b.OutN
	w.Stock[ItemNone] = 0
	w.mark(b, 0)
	delete(w.Buildings, id)
	w.dirty = true
	w.refreshAround(b.Pos)
	return nil
}

// Rotate turns a building clockwise in place.
func (w *World) Rotate(id int32) {
	if b := w.Buildings[id]; b != nil && b.Kind != KCore {
		b.Dir = b.Dir.CW()
		w.refreshAround(b.Pos)
	}
}

// SetRecipe selects a recipe for a machine, returning buffered inputs to the core.
func (w *World) SetRecipe(id int32, r int) {
	b := w.Buildings[id]
	if b == nil || b.Recipe == r || (r >= 0 && !w.RecipeUnlocked(r)) {
		return
	}
	for _, s := range b.InBuf {
		w.Stock[s.Item] += s.N
	}
	b.InBuf = nil
	b.Working = false
	b.Progress = 0
	b.Recipe = r
	b.AutoRecipe = false
	if r >= 0 {
		for _, in := range Recipes[r].In {
			b.InBuf = append(b.InBuf, Stack{in.Item, 0})
		}
	}
}

// feeds reports whether the building at q is a belt pointing into tile p.
func (w *World) feeds(q, p Pos) bool {
	b := w.At(q)
	return b != nil && b.Kind == KBelt && q.Step(b.Dir) == p
}

// refreshAround recomputes belt curve flags near a changed tile.
func (w *World) refreshAround(p Pos) {
	for y := -2; y <= 4; y++ {
		for x := -2; x <= 4; x++ {
			if b := w.At(p.Add(x, y)); b != nil && b.Kind == KBelt {
				w.refreshCurve(b)
			}
		}
	}
}

func (w *World) refreshCurve(b *Building) {
	p := b.Pos
	b.Curve = 0
	if w.feeds(p.Step(b.Dir.Opp()), p) {
		return
	}
	left := w.feeds(p.Step(b.Dir.CCW()), p)
	right := w.feeds(p.Step(b.Dir.CW()), p)
	if left && !right {
		b.Curve = -1
	} else if right && !left {
		b.Curve = 1
	}
}

// beltPrev returns the belt feeding b as part of the same line (straight or
// curve), or nil.
func (w *World) beltPrev(b *Building) *Building {
	p := b.Pos
	var q Pos
	switch b.Curve {
	case 0:
		q = p.Step(b.Dir.Opp())
	case -1:
		q = p.Step(b.Dir.CCW())
	default:
		q = p.Step(b.Dir.CW())
	}
	if w.feeds(q, p) {
		return w.At(q)
	}
	return nil
}

func (w *World) beltNext(b *Building) *Building {
	n := w.At(b.Pos.Step(b.Dir))
	if n != nil && n.Kind == KBelt && w.beltPrev(n) == b {
		return n
	}
	return nil
}

// BeltLine returns the connected belt line through b in flow order.
func (w *World) BeltLine(b *Building) (line []*Building, ring bool) {
	if b == nil || b.Kind != KBelt {
		return nil, false
	}
	start := b
	seen := map[int32]bool{b.ID: true}
	for {
		p := w.beltPrev(start)
		if p == nil {
			break
		}
		if seen[p.ID] {
			ring = p == b || ring
			break
		}
		seen[p.ID] = true
		start = p
	}
	if ring {
		start = b
	}
	seen = map[int32]bool{}
	for c := start; c != nil && !seen[c.ID]; c = w.beltNext(c) {
		seen[c.ID] = true
		line = append(line, c)
	}
	return line, ring
}

// ReverseLine flips the direction of the whole belt line through b.
func (w *World) ReverseLine(b *Building) int {
	line, ring := w.BeltLine(b)
	if len(line) == 0 {
		return 0
	}
	dirs := make([]Dir, len(line))
	for i, c := range line {
		switch {
		case i > 0:
			dirs[i], _ = DirBetween(c.Pos, line[i-1].Pos)
		case ring:
			dirs[i], _ = DirBetween(c.Pos, line[len(line)-1].Pos)
		default:
			dirs[i] = c.Dir.Opp()
		}
	}
	for i, c := range line {
		c.Dir = dirs[i]
		n := len(c.Belt)
		for j := 0; j < n/2; j++ {
			c.Belt[j], c.Belt[n-1-j] = c.Belt[n-1-j], c.Belt[j]
		}
		for j := range c.Belt {
			c.Belt[j].P = 1 - c.Belt[j].P
		}
	}
	for _, c := range line {
		w.refreshAround(c.Pos)
	}
	return len(line)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// MinerRate is a miner's output in items per second (before research bonuses).
func MinerRate(b *Building) float64 {
	n := b.Size() * b.Size()
	if b.MinerTiles == 0 || n == 0 {
		return 0
	}
	return SpeedOf(b.Kind) * float64(b.MinerTiles) / float64(n) / MinerTime
}
