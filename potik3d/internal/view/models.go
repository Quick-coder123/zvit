package view

import (
	"image/color"
	"math"
	"sort"

	"potik3d/internal/world"
)

// drawBuilding draws a building's low-poly model.
func drawBuilding(b *Batch, w *world.World, bd *world.Building, base, t float64, selected bool) {
	x0, y0 := float64(bd.X), float64(bd.Y)
	s := float64(bd.Size())
	if bd.Kind == world.KStairs {
		drawStairs(b, bd, base, selected)
		return
	}
	if bd.Site {
		drawSite(b, bd, x0, y0, s, base)
		return
	}
	wall := colWall
	if selected {
		wall = mix(wall, rgb(0xF2D16B), 0.45)
	}
	switch bd.Kind {
	case world.KHall:
		ax0, ay0, ax1, ay1 := x0+0.15, y0+0.15, x0+s-0.15, y0+s-0.15
		top := base + 0.9
		b.Box(ax0, ay0, base, ax1, ay1, top, wall, wall)
		for side := 0; side < 4; side++ {
			b.WallRect(ax0, ay0, ax1, ay1, side, 0, 1, base, base+0.14, colStone)
			for _, u := range []float64{0.18, 0.38, 0.62, 0.82} {
				if side == sideYP && (u == 0.38 || u == 0.62) {
					continue
				}
				b.window(ax0, ay0, ax1, ay1, side, u, base+0.38, 0.14, 0.24)
			}
		}
		b.door(ax0, ay0, ax1, ay1, sideYP, 0.5, base, 0.3, 0.46)
		apex := top + 0.9
		b.Pyramid(x0+s/2, y0+s/2, top, s*0.62, 0.9, 4, math.Pi/4, rgb(0x4F7A8C))
		// tower and flag stand on the apex, above the whole roof
		cx, cy := x0+s/2, y0+s/2
		tower := rgb(0x6B4F3A)
		b.Box(cx-0.13, cy-0.13, apex-0.02, cx+0.13, cy+0.13, apex+0.4, tower, tower)
		b.WallRect(cx-0.13, cy-0.13, cx+0.13, cy+0.13, sideYP, 0.3, 0.7, apex+0.14, apex+0.3, colGlass)
		b.WallRect(cx-0.13, cy-0.13, cx+0.13, cy+0.13, sideXP, 0.3, 0.7, apex+0.14, apex+0.3, colGlass)
		b.Pyramid(cx, cy, apex+0.4, 0.24, 0.3, 4, math.Pi/4, rgb(0x4FA3A5))
		pz := apex + 0.7
		b.Box(cx-0.015, cy-0.015, pz, cx+0.015, cy+0.015, pz+0.45, rgb(0x5A4636), rgb(0x5A4636))
		fz := pz + 0.45
		wv := 0.04 * math.Sin(t*3+float64(bd.ID))
		flag := rgb(0xE07A5F)
		b.Poly(flag, [3]float64{cx, cy, fz}, [3]float64{cx + 0.17, cy + wv, fz - 0.02},
			[3]float64{cx + 0.34, cy - wv, fz - 0.04}, [3]float64{cx + 0.34, cy - wv, fz - 0.17},
			[3]float64{cx + 0.17, cy + wv, fz - 0.15}, [3]float64{cx, cy, fz - 0.15})
		if wantsSmoke(bd.ID, bd.Site, bd.Kind, 0) {
			smoke(b, cx-0.08, cy, pz, t, bd.ID)
		}
	case world.KHouse:
		drawHouse(b, w, bd, x0, y0, base, t, selected)
	case world.KWarehouse:
		col := rgb(0xB08A5A)
		if selected {
			col = mix(col, rgb(0xF2D16B), 0.45)
		}
		ax0, ay0, ax1, ay1 := x0+0.1, y0+0.1, x0+s-0.1, y0+s-0.42
		crates := func() {
			n := 0
			if bd.Used() > 0 {
				n = int(math.Ceil(4 * float64(bd.Used()) / float64(bd.Def().Cap)))
			}
			cc := rgb(0xC8A06A)
			for k := 0; k < min(n, 4); k++ {
				cx := x0 + 0.25 + float64(k%3)*0.32
				cz := base
				if k == 3 {
					cx, cz = x0+0.41, base+0.22
				}
				b.Box(cx-0.11, y0+s-0.36, cz, cx+0.11, y0+s-0.14, cz+0.22, cc, rgb(0xD9B27C))
			}
		}
		if b.propFirst(0, 1) {
			crates()
		}
		b.Box(ax0, ay0, base, ax1, ay1, base+0.75, col, col)
		b.WallRect(ax0, ay0, ax1, ay1, sideYP, 0.32, 0.68, base, base+0.56, colFrame)
		b.WallRect(ax0, ay0, ax1, ay1, sideYP, 0.34, 0.495, base, base+0.53, colDoor)
		b.WallRect(ax0, ay0, ax1, ay1, sideYP, 0.505, 0.66, base, base+0.53, colDoor)
		for _, side := range []int{sideXP, sideXM} {
			b.window(ax0, ay0, ax1, ay1, side, 0.5, base+0.42, 0.18, 0.14)
		}
		b.Prism(x0+0.02, y0+0.02, x0+s-0.02, ay1+0.08, base+0.75, 0.55, rgb(0x7A5536), col)
		if !b.propFirst(0, 1) {
			crates()
		}
	case world.KTank:
		cx, cy := x0+s/2, y0+s/2
		b.Prismoid(cx, cy, base, 0.38, 0.9, 8, mix(rgb(0x9C7A55), wall, boolf(selected)*0.3))
		for _, z := range []float64{0.2, 0.62} {
			b.Ring(cx, cy, base+z, 0.395, 0.05, 8, rgb(0x6E747C))
		}
		fill := float64(bd.Used()) / float64(bd.Def().Cap)
		b.Prismoid(cx, cy, base+0.9, 0.33, 0.02, 8, mix(rgb(0x8A6A48), colWater, fill))
	case world.KLumber:
		logs := func() {
			for k := 0; k < min(3, (bd.OutN+1)/2); k++ {
				b.Box(x0+0.22, y0+0.84, base+float64(k)*0.09, x0+0.78, y0+0.96, base+float64(k)*0.09+0.09, rgb(0x9C6B45), rgb(0xE3C79B))
			}
		}
		stump := func() {
			b.Prismoid(x0+0.1, y0+0.1, base, 0.07, 0.12, 6, rgb(0x7A5536))
			b.Box(x0+0.09, y0+0.06, base+0.12, x0+0.11, y0+0.14, base+0.2, rgb(0x6B4F3A), rgb(0x6B4F3A))
			b.Box(x0+0.08, y0+0.04, base+0.17, x0+0.12, y0+0.08, base+0.24, rgb(0x9AA0AC), rgb(0xB8C0CC))
		}
		if b.propFirst(0, 1) {
			logs()
		}
		if b.propFirst(-1, -1) {
			stump()
		}
		ax0, ay0, ax1, ay1 := x0+0.2, y0+0.2, x0+s-0.2, y0+s-0.2
		b.Box(ax0, ay0, base, ax1, ay1, base+0.45, rgb(0xB08A5A), rgb(0xB08A5A))
		b.door(ax0, ay0, ax1, ay1, sideXP, 0.5, base, 0.14, 0.28)
		b.window(ax0, ay0, ax1, ay1, sideYM, 0.5, base+0.2, 0.12, 0.12)
		b.Pyramid(x0+s/2, y0+s/2, base+0.45, s*0.55, 0.35, 4, math.Pi/4, rgb(0x7A5536))
		if !b.propFirst(0, 1) {
			logs()
		}
		if !b.propFirst(-1, -1) {
			stump()
		}
	case world.KForester:
		pots := func() {
			for _, px := range []float64{0.32, 0.68} {
				b.Prismoid(x0+px, y0+0.9, base, 0.06, 0.08, 6, rgb(0xB5654A))
				b.Pyramid(x0+px, y0+0.9, base+0.08, 0.07, 0.2, 5, px, colLeaf)
			}
		}
		if b.propFirst(0, 1) {
			pots()
		}
		ax0, ay0, ax1, ay1 := x0+0.2, y0+0.2, x0+s-0.2, y0+s-0.2
		b.Box(ax0, ay0, base, ax1, ay1, base+0.45, rgb(0xC8B48A), rgb(0xC8B48A))
		b.door(ax0, ay0, ax1, ay1, sideXP, 0.5, base, 0.14, 0.28)
		b.window(ax0, ay0, ax1, ay1, sideYP, 0.5, base+0.2, 0.12, 0.12)
		b.WallRect(ax0, ay0, ax1, ay1, sideYM, 0.25, 0.75, base+0.3, base+0.4, rgb(0x5E8A55))
		b.Pyramid(x0+s/2, y0+s/2, base+0.45, s*0.55, 0.35, 4, math.Pi/4, rgb(0x5E8A55))
		if !b.propFirst(0, 1) {
			pots()
		}
	case world.KGatherer:
		basket := func() {
			b.Prismoid(x0+0.9, y0+0.5, base, 0.08, 0.1, 6, rgb(0xC8A97E))
			for k := 0; k < min(3, bd.OutN); k++ {
				a := float64(k) * 2.1
				px, py := x0+0.9+math.Cos(a)*0.03, y0+0.5+math.Sin(a)*0.03
				b.Box(px-0.03, py-0.03, base+0.1, px+0.03, py+0.03, base+0.15, rgb(0xC0504D), rgb(0xD9605A))
			}
		}
		if b.propFirst(1, 0) {
			basket()
		}
		ax0, ay0, ax1, ay1 := x0+0.2, y0+0.2, x0+s-0.2, y0+s-0.2
		b.Box(ax0, ay0, base, ax1, ay1, base+0.4, rgb(0xD9C29A), rgb(0xD9C29A))
		b.door(ax0, ay0, ax1, ay1, sideYP, 0.5, base, 0.14, 0.27)
		b.window(ax0, ay0, ax1, ay1, sideXM, 0.5, base+0.18, 0.12, 0.11)
		b.WallRect(ax0, ay0, ax1, ay1, sideYM, 0.3, 0.7, base+0.24, base+0.34, rgb(0xC0504D))
		b.Pyramid(x0+s/2, y0+s/2, base+0.4, s*0.55, 0.32, 4, math.Pi/4, rgb(0x9C6B45))
		if !b.propFirst(1, 0) {
			basket()
		}
	case world.KFarm:
		b.Box(x0+0.04, y0+0.04, base, x0+s-0.04, y0+s-0.04, base+0.04, rgb(0x8A6A48), rgb(0x8A6A48))
		dry := !w.Moist(world.Idx(bd.X, bd.Y))
		g := float64(bd.Progress)
		for r := 0; r < 3; r++ {
			for k := 0; k < 3; k++ {
				px, py := x0+0.2+float64(k)*0.3, y0+0.2+float64(r)*0.3
				hgt := 0.08 + 0.2*g + 0.02*math.Sin(t*2+px+py)
				col := rgb(0x7FB35A)
				if g > 0.7 && (r+k)%2 == 0 {
					col = rgb(0xD9B44A)
				}
				if dry {
					col = rgb(0xB8A26A)
				}
				b.Pyramid(px, py, base+0.04, 0.09, hgt, 4, 0.3, col)
			}
		}
	case world.KPump:
		bucket := func() {
			b.Prismoid(x0+0.5, y0+0.86, base, 0.06, 0.1, 6, rgb(0x8C99A6))
		}
		if b.propFirst(0, 1) {
			bucket()
		}
		b.Box(x0+0.25, y0+0.25, base, x0+s-0.25, y0+s-0.25, base+0.3, rgb(0xA8A39A), colWater)
		b.Box(x0+s/2-0.05, y0+s/2-0.05, base+0.3, x0+s/2+0.05, y0+s/2+0.05, base+0.8, rgb(0x6B4F3A), rgb(0x6B4F3A))
		arm := 0.0
		if bd.Working {
			arm = 0.2 * math.Sin(t*3)
		}
		b.Box(x0+s/2-0.3, y0+s/2-0.04, base+0.8+arm*0.3, x0+s/2+0.3, y0+s/2+0.04, base+0.86+arm*0.3, rgb(0x6B4F3A), rgb(0x7A5536))
		if !b.propFirst(0, 1) {
			bucket()
		}
	case world.KSawmill:
		shed := func() {
			for _, px := range []float64{0.08, 0.92} {
				b.Box(x0+px-0.03, y0+0.1, base, x0+px+0.03, y0+0.16, base+0.85, colWood, colWood)
			}
			b.Box(x0+0.02, y0+0.04, base+0.85, x0+0.98, y0+0.4, base+0.9, rgb(0x7A5536), rgb(0x8C6B4A))
		}
		if b.propFirst(0, -1) {
			shed()
		}
		b.Box(x0+0.1, y0+0.2, base, x0+s-0.1, y0+s-0.2, base+0.35, colWood, colWoodLt)
		for k := 0; k < min(3, bd.InN); k++ {
			ly := y0 + 0.3 + float64(k)*0.13
			b.Box(x0+0.12, ly, base+0.35, x0+0.3, ly+0.1, base+0.45, rgb(0x9C6B45), rgb(0xE3C79B))
		}
		r := 0.22
		ang := 0.0
		if bd.Working {
			ang = t * 6
		}
		for k := 0; k < 6; k++ {
			a := ang + float64(k)*math.Pi/3
			px := x0 + s/2 + math.Cos(a)*r
			pz := base + 0.55 + math.Sin(a)*r
			b.Box(px-0.03, y0+s/2-0.01, pz-0.03, px+0.03, y0+s/2+0.01, pz+0.03, rgb(0x9AA0AC), rgb(0xB8C0CC))
		}
		if bd.OutN > 0 {
			b.Box(x0+0.5, y0+s-0.2, base, x0+0.9, y0+s-0.05, base+0.08*float64(min(bd.OutN, 4)), rgb(0xD9B27C), rgb(0xE3C79B))
		}
		if !b.propFirst(0, -1) {
			shed()
		}
	case world.KDam:
		b.Box(x0, y0+0.15, base, x0+s, y0+s-0.15, base+LevelH, rgb(0x8C6B4A), rgb(0x9C7A55))
	case world.KCampfire:
		seats := func(first bool) {
			for _, d := range []float64{-1, 1} {
				if b.propFirst(d, 0) != first {
					continue
				}
				b.Box(x0+0.5+d*0.38-0.06, y0+0.3, base, x0+0.5+d*0.38+0.06, y0+0.7, base+0.1, rgb(0x9C6B45), rgb(0xC8A06A))
			}
		}
		seats(true)
		for k := 0; k < 6; k++ {
			a := float64(k) * math.Pi / 3
			b.Box(x0+0.5+math.Cos(a)*0.2-0.05, y0+0.5+math.Sin(a)*0.2-0.05, base, x0+0.5+math.Cos(a)*0.2+0.05, y0+0.5+math.Sin(a)*0.2+0.05, base+0.08, colRock, colRock)
		}
		fl := 0.85 + 0.15*math.Sin(t*9)
		b.Pyramid(x0+0.5, y0+0.5, base, 0.14, 0.45*fl, 5, t, rgb(0xF2A65A))
		b.Pyramid(x0+0.5, y0+0.5, base, 0.08, 0.3*fl, 5, -t, rgb(0xF7D774))
		seats(false)
	}
}

