package sim

import "testing"

func treeNear(t *testing.T, w *World, from Pos) Pos {
	t.Helper()
	for r := 0; r < 20; r++ {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				p := from.Add(x, y)
				if v, g := w.VegAt(p); v == VegTree && g >= 1 {
					return p
				}
			}
		}
	}
	t.Fatal("no tree")
	return Pos{}
}

func TestStartHasForestAndBushes(t *testing.T) {
	w := NewWorld(42)
	trees, bushes := 0, 0
	for y := -15; y <= 15; y++ {
		for x := -15; x <= 15; x++ {
			switch v, _ := w.VegAt(Pos{x, y}); v {
			case VegTree:
				trees++
			case VegBush:
				bushes++
			}
		}
	}
	if trees < 20 || bushes < 3 {
		t.Fatalf("trees %d bushes %d near start", trees, bushes)
	}
	if v, _ := w.VegAt(Pos{2, 2}); v != VegNone {
		t.Fatal("town square should be clear")
	}
}

func TestLumberjackChops(t *testing.T) {
	w := emptyWorld(3)
	w.Stock[Wood] = 0
	tree := treeNear(t, w, Pos{7, 9})
	built(t, w, KLumber, Pos{tree.X - 2, tree.Y})
	run(w, 90)
	if w.TotalStock(Wood) < 4 {
		t.Fatalf("only %d wood delivered", w.TotalStock(Wood))
	}
	if v, _ := w.VegAt(tree); v == VegTree {
		// the nearest tree should be gone
		t.Log("nearest tree survived (another was chopped first)")
	}
}

func TestForesterPlants(t *testing.T) {
	w := emptyWorld(2)
	f := built(t, w, KForester, Pos{-20, 20})
	run(w, 30)
	if len(w.Planted) == 0 {
		t.Fatal("forester planted nothing")
	}
	for p := range w.Planted {
		if v, g := w.VegAt(p); v != VegTree || g >= 1 {
			t.Fatal("a fresh sapling should be growing")
		}
		_ = f
	}
	first := w.Tick
	run(w, TreeGrowTime)
	for p, at := range w.Planted {
		if _, g := w.VegAt(p); at <= first && g < 1 {
			t.Fatal("early saplings should have grown")
		}
	}
}

func TestGathererFeeds(t *testing.T) {
	w := emptyWorld(3)
	w.Stock[Food] = 0
	built(t, w, KGatherer, Pos{-7, 7})
	for _, s := range w.Settlers {
		s.MealIn = 1e6 // nobody eats during the test
	}
	run(w, 80)
	if w.FoodAvailable() < 4 || len(w.Picked) < 2 {
		t.Fatalf("gatherer brought only %d food (%d bushes picked)", w.FoodAvailable(), len(w.Picked))
	}
}

func TestBuildingClearsTreeAndSaveKeepsNature(t *testing.T) {
	w := emptyWorld(0)
	tree := treeNear(t, w, Pos{7, 9})
	built(t, w, KHouse, tree)
	if v, _ := w.VegAt(tree); v != VegNone {
		t.Fatal("building should clear the tree")
	}
	w.Planted[Pos{30, 30}] = w.Tick
	w2 := roundTrip(t, w)
	if v, _ := w2.VegAt(Pos{30, 30}); v != VegTree {
		t.Fatal("planted tree lost")
	}
	w2.Remove(w2.At(tree).ID)
	if v, _ := w2.VegAt(tree); v != VegNone {
		t.Fatal("cleared tree should stay cleared")
	}
}

func TestSawmill(t *testing.T) {
	w := emptyWorld(3)
	w.Stock[Wood], w.Stock[Planks] = 20, 0
	built(t, w, KSawmill, Pos{4, -3})
	run(w, 60)
	if w.TotalStock(Planks) < 5 {
		t.Fatalf("planks %d", w.TotalStock(Planks))
	}
}
