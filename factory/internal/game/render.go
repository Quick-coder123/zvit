package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"potik/internal/sim"
)

// Camera maps tile coordinates to screen pixels.
type Camera struct {
	X, Y  float64 // tile coordinate at screen centre
	Scale float64 // pixels per tile
	W, H  float64 // screen size
}

func (c *Camera) ToScreen(x, y float64) (float32, float32) {
	return float32((x-c.X)*c.Scale + c.W/2), float32((y-c.Y)*c.Scale + c.H/2)
}

func (c *Camera) ToWorld(sx, sy float64) (float64, float64) {
	return (sx-c.W/2)/c.Scale + c.X, (sy-c.H/2)/c.Scale + c.Y
}

func (g *Game) drawWorld(dst *ebiten.Image) {
	dst.Fill(colBG)
	g.drawGround(dst)
	cam := &g.cam
	x0, y0 := cam.ToWorld(0, 0)
	x1, y1 := cam.ToWorld(cam.W, cam.H)
	tx0, ty0 := int(math.Floor(x0))-1, int(math.Floor(y0))-1
	tx1, ty1 := int(math.Ceil(x1))+1, int(math.Ceil(y1))+1
	s := float32(cam.Scale)

	g.drawTerrain(dst, tx0, ty0, tx1, ty1)

	// ore deposits
	for y := ty0; y <= ty1; y++ {
		for x := tx0; x <= tx1; x++ {
			it := g.w.OreAt(sim.Pos{X: x, Y: y})
			if it == sim.ItemNone {
				continue
			}
			if g.drawOreTile(dst, it, x, y) {
				continue
			}
			sx, sy := cam.ToScreen(float64(x), float64(y))
			c := rgb(sim.Items[it].Color)
			if cam.Scale < 14 {
				fillRect(dst, sx, sy, s+0.5, s+0.5, mix(colBG, c, 0.35))
				continue
			}
			fillRoundRect(dst, sx+s*0.04, sy+s*0.04, s*0.92, s*0.92, s*0.22, mix(colBG, c, 0.2))
			// a couple of pebbles per tile, stable per position
			h := uint32(x*73856093) ^ uint32(y*19349663)
			for k := 0; k < 2; k++ {
				h = h*1664525 + 1013904223
				px := float32(h>>8&0xFF) / 255
				h = h*1664525 + 1013904223
				py := float32(h>>8&0xFF) / 255
				fillCircle(dst, sx+s*(0.25+px*0.5), sy+s*(0.25+py*0.5), s*(0.07+0.03*float32(k)), mix(colBG, c, 0.55))
			}
		}
	}

	// grid
	if cam.Scale >= 18 {
		gc := rgba(0xD9D1C2, 0x70)
		for x := tx0; x <= tx1; x++ {
			sx, _ := cam.ToScreen(float64(x), 0)
			fillRect(dst, sx, 0, 1, float32(cam.W), gc)
		}
		for y := ty0; y <= ty1; y++ {
			_, sy := cam.ToScreen(0, float64(y))
			fillRect(dst, 0, sy, float32(cam.W), 1, gc)
		}
	}

	g.drawWater(dst, tx0, ty0, tx1, ty1)
	g.drawVegetation(dst, tx0, ty0, tx1, ty1)
	g.drawRuins(dst)

	view := func(b *sim.Building) bool {
		sz := b.Size()
		return b.Pos.X+sz >= tx0 && b.Pos.X <= tx1 && b.Pos.Y+sz >= ty0 && b.Pos.Y <= ty1
	}
	order := g.w.Order()
	for _, id := range order {
		if b := g.w.Buildings[id]; view(b) {
			g.drawBuilding(dst, b, 1)
		}
	}
	g.drawSettlers(dst)
	g.drawEnemies(dst)
	g.drawShots(dst)
	for _, id := range order {
		if b := g.w.Buildings[id]; b.Dmg > 0 && view(b) {
			g.drawHPBar(dst, b)
		}
	}
	g.drawStationRange(dst)
	g.drawThreats(dst)
}

