package sim

import (
	"bytes"
	"testing"
)

func emptyWorld() *World {
	w := &World{Seed: 1, Buildings: map[int32]*Building{}}
	w.initCaches()
	for i := range w.Stock {
		w.Stock[i] = 100000
	}
	return w
}

func mustPlace(t *testing.T, w *World, k Kind, p Pos, d Dir) *Building {
	t.Helper()
	b, err := w.Place(k, p, d, 0)
	if err != nil {
		t.Fatalf("place %v at %v: %v", Builds[k].Name, p, err)
	}
	return b
}

func run(w *World, seconds float64) {
	for i := 0; i < int(seconds*TPS); i++ {
		w.Update(nil)
	}
}

// A chest feeding a long belt into the core should reach full belt throughput.
func TestBeltThroughput(t *testing.T) {
	w := emptyWorld()
	core := w.add(&Building{Kind: KCore, Pos: Pos{20, -1}, Recipe: -1})
	_ = core
	src := mustPlace(t, w, KChest, Pos{0, 0}, East)
	src.Store = []Stack{{IronPlate, 1000}}
	for x := 1; x < 20; x++ {
		mustPlace(t, w, KBelt, Pos{x, 0}, East)
	}
	run(w, 15) // warm up
	before := w.Stock[IronPlate]
	run(w, 20)
	got := float64(w.Stock[IronPlate]-before) / 20
	want := BeltSpeed / BeltSpacing
	if got < want*0.95 {
		t.Fatalf("throughput %.2f items/s, want ~%.2f", got, want)
	}
}

// Belts turning corners and side-loading must keep items moving.
func TestCornerAndSideload(t *testing.T) {
	w := emptyWorld()
	w.add(&Building{Kind: KCore, Pos: Pos{5, 5}, Recipe: -1})
	a := mustPlace(t, w, KChest, Pos{0, 0}, East)
	a.Store = []Stack{{Coal, 50}}
	mustPlace(t, w, KBelt, Pos{1, 0}, East)
	mustPlace(t, w, KBelt, Pos{2, 0}, East)
	corner := mustPlace(t, w, KBelt, Pos{3, 0}, South)
	for y := 1; y < 5; y++ {
		mustPlace(t, w, KBelt, Pos{3, y}, South)
	}
	mustPlace(t, w, KBelt, Pos{3, 5}, East)
	mustPlace(t, w, KBelt, Pos{4, 5}, East)
	// side feeder into the vertical line
	bch := mustPlace(t, w, KChest, Pos{1, 3}, East)
	bch.Store = []Stack{{Stone, 50}}
	mustPlace(t, w, KBelt, Pos{2, 3}, East)
	if corner.Curve != 1 {
		t.Fatalf("corner curve = %d, want 1", corner.Curve)
	}
	run(w, 40)
	if w.Stock[Coal] != 100050 || w.Stock[Stone] != 100050 {
		t.Fatalf("delivered coal=%d stone=%d", w.Stock[Coal]-100000, w.Stock[Stone]-100000)
	}
}

// Miner -> belt -> furnace -> belt -> core produces plates.
func TestSmeltingChain(t *testing.T) {
	w := NewWorld(42)
	for i := range w.Stock {
		w.Stock[i] = 1000
	}
	// find an iron patch spot
	var spot Pos
	found := false
	for y := -20; y < 20 && !found; y++ {
		for x := -25; x < 25 && !found; x++ {
			if it, n := w.minerOre(Pos{x, y}); it == IronOre && n == 4 && x > 2 {
				spot, found = Pos{x, y}, true
			}
		}
	}
	if !found {
		t.Fatal("no iron near start")
	}
	m := mustPlace(t, w, KMiner, spot, West)
	_ = m
	// belt from miner west to a furnace, furnace outputs west to belt into core
	x := spot.X - 1
	y := spot.Y
	fur := Pos{x - 2, y}
	for cx := x; cx > fur.X+1; cx-- {
		mustPlace(t, w, KBelt, Pos{cx, y}, West)
	}
	f := mustPlace(t, w, KFurnace, fur, West)
	f.Dir = West
	// route output from furnace to the core at (-1..1,-1..1)
	p := Pos{fur.X - 1, y}
	for p.X > 2 {
		mustPlace(t, w, KBelt, p, West)
		p.X--
	}
	// go vertical toward y=0 then into core
	for p.Y != 0 {
		d := South
		if p.Y > 0 {
			d = North
		}
		mustPlace(t, w, KBelt, p, d)
		p = p.Step(d)
	}
	mustPlace(t, w, KBelt, p, West)
	before := w.Stock[IronPlate]
	run(w, 60)
	got := w.Stock[IronPlate] - before
	if got < 25 {
		t.Fatalf("only %d iron plates reached the core", got)
	}
}

func TestPlacementRules(t *testing.T) {
	w := NewWorld(3)
	if _, err := w.Place(KFurnace, Pos{0, 0}, East, 0); err != ErrOccupied {
		t.Fatalf("placing on core: %v", err)
	}
	if _, err := w.Place(KMiner, Pos{0, 4}, East, 0); err != ErrNoOre {
		t.Fatalf("miner off ore: %v", err)
	}
	w.Stock[IronPlate] = 0
	if _, err := w.Place(KBelt, Pos{5, 5}, East, 0); err != ErrNoFunds {
		t.Fatalf("no funds: %v", err)
	}
	w.Stock[IronPlate] = 1
	b := mustPlace(t, w, KBelt, Pos{5, 5}, East)
	if w.Stock[IronPlate] != 0 {
		t.Fatal("belt not paid")
	}
	if _, err := w.Place(KBelt, Pos{5, 5}, North, 0); err != nil || b.Dir != North {
		t.Fatalf("reorient belt: %v", err)
	}
	if err := w.Remove(b.ID); err != nil || w.Stock[IronPlate] != 1 {
		t.Fatal("refund failed")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	w := NewWorld(9)
	mustPlace(t, w, KBelt, Pos{4, 4}, South)
	mustPlace(t, w, KChest, Pos{4, 6}, East).Store = []Stack{{Gear, 7}}
	var buf bytes.Buffer
	if err := w.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	w2, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(w2.Buildings) != len(w.Buildings) || w2.Stock != w.Stock {
		t.Fatal("mismatch after load")
	}
	if c := w2.At(Pos{4, 6}); c == nil || c.StoreCount() != 7 {
		t.Fatal("chest lost")
	}
	if w2.OreAt(Pos{13, -5}) != IronOre {
		t.Fatal("ore not regenerated from seed")
	}
}
