package sim

// BuildDemo lays out a small working settlement near the town hall. It is
// used for debug screenshots and as a smoke test of the full economy.
func BuildDemo(w *World) {
	for i := range w.Stock {
		w.Stock[i] += 15
	}
	place := func(k Kind, x, y int) *Building {
		b, _ := w.Place(k, Pos{x, y}, East, 0)
		if b != nil {
			b.Site, b.Need = false, nil
		}
		return b
	}
	for y := -8; y <= -4; y++ {
		place(KMiner, 12, y)
	}
	place(KMinerBig, 14, -8)
	place(KLab, -3, -5)
	place(KChest, 10, -6)
	place(KChest, -10, -4)
	place(KChest, 3, -1)
	place(KStation, 9, -4)
	place(KFurnace, 7, -3)
	place(KFurnace, 7, -5)
	place(KFurnaceBig, 4, -8)
	for y := -7; y <= -5; y++ {
		place(KMiner, -13, y)
	}
	place(KFurnace, -9, -2)
	a := place(KAssembler, -6, -2)
	w.SetRecipe(a.ID, 6)
	place(KAssemblerBig, 1, -8)
	for i, p := range []Pos{{-4, 3}, {-2, 4}, {0, 4}, {2, 4}, {4, 3}, {-5, 5}} {
		place(KHouse, p.X, p.Y)
		_ = i
	}
	place(KLumber, 5, 10)
	place(KForester, 9, 12)
	place(KGatherer, -6, 9)
	place(KSawmill, 6, 3)
	place(KFarm, -1, 7)
	place(KFarm, 1, 7)
	place(KFarm, 3, 7)
	// an unfinished site waiting for a builder
	w.Place(KFurnace, Pos{5, 0}, East, 0)
	c := w.Core()
	for i := 0; i < 14; i++ {
		s := w.spawnSettler(c)
		s.X += float64(i%5) - 2
	}
	// let jobs settle
	for i := 0; i < 20*TPS; i++ {
		w.Update(nil)
	}
}

// BuildCombatDemo adds defences east of the iron miners and an incoming wave.
func BuildCombatDemo(w *World) {
	place := func(k Kind, x, y int) *Building {
		b, _ := w.Place(k, Pos{x, y}, East, 0)
		if b != nil {
			b.Site, b.Need = false, nil
		}
		return b
	}
	for y := -9; y <= -1; y++ {
		place(KWall, 17, y)
	}
	for _, b := range []*Building{place(KTurret, 16, -9), place(KTurretBig, 15, -6), place(KTurret, 16, -2)} {
		if b != nil {
			b.Ammo = 120
		}
	}
	if f := place(KFlamer, 16, -4); f != nil {
		f.Ammo = 160
	}
	c := w.Core()
	for i := 0; i < 6; i++ {
		w.spawnSettler(c)
	}
	for i := 0; i < 3*TPS; i++ {
		w.Update(nil)
	}
	w.Peaceful = false
	w.Wave = 5
	for i := 0; i < 14; i++ {
		k := Crawler
		if i%4 == 0 {
			k = Swarmer
		}
		if i == 3 || i == 9 {
			k = Brute
		}
		w.Enemies = append(w.Enemies, &Enemy{Kind: k, X: 26 + float64(i%5)*1.3, Y: -9 + float64(i)*0.8, HP: EnemyDefs[k].HP, Phase: float32(i)})
	}
	w.Ruins = append(w.Ruins, Spec{Kind: KWall, Pos: Pos{17, 0}, Dir: East})
}