func boolf(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

// drawSite shows a construction site: posts, beams and walls that rise
// with progress.
func drawSite(b *Batch, bd *world.Building, x0, y0, s, base float64) {
	total, left := 0, 0
	for _, c := range bd.Def().Cost {
		total += c.N
	}
	for _, c := range bd.Need {
		left += c.N
	}
	prog := 0.0
	if total > 0 {
		prog = 0.5 * float64(total-left) / float64(total)
	}
	prog += 0.5 * float64(bd.Work)
	if bd.Kind == world.KDam {
		b.Box(x0+0.1, y0+0.25, base, x0+s-0.1, y0+s-0.25, base+LevelH*prog, colWood, colWoodLt)
		return
	}
	h := 0.8 * s / 2
	if s == 1 {
		h = 0.6
	}
	for _, p := range [][2]float64{{0.12, 0.12}, {s - 0.12, 0.12}, {0.12, s - 0.12}, {s - 0.12, s - 0.12}} {
		b.Box(x0+p[0]-0.04, y0+p[1]-0.04, base, x0+p[0]+0.04, y0+p[1]+0.04, base+h, colWoodLt, colWoodLt)
	}
	if prog > 0.02 {
		b.Box(x0+0.16, y0+0.16, base, x0+s-0.16, y0+s-0.16, base+h*prog, colWall, rgb(0xD8CCB2))
	}
	b.Box(x0+0.08, y0+0.08, base+h-0.03, x0+s-0.08, y0+0.16, base+h+0.03, colWoodLt, colWoodLt)
	b.Box(x0+0.08, y0+s-0.16, base+h-0.03, x0+s-0.08, y0+s-0.08, base+h+0.03, colWoodLt, colWoodLt)
}

// drawSettler draws a small low-poly person, bobbing as they walk.
func drawSettler(b *Batch, w *world.World, st *world.Settler, t float64) {
	z := st.Z * LevelH
	ph := float64(st.Seed)
	bob := 0.03 * math.Abs(math.Sin(t*8+ph))
	x, y := st.X, st.Y
	skin := rgb(0xF1D2B0)
	tunic := rgb(0x4FA3A5) // free
	switch {
	case st.Task == world.TaskBuild:
		tunic = rgb(0xC8A06A)
	case st.Job != 0:
		tunic = rgb(0x6C8EBF)
	}
	if st.Suffering() {
		tunic = mix(tunic, rgb(0xE07A5F), 0.5)
	}
	leg := 0.04 * math.Sin(t*8+ph)
	b.Box(x-0.06+leg, y-0.04, z, x-0.01+leg, y+0.04, z+0.16, rgb(0x5A4636), rgb(0x5A4636))
	b.Box(x+0.01-leg, y-0.04, z, x+0.06-leg, y+0.04, z+0.16, rgb(0x5A4636), rgb(0x5A4636))
	b.Box(x-0.09, y-0.06, z+0.16+bob, x+0.09, y+0.06, z+0.38+bob, tunic, tunic)
	b.Box(x-0.07, y-0.07, z+0.38+bob, x+0.07, y+0.07, z+0.52+bob, skin, rgb(0x6B4A2E))
	carry := st.Carry.Item
	if st.Task == world.TaskBuild && st.Carry.N == 0 {
		// hammer in hand while building
		a := math.Sin(t * 10)
		b.Box(x+0.1, y-0.02, z+0.35+0.08*a, x+0.24, y+0.02, z+0.4+0.08*a, rgb(0x6B4F3A), rgb(0x9AA0AC))
	}
	if st.Carry.N == 0 {
		return
	}
	c := itemColor(carry)
	switch carry {
	case world.Log:
		b.Box(x-0.22, y-0.05, z+0.4+bob, x+0.22, y+0.05, z+0.48+bob, c, c)
	case world.Plank:
		b.Box(x-0.2, y-0.07, z+0.4+bob, x+0.2, y+0.07, z+0.45+bob, c, rgb(0xE3C79B))
	case world.Water:
		b.Box(x+0.08, y-0.04, z+0.12+bob, x+0.16, y+0.04, z+0.24+bob, rgb(0x8C99A6), c)
	default:
		b.Box(x-0.08, y-0.1, z+0.28+bob, x+0.08, y+0.1, z+0.36+bob, rgb(0xC8A97E), c)
	}
}

func itemColor(it world.Item) color.RGBA {
	v := world.Items[it].Color
	return rgb(v)
}

// siteProgress is how far a construction site has got (0..1): half for
// delivered materials, half for building work.
func siteProgress(bd *world.Building) float64 {
	if !bd.Site {
		return 1
	}
	total, left := 0, 0
	for _, c := range bd.Def().Cost {
		total += c.N
	}
	for _, c := range bd.Need {
		left += c.N
	}
	prog := 0.0
	if total > 0 {
		prog = 0.5 * float64(total-left) / float64(total)
	}
	return prog + 0.5*float64(bd.Work)
}

// drawSupport draws a platform or bridge standing on the ground at level
// height ground; levels is how high its deck is (for previews, the target).
func drawSupport(b *Batch, w *world.World, bd *world.Building, ground, t float64, hi bool, levels int) {
	x0, y0 := float64(bd.X), float64(bd.Y)
	deck, post := colWoodLt, colWood
	if hi {
		deck = mix(deck, rgb(0xF2D16B), 0.45)
	}
	full := float64(levels) * LevelH
	done := full
	prog := siteProgress(bd)
	if bd.Site {
		if bd.Kind == world.KPlatform {
			done = float64(levels-1) * LevelH
		} else {
			done = 0
		}
	}
	switch bd.Kind {
	case world.KPlatform:
		// finished levels: a timber crate with a plank deck
		if done > 0 {
			b.Box(x0, y0, ground, x0+1, y0+1, ground+done-0.1, rgb(0x8C6B4A), rgb(0x8C6B4A))
			for lv := 1; float64(lv)*LevelH < done+0.01; lv++ {
				z := ground + float64(lv)*LevelH
				b.Box(x0, y0, z-0.12, x0+1, y0+1, z, mix(post, deck, 0.4), deck)
			}
		}
		if bd.Site {
			// the new level: corner posts rising, then the deck
			h := (full - done) * math.Min(1, prog*1.6)
			for _, p := range [][2]float64{{0.08, 0.08}, {0.92, 0.08}, {0.08, 0.92}, {0.92, 0.92}} {
				b.Box(x0+p[0]-0.05, y0+p[1]-0.05, ground+done, x0+p[0]+0.05, y0+p[1]+0.05, ground+done+h, post, post)
			}
			if prog > 0.65 {
				z := ground + full
				b.Box(x0, y0, z-0.1, x0+(prog-0.65)/0.35, y0+1, z, mix(post, deck, 0.4), deck)
			}
		}
	case world.KBridge:
		z := ground + full
		posts := [][2]float64{{0.12, 0.12}, {0.88, 0.12}, {0.12, 0.88}, {0.88, 0.88}}
		for _, p := range posts {
			b.Box(x0+p[0]-0.05, y0+p[1]-0.05, ground-0.2, x0+p[0]+0.05, y0+p[1]+0.05, z-0.08, post, post)
		}
		if bd.Site {
			if prog > 0.4 {
				b.Box(x0, y0, z-0.08, x0+1, y0+(prog-0.4)/0.6, z, mix(post, deck, 0.4), deck)
			}
			return
		}
		b.Box(x0, y0, z-0.08, x0+1, y0+1, z, mix(post, deck, 0.4), deck)
		// rails on edges that drop away
		top := w.Surface(world.Idx(bd.X, bd.Y))
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := bd.X+d[0], bd.Y+d[1]
			if world.In(nx, ny) && w.Surface(world.Idx(nx, ny)) >= top {
				continue
			}
			var rx0, ry0, rx1, ry1 float64
			switch {
			case d[0] == 1:
				rx0, ry0, rx1, ry1 = x0+0.92, y0, x0+0.98, y0+1
			case d[0] == -1:
				rx0, ry0, rx1, ry1 = x0+0.02, y0, x0+0.08, y0+1
			case d[1] == 1:
				rx0, ry0, rx1, ry1 = x0, y0+0.92, x0+1, y0+0.98
			default:
				rx0, ry0, rx1, ry1 = x0, y0+0.02, x0+1, y0+0.08
			}
			b.Box(rx0, ry0, z+0.22, rx1, ry1, z+0.28, post, deck)
			for _, f := range []float64{0.1, 0.5, 0.9} {
				px, py := rx0+(rx1-rx0)*f, ry0+(ry1-ry0)*f
				if d[0] != 0 {
					px = (rx0 + rx1) / 2
				} else {
					py = (ry0 + ry1) / 2
				}
				b.Box(px-0.03, py-0.03, z, px+0.03, py+0.03, z+0.22, post, post)
			}
		}
	}
}

