package sim

// BuildDemo lays out a small working factory near the core. It is used for
// debug screenshots and as a smoke test of the full production chain.
func BuildDemo(w *World) {
	for i := range w.Stock {
		w.Stock[i] += 500
	}
	place := func(k Kind, x, y int, d Dir) *Building {
		b, _ := w.Place(k, Pos{x, y}, d, 0)
		return b
	}
	line := func(x0, y0, x1, y1 int) {
		for x0 != x1 || y0 != y1 {
			var d Dir
			switch {
			case x1 > x0:
				d = East
			case x1 < x0:
				d = West
			case y1 > y0:
				d = South
			default:
				d = North
			}
			place(KBelt, x0, y0, d)
			x0, y0 = x0+d.DX(), y0+d.DY()
		}
	}
	// iron: a column of 1×1 miners outputs west onto a collector belt
	for y := -8; y <= -4; y++ {
		place(KMiner, 12, y, West)
	}
	line(11, -8, 11, -3)
	line(11, -3, 8, -3)
	place(KFurnace, 8, -3, West)
	line(7, -3, 2, -3)
	line(2, -3, 2, 0)
	place(KBelt, 2, 0, West)
	// upgraded 2×2 machines next to the small ones
	place(KMinerBig, 14, -8, West)
	place(KFurnaceBig, 5, -8, West)
	place(KAssemblerBig, 2, -8, West)
	// copper: miners -> furnace -> wire assembler -> core
	for y := -7; y <= -5; y++ {
		place(KMiner, -13, y, East)
	}
	line(-12, -7, -12, -2)
	place(KBelt, -12, -2, East)
	line(-11, -2, -9, -2)
	place(KFurnace, -9, -2, East)
	line(-8, -2, -5, -2)
	a := place(KAssembler, -5, -2, East)
	w.SetRecipe(a.ID, 6)
	line(-4, -2, -2, -2)
	line(-2, -2, -2, 0)
	place(KBelt, -2, 0, East)
	place(KChest, 4, 4, East)
}

// LogisticsDemoOrigin is where BuildLogisticsDemo lays out its showcase.
var LogisticsDemoOrigin = Pos{-34, 22}

// BuildLogisticsDemo shows sorter, splitter, merger and tunnel working.
func BuildLogisticsDemo(w *World) {
	o := LogisticsDemoOrigin
	place := func(k Kind, x, y int, d Dir) *Building {
		b, _ := w.Place(k, o.Add(x, y), d, 0)
		return b
	}
	src := place(KChest, 0, 0, East)
	for i := 0; i < 20; i++ {
		src.Store = append(src.Store, Stack{Coal, 3}, Stack{IronPlate, 4}, Stack{CopperPlate, 3})
	}
	place(KBelt, 1, 0, East)
	place(KBelt, 2, 0, East)
	so := place(KSorter, 3, 0, East)
	w.SetFilter(so.ID, Coal)
	place(KBelt, 3, -1, North)
	place(KChest, 3, -2, North)
	place(KBelt, 3, 1, South)
	place(KChest, 3, 2, South)
	place(KBelt, 4, 0, East)
	place(KBelt, 5, 0, East)
	place(KSplitter, 6, 0, East)
	place(KBelt, 7, 0, East)
	place(KBelt, 8, 0, East)
	place(KBelt, 6, -1, North)
	for x := 6; x <= 8; x++ {
		place(KBelt, x, -2, East)
	}
	place(KBelt, 9, -2, South)
	place(KBelt, 9, -1, South)
	place(KBelt, 6, 1, South)
	for x := 6; x <= 8; x++ {
		place(KBelt, x, 2, East)
	}
	place(KBelt, 9, 2, North)
	place(KBelt, 9, 1, North)
	place(KMerger, 9, 0, East)
	place(KBelt, 10, 0, East)
	place(KTunnel, 11, 0, East)
	cross := place(KChest, 12, -4, South)
	cross.Store = []Stack{{Stone, 150}}
	for y := -3; y <= 3; y++ {
		place(KBelt, 12, y, South)
	}
	place(KChest, 12, 4, South)
	place(KTunnel, 14, 0, East)
	place(KBelt, 15, 0, East)
	place(KBelt, 16, 0, East)
	place(KChest, 17, 0, East)
}

// BuildCombatDemo adds defences east of the iron miners and an incoming wave.
func BuildCombatDemo(w *World) {
	for y := -9; y <= -1; y++ {
		w.Place(KWall, Pos{17, y}, East, 0)
	}
	t1, _ := w.Place(KTurret, Pos{16, -9}, East, 0)
	t2, _ := w.Place(KTurretBig, Pos{15, -6}, East, 0)
	t3, _ := w.Place(KTurret, Pos{16, -2}, East, 0)
	f, _ := w.Place(KFlamer, Pos{16, -4}, East, 0)
	for _, b := range []*Building{t1, t2, t3} {
		if b != nil {
			b.Ammo = 120
		}
	}
	if f != nil {
		f.Ammo = 160
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