// drawBuilding draws a machine. alpha < 1 is used for placement ghosts.
func (g *Game) drawBuilding(dst *ebiten.Image, b *sim.Building, alpha float64) {
	cam := &g.cam
	s := float32(cam.Scale)
	sz := float32(b.Size())
	x, y := cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
	w := s * sz
	acc := accentOf(b.Kind)
	fade := func(c color.RGBA) color.RGBA {
		if alpha >= 1 {
			return c
		}
		return color.RGBA{uint8(float64(c.R) * alpha), uint8(float64(c.G) * alpha), uint8(float64(c.B) * alpha), uint8(float64(c.A) * alpha)}
	}
	pad := s * 0.08
	bx, by, bw := x+pad, y+pad, w-2*pad
	t := float64(g.w.Tick) / sim.TPS

	if b.Site && alpha >= 1 {
		g.drawSite(dst, b)
		return
	}

	switch b.Kind {
	case sim.KDam, sim.KLevee:
		c := rgb(0x9C7A55)
		if b.Kind == sim.KLevee {
			c = rgb(0xA89C8C)
		}
		fillRoundRect(dst, x+s*0.02, y+s*0.06, s*0.96, s*0.96, s*0.1, fade(rgba(0x2E3440, 0x26)))
		fillRoundRect(dst, x+s*0.02, y+s*0.02, s*0.96, s*0.96, s*0.1, fade(c))
		lc := fade(mix(c, colInk, 0.3))
		if b.Kind == sim.KDam {
			for k := 1; k < 4; k++ {
				fx := x + s*float32(k)*0.25
				line(dst, fx, y+s*0.08, fx, y+s*0.92, max32(1, s*0.04), lc)
			}
			fillRect(dst, x+s*0.06, y+s*0.06, s*0.88, s*0.12, fade(rgba(0x5DA9D6, 0x99)))
		} else {
			fillRoundRect(dst, x+s*0.16, y+s*0.16, s*0.68, s*0.68, s*0.2, fade(mix(c, colBody, 0.25)))
		}
		return
	case sim.KWall:
		stone := rgb(0xB9AE9E)
		fillRoundRect(dst, x+s*0.03, y+s*0.07, s*0.94, s*0.94, s*0.12, fade(rgba(0x2E3440, 0x26)))
		fillRoundRect(dst, x+s*0.03, y+s*0.03, s*0.94, s*0.94, s*0.12, fade(stone))
		lc := fade(mix(stone, colInk, 0.3))
		lw := max32(1, s*0.04)
		line(dst, x+s*0.06, y+s*0.36, x+s*0.94, y+s*0.36, lw, lc)
		line(dst, x+s*0.06, y+s*0.66, x+s*0.94, y+s*0.66, lw, lc)
		line(dst, x+s*0.5, y+s*0.06, x+s*0.5, y+s*0.36, lw, lc)
		line(dst, x+s*0.28, y+s*0.36, x+s*0.28, y+s*0.66, lw, lc)
		line(dst, x+s*0.72, y+s*0.36, x+s*0.72, y+s*0.66, lw, lc)
		line(dst, x+s*0.5, y+s*0.66, x+s*0.5, y+s*0.94, lw, lc)
		return
	}

	// shadow + body
	fillRoundRect(dst, bx+s*0.03, by+s*0.06, bw, bw, s*0.18, fade(rgba(0x2E3440, 0x22)))
	body := colBody
	if b.Kind == sim.KCore {
		body = rgb(0x3B4252)
	}
	fillRoundRect(dst, bx, by, bw, bw, s*0.18, fade(body))
	strokeRoundRect(dst, bx, by, bw, bw, s*0.18, max32(1, s*0.05), fade(mix(acc, colInk, 0.35)))
	cx, cy := x+w/2, y+w/2

	switch b.Kind.Base() {
	case sim.KMiner:
		fillCircle(dst, cx, cy, w*0.27, fade(mix(colBody, acc, 0.18)))
		rot := 0.0
		if b.Working {
			rot = t * 3
		}
		for k := 0; k < 3; k++ {
			a := rot + float64(k)*2*math.Pi/3
			ex, ey := cx+float32(math.Cos(a))*w*0.24, cy+float32(math.Sin(a))*w*0.24
			line(dst, cx, cy, ex, ey, max32(1.5, s*0.12), fade(acc))
		}
		fillCircle(dst, cx, cy, w*0.08, fade(colInk))
		if b.MinerItem != sim.ItemNone && s >= 12 && b.Size() > 1 {
			drawItem(dst, b.MinerItem, bx+bw*0.14, by+bw*0.14, s*0.32)
		}
	case sim.KFurnace:
		glow := 0.0
		if b.Working {
			glow = 0.55 + 0.45*math.Sin(t*5)*0.5 + 0.2
		}
		mw, mh := w*0.5, w*0.36
		fillRoundRect(dst, cx-mw/2, cy-mh/2+w*0.04, mw, mh, s*0.15, fade(mix(rgb(0x4C566A), acc, glow)))
		fillRect(dst, cx-mw*0.3, cy-w*0.33, mw*0.6, max32(1, s*0.08), fade(mix(colBody, acc, 0.6)))
	case sim.KAssembler:
		rot := 0.0
		if b.Working {
			rot = t * 1.5
		}
		r := w * 0.26
		for k := 0; k < 8; k++ {
			a := rot + float64(k)*math.Pi/4
			fillCircle(dst, cx+float32(math.Cos(a))*r, cy+float32(math.Sin(a))*r, w*0.05, fade(acc))
		}
		strokeCircle(dst, cx, cy, r, max32(1, s*0.08), fade(acc))
	case sim.KChest:
		fillRoundRect(dst, bx+bw*0.1, by+bw*0.18, bw*0.8, bw*0.64, s*0.08, fade(mix(colBody, acc, 0.35)))
		line(dst, bx+bw*0.1, by+bw*0.38, bx+bw*0.9, by+bw*0.38, max32(1, s*0.05), fade(mix(acc, colInk, 0.4)))
		fill := float32(b.StoreCount()) / float32(g.w.StorageCap(b))
		if fill > 0 {
			fillRect(dst, bx+bw*0.15, by+bw*0.76-bw*0.3*fill, bw*0.7, bw*0.3*fill, fade(acc))
		}
	case sim.KHouse:
		roof := fade(rgb(0xC97B4A))
		path.MoveTo(cx, by+bw*0.12)
		path.LineTo(bx+bw*0.88, cy)
		path.LineTo(bx+bw*0.12, cy)
		path.Close()
		fillPath(dst, roof)
		fillRect(dst, bx+bw*0.22, cy, bw*0.56, bw*0.34, fade(mix(colBody, rgb(0xC97B4A), 0.15)))
		fillRoundRect(dst, cx-bw*0.08, cy+bw*0.12, bw*0.16, bw*0.22, s*0.03, fade(rgb(0x6B4F3A)))
	case sim.KFarm:
		field := fade(rgb(0xE8D9A0))
		fillRoundRect(dst, bx+bw*0.08, by+bw*0.08, bw*0.84, bw*0.84, s*0.1, field)
		for row := 0; row < 3; row++ {
			ry := by + bw*(0.25+float32(row)*0.25)
			for k := 0; k < 4; k++ {
				sway := float32(math.Sin(t*2+float64(row+k))) * bw * 0.02
				px := bx + bw*(0.2+float32(k)*0.2)
				line(dst, px, ry+bw*0.08, px+sway, ry-bw*0.06, max32(1, s*0.05), fade(rgb(0x8DB360)))
				if b.Progress > 0.5 {
					fillCircle(dst, px+sway, ry-bw*0.07, max32(1, s*0.04), fade(rgb(0xC9A227)))
				}
			}
		}
	case sim.KLab:
		fh := bw * 0.52
		path.MoveTo(cx-fh*0.14, cy-fh*0.55)
		path.LineTo(cx-fh*0.14, cy-fh*0.1)
		path.LineTo(cx-fh*0.5, cy+fh*0.45)
		path.LineTo(cx+fh*0.5, cy+fh*0.45)
		path.LineTo(cx+fh*0.14, cy-fh*0.1)
		path.LineTo(cx+fh*0.14, cy-fh*0.55)
		strokePath(dst, max32(1.2, s*0.06), fade(mix(acc, colInk, 0.4)))
		path.MoveTo(cx-fh*0.32, cy+fh*0.2)
		path.LineTo(cx-fh*0.44, cy+fh*0.4)
		path.LineTo(cx+fh*0.44, cy+fh*0.4)
		path.LineTo(cx+fh*0.32, cy+fh*0.2)
		path.Close()
		fillPath(dst, fade(acc))
		if b.Working {
			for k := 0; k < 3; k++ {
				ph := math.Mod(t*0.8+float64(k)/3, 1)
				fillCircle(dst, cx+float32(math.Sin(ph*6+float64(k)))*fh*0.12, cy-fh*float32(0.1+ph*0.6), max32(1, s*0.035), fade(rgba(0x8E7CC3, uint8(220*(1-ph)))))
			}
		}
	case sim.KCampfire:
		for k := 0; k < 6; k++ {
			a := float64(k) * math.Pi / 3
			fillCircle(dst, cx+float32(math.Cos(a))*bw*0.3, cy+float32(math.Sin(a))*bw*0.3, bw*0.08, fade(rgb(0x9AA0AC)))
		}
		fl := float32(0.85 + 0.15*math.Sin(t*9))
		path.MoveTo(cx, cy-bw*0.3*fl)
		path.QuadTo(cx+bw*0.2, cy, cx, cy+bw*0.15)
		path.QuadTo(cx-bw*0.2, cy, cx, cy-bw*0.3*fl)
		path.Close()
		fillPath(dst, fade(acc))
		fillCircle(dst, cx, cy+bw*0.02, bw*0.07, fade(rgb(0xF7D774)))
	case sim.KFlowerbed:
		fillRoundRect(dst, cx-bw*0.36, cy-bw*0.36, bw*0.72, bw*0.72, s*0.12, fade(rgb(0x8DB360)))
		cols := []color.RGBA{rgb(0xD97BA6), rgb(0xF2D16B), rgb(0xE07A5F), rgb(0xFBF9F4)}
		for k := 0; k < 5; k++ {
			a := float64(k)*1.256 + 0.3
			fx := cx + float32(math.Cos(a))*bw*0.2
			fy := cy + float32(math.Sin(a))*bw*0.2
			fillCircle(dst, fx, fy, bw*0.09, fade(cols[k%len(cols)]))
		}
	case sim.KStatue:
		fillRoundRect(dst, cx-bw*0.3, cy+bw*0.12, bw*0.6, bw*0.2, s*0.04, fade(rgb(0xB9AE9E)))
		fillCircle(dst, cx, cy-bw*0.22, bw*0.13, fade(acc))
		fillRoundRect(dst, cx-bw*0.15, cy-bw*0.1, bw*0.3, bw*0.24, s*0.06, fade(acc))
	case sim.KBakery:
		fillRoundRect(dst, cx-bw*0.32, cy-bw*0.05, bw*0.64, bw*0.36, s*0.15, fade(rgb(0x6B4F3A)))
		if b.Working {
			fillRoundRect(dst, cx-bw*0.2, cy+bw*0.05, bw*0.4, bw*0.18, s*0.08, fade(rgb(0xF2A65A)))
		}
		drawItem(dst, sim.Bread, cx, cy-bw*0.22, s*0.3)
	case sim.KPump:
		fillRoundRect(dst, cx-bw*0.12, cy-bw*0.3, bw*0.24, bw*0.55, s*0.05, fade(rgb(0x6B7385)))
		line(dst, cx, cy-bw*0.25, cx+bw*0.3, cy-bw*0.35, max32(1.5, s*0.06), fade(colInk))
		drawItem(dst, sim.Water, cx-bw*0.24, cy+bw*0.22, s*0.3)
		if b.Working {
			ph := float32(math.Mod(t*2, 1))
			fillCircle(dst, cx+bw*0.18, cy+bw*(0.05+0.25*ph), max32(1, s*0.04), fade(rgba(0x5DA9D6, uint8(255*(1-ph)))))
		}
	case sim.KLumber:
		// axe
		line(dst, cx-bw*0.2, cy+bw*0.25, cx+bw*0.15, cy-bw*0.2, max32(1.5, s*0.08), fade(rgb(0x6B4F3A)))
		path.MoveTo(cx+bw*0.05, cy-bw*0.3)
		path.LineTo(cx+bw*0.32, cy-bw*0.22)
		path.LineTo(cx+bw*0.22, cy+bw*0.02)
		path.Close()
		fillPath(dst, fade(rgb(0x8C99A6)))
		drawItem(dst, sim.Wood, cx-bw*0.18, cy+bw*0.3, s*0.26)
	case sim.KForester:
		line(dst, cx, cy+bw*0.3, cx, cy-bw*0.05, max32(1.5, s*0.07), fade(rgb(0x6B4F3A)))
		fillCircle(dst, cx-bw*0.12, cy-bw*0.1, bw*0.14, fade(acc))
		fillCircle(dst, cx+bw*0.12, cy-bw*0.16, bw*0.14, fade(mix(acc, colBody, 0.25)))
		fillCircle(dst, cx, cy-bw*0.28, bw*0.14, fade(acc))
	case sim.KGatherer:
		fillRoundRect(dst, cx-bw*0.3, cy-bw*0.05, bw*0.6, bw*0.32, s*0.08, fade(rgb(0xC8A97E)))
		for k := 0; k < 4; k++ {
			fillCircle(dst, cx-bw*0.18+float32(k)*bw*0.12, cy-bw*0.08, bw*0.07, fade(acc))
		}
		line(dst, cx-bw*0.28, cy-bw*0.05, cx, cy-bw*0.35, max32(1, s*0.05), fade(rgb(0x8C6B4A)))
		line(dst, cx, cy-bw*0.35, cx+bw*0.28, cy-bw*0.05, max32(1, s*0.05), fade(rgb(0x8C6B4A)))
	case sim.KSawmill:
		rot := 0.0
		if b.Working {
			rot = t * 4
		}
		r := bw * 0.26
		strokeCircle(dst, cx, cy, r, max32(1, s*0.06), fade(mix(acc, colInk, 0.3)))
		for k := 0; k < 8; k++ {
			a := rot + float64(k)*math.Pi/4
			triangle(dst, cx+float32(math.Cos(a))*r, cy+float32(math.Sin(a))*r, float32(a+math.Pi/2), s*0.12, fade(mix(acc, colInk, 0.3)))
		}
		fillCircle(dst, cx, cy, bw*0.07, fade(colInk))
	case sim.KStation:
		// a little cart
		fillRoundRect(dst, cx-bw*0.3, cy-bw*0.2, bw*0.6, bw*0.32, s*0.06, fade(acc))
		fillCircle(dst, cx-bw*0.18, cy+bw*0.18, bw*0.1, fade(colInk))
		fillCircle(dst, cx+bw*0.18, cy+bw*0.18, bw*0.1, fade(colInk))
		line(dst, cx+bw*0.3, cy-bw*0.12, cx+bw*0.42, cy-bw*0.28, max32(1, s*0.05), fade(colInk))
	case sim.KCore:
		pulse := float32(0.5 + 0.5*math.Sin(t*1.2))
		r := w * (0.2 + 0.02*pulse)
		path.MoveTo(cx, cy-r)
		path.LineTo(cx+r, cy)
		path.LineTo(cx, cy+r)
		path.LineTo(cx-r, cy)
		path.Close()
		fillPath(dst, fade(colOK))
		strokeCircle(dst, cx, cy, w*0.33, max32(1, s*0.05), fade(rgba(0x4FA3A5, 0x88)))
		if s >= 16 {
			drawText(dst, "РАТУША", float64(cx), float64(by+bw*0.06), float64(s)*0.3, true, fade(rgb(0xD8DEE9)), alignCenter)
		}
	}

	if base := b.Kind.Base(); base == sim.KTurret || base == sim.KFlamer || base == sim.KCore {
		g.drawWeapon(dst, b, cx, cy, w, fade)
	}

	// output product badge for crafters
	if b.Kind.IsCrafter() && b.Recipe >= 0 && s >= 12 {
		if b.Size() == 1 {
			drawItem(dst, sim.Recipes[b.Recipe].Out.Item, bx+bw*0.14, by+bw*0.14, s*0.24)
		} else {
			drawItem(dst, sim.Recipes[b.Recipe].Out.Item, bx+bw*0.16, by+bw*0.16, s*0.34)
		}
	}

	if alpha >= 1 && !g.iconMode {
		g.drawStatus(dst, b, x, y, w)
	}
}

