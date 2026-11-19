package view

import (
	"image/color"
	"math"

	"potik3d/internal/world"
)

// drawBuilding draws a building's low-poly model.
func drawBuilding(b *Batch, w *world.World, bd *world.Building, base, t float64, selected bool) {
	x0, y0 := float64(bd.X), float64(bd.Y)
	s := float64(bd.Size())
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
		b.Box(x0+0.15, y0+0.15, base, x0+s-0.15, y0+s-0.15, base+0.9, wall, wall)
		b.Pyramid(x0+s/2, y0+s/2, base+0.9, s*0.62, 0.9, 4, math.Pi/4, rgb(0x4F7A8C))
		b.Box(x0+s/2-0.12, y0+s/2-0.12, base+1.6, x0+s/2+0.12, y0+s/2+0.12, base+2.2, rgb(0x6B4F3A), rgb(0x6B4F3A))
		b.Pyramid(x0+s/2, y0+s/2, base+2.2, 0.35, 0.3, 4, math.Pi/4, rgb(0x4FA3A5))
	case world.KHouse:
		b.Box(x0+0.15, y0+0.15, base, x0+s-0.15, y0+s-0.15, base+0.55, wall, wall)
		b.Pyramid(x0+s/2, y0+s/2, base+0.55, s*0.6, 0.5, 4, math.Pi/4, colRoof)
	case world.KWarehouse:
		col := rgb(0xB08A5A)
		if selected {
			col = mix(col, rgb(0xF2D16B), 0.45)
		}
		b.Box(x0+0.1, y0+0.1, base, x0+s-0.1, y0+s-0.1, base+0.7, col, col)
		b.Prism(x0+0.02, y0+0.02, x0+s-0.02, y0+s-0.02, base+0.7, 0.55, rgb(0x7A5536), col)
	case world.KTank:
		b.Prismoid(x0+s/2, y0+s/2, base, 0.38, 0.9, 8, mix(rgb(0x9C7A55), wall, boolf(selected)*0.3))
		fill := float64(bd.Used()) / float64(bd.Def().Cap)
		b.Prismoid(x0+s/2, y0+s/2, base+0.9, 0.33, 0.02, 8, mix(rgb(0x8A6A48), colWater, fill))
	case world.KLumber:
		b.Box(x0+0.2, y0+0.2, base, x0+s-0.2, y0+s-0.2, base+0.45, rgb(0xB08A5A), rgb(0xB08A5A))
		b.Pyramid(x0+s/2, y0+s/2, base+0.45, s*0.55, 0.35, 4, math.Pi/4, rgb(0x7A5536))
		for k := 0; k < min(3, bd.OutN/2+1); k++ {
			ly := y0 + s - 0.12 - float64(k)*0.12
			b.Box(x0+0.08, ly-0.05, base, x0+s*0.6, ly+0.05, base+0.1, rgb(0x9C6B45), rgb(0xE3C79B))
		}
	case world.KForester:
		b.Box(x0+0.2, y0+0.2, base, x0+s-0.2, y0+s-0.2, base+0.45, rgb(0xC8B48A), rgb(0xC8B48A))
		b.Pyramid(x0+s/2, y0+s/2, base+0.45, s*0.55, 0.35, 4, math.Pi/4, rgb(0x5E8A55))
		b.Pyramid(x0+0.2, y0+s-0.2, base, 0.12, 0.3, 5, 0, colLeaf)
	case world.KGatherer:
		b.Box(x0+0.2, y0+0.2, base, x0+s-0.2, y0+s-0.2, base+0.4, rgb(0xD9C29A), rgb(0xD9C29A))
		b.Pyramid(x0+s/2, y0+s/2, base+0.4, s*0.55, 0.32, 4, math.Pi/4, rgb(0x9C6B45))
		if bd.OutN > 0 {
			b.Box(x0+0.65, y0+0.65, base, x0+0.9, y0+0.9, base+0.12, rgb(0xC8A97E), rgb(0xC0504D))
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
		b.Box(x0+0.25, y0+0.25, base, x0+s-0.25, y0+s-0.25, base+0.3, rgb(0xA8A39A), colWater)
		b.Box(x0+s/2-0.05, y0+s/2-0.05, base+0.3, x0+s/2+0.05, y0+s/2+0.05, base+0.8, rgb(0x6B4F3A), rgb(0x6B4F3A))
		arm := 0.0
		if bd.Working {
			arm = 0.2 * math.Sin(t*3)
		}
		b.Box(x0+s/2-0.3, y0+s/2-0.04, base+0.8+arm*0.3, x0+s/2+0.3, y0+s/2+0.04, base+0.86+arm*0.3, rgb(0x6B4F3A), rgb(0x7A5536))
	case world.KSawmill:
		b.Box(x0+0.1, y0+0.2, base, x0+s-0.1, y0+s-0.2, base+0.35, colWood, colWoodLt)
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
			b.Box(x0+0.1, y0+s-0.2, base, x0+0.6, y0+s-0.05, base+0.08*float64(min(bd.OutN, 4)), rgb(0xD9B27C), rgb(0xE3C79B))
		}
	case world.KDam:
		b.Box(x0, y0+0.15, base, x0+s, y0+s-0.15, base+LevelH, rgb(0x8C6B4A), rgb(0x9C7A55))
	case world.KCampfire:
		for k := 0; k < 6; k++ {
			a := float64(k) * math.Pi / 3
			b.Box(x0+0.5+math.Cos(a)*0.28-0.05, y0+0.5+math.Sin(a)*0.28-0.05, base, x0+0.5+math.Cos(a)*0.28+0.05, y0+0.5+math.Sin(a)*0.28+0.05, base+0.08, colRock, colRock)
		}
		fl := 0.85 + 0.15*math.Sin(t*9)
		b.Pyramid(x0+0.5, y0+0.5, base, 0.16, 0.45*fl, 5, t, rgb(0xF2A65A))
		b.Pyramid(x0+0.5, y0+0.5, base, 0.09, 0.3*fl, 5, -t, rgb(0xF7D774))
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
