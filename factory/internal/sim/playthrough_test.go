package sim

import (
	"math"
	"testing"
)

// nearestOre finds the free ore tile of a type closest to the town hall.
func nearestOre(w *World, it Item) Pos {
	best, bd := Pos{}, math.MaxFloat64
	for y := -25; y <= 25; y++ {
		for x := -25; x <= 25; x++ {
			p := Pos{x, y}
			if w.OreAt(p) == it && w.At(p) == nil {
				if d := math.Hypot(float64(x), float64(y)); d < bd {
					best, bd = p, d
				}
			}
		}
	}
	return best
}

func freeSpot(w *World, near Pos) Pos {
	for r := 0; r < 20; r++ {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				p := near.Add(x, y)
				if w.At(p) == nil && w.OreAt(p) == ItemNone && w.At(p.Add(0, 1)) == nil {
					return p
				}
			}
		}
	}
	return near
}

// TestEarlyGamePlaythrough follows a simple build order for 15 minutes and
// checks the economy holds up through the first wave.
func TestEarlyGamePlaythrough(t *testing.T) {
	w := NewWorld(42)
	try := func(k Kind, p Pos) {
		if _, err := w.Place(k, p, East, 0); err != nil {
			t.Logf("%5.0fs place %s: %v", float64(w.Tick)/TPS, Builds[k].Name, err)
		}
	}
	try(KMiner, nearestOre(w, IronOre))
	try(KFarm, freeSpot(w, Pos{-3, 3}))
	try(KHouse, freeSpot(w, Pos{2, 3}))
	step := 0
	for sec := 0; sec < 15*60; sec++ {
		run(w, 1)
		switch {
		case step == 0 && sec >= 40:
			try(KMiner, nearestOre(w, Stone))
			step++
		case step == 1 && w.TotalStock(Stone) >= 8:
			try(KHouse, freeSpot(w, Pos{4, 3}))
			step++
		case step == 2 && w.Research("furnace") == nil:
			try(KFurnace, freeSpot(w, Pos{4, -3}))
			step++
		case step == 3 && w.Research("walls") == nil:
			step++
		case step == 4:
			try(KMiner, nearestOre(w, IronOre))
			try(KFarm, freeSpot(w, Pos{-5, 4}))
			step++
		case step == 5 && w.TotalStock(Stone) >= 20:
			try(KHouse, freeSpot(w, Pos{0, 5}))
			try(KHouse, freeSpot(w, Pos{2, 6}))
			step++
		case step == 6 && w.Research("station") == nil:
			step++
		}
		if sec%30 == 29 && len(w.Ruins) > 0 && len(w.Enemies) == 0 {
			w.RebuildRuins()
		}
		if sec%60 == 59 {
			free, work, car, starving := w.RoleCounts()
			t.Logf("%2d min: pop %2d/%2d free %d work %d carriers %d starving %d food %3d ore %3d stone %3d plates %3d wave %d deaths %d step %d",
				(sec+1)/60, w.Population(), w.Housing(), free, work, car, starving, w.TotalStock(Food),
				w.TotalStock(IronOre), w.TotalStock(Stone), w.TotalStock(IronPlate), w.Wave, w.Deaths, step)
		}
	}
	if w.Deaths > 0 {
		t.Errorf("%d settlers died in a sensible early game", w.Deaths)
	}
	if w.Population() < 6 {
		t.Errorf("population %d after 15 min, expected growth", w.Population())
	}
	if c := w.Core(); c.Dmg >= w.MaxHP(c)*0.5 {
		t.Errorf("town hall badly damaged by the first waves")
	}
	if step < 6 {
		t.Errorf("build order stalled at step %d", step)
	}
}
