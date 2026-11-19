package world

import (
	"bytes"
	"testing"
)

// flatRun finds n tiles in a row (x..x+n-1, y) of equal dry ground where
// everything can be built, plus a free margin around them.
func flatRun(t *testing.T, w *World, n int) (int, int) {
	t.Helper()
	h := w.hall()
	for r := 4; r < 40; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				x, y := h.X+dx, h.Y+dy
				ok := true
				for k := -1; k <= n && ok; k++ {
					for e := -1; e <= 1 && ok; e++ {
						if w.CheckPlace(KHouse, x+k, y+e) != nil || w.Height[Idx(x+k, y+e)] != w.Height[Idx(x, y)] {
							ok = false
						}
					}
				}
				if ok {
					return x, y
				}
			}
		}
	}
	t.Fatal("no flat run")
	return 0, 0
}

func mustPlace(t *testing.T, w *World, k Kind, x, y int) *Building {
	t.Helper()
	b, err := w.Place(k, x, y)
	if err != nil {
		t.Fatalf("place %s at %d,%d: %v", Builds[k].Name, x, y, err)
	}
	return b
}

func TestPlatformHouse(t *testing.T) {
	w := New(7)
	x, y := flatRun(t, w, 1)
	i := Idx(x, y)
	h0 := w.WalkHeight(i)
	p := mustPlace(t, w, KPlatform, x, y)
	if w.WalkHeight(i) != h0 {
		t.Fatal("a platform site must not raise the ground yet")
	}
	w.finishSite(p)
	if w.WalkHeight(i) != h0+1 {
		t.Fatalf("finished platform: walk height %d, want %d", w.WalkHeight(i), h0+1)
	}
	house := mustPlace(t, w, KHouse, x, y)
	if house.Site == false || w.BuildingAt(x, y) != house || w.SupportAt(x, y) != p {
		t.Fatal("house should stand on the platform")
	}
	// settlers must bring materials up and build it
	run(w, 120)
	if house.Site {
		t.Fatalf("house on a platform was not built (still needs %v)", house.Need)
	}
}

func TestPlatformStack(t *testing.T) {
	w := New(7)
	x, y := flatRun(t, w, 1)
	i := Idx(x, y)
	h0 := int(w.Height[i])
	p := mustPlace(t, w, KPlatform, x, y)
	w.finishSite(p)
	for lv := 2; lv <= 3; lv++ {
		pl, err := w.PlanPlacement(KPlatform, x, y)
		if err != nil || !pl.Upgrade || pl.Levels != lv {
			t.Fatalf("upgrade to %d: %+v %v", lv, pl, err)
		}
		b := mustPlace(t, w, KPlatform, x, y)
		if b != p || !p.Site || w.Surface(i) != h0+lv-1 {
			t.Fatalf("upgrade should reuse the platform and keep the old height while building")
		}
		w.finishSite(p)
		if w.Surface(i) != h0+lv {
			t.Fatalf("surface %d after level %d", w.Surface(i), lv)
		}
	}
	if _, err := w.Place(KPlatform, x, y); err != ErrTooHigh {
		t.Fatalf("4th level: %v", err)
	}
	for _, k := range []Kind{KFarm, KPump, KDam} {
		if err := w.CheckPlace(k, x, y); err != ErrGroundOnly {
			t.Errorf("%s on a platform: %v", Builds[k].Name, err)
		}
	}
	w.finishSite(mustPlace(t, w, KHouse, x, y))
	p.Levels = 2 // pretend it is lower to test the top-building rule
	if _, err := w.Place(KPlatform, x, y); err == nil {
		t.Fatal("platform upgraded under a house")
	}
}

func TestStairsPath(t *testing.T) {
	w := New(7)
	x, y := flatRun(t, w, 3)
	start, stair, top := Idx(x, y), Idx(x+1, y), Idx(x+2, y)
	p := mustPlace(t, w, KPlatform, x+2, y)
	w.finishSite(p)
	mustPlace(t, w, KPlatform, x+2, y)
	w.finishSite(p)
	goal := func(i int) bool { return i == top }
	if w.FindPath(start, goal, float64(x)+2.5, float64(y)+0.5, 0) != nil {
		t.Fatal("reached a 2-level platform without stairs")
	}
	if w.CanStep(stair, top, 0) {
		t.Fatal("CanStep allows a 2-level climb without stairs")
	}
	pl, err := w.PlanPlacement(KStairs, x+1, y)
	if err != nil || pl.Dir != 0 || pl.Levels != 2 {
		t.Fatalf("stairs preview %+v %v", pl, err)
	}
	st := mustPlace(t, w, KStairs, x+1, y)
	if st.Dir != pl.Dir || st.Levels != pl.Levels {
		t.Fatal("placed stairs differ from the preview")
	}
	if w.FindPath(start, goal, float64(x)+2.5, float64(y)+0.5, 0) != nil {
		t.Fatal("unfinished stairs already work")
	}
	w.finishSite(st)
	path := w.FindPath(start, goal, float64(x)+2.5, float64(y)+0.5, 0)
	if path == nil {
		t.Fatal("no way up the stairs")
	}
	if !w.CanStep(stair, top, 0) || !w.CanStep(top, stair, 0) {
		t.Fatal("stairs should work both ways")
	}
	// stairs never allow a diagonal climb
	if w.CanStep(Idx(x+1, y+1), top, 0) {
		t.Fatal("diagonal climb allowed")
	}
}