// drawStatus marks paused buildings and missing workers.
func (g *Game) drawStatus(dst *ebiten.Image, b *sim.Building, x, y, w float32) {
	s := float32(g.cam.Scale)
	if s < 14 {
		return
	}
	r := max32(5, s*0.17)
	cx, cy := x+w-r*0.6, y+r*0.6
	switch {
	case b.IsFlooded():
		fillCircle(dst, cx, cy, r, rgba(0x2F7FB8, 0xE6))
		drawItem(dst, sim.Water, cx, cy, r*1.3)
	case b.Kind == sim.KFarm && !b.Site && !g.w.MoistAt(b.Pos):
		fillCircle(dst, cx, cy, r, rgba(0xC77B2E, 0xE6))
		fillRect(dst, cx-r*0.5, cy-r*0.08, r, r*0.16, colBody)
	case b.Paused:
		fillCircle(dst, cx, cy, r, rgba(0x2E3440, 0xDD))
		fillRect(dst, cx-r*0.4, cy-r*0.45, r*0.28, r*0.9, colBG)
		fillRect(dst, cx+r*0.12, cy-r*0.45, r*0.28, r*0.9, colBG)
	case sim.Builds[b.Kind].Workers > 0 && b.Kind != sim.KStation && !(b.Kind.IsFieldKind() && b.Staff() >= sim.Builds[b.Kind].Workers) && b.Inside() < sim.Builds[b.Kind].Workers:
		fillCircle(dst, cx, cy, r, rgba(0xE07A5F, 0xE6))
		personGlyph(dst, cx, cy, r*0.9, colBody)
	}
}

