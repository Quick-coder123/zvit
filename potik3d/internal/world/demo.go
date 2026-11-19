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

// finishSite completes a construction site at once (demos and tests).
func (w *World) finishSite(b *Building) {
	b.Site, b.Need, b.Work = false, nil, 0
	w.pathEpoch++
}

// BuildVerticalDemo adds a raised terrace with a house, stairs up to it and
// a bridge across the river near the hall, all finished.
func BuildVerticalDemo(w *World) {
	h := w.hall()
	if h == nil {
		return
	}
	place := func(k Kind, x, y int) *Building {
		b, err := w.Place(k, x, y)
		if err != nil {
			return nil
		}
		w.finishSite(b)
		return b
	}
	// a 2x2 terrace two levels up, with a house on one corner
	for r := 0; r < 30; r++ {
		x, y, ok := spot(w, KHouse, h.X+5+r%5, h.Y-5+r/5, func(i int) bool {
			x, y := XY(i)
			for dy := 0; dy < 2; dy++ {
				for dx := -1; dx < 2; dx++ {
					if w.CheckPlace(KHouse, x+dx, y+dy) != nil {
						return false
					}
				}
			}
			return true
		})
		if !ok {
			break
		}
		for lv := 0; lv < 2; lv++ {
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					place(KPlatform, x+dx, y+dy)
				}
			}
		}
		place(KHouse, x+1, y)
		place(KStairs, x-1, y+1)
		break
	}
	// a bridge across the river on a free row near the hall
	for _, dy := range []int{0, 1, -1, 2, -2, 3, -3, 4, -4, 5, -5, 6, -6} {
		y := h.Y + dy
		x := h.X - 1
		for x > 0 && w.passable(Idx(x, y), 0) && w.owner[Idx(x, y)] == 0 && w.Height[Idx(x, y)] > 0 {
			x--
		}
		if x <= 0 || w.Height[Idx(x, y)] != 0 || w.owner[Idx(x, y)] != 0 || w.passable(Idx(x, y), 0) {
			continue
		}
		n := 0
		for ; x > 0 && w.CheckPlace(KBridge, x, y) == nil; x-- {
			place(KBridge, x, y)
			n++
			if w.passable(Idx(x-1, y), 0) && w.Height[Idx(x-1, y)] >= 1 {
				break
			}
		}
		if n > 0 {
			break
		}
	}
	w.Notices = w.Notices[:0]
}
