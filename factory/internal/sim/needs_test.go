package sim

import "testing"

func TestVariedDiet(t *testing.T) {
	w := emptyWorld(1)
	s := w.Settlers[0]
	w.Stock[Food], w.Stock[Berries], w.Stock[Bread] = 5, 5, 0
	w.eat(s)
	w.eat(s)
	if s.LastFood == s.PrevFood || s.LastFood == ItemNone {
		t.Fatalf("settler should alternate foods: %v %v", s.PrevFood, s.LastFood)
	}
	if n := w.needScores(s); n[NeedVariety] != 1 {
		t.Fatal("varied diet should be met")
	}
	w.Stock[Berries] = 0
	w.eat(s)
	w.eat(s)
	if n := w.needScores(s); n[NeedVariety] != 0 {
		t.Fatal("one food only should not count as varied")
	}
}

func TestLeisureBeautyAndProductivity(t *testing.T) {
	w := emptyWorld(4)
	built(t, w, KHouse, Pos{5, 5})
	run(w, 2)
	base := w.Productivity()
	built(t, w, KCampfire, Pos{7, 5})
	built(t, w, KStatue, Pos{3, 7})
	for _, s := range w.Settlers {
		s.Home = w.At(Pos{5, 5}).ID
		s.LastFood, s.PrevFood = Food, Berries
	}
	w.updateWellbeing()
	if w.NeedShare(NeedLeisure) != 1 || w.NeedShare(NeedBeauty) != 1 {
		t.Fatalf("leisure %.2f beauty %.2f", w.NeedShare(NeedLeisure), w.NeedShare(NeedBeauty))
	}
	if w.Wellbeing() < 4.9 || w.Productivity() <= base {
		t.Fatalf("wellbeing %.2f productivity %.2f (was %.2f)", w.Wellbeing(), w.Productivity(), base)
	}
}

func TestBakery(t *testing.T) {
	w := emptyWorld(3)
	w.Researched["bakery"] = true
	w.Stock[Food], w.Stock[Wood], w.Stock[Bread] = 30, 20, 0
	built(t, w, KBakery, Pos{4, -3})
	run(w, 60)
	if w.TotalStock(Bread) < 6 {
		t.Fatalf("bread %d", w.TotalStock(Bread))
	}
}