func personGlyph(dst *ebiten.Image, cx, cy, r float32, c color.RGBA) {
	fillCircle(dst, cx, cy-r*0.35, r*0.25, c)
	fillRoundRect(dst, cx-r*0.32, cy-r*0.05, r*0.64, r*0.55, r*0.2, c)
}

// drawSite shows a construction site: a faint outline of the future
// building with progress.
func (g *Game) drawSite(dst *ebiten.Image, b *sim.Building) {
	s := float32(g.cam.Scale)
	x, y := g.cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
	w := s * float32(b.Size())
	g.drawBuilding(dst, b, 0.35)
	// scaffolding corners
	c := rgba(0xA3785B, 0xDD)
	lw := max32(1.5, s*0.06)
	m := w * 0.22
	for _, p := range [][4]float32{{x + 2, y + 2, 1, 1}, {x + w - 2, y + 2, -1, 1}, {x + 2, y + w - 2, 1, -1}, {x + w - 2, y + w - 2, -1, -1}} {
		line(dst, p[0], p[1], p[0]+p[2]*m, p[1], lw, c)
		line(dst, p[0], p[1], p[0], p[1]+p[3]*m, lw, c)
	}
	// progress: materials first, then building work
	total, left := 0, 0
	for _, st := range sim.Builds[b.Kind].Cost {
		total += st.N
	}
	for _, st := range b.Need {
		left += st.N
	}
	frac := float32(1)
	if total > 0 && !b.FreeBuilt {
		frac = float32(total-left) / float32(total)
	}
	bx, by, bw := x+w*0.12, y+w-max32(5, s*0.16), w*0.76
	bh := max32(3, s*0.08)
	fillRoundRect(dst, bx, by, bw, bh, bh/2, rgba(0x2E3440, 0x66))
	fillRoundRect(dst, bx, by, bw*frac*0.5+bw*0.5*b.BuildWork, bh, bh/2, rgb(0xC9A227))
}