func TestStairRemovalReplans(t *testing.T) {
	w := New(7)
	x, y := flatRun(t, w, 6)
	top := Idx(x+5, y)
	p := mustPlace(t, w, KPlatform, x+5, y)
	w.finishSite(p)
	mustPlace(t, w, KPlatform, x+5, y)
	w.finishSite(p)
	st := mustPlace(t, w, KStairs, x+4, y)
	w.finishSite(st)
	s := w.spawnSettler(w.hall())
	s.X, s.Y = float64(x)+0.5, float64(y)+0.5
	removed := false
	prev := s.tile()
	for k := 0; k < 60*20; k++ {
		if !removed && s.tile() == Idx(x+2, y) {
			if err := w.Remove(st.ID); err != nil {
				t.Fatal(err)
			}
			removed = true
		}
		w.walkToTile(s, top)
		cur := s.tile()
		if cur != prev {
			if absi(w.WalkHeight(cur)-w.WalkHeight(prev)) > 1 {
				t.Fatalf("settler jumped %d levels after the stairs were removed", w.WalkHeight(cur)-w.WalkHeight(prev))
			}
			prev = cur
		}
		if cur == top {
			t.Fatal("settler reached the platform without stairs")
		}
	}
	if !removed {
		t.Fatal("settler never walked towards the stairs")
	}
}

// oldEdge is the A* edge rule before vertical building.
func oldEdge(w *World, i, j int) bool {
	oldPass := func(k int) bool {
		if w.Water[k] > 0.6 {
			if b := w.Owner(k); b == nil || b.Kind != KDam || b.Site {
				return false
			}
		}
		if b := w.Owner(k); b != nil && !b.Site && b.Kind != KDam && b.Kind != KFarm {
			return false
		}
		return true
	}
	x, y := XY(i)
	nx, ny := XY(j)
	if !oldPass(j) {
		return false
	}
	hc, hj := int(w.Height[i]), int(w.Height[j])
	if absi(hj-hc) > 1 {
		return false
	}
	if nx != x && ny != y {
		a, b := Idx(nx, y), Idx(x, ny)
		return oldPass(a) && oldPass(b) && int(w.Height[a]) == hc && int(w.Height[b]) == hc && hj == hc
	}
	return true
}

func TestCanStepMatchesOldRule(t *testing.T) {
	w := New(7)
	for i := 0; i < Size*Size; i++ {
		x, y := XY(i)
		for _, d := range dirs8 {
			if !In(x+d[0], y+d[1]) {
				continue
			}
			j := Idx(x+d[0], y+d[1])
			if got, want := w.CanStep(i, j, 0), oldEdge(w, i, j); got != want {
				t.Fatalf("(%d,%d)->(%d,%d): CanStep %v, old rule %v", x, y, x+d[0], y+d[1], got, want)
			}
		}
	}
}

func totalWater(w *World) float64 {
	t := 0.0
	for _, v := range w.Water {
		t += float64(v)
	}
	return t
}

func TestBridgeCross(t *testing.T) {
	w := New(7)
	plain := New(7)
	h := w.hall()
	y := h.Y
	east := h.X - 1
	for east > 0 && w.passable(Idx(east, y), 0) {
		east--
	}
	bank := Idx(east+1, y)
	far := -1
	for x := east; x > 0; x-- {
		if w.passable(Idx(x, y), 0) && w.Height[Idx(x, y)] >= 1 {
			far = Idx(x, y)
			break
		}
	}
	if far < 0 {
		t.Fatal("no far bank")
	}
	fx, fy := XY(far)
	goal := func(i int) bool { return i == far }
	// the river has fords elsewhere, so compare with the detour
	before := w.FindPath(bank, goal, float64(fx)+0.5, float64(fy)+0.5, 0)
	var bridges []*Building
	for x := east; Idx(x, y) != far; x-- {
		pl, err := w.PlanPlacement(KBridge, x, y)
		if err != nil {
			t.Fatalf("bridge at %d: %v", x, err)
		}
		b := mustPlace(t, w, KBridge, x, y)
		if b.Levels != pl.Levels || b.Levels < 1 {
			t.Fatalf("bridge levels %d, preview %d", b.Levels, pl.Levels)
		}
		bridges = append(bridges, b)
	}
	for _, b := range bridges {
		if i := Idx(b.X, b.Y); w.Height[i] == 0 && w.passable(i, 0) {
			t.Fatal("deep river tile walkable before the bridge is finished")
		}
		w.finishSite(b)
	}
	for _, b := range bridges {
		if !w.passable(Idx(b.X, b.Y), 0) {
			t.Fatal("finished bridge not walkable")
		}
	}
	after := w.FindPath(bank, goal, float64(fx)+0.5, float64(fy)+0.5, 0)
	if after == nil || len(after) > len(bridges)+1 {
		t.Fatalf("bridge does not give a direct crossing: %d steps for %d bridges", len(after), len(bridges))
	}
	if before != nil && len(before) <= len(after) {
		t.Fatalf("no shorter way with the bridge: %d vs %d", len(after), len(before))
	}
	for k := 0; k < 600; k++ {
		w.StepWater()
		plain.StepWater()
	}
	a, b := totalWater(w), totalWater(plain)
	if d := (a - b) / b; d > 0.01 || d < -0.01 {
		t.Fatalf("bridges changed the water volume by %.2f%%", d*100)
	}
}

