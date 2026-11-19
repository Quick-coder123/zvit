package world

// spot finds the free tile nearest to (x, y) where k can be placed.
func spot(w *World, k Kind, x, y int, extra func(int) bool) (int, int, bool) {
	for r := 0; r < 40; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				px, py := x+dx, y+dy
				if w.CheckPlace(k, px, py) == nil && (extra == nil || extra(Idx(px, py))) {
					return px, py, true
				}
			}
		}
	}
	return 0, 0, false
}

func near(w *World, prop Prop, r int, x, y int) func(int) bool {
	return func(i int) bool {
		cx, cy := XY(i)
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if In(cx+dx, cy+dy) && w.Props[Idx(cx+dx, cy+dy)] == prop {
					return true
				}
			}
		}
		return false
	}
}

// BuildDemo grows a small town for screenshots: it places a sensible set of
// buildings and runs the simulation for the given number of seconds.
func BuildDemo(w *World, seconds int) {
	h := w.hall()
	hx, hy := h.X, h.Y
	moist := func(i int) bool { return w.moist[i] }
	put := func(k Kind, x, y int, extra func(int) bool) {
		if px, py, ok := spot(w, k, x, y, extra); ok {
			w.Place(k, px, py)
		}
	}
	put(KPump, hx-4, hy, nil)
	put(KLumber, hx, hy, near(w, PropTree, 3, 0, 0))
	put(KGatherer, hx, hy, near(w, PropBush, 4, 0, 0))
	put(KHouse, hx+3, hy, nil)
	put(KFarm, hx-3, hy+2, moist)
	stage := 0
	for sec := 1; sec <= seconds; sec++ {
		for i := 0; i < TPS; i++ {
			w.Update()
		}
		switch {
		case stage == 0 && sec > 60:
			put(KHouse, hx+3, hy+2, nil)
			put(KFarm, hx-3, hy-2, moist)
			put(KForester, hx, hy, near(w, PropTree, 3, 0, 0))
			stage++
		case stage == 1 && w.Total(Log) >= 20:
			put(KSawmill, hx+4, hy-2, nil)
			put(KHouse, hx+4, hy+3, nil)
			stage++
		case stage == 2 && w.Total(Plank) >= 8 && w.Total(Log) >= 16:
			put(KWarehouse, hx+1, hy+5, nil)
			put(KTank, hx-1, hy+3, nil)
			put(KCampfire, hx+3, hy-3, nil)
			stage++
		}
	}
	w.Notices = w.Notices[:0]
}