// drawSettlers draws everyone who is outside a building.
func (g *Game) drawSettlers(dst *ebiten.Image) {
	s := float32(g.cam.Scale)
	t := float64(g.w.Tick) / sim.TPS
	for _, st := range g.w.Settlers {
		if st.Inside {
			continue
		}
		x, y := g.cam.ToScreen(st.X, st.Y)
		if float64(x) < -20 || float64(y) < -20 || float64(x) > g.cam.W+20 || float64(y) > g.cam.H+20 {
			continue
		}
		r := max32(3, s*0.18)
		body := rgb(0x4FA3A5) // free
		switch {
		case g.isCarrier(st):
			body = rgb(0xD4A373)
		case st.Job != 0:
			body = rgb(0x6C8EBF)
		case st.Task == sim.TaskBuild:
			body = rgb(0xA3785B)
		}
		bob := float32(math.Sin(t*9+float64(st.Seed))) * r * 0.15
		fillCircle(dst, x, y+r*0.9, r*0.9, rgba(0x2E3440, 0x22))
		fillRoundRect(dst, x-r*0.7, y-r*0.2+bob, r*1.4, r*1.3, r*0.5, body)
		fillCircle(dst, x, y-r*0.65+bob, r*0.55, rgb(0xF1D9C0))
		if st.Starved() {
			strokeCircle(dst, x, y, r*1.5, max32(1, s*0.03), colBad)
		}
		if st.Carry.N > 0 {
			drawItem(dst, st.Carry.Item, x, y-r*1.7+bob, max32(6, s*0.3))
		}
		if st.Task == sim.TaskBuild && s >= 16 {
			a := float32(math.Sin(t * 10))
			line(dst, x+r*0.6, y-r*0.2, x+r*1.3, y-r*(0.8+0.4*a), max32(1, s*0.04), colInk)
		}
	}
}

