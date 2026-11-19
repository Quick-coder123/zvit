package world

import (
	"bytes"
	"math"
	"testing"
)

func run(w *World, sec float64) {
	for i := 0; i < int(sec*TPS); i++ {
		w.Update()
	}
}

func place(t *testing.T, w *World, k Kind, x, y int, extra func(int) bool) *Building {
	t.Helper()
	px, py, ok := spot(w, k, x, y, extra)
	if !ok {
		t.Fatalf("no spot for %s", Builds[k].Name)
	}
	b, err := w.Place(k, px, py)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestStartTown(t *testing.T) {
	w := New(7)
	h := w.hall()
	if h == nil || len(w.Settlers) != 6 || w.Housing() != 6 {
		t.Fatal("start: hall and six settlers")
	}
	if w.Total(Log) != 40 {
		t.Fatal("starting logs")
	}
	// the hall should be on dry ground with water within reach
	hx, hy := h.Center()
	if d := math.Abs(hx - w.RiverX(hy)); d > 20 {
		t.Fatalf("hall %v tiles from the river", d)
	}
}

func TestPathAroundTerrain(t *testing.T) {
	w := New(7)
	h := w.hall()
	s := w.Settlers[0]
	far := Idx(5, 5)
	for w.Height[far] == 0 || w.Water[far] > 0.1 {
		far++
	}
	p := w.FindPath(s.tile(), func(i int) bool { return i == far }, 5.5, 5.5, 0)
	if p == nil {
		t.Skip("far corner unreachable on this map")
	}
	prev := s.tile()
	for _, i := range p {
		if absi(w.WalkHeight(i)-w.WalkHeight(prev)) > 1 {
			t.Fatal("path climbs more than one level at once")
		}
		if b := w.Owner(i); b != nil && b != h {
			t.Fatal("path walks through a building")
		}
		prev = i
	}
}

// TestEarlyGame plays a simple build order for 15 minutes.
func TestEarlyGame(t *testing.T) {
	w := New(7)
	h := w.hall()
	hx, hy := h.X, h.Y
	moist := func(i int) bool { return w.moist[i] }
	place(t, w, KPump, hx-4, hy, nil)
	place(t, w, KLumber, hx, hy, near(w, PropTree, 3, 0, 0))
	place(t, w, KGatherer, hx, hy, near(w, PropBush, 4, 0, 0))
	place(t, w, KHouse, hx+3, hy, nil)
	place(t, w, KFarm, hx-3, hy+2, moist)
	stage := 0
	for sec := 1; sec <= 15*60; sec++ {
		run(w, 1)
		switch {
		case stage == 0 && sec > 60:
			place(t, w, KHouse, hx+3, hy+2, nil)
			place(t, w, KFarm, hx-3, hy-2, moist)
			place(t, w, KForester, hx, hy, near(w, PropTree, 3, 0, 0))
			stage++
		case stage == 1 && w.Total(Log) >= 20:
			place(t, w, KSawmill, hx+4, hy-2, nil)
			place(t, w, KHouse, hx+4, hy+3, nil)
			stage++
		case stage == 2 && w.Total(Plank) >= 8 && w.Total(Log) >= 16:
			place(t, w, KWarehouse, hx+1, hy+5, nil)
			place(t, w, KTank, hx-1, hy+3, nil)
			stage++
		}
		if sec%60 == 0 {
			free, work, suf := w.Roles()
			used, total := w.Space()
			sites := 0
			for _, b := range w.Buildings {
				if b.Site {
					sites++
				}
			}
			t.Logf("%2d min pop %2d/%2d free %d work %d suffer %d logs %3d planks %3d berries %3d crops %3d water %3d space %d/%d sites %d drought %v deaths %d",
				sec/60, len(w.Settlers), w.Housing(), free, work, suf, w.Total(Log), w.Total(Plank), w.Available(Berries),
				w.Available(Crop), w.Available(Water), used, total, sites, w.Drought, w.Deaths)
		}
	}
	if w.Deaths > 0 {
		t.Errorf("%d deaths in a sensible early game", w.Deaths)
	}
	if len(w.Settlers) <= 6 {
		t.Errorf("no growth: %d settlers", len(w.Settlers))
	}
	for _, b := range w.Buildings {
		if b.Site {
			t.Errorf("%s still unfinished", b.Def().Name)
		}
	}
}

func TestSaveRoundTrip(t *testing.T) {
	w := New(7)
	run(w, 5)
	var buf bytes.Buffer
	if err := w.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	w2, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(w2.Buildings) != len(w.Buildings) || len(w2.Settlers) != len(w.Settlers) || w2.Total(Log) != w.Total(Log) {
		t.Fatal("mismatch after load")
	}
	run(w2, 5)
}
