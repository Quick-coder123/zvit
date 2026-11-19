package sim

import (
	"math"
	"testing"
)

func riverStats(w *World, y int) (maxDepth float64, wet int) {
	for x := -40; x <= 10; x++ {
		d := w.WaterAt(Pos{x, y})
		maxDepth = math.Max(maxDepth, d)
		if d > 0.05 {
			wet++
		}
	}
	return
}

func TestRiverFlows(t *testing.T) {
	w := NewWorld(42)
	for _, y := range []int{-60, -20, 0, 20, 40} {
		d, wet := riverStats(w, y)
		t.Logf("row %3d: max depth %.2f, wet tiles %d", y, d, wet)
		if d < 0.3 || wet < 3 || wet > 9 {
			t.Errorf("row %d: river depth %.2f wet %d", y, d, wet)
		}
	}
	if w.WaterAt(Pos{0, 0}) != 0 || w.WaterAt(Pos{10, 0}) != 0 {
		t.Fatal("town must stay dry")
	}
	if !w.MoistAt(Pos{int(riverX(0.5)) + 5, 0}) || w.MoistAt(Pos{5, 0}) {
		t.Fatal("moisture should be next to the river only")
	}
}

func TestDroughtDriesAndDamHolds(t *testing.T) {
	w := NewWorld(42)
	// dam the river across row 10
	for x := -25; x <= -5; x++ {
		p := Pos{x, 10}
		if w.HeightAt(p) <= 1 {
			b := built(t, w, KDam, p)
			_ = b
		}
	}
	w.wallsDirty = true
	run(w, 60) // let water pile up behind the dam
	up, _ := riverStats(w, 5)
	down, _ := riverStats(w, 30)
	t.Logf("wet: upstream %.2f downstream %.2f", up, down)
	w.Drought = true
	w.SeasonLeft = 1e9
	run(w, 180)
	up2, _ := riverStats(w, 5)
	down2, _ := riverStats(w, 30)
	free, _ := riverStats(w, -40)
	t.Logf("after 3 min of drought: upstream %.2f downstream %.2f free-flowing %.2f", up2, down2, free)
	if up2 < 0.3 {
		t.Errorf("dam should keep water through a drought, depth %.2f", up2)
	}
	if down2 > 0.15 || free > 0.2 {
		t.Errorf("undammed river should run dry, downstream %.2f free %.2f", down2, free)
	}
}

func TestSeasonsCycle(t *testing.T) {
	w := NewWorld(1)
	if w.Drought || w.SeasonLeft != 600 {
		t.Fatal("start in the wet season with 10 min")
	}
	w.SeasonLeft = DT / 2
	w.Update(nil)
	if !w.Drought || w.SeasonLeft != DroughtLength(0) {
		t.Fatal("drought should start")
	}
	w.SeasonLeft = DT / 2
	w.Update(nil)
	if w.Drought || w.Cycle != 1 {
		t.Fatal("drought should end")
	}
}

func TestPumpAndThirst(t *testing.T) {
	w := emptyWorld(3)
	w.Stock[Water] = 0
	for _, s := range w.Settlers {
		s.DrinkIn = 1
	}
	run(w, ThirstDeath+5)
	if w.Deaths != 3 {
		t.Fatalf("thirst deaths %d, want 3", w.Deaths)
	}
	w2 := emptyWorld(3)
	w2.Stock[Water] = 0
	if _, err := w2.Place(KPump, Pos{0, 3}, East, 0); err != ErrNoWater {
		t.Fatalf("pump away from water: %v", err)
	}
	built(t, w2, KPump, pumpSpot(w2, Pos{-4, 0}))
	run(w2, 200)
	if w2.Deaths != 0 || w2.WaterAvailable() == 0 {
		t.Fatalf("pump should keep people alive: deaths %d water %d", w2.Deaths, w2.WaterAvailable())
	}
}

func TestFarmNeedsMoisture(t *testing.T) {
	w := emptyWorld(2)
	if _, err := w.Place(KFarm, Pos{4, 4}, East, 0); err != ErrDry {
		t.Fatalf("farm on dry soil: %v", err)
	}
	f := built(t, w, KFarm, moistSpot(w, Pos{-4, 4}))
	run(w, 30)
	if f.OutN == 0 && f.Progress == 0 {
		t.Fatal("moist farm should grow")
	}
	// a drought dries the soil and stops the farm
	w.Drought, w.SeasonLeft = true, 1e9
	run(w, 240)
	if w.MoistAt(f.Pos) {
		t.Fatal("soil should dry out in a long drought")
	}
	p := f.Progress
	run(w, 10)
	if f.Progress != p {
		t.Fatal("farm on dry soil must not grow")
	}
}

func TestBuildingsFloodAndWaterSaved(t *testing.T) {
	w := NewWorld(42)
	if _, err := w.Place(KHouse, Pos{int(riverX(0.5)), 0}, East, 0); err != ErrWater {
		t.Fatalf("house in the river: %v", err)
	}
	w2 := roundTrip(t, w)
	if math.Abs(w2.WaterAt(Pos{int(riverX(-19.5)), -20})-w.WaterAt(Pos{int(riverX(-19.5)), -20})) > 1e-6 {
		t.Fatal("water not saved")
	}
}

func TestMuddyWaterSeason(t *testing.T) {
	w := NewWorld(42)
	pump := built(t, w, KPump, pumpSpot(w, Pos{-6, -40}))
	w.spawnSettler(w.Core())
	w.Polluted, w.SeasonLeft = true, 1e9
	run(w, 120)
	upstream := Pos{int(riverX(-49.5)), -50}
	if w.ContAt(upstream) < 0.5 {
		t.Fatalf("river should turn muddy, contamination %.2f", w.ContAt(upstream))
	}
	if _, ok := w.pumpSource(pump); ok {
		t.Fatal("a pump must not take muddy water")
	}
	pur := &Building{Kind: KPurifier, Pos: pump.Pos}
	if _, ok := w.pumpSource(pur); !ok {
		t.Fatal("a purifier takes any water")
	}
	if w.MoistAt(Pos{upstream.X + 4, upstream.Y}) {
		t.Fatal("muddy water must not moisten soil")
	}
	// after the season the river clears
	w.Polluted = false
	run(w, 240)
	if c := w.ContAt(upstream); c > 0.2 {
		t.Fatalf("river should clear after the season, contamination %.2f", c)
	}
}

func TestFloodgate(t *testing.T) {
	w := NewWorld(42)
	var gates []*Building
	for x := -25; x <= -5; x++ {
		if p := (Pos{x, 10}); w.HeightAt(p) <= 1 {
			gates = append(gates, built(t, w, KGate, p))
		}
	}
	w.wallsDirty = true
	run(w, 40)
	closed, _ := riverStats(w, 25)
	for _, g := range gates {
		w.SetGate(g.ID, true)
	}
	run(w, 40)
	open, _ := riverStats(w, 25)
	t.Logf("downstream depth: closed %.2f open %.2f", closed, open)
	if closed > 0.1 || open < 0.3 {
		t.Fatalf("gate should block when closed and pass when open: %.2f / %.2f", closed, open)
	}
}

func TestPollutedSeasonsComeLater(t *testing.T) {
	w := NewWorld(42)
	for c := 0; c < 2; c++ {
		w.Cycle = c
		if w.nextPolluted() {
			t.Fatal("no muddy water in the first two cycles")
		}
	}
	n := 0
	for c := 2; c < 40; c++ {
		w.Cycle = c
		if w.nextPolluted() {
			n++
		}
	}
	if n < 5 || n > 25 {
		t.Fatalf("muddy seasons %d of 38", n)
	}
}