func (g *Game) isCarrier(st *sim.Settler) bool {
	b := g.w.Buildings[st.Job]
	return b != nil && b.Kind == sim.KStation
}

// workRadius is the reach of stations and field huts (0 for others).
func workRadius(k sim.Kind) float64 {
	switch {
	case k == sim.KStation:
		return sim.StationRadius
	case k.IsFieldKind():
		return sim.FieldRadius + 0.5
	case k == sim.KCampfire:
		return sim.LeisureRadius
	case k == sim.KFlowerbed:
		return sim.BeautyRadius
	case k == sim.KStatue:
		return sim.StatueRadius
	}
	return 0
}

// drawStationRange shows the reach of the selected or hovered station or
// field hut, or of the one about to be placed.
func (g *Game) drawStationRange(dst *ebiten.Image) {
	var k sim.Kind
	var x, y float64
	if sel := g.w.Buildings[g.selected]; sel != nil && workRadius(sel.Kind) > 0 {
		k, x, y = sel.Kind, float64(sel.Pos.X)+0.5, float64(sel.Pos.Y)+0.5
	} else if h := g.w.At(g.hover); h != nil && workRadius(h.Kind) > 0 && g.tool == sim.KNone {
		k, x, y = h.Kind, float64(h.Pos.X)+0.5, float64(h.Pos.Y)+0.5
	} else if workRadius(g.tool) > 0 && !g.overUI {
		p := g.footprint(g.tool)
		k, x, y = g.tool, float64(p.X)+0.5, float64(p.Y)+0.5
	} else {
		return
	}
	s := float32(g.cam.Scale)
	cx, cy := g.cam.ToScreen(x, y)
	c := accentOf(k)
	r := s * float32(workRadius(k))
	fillCircle(dst, cx, cy, r, color.RGBA{c.R / 10, c.G / 10, c.B / 10, 0x18})
	strokeCircle(dst, cx, cy, r, max32(1.5, s*0.05), color.RGBA{uint8(int(c.R) * 0xAA / 255), uint8(int(c.G) * 0xAA / 255), uint8(int(c.B) * 0xAA / 255), 0xAA})
}

