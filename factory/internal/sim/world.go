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

	grid   map[Pos]int32
	chunks map[Pos]*[chunkSize * chunkSize]Item
	order  []int32
	dirty  bool
	retry  []*Building
}

// NewWorld creates a fresh world with a core at the origin and a starter stock.
func NewWorld(seed int64) *World {
	w := &World{Seed: seed, Buildings: map[int32]*Building{}}
	w.initCaches()
	w.add(&Building{Kind: KCore, Pos: Pos{-1, -1}, Recipe: -1})
	w.Stock[IronPlate] = 140
	w.Stock[Gear] = 30
	w.Stock[Stone] = 60
	w.Stock[Wire] = 24
	w.Stock[CopperPlate] = 20
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
)

// CanAfford checks the core stock for a building cost.
func (w *World) CanAfford(k Kind) bool {
	for _, c := range Builds[k].Cost {
		if w.Stock[c.Item] < c.N {
			return false
		}
	}
	return true
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
	if k == KMiner {
		if it, _ := w.minerOre(p); it == ItemNone {
			return ErrNoOre
		}
	}
	if !w.CanAfford(k) {
		return ErrNoFunds
	}
	return nil
}

func (w *World) minerOre(p Pos) (Item, int) {
	var cnt [ItemCount]int
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
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

// Place builds a structure, paying its cost from the core stock.
func (w *World) Place(k Kind, p Pos, d Dir, owner uint8) (*Building, error) {
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
	for _, c := range Builds[k].Cost {
		w.Stock[c.Item] -= c.N
	}
	b := &Building{Kind: k, Pos: p, Dir: d, Owner: owner, Recipe: -1}
	switch k {
	case KMiner:
		b.MinerItem, b.MinerTiles = w.minerOre(p)
	case KFurnace:
		b.AutoRecipe = true
	}
	w.add(b)
	w.refreshAround(p)
	return b, nil
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
	for _, c := range Builds[b.Kind].Cost {
		w.Stock[c.Item] += c.N
	}
	for _, it := range b.Belt {
		w.Stock[it.Item]++
	}
	for _, s := range b.InBuf {
		w.Stock[s.Item] += s.N
	}
	for _, s := range b.Store {
		w.Stock[s.Item] += s.N
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
	if b == nil || b.Recipe == r {
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
