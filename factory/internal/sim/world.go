package sim

import (
	"errors"
	"sort"
)

// Building is any placed structure. Fields are exported for gob saves.
type Building struct {
	ID    int32
	Kind  Kind
	Pos   Pos // top-left tile
	Dir   Dir
	Owner uint8

	// Construction: a placed building is a site until materials are delivered
	// and a builder has worked on it.
	Site      bool
	Need      []Stack // materials still to be delivered to the site
	BuildWork float32 // 0..1
	Paused    bool    // player switched it off; its workers are freed

	// Machines
	Recipe     int // -1 none
	AutoRecipe bool
	Progress   float32
	Working    bool
	InBuf      []Stack
	OutItem    Item
	OutN       int
	MinerItem  Item
	MinerTiles int

	// Storehouse contents
	Store []Stack

	// Combat
	Dmg       float32 // damage taken (0 = full health)
	LastHit   uint64  // tick of the last damage
	Ammo      int32   // turret shots or flamer fuel
	Aim       float32 // turret barrel angle
	Cool      float32 // weapon cooldown
	FreeBuilt bool    // built with a starter token; refunds the token
	GateOpen  bool    // floodgate lets water through

	// transient, rebuilt every allocation pass / on load
	staff   int          // settlers employed here
	inside  int          // employed settlers currently inside
	pend    map[Item]int // items on their way here
	res     map[Item]int // items promised to haulers (outputs, storage)
	builder int32        // settler building this site
	hungry  bool         // a worker inside is starving
	flooded bool         // under water
}

func (b *Building) Size() int { return Builds[b.Kind].Size }

// Covers reports whether the building occupies tile p.
func (b *Building) Covers(p Pos) bool {
	s := b.Size()
	return p.X >= b.Pos.X && p.Y >= b.Pos.Y && p.X < b.Pos.X+s && p.Y < b.Pos.Y+s
}

// Staff and Inside report employment (for the UI).
func (b *Building) Staff() int  { return b.staff }
func (b *Building) Inside() int { return b.inside }

// Running reports whether the building is complete, switched on, staffed
// and not under water.
func (b *Building) Running() bool {
	return !b.Site && !b.Paused && !b.flooded && b.inside >= Builds[b.Kind].Workers
}

// IsFlooded reports whether water stopped the building (for the UI).
func (b *Building) IsFlooded() bool { return b.flooded }

// StoreCount is the total number of items held in a storehouse.
func (b *Building) StoreCount() int {
	n := 0
	for _, s := range b.Store {
		n += s.N
	}
	return n
}

func (b *Building) storeHas(it Item) int {
	for _, s := range b.Store {
		if s.Item == it {
			return s.N
		}
	}
	return 0
}

func (b *Building) storeAdd(it Item, n int) {
	for i := range b.Store {
		if b.Store[i].Item == it {
			b.Store[i].N += n
			return
		}
	}
	b.Store = append(b.Store, Stack{it, n})
}

func (b *Building) storeTake(it Item, n int) int {
	for i := range b.Store {
		if b.Store[i].Item == it {
			if n > b.Store[i].N {
				n = b.Store[i].N
			}
			b.Store[i].N -= n
			if b.Store[i].N == 0 {
				b.Store = append(b.Store[:i], b.Store[i+1:]...)
			}
			return n
		}
	}
	return 0
}