func TestDemolishSupport(t *testing.T) {
	w := New(7)
	x, y := flatRun(t, w, 1)
	p := mustPlace(t, w, KPlatform, x, y)
	w.finishSite(p)
	house := mustPlace(t, w, KHouse, x, y)
	w.finishSite(house)
	w.hall().Store = []Stack{{Log, 10}}
	if err := w.Remove(p.ID); err != ErrTopFirst {
		t.Fatalf("removed a platform under a house: %v", err)
	}
	if err := w.Remove(house.ID); err != nil {
		t.Fatal(err)
	}
	planks := w.Total(Plank)
	if err := w.Remove(p.ID); err != nil {
		t.Fatal(err)
	}
	if w.Total(Plank) != planks+3 || w.SupportAt(x, y) != nil {
		t.Fatalf("platform refund: planks %d -> %d", planks, w.Total(Plank))
	}
}

func TestFieldWorkSkipsSupports(t *testing.T) {
	w := New(7)
	BuildDemo(w, 240)
	var fo *Building
	for _, id := range w.Order() {
		if b := w.Buildings[id]; b.Kind == KForester && !b.Site {
			fo = b
		}
	}
	if fo == nil {
		t.Fatal("no forester in the demo")
	}
	for k := 0; k < 5; k++ {
		tgt := w.fieldTarget(fo, nil)
		if tgt < 0 {
			return
		}
		x, y := XY(tgt)
		var b *Building
		var err error
		if b, err = w.Place(KPlatform, x, y); err != nil {
			if b, err = w.Place(KBridge, x, y); err != nil {
				continue
			}
		}
		w.finishSite(b)
		if w.fieldTarget(fo, nil) == tgt {
			t.Fatalf("forester still targets a tile under a %s", b.Def().Name)
		}
	}
}

func TestVerticalSaveRoundTrip(t *testing.T) {
	w := New(7)
	BuildVerticalDemo(w)
	var buf bytes.Buffer
	if err := w.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	w2, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for id, b := range w.Buildings {
		b2 := w2.Buildings[id]
		if b2 == nil || b2.Kind != b.Kind || b2.Levels != b.Levels || b2.Dir != b.Dir || b2.Site != b.Site {
			t.Fatalf("building %d differs after load", id)
		}
		if b.Kind.IsSupport() {
			n++
			if w2.SupportAt(b.X, b.Y) != b2 || w2.Surface(Idx(b.X, b.Y)) != w.Surface(Idx(b.X, b.Y)) {
				t.Fatal("support layer not restored")
			}
		} else if w2.BuildingAt(b.X, b.Y) != b2 {
			t.Fatal("top layer not restored")
		}
	}
	if n < 5 {
		t.Fatalf("demo built only %d supports", n)
	}
}

// Dams keep their original rule: they go across the river (level 0, in
// water) but never on a platform.
func TestDamPlacementUnchanged(t *testing.T) {
	w := New(7)
	river := -1
	for i := 0; i < Size*Size && river < 0; i++ {
		if w.Height[i] == 0 && w.Water[i] > 0.6 && w.owner[i] == 0 {
			river = i
		}
	}
	x, y := XY(river)
	if err := w.CheckPlace(KDam, x, y); err != nil {
		t.Fatalf("dam across the river refused: %v", err)
	}
	px, py := flatRun(t, w, 1)
	w.finishSite(mustPlace(t, w, KPlatform, px, py))
	if err := w.CheckPlace(KDam, px, py); err != ErrGroundOnly {
		t.Fatalf("dam on a platform: %v", err)
	}
}

func TestPlanPlacementBadKind(t *testing.T) {
	w := New(7)
	for _, k := range []Kind{KNone, KindCount, 255, KHall} {
		if _, err := w.PlanPlacement(k, 10, 10); err != ErrBlocked {
			t.Errorf("kind %d: %v", k, err)
		}
	}
}