// drawVegetation draws trees and berry bushes.
func (g *Game) drawVegetation(dst *ebiten.Image, tx0, ty0, tx1, ty1 int) {
	s := float32(g.cam.Scale)
	t := float64(g.w.Tick) / sim.TPS
	for y := ty0; y <= ty1; y++ {
		for x := tx0; x <= tx1; x++ {
			p := sim.Pos{X: x, Y: y}
			v, grow := g.w.VegAt(p)
			if v == sim.VegNone || g.w.At(p) != nil {
				continue
			}
			sx, sy := g.cam.ToScreen(float64(x)+0.5, float64(y)+0.5)
			h := uint32(x*73856093) ^ uint32(y*19349663)
			jx := (float32(h&0xFF)/255 - 0.5) * s * 0.2
			jy := (float32(h>>8&0xFF)/255 - 0.5) * s * 0.2
			sx, sy = sx+jx, sy+jy
			switch v {
			case sim.VegTree:
				size := float32(0.25 + 0.75*grow)
				r := s * 0.42 * size
				sway := float32(math.Sin(t*0.8+float64(h%100))) * s * 0.015
				fillCircle(dst, sx+s*0.05, sy+s*0.1, r, rgba(0x2E3440, 0x22))
				if s >= 10 {
					fillRect(dst, sx-s*0.05, sy, s*0.1, s*0.25*size, rgb(0x7A5536))
				}
				dark, light := rgb(0x4E8A55), rgb(0x6FA06A)
				if h%3 == 0 {
					dark, light = rgb(0x3F7A4A), rgb(0x5E9A62)
				}
				fillCircle(dst, sx+sway, sy-s*0.08*size, r, dark)
				fillCircle(dst, sx-r*0.25+sway, sy-s*0.08*size-r*0.25, r*0.62, light)
			case sim.VegBush:
				r := s * 0.3
				fillCircle(dst, sx+s*0.04, sy+s*0.07, r, rgba(0x2E3440, 0x20))
				fillCircle(dst, sx, sy, r, rgb(0x5E9A62))
				fillCircle(dst, sx-r*0.3, sy-r*0.3, r*0.55, rgb(0x76B06F))
				if grow >= 1 && s >= 10 {
					for k := 0; k < 4; k++ {
						a := float64(k)*1.7 + float64(h%7)
						fillCircle(dst, sx+float32(math.Cos(a))*r*0.55, sy+float32(math.Sin(a))*r*0.55, max32(1.2, s*0.05), rgb(0xC0504D))
					}
				}
			}
		}
	}
}

