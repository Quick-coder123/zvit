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
	// iron: patch around (13,-5); miners output west onto a collector belt
	for _, y := range []int{-8, -6, -4} {
		place(KMiner, 12, y, West)
	}
	line(11, -8, 11, -3)
	line(11, -3, 8, -3)
	place(KFurnace, 7, -4, West)
	line(6, -3, 2, -3)
	line(2, -3, 2, 0)
	place(KBelt, 2, 0, West)
	// copper from (-13,-6) east into a furnace, then an assembler making wire
	place(KMiner, -14, -7, East)
	place(KMiner, -14, -5, East)
	line(-12, -7, -12, -2)
	place(KBelt, -12, -2, East)
	line(-11, -2, -9, -2)
	f := place(KFurnace, -9, -3, East)
	_ = f
	line(-7, -2, -5, -2)
	a := place(KAssembler, -5, -3, East)
	w.SetRecipe(a.ID, 6)
	line(-3, -2, -2, -2)
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