type World struct {
	Seed      int64
	Tick      uint64
	NextID    int32
	Buildings map[int32]*Building
	Stock     [ItemCount]int // contents of the town hall (unlimited)

	Settlers    []*Settler
	NextSettler int32
	WanderTimer float32

	Free       [KindCount]int  // starter buildings that cost nothing
	Researched map[string]bool // finished research

	// Vegetation changes on top of the generated map.
	Planted map[Pos]uint64 // tick a sapling was planted
	Cleared map[Pos]bool   // natural trees/bushes removed
	Picked  map[Pos]uint64 // tick a bush was last picked

	// Water and seasons
	Water      []float32 // depth per tile of the water region
	Drought    bool
	Polluted   bool      // muddy-water season
	Cont       []float32 // contamination of the water per tile (0..1)
	SeasonLeft float64   // seconds left in the current season
	Cycle      int       // completed wet+drought cycles

	ResearchQueue []string // techs to research; the first is active
	ResearchNeed  []Stack  // materials still to deliver for the active one
	ResearchWork  float32  // 0..1 once materials are in
	Enemies       []*Enemy
	Wave          int     // waves spawned so far
	NextWave      float64 // seconds until the next wave
	WaveAngle     float64 // direction the next wave comes from (radians)
	Peaceful      bool    // no waves
	Ruins         []Spec  // destroyed buildings that can be rebuilt
	Kills         int
	Deaths        int

	Shots   []Shot   // transient weapon effects (not saved)
	Notices []string // messages for the UI (not saved)

	grid       map[Pos]int32
	chunks     map[Pos]*[chunkSize * chunkSize]Item
	fullWarned uint64
	wellbeing  float64
	needShare  [NeedCount]float64

	height     []uint8
	wall       []float32
	moist      []bool
	springs    []int
	sinks      []int
	flux       []float32 // 4 outflows per water cell
	massBuf    []float32
	wallsDirty bool
	vegChunks  map[Pos]*[vegChunkTiles]Veg
	order      []int32
	dirty      bool
}

// NewWorld creates a fresh settlement: a town hall, four settlers and a few
// starter supplies.
func NewWorld(seed int64) *World {
	w := &World{Seed: seed, Buildings: map[int32]*Building{}, Researched: map[string]bool{}}
	w.initCaches()
	core := w.add(&Building{Kind: KCore, Pos: Pos{-1, -1}, Recipe: -1})
	w.Free[KMiner] = 1
	w.Free[KFarm] = 1
	w.Free[KHouse] = 1
	w.Free[KLab] = 1
	w.Free[KLumber] = 1
	w.Free[KPump] = 1
	w.Free[KCampfire] = 1
	w.Stock[Wood] = 30
	w.Stock[Water] = 40
	w.SeasonLeft = w.WetLength(0)
	w.Stock[Food] = 40
	w.Stock[Stone] = 30
	w.Stock[IronOre] = 20
	for i := 0; i < 4; i++ {
		s := w.spawnSettler(core)
		s.X += float64(i%2) - 0.5
		s.Y += 2.2
	}
	w.NextWave = FirstWaveDelay
	w.WaveAngle = w.rollWaveAngle()
	w.Peaceful = true // enemies are optional; switch them on in the menu
	return w
}

