package sim

import "testing"

func core(w *World, x, y int) *Building {
	return w.add(&Building{Kind: KCore, Pos: Pos{x, y}, Recipe: -1})
}

func belts(t *testing.T, w *World, from Pos, d Dir, n int) Pos {
	t.Helper()
	p := from
	for i := 0; i < n; i++ {
		mustPlace(t, w, KBelt, p, d)
		p = p.Step(d)
	}
	return p
}

// Splitter feeding two chests should split evenly.
func TestSplitterEven(t *testing.T) {
	w := emptyWorld()
	src := mustPlace(t, w, KChest, Pos{0, 0}, East)
	src.Store = []Stack{{IronPlate, 100}}
	belts(t, w, Pos{1, 0}, East, 2)
	mustPlace(t, w, KSplitter, Pos{3, 0}, East)
	// left of East is North
	belts(t, w, Pos{3, -1}, North, 2)
	up := mustPlace(t, w, KChest, Pos{3, -3}, North)
	belts(t, w, Pos{3, 1}, South, 2)
	down := mustPlace(t, w, KChest, Pos{3, 3}, South)
	// chests push out the front; block that by having nothing there
	run(w, 30)
	a, b := up.StoreCount(), down.StoreCount()
	if a+b != 100 || a < 45 || b < 45 {
		t.Fatalf("split %d / %d", a, b)
	}
}

// Merger should take fairly from two saturated inputs.
func TestMergerFair(t *testing.T) {
	w := emptyWorld()
	core(w, 6, -1)
	l := mustPlace(t, w, KChest, Pos{3, -3}, South)
	l.Store = []Stack{{Coal, 1000}}
	mustPlace(t, w, KBelt, Pos{3, -2}, South)
	mustPlace(t, w, KBelt, Pos{3, -1}, South)
	r := mustPlace(t, w, KChest, Pos{0, 0}, East)
	r.Store = []Stack{{Stone, 1000}}
	belts(t, w, Pos{1, 0}, East, 2)
	mustPlace(t, w, KMerger, Pos{3, 0}, East)
	belts(t, w, Pos{4, 0}, East, 2)
	run(w, 20)
	c, s := w.Stock[Coal]-100000, w.Stock[Stone]-100000
	if c < 20 || s < 20 || c-s > 3 || s-c > 3 {
		t.Fatalf("merge coal=%d stone=%d", c, s)
	}
	if c+s < int(20*BeltSpeed/BeltSpacing*0.8) {
		t.Fatalf("merger throughput too low: %d", c+s)
	}
}

// Tunnel carries items under a crossing belt; pairs automatically.
func TestTunnel(t *testing.T) {
	w := emptyWorld()
	core(w, 10, -1)
	src := mustPlace(t, w, KChest, Pos{0, 0}, East)
	src.Store = []Stack{{Gear, 40}}
	mustPlace(t, w, KBelt, Pos{1, 0}, East)
	in := mustPlace(t, w, KTunnel, Pos{2, 0}, East)
	// a crossing belt and a machine in the way
	for y := -3; y <= 3; y++ {
		mustPlace(t, w, KBelt, Pos{4, y}, South)
	}
	out := mustPlace(t, w, KTunnel, Pos{7, 0}, East)
	if in.Exit || !out.Exit {
		t.Fatalf("pairing: in.Exit=%v out.Exit=%v", in.Exit, out.Exit)
	}
	if ex, k := w.TunnelExit(in); ex != out || k != 5 {
		t.Fatalf("exit lookup k=%d", k)
	}
	belts(t, w, Pos{8, 0}, East, 2)
	before := w.Stock[Gear]
	run(w, 20)
	if got := w.Stock[Gear] - before; got != 40 {
		t.Fatalf("tunnel delivered %d", got)
	}
	// too far: no pairing
	far := mustPlace(t, w, KTunnel, Pos{20, 5}, East)
	far2 := mustPlace(t, w, KTunnel, Pos{27, 5}, East)
	if far2.Exit || far.Exit {
		t.Fatal("tunnels 7 apart must not pair")
	}
}

func TestSorter(t *testing.T) {
	w := emptyWorld()
	core(w, 6, -1)
	src := mustPlace(t, w, KChest, Pos{0, 0}, East)
	src.Store = []Stack{{Coal, 10}, {Stone, 10}, {Coal, 10}}
	belts(t, w, Pos{1, 0}, East, 2)
	so := mustPlace(t, w, KSorter, Pos{3, 0}, East)
	w.SetFilter(so.ID, Coal)
	belts(t, w, Pos{4, 0}, East, 2) // to core: stone
	belts(t, w, Pos{3, 1}, South, 2)
	side := mustPlace(t, w, KChest, Pos{3, 3}, South)
	run(w, 20)
	if side.StoreCount() != 20 || side.Store[0].Item != Coal {
		t.Fatalf("side got %v", side.Store)
	}
	if w.Stock[Stone]-100000 != 10 {
		t.Fatalf("stone to core %d", w.Stock[Stone]-100000)
	}
}

func TestReverseLine(t *testing.T) {
	w := emptyWorld()
	a := mustPlace(t, w, KBelt, Pos{0, 0}, East)
	b := mustPlace(t, w, KBelt, Pos{1, 0}, South)
	c := mustPlace(t, w, KBelt, Pos{1, 1}, South)
	b.Belt = []BeltItem{{Coal, 0.8}, {Stone, 0.3}}
	if n := w.ReverseLine(b); n != 3 {
		t.Fatalf("reversed %d belts", n)
	}
	if c.Dir != North || b.Dir != West || a.Dir != West {
		t.Fatalf("dirs a=%v b=%v c=%v", a.Dir, b.Dir, c.Dir)
	}
	if b.Belt[0].Item != Stone || b.Belt[0].P != 0.7 {
		t.Fatalf("items %v", b.Belt)
	}
	if b.Curve == 0 {
		t.Fatal("reversed corner should be a curve")
	}
}

func TestPasteSpec(t *testing.T) {
	w := emptyWorld()
	a, err := w.PlaceSpec(Spec{Kind: KAssembler, Pos: Pos{0, 0}, Dir: East, Recipe: 6}, 0)
	if err != nil || a.Recipe != 6 {
		t.Fatalf("recipe not copied: %v", err)
	}
	// exit placed before entrance must stay an exit when ExitSet
	ex, _ := w.PlaceSpec(Spec{Kind: KTunnel, Pos: Pos{8, 0}, Dir: East, Exit: true, ExitSet: true}, 0)
	en, _ := w.PlaceSpec(Spec{Kind: KTunnel, Pos: Pos{5, 0}, Dir: East, ExitSet: true}, 0)
	if got, _ := w.TunnelExit(en); got != ex {
		t.Fatal("pasted tunnel pair not linked")
	}
}
