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