func (w *World) initCaches() {
	w.grid = map[Pos]int32{}
	w.chunks = map[Pos]*[chunkSize * chunkSize]Item{}
	w.vegChunks = map[Pos]*[vegChunkTiles]Veg{}
	if w.Planted == nil {
		w.Planted = map[Pos]uint64{}
	}
	if w.Cleared == nil {
		w.Cleared = map[Pos]bool{}
	}
	if w.Picked == nil {
		w.Picked = map[Pos]uint64{}
	}
	w.dirty = true
	for _, b := range w.Buildings {
		w.mark(b, b.ID)
	}
	w.initWater()
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
	w.wallsDirty = true
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

// --- stock across all storages ------------------------------------------------

// TotalStock counts an item in the town hall and all storehouses.
func (w *World) TotalStock(it Item) int {
	n := w.Stock[it]
	for _, b := range w.Buildings {
		if b.Kind == KChest && !b.Site {
			n += b.storeHas(it)
		}
	}
	return n
}

// TakeStock removes items from storage (town hall first). It reports whether
// the full amount was available; nothing is taken otherwise.
func (w *World) TakeStock(it Item, n int) bool {
	if w.TotalStock(it) < n {
		return false
	}
	t := min(n, w.Stock[it])
	w.Stock[it] -= t
	n -= t
	for _, id := range w.Order() {
		if n == 0 {
			break
		}
		if b := w.Buildings[id]; b.Kind == KChest && !b.Site {
			n -= b.storeTake(it, n)
		}
	}
	return true
}

func (w *World) affordable(cost []Stack) bool {
	for _, c := range cost {
		if w.TotalStock(c.Item) < c.N {
			return false
		}
	}
	return true
}

// StorageCap is how many items a storage building holds, including research.
func (w *World) StorageCap(b *Building) int {
	c := 0
	switch b.Kind {
	case KCore:
		c = CoreCap
	case KChest:
		c = ChestCap
	}
	if w.Researched["chest"] {
		c *= 2
	}
	return c
}

// StorageUsed is how many items a storage building currently holds.
func (w *World) StorageUsed(b *Building) int {
	if b.Kind == KCore {
		n := 0
		for _, v := range w.Stock {
			n += v
		}
		return n
	}
	return b.StoreCount()
}

// StorageTotals sums used and total space over all finished storages.
func (w *World) StorageTotals() (used, total int) {
	for _, b := range w.Buildings {
		if b.Kind.IsStorage() && !b.Site {
			used += w.StorageUsed(b)
			total += w.StorageCap(b)
		}
	}
	return
}

// --- placing and removing --------------------------------------------------------

var (
	ErrOccupied = errors.New("місце зайняте")
	ErrNoOre    = errors.New("бур треба ставити на поклад")
	ErrOnOre    = errors.New("ферму треба ставити на чисту землю")
	ErrNoFunds  = errors.New("бракує ресурсів")
	ErrCore     = errors.New("ратушу не можна знести")
	ErrLocked   = errors.New("спершу відкрий це в дослідженнях (T)")
	ErrLegacy   = errors.New("цієї будівлі більше немає")
	ErrWater    = errors.New("тут вода — будувати можна лише греблі, насипи й насоси")
	ErrNoWater  = errors.New("насос треба ставити біля води")
	ErrDry      = errors.New("ферма потребує вологого ґрунту біля води")
)

// CanAfford reports whether the materials for a building exist in storage
// (or a starter token is left). Building is allowed either way: the site
// simply waits for materials.
func (w *World) CanAfford(k Kind) bool {
	return w.Free[k] > 0 || w.affordable(Builds[k].Cost)
}

// CheckPlace validates a placement without changing anything.
func (w *World) CheckPlace(k Kind, p Pos, d Dir) error {
	if k.Legacy() || k == KCore || k == KNone || k >= KindCount {
		return ErrLegacy
	}
	s := Builds[k].Size
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if w.At(p.Add(x, y)) != nil {
				return ErrOccupied
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
	if k == KFarm && w.OreAt(p) != ItemNone {
		return ErrOnOre
	}
	if k != KPump && k != KPurifier && waterBlock(k) == 0 {
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				if w.WaterAt(p.Add(x, y)) > BuildMaxWet {
					return ErrWater
				}
			}
		}
	}
	if (k == KPump || k == KPurifier) && !w.nearWater(p, s) {
		return ErrNoWater
	}
	if k == KFarm && !w.MoistAt(p) {
		return ErrDry
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
	Kind   Kind
	Pos    Pos
	Dir    Dir
	Recipe int // -1: none / automatic
}

// Place puts down a construction site.
func (w *World) Place(k Kind, p Pos, d Dir, owner uint8) (*Building, error) {
	return w.PlaceSpec(Spec{Kind: k, Pos: p, Dir: d, Recipe: -1}, owner)
}

// PlaceSpec puts down a construction site from a full spec (copy/paste).
func (w *World) PlaceSpec(sp Spec, owner uint8) (*Building, error) {
	k, p, d := sp.Kind, sp.Pos, sp.Dir
	if err := w.CheckPlace(k, p, d); err != nil {
		return nil, err
	}
	b := &Building{Kind: k, Pos: p, Dir: d, Owner: owner, Recipe: -1, Site: true}
	if w.Free[k] > 0 {
		w.Free[k]--
		b.FreeBuilt = true
	} else {
		b.Need = append([]Stack(nil), Builds[k].Cost...)
	}
	switch k.Base() {
	case KMiner:
		b.MinerItem, b.MinerTiles = w.minerOre(p, b.Size())
	case KFurnace:
		b.AutoRecipe = true
	}
	w.add(b)
	if k == KSawmill {
		w.SetRecipe(b.ID, sawmillRecipe)
	}
	if k == KBakery {
		w.SetRecipe(b.ID, bakeryRecipe)
	}
	for y := 0; y < b.Size(); y++ {
		for x := 0; x < b.Size(); x++ {
			w.clearVeg(p.Add(x, y))
		}
	}
	if sp.Recipe >= 0 && sp.Recipe < len(Recipes) && Recipes[sp.Recipe].Machine == k.Base() && w.RecipeUnlocked(sp.Recipe) {
		w.SetRecipe(b.ID, sp.Recipe)
	}
	return b, nil
}

// SpecOf captures a building's settings for copying.
func SpecOf(b *Building) Spec {
	return Spec{Kind: b.Kind, Pos: b.Pos, Dir: b.Dir, Recipe: b.Recipe}
}

// Remove deletes a building and refunds its cost and contents to the town hall.
func (w *World) Remove(id int32) error {
	b := w.Buildings[id]
	if b == nil {
		return nil
	}
	if b.Kind == KCore {
		return ErrCore
	}
	switch {
	case b.FreeBuilt:
		w.Free[b.Kind]++
	case b.Site:
		// refund what was already delivered
		for _, c := range Builds[b.Kind].Cost {
			left := 0
			for _, n := range b.Need {
				if n.Item == c.Item {
					left = n.N
				}
			}
			w.Stock[c.Item] += c.N - left
		}
	default:
		for _, c := range Builds[b.Kind].Cost {
			w.Stock[c.Item] += c.N
		}
	}
	w.dropContents(b)
	w.unplace(b)
	return nil
}

// dropContents moves everything a building holds into the town hall.
func (w *World) dropContents(b *Building) {
	switch b.Kind.Base() {
	case KTurret:
		w.Stock[Ammo] += int(b.Ammo) / AmmoShots
	case KFlamer:
		w.Stock[Coal] += int(b.Ammo) / CoalFuel
	}
	for _, s := range b.InBuf {
		w.Stock[s.Item] += s.N
	}
	for _, s := range b.Store {
		w.Stock[s.Item] += s.N
	}
	w.Stock[b.OutItem] += b.OutN
	w.Stock[ItemNone] = 0
}

// unplace takes a building off the map and releases the people tied to it.
func (w *World) unplace(b *Building) {
	w.mark(b, 0)
	delete(w.Buildings, b.ID)
	w.dirty = true
	w.wallsDirty = true
	for _, s := range w.Settlers {
		if s.Job == b.ID {
			s.Job, s.Inside = 0, false
		}
		if s.Home == b.ID {
			s.Home = 0
		}
	}
}

// Rotate turns a building clockwise in place (cosmetic).
func (w *World) Rotate(id int32) {
	if b := w.Buildings[id]; b != nil && b.Kind != KCore {
		b.Dir = b.Dir.CW()
	}
}

// SetPaused switches a building off (freeing its workers) or back on.
func (w *World) SetPaused(id int32, paused bool) {
	if b := w.Buildings[id]; b != nil && b.Kind != KCore {
		b.Paused = paused
		if paused {
			for _, s := range w.Settlers {
				if s.Job == b.ID {
					s.Job, s.Inside = 0, false
				}
			}
			b.staff, b.inside = 0, 0
		}
	}
}

// SetRecipe selects a recipe for a machine, returning buffered inputs to the town hall.
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

var bakeryRecipe = func() int {
	for i, r := range Recipes {
		if r.Machine == KBakery {
			return i
		}
	}
	return -1
}()

// sawmillRecipe is the only recipe of a sawmill (wood -> planks).
var sawmillRecipe = func() int {
	for i, r := range Recipes {
		if r.Machine == KSawmill {
			return i
		}
	}
	return -1
}()

// SetGate opens or closes a floodgate.
func (w *World) SetGate(id int32, open bool) {
	if b := w.Buildings[id]; b != nil && b.Kind == KGate {
		b.GateOpen = open
		w.wallsDirty = true
	}
}