// drawStairs draws steps climbing rise levels towards dir.
func drawStairs(b *Batch, bd *world.Building, base float64, selected bool) {
	x0, y0 := float64(bd.X), float64(bd.Y)
	rise := float64(max(1, bd.Levels)) * LevelH
	n := 4 * max(1, bd.Levels)
	if bd.Site {
		n = int(math.Ceil(float64(n) * siteProgress(bd)))
	}
	col, topc := colWood, colWoodLt
	if selected {
		topc = mix(topc, rgb(0xF2D16B), 0.45)
	}
	type step struct {
		x0, y0, x1, y1, z float64
		depth             float64
	}
	steps := make([]step, 0, n)
	for k := 0; k < n; k++ {
		f0 := float64(k) / float64(4*max(1, bd.Levels))
		f1 := float64(k+1) / float64(4*max(1, bd.Levels))
		z := base + rise*f1
		var s step
		switch bd.Dir {
		case 0: // +x
			s = step{x0 + f0, y0 + 0.1, x0 + f1, y0 + 0.9, z, 0}
		case 1: // -x
			s = step{x0 + 1 - f1, y0 + 0.1, x0 + 1 - f0, y0 + 0.9, z, 0}
		case 2: // +y
			s = step{x0 + 0.1, y0 + f0, x0 + 0.9, y0 + f1, z, 0}
		default: // -y
			s = step{x0 + 0.1, y0 + 1 - f1, x0 + 0.9, y0 + 1 - f0, z, 0}
		}
		_, _, s.depth = b.cam.Project((s.x0+s.x1)/2, (s.y0+s.y1)/2, base)
		steps = append(steps, s)
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].depth < steps[j].depth })
	for _, s := range steps {
		b.Box(s.x0, s.y0, base, s.x1, s.y1, s.z, col, topc)
	}
}