var heightTint = [5]color.RGBA{
	rgba(0xB8A98A, 0x90), // river bed
	rgba(0xCDBF9F, 0x60), // bank
	{},                   // plain
	rgba(0xFFFFFF, 0x30),
	rgba(0xFFFFFF, 0x58),
}

// drawTerrain tints the ground by height and shades the steps between levels.
func (g *Game) drawTerrain(dst *ebiten.Image, tx0, ty0, tx1, ty1 int) {
	s := float32(g.cam.Scale)
	w := g.w
	for y := ty0; y <= ty1; y++ {
		for x := tx0; x <= tx1; x++ {
			p := sim.Pos{X: x, Y: y}
			h := w.HeightAt(p)
			sx, sy := g.cam.ToScreen(float64(x), float64(y))
			if h != 2 && h >= 0 && h < len(heightTint) {
				fillRect(dst, sx, sy, s+0.5, s+0.5, heightTint[h])
			}
			if w.Drought && h >= 1 {
				fillRect(dst, sx, sy, s+0.5, s+0.5, rgba(0xE0B865, 0x1E))
			} else if h >= 1 && w.MoistAt(p) && w.WaterAt(p) < 0.05 {
				fillRect(dst, sx, sy, s+0.5, s+0.5, rgba(0x7FAE5A, 0x22))
			}
			// steps between heights: shadow below, highlight on top
			if hb := w.HeightAt(sim.Pos{X: x, Y: y + 1}); hb < h {
				fillRect(dst, sx, sy+s*0.82, s+0.5, s*0.18, rgba(0x2E3440, uint8(0x22*min(h-hb, 3))))
			}
			if hr := w.HeightAt(sim.Pos{X: x + 1, Y: y}); hr < h {
				fillRect(dst, sx+s*0.88, sy, s*0.12, s+0.5, rgba(0x2E3440, uint8(0x16*min(h-hr, 3))))
			}
		}
	}
}

// drawWater draws water depth with a gentle shimmer.
func (g *Game) drawWater(dst *ebiten.Image, tx0, ty0, tx1, ty1 int) {
	s := float32(g.cam.Scale)
	t := float64(g.w.Tick) / sim.TPS
	shallow, deep := rgb(0x8CCBE8), rgb(0x2F7FB8)
	for y := ty0; y <= ty1; y++ {
		for x := tx0; x <= tx1; x++ {
			d := g.w.WaterAt(sim.Pos{X: x, Y: y})
			if d < 0.02 {
				continue
			}
			sx, sy := g.cam.ToScreen(float64(x), float64(y))
			k := math.Min(1, d/1.6)
			c := mix(shallow, deep, k)
			a := 0.45 + 0.45*math.Min(1, d*1.5)
			c = color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(255 * a)}
			fillRect(dst, sx, sy, s+0.5, s+0.5, c)
			if s >= 14 && d > 0.15 {
				ph := math.Mod(t*0.6+float64(x)*0.37+float64(y)*0.21, 1)
				lx := sx + s*float32(ph)
				line(dst, lx-s*0.18, sy+s*0.4, lx+s*0.12, sy+s*0.4, max32(1, s*0.04), rgba(0xFFFFFF, 0x44))
			}
		}
	}
}
