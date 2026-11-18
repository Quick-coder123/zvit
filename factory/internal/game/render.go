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
	cam := &g.cam
	x0, y0 := cam.ToWorld(0, 0)
	x1, y1 := cam.ToWorld(cam.W, cam.H)
	tx0, ty0 := int(math.Floor(x0))-1, int(math.Floor(y0))-1
	tx1, ty1 := int(math.Ceil(x1))+1, int(math.Ceil(y1))+1
	s := float32(cam.Scale)

	// grid
	if cam.Scale >= 18 {
		gc := colGrid
		for x := tx0; x <= tx1; x++ {
			sx, _ := cam.ToScreen(float64(x), 0)
			fillRect(dst, sx, 0, 1, float32(cam.W), gc)
		}
		for y := ty0; y <= ty1; y++ {
			_, sy := cam.ToScreen(0, float64(y))
			fillRect(dst, 0, sy, float32(cam.W), 1, gc)
		}
	}

	// ore deposits
	for y := ty0; y <= ty1; y++ {
		for x := tx0; x <= tx1; x++ {
			it := g.w.OreAt(sim.Pos{X: x, Y: y})
			if it == sim.ItemNone {
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

	g.drawRuins(dst)

	view := func(b *sim.Building) bool {
		sz := b.Size()
		return b.Pos.X+sz >= tx0 && b.Pos.X <= tx1 && b.Pos.Y+sz >= ty0 && b.Pos.Y <= ty1
	}
	order := g.w.Order()
	// belts first, then their items, then machines on top
	for _, id := range order {
		b := g.w.Buildings[id]
		if b.Kind == sim.KBelt && view(b) {
			g.drawBelt(dst, b)
		}
	}
	for _, id := range order {
		b := g.w.Buildings[id]
		if b.Kind == sim.KBelt && view(b) {
			for _, bi := range b.Belt {
				x, y := beltItemPos(b, bi.P)
				sx, sy := cam.ToScreen(x, y)
				drawItem(dst, bi.Item, sx, sy, s*0.38)
			}
		}
	}
	for _, id := range order {
		b := g.w.Buildings[id]
		if b.Kind != sim.KBelt && view(b) {
			g.drawBuilding(dst, b, 1)
		}
	}
	g.drawEnemies(dst)
	g.drawShots(dst)
	for _, id := range order {
		if b := g.w.Buildings[id]; b.Dmg > 0 && view(b) {
			g.drawHPBar(dst, b)
		}
	}
	g.drawThreats(dst)
}

// beltItemPos returns tile-space coordinates for an item at progress p.
func beltItemPos(b *sim.Building, p float32) (float64, float64) {
	cx, cy := float64(b.Pos.X)+0.5, float64(b.Pos.Y)+0.5
	d := b.Dir
	dx, dy := float64(d.DX()), float64(d.DY())
	if b.Curve == 0 {
		t := float64(p) - 0.5
		return cx + dx*t, cy + dy*t
	}
	side := d.CCW()
	if b.Curve == 1 {
		side = d.CW()
	}
	sx, sy := float64(side.DX()), float64(side.DY())
	// arc centred on the corner between entry side and exit side
	ccx, ccy := cx+sx*0.5+dx*0.5, cy+sy*0.5+dy*0.5
	a := float64(p) * math.Pi / 2
	ux, uy := -dx, -dy
	vx, vy := -sx, -sy
	return ccx + 0.5*(ux*math.Cos(a)+vx*math.Sin(a)), ccy + 0.5*(uy*math.Cos(a)+vy*math.Sin(a))
}

func (g *Game) drawBelt(dst *ebiten.Image, b *sim.Building) {
	cam := &g.cam
	s := float32(cam.Scale)
	sx, sy := cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
	phase := float32(math.Mod(float64(g.w.Tick)*sim.BeltSpeed/sim.TPS, 0.5))
	if b.Curve == 0 {
		if b.Dir == sim.North || b.Dir == sim.South {
			fillRoundRect(dst, sx+s*0.1, sy+s*0.02, s*0.8, s*0.96, s*0.1, colBeltBase)
		} else {
			fillRoundRect(dst, sx+s*0.02, sy+s*0.1, s*0.96, s*0.8, s*0.1, colBeltBase)
		}
		if s >= 10 {
			a := dirAngle(b.Dir)
			for k := 0; k < 2; k++ {
				t := float32(k)*0.5 + phase + 0.25
				x, y := beltItemPos(b, t)
				px, py := cam.ToScreen(x, y)
				chevron(dst, px, py, a, s*0.3, max32(1, s*0.06), colChevron)
			}
		}
		return
	}
	// curved belt: thick quarter arc
	const steps = 10
	for i := 0; i <= steps; i++ {
		x, y := beltItemPos(b, float32(i)/steps)
		px, py := cam.ToScreen(x, y)
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	strokePath(dst, s*0.8, colBeltBase)
	if s >= 10 {
		for k := 0; k < 2; k++ {
			t := float32(k)*0.5 + phase + 0.25
			x, y := beltItemPos(b, t)
			x2, y2 := beltItemPos(b, t+0.01)
			px, py := cam.ToScreen(x, y)
			a := float32(math.Atan2(y2-y, x2-x))
			chevron(dst, px, py, a, s*0.3, max32(1, s*0.06), colChevron)
		}
	}
}

// drawBuilding draws a machine. alpha < 1 is used for placement ghosts.
func (g *Game) drawBuilding(dst *ebiten.Image, b *sim.Building, alpha float64) {
	cam := &g.cam
	s := float32(cam.Scale)
	sz := float32(b.Size())
	x, y := cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
	w := s * sz
	acc := colAccent[b.Kind]
	fade := func(c color.RGBA) color.RGBA {
		if alpha >= 1 {
			return c
		}
		return color.RGBA{uint8(float64(c.R) * alpha), uint8(float64(c.G) * alpha), uint8(float64(c.B) * alpha), uint8(float64(c.A) * alpha)}
	}
	pad := s * 0.08
	bx, by, bw := x+pad, y+pad, w-2*pad
	t := float64(g.w.Tick) / sim.TPS

	switch b.Kind {
	case sim.KBelt:
		fillRoundRect(dst, bx, by, bw, bw, s*0.12, fade(colBeltBase))
		triangle(dst, x+w/2, y+w/2, dirAngle(b.Dir), s*0.4, fade(colChevron))
		return
	case sim.KTunnel:
		g.drawTunnel(dst, b, x, y, s, fade)
		return
	case sim.KSplitter, sim.KMerger, sim.KSorter:
		g.drawRouter(dst, b, x, y, s, fade)
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

	switch b.Kind {
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
		if b.MinerItem != sim.ItemNone && s >= 12 {
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
		fill := float32(b.StoreCount()) / sim.ChestCap
		if fill > 0 {
			fillRect(dst, bx+bw*0.15, by+bw*0.76-bw*0.3*fill, bw*0.7, bw*0.3*fill, fade(acc))
		}
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
			drawText(dst, "ЯДРО", float64(cx), float64(by+bw*0.06), float64(s)*0.32, true, fade(rgb(0xD8DEE9)), alignCenter)
		}
	}

	if b.Kind == sim.KTurret || b.Kind == sim.KFlamer || b.Kind == sim.KCore {
		g.drawWeapon(dst, b, cx, cy, w, fade)
	}

	// output product badge for crafters
	if (b.Kind == sim.KFurnace || b.Kind == sim.KAssembler) && b.Recipe >= 0 && s >= 12 {
		drawItem(dst, sim.Recipes[b.Recipe].Out.Item, bx+bw*0.16, by+bw*0.16, s*0.34)
	}

	// output arrow on the front face
	if b.Kind != sim.KCore && b.Kind != sim.KTurret && b.Kind != sim.KFlamer {
		a := dirAngle(b.Dir)
		ax := cx + float32(b.Dir.DX())*(bw/2-s*0.02)
		ay := cy + float32(b.Dir.DY())*(bw/2-s*0.02)
		triangle(dst, ax, ay, a, s*0.28, fade(mix(acc, colInk, 0.35)))
	}
}

// straightBase draws a belt strip across a 1x1 tile in direction d.
func straightBase(dst *ebiten.Image, x, y, s float32, d sim.Dir, c color.RGBA) {
	if d == sim.North || d == sim.South {
		fillRoundRect(dst, x+s*0.1, y+s*0.02, s*0.8, s*0.96, s*0.1, c)
	} else {
		fillRoundRect(dst, x+s*0.02, y+s*0.1, s*0.96, s*0.8, s*0.1, c)
	}
}

// halfRect returns the half of a tile on side d (toward d).
func halfRect(x, y, s float32, d sim.Dir) (float32, float32, float32, float32) {
	switch d {
	case sim.East:
		return x + s*0.46, y + s*0.06, s * 0.5, s * 0.88
	case sim.West:
		return x + s*0.04, y + s*0.06, s * 0.5, s * 0.88
	case sim.South:
		return x + s*0.06, y + s*0.46, s * 0.88, s * 0.5
	default:
		return x + s*0.06, y + s*0.04, s * 0.88, s * 0.5
	}
}

func (g *Game) drawTunnel(dst *ebiten.Image, b *sim.Building, x, y, s float32, fade func(color.RGBA) color.RGBA) {
	acc := colAccent[sim.KTunnel]
	straightBase(dst, x, y, s, b.Dir, fade(colBeltBase))
	side := b.Dir
	if b.Exit {
		side = b.Dir.Opp()
	}
	hx, hy, hw, hh := halfRect(x, y, s, side)
	fillRoundRect(dst, hx+s*0.02, hy+s*0.05, hw, hh, s*0.14, fade(rgba(0x2E3440, 0x22)))
	fillRoundRect(dst, hx, hy, hw, hh, s*0.14, fade(acc))
	// dark mouth facing the open belt side
	cx, cy := hx+hw/2, hy+hh/2
	ox := -float32(side.DX()) * s * 0.17
	oy := -float32(side.DY()) * s * 0.17
	if side == sim.East || side == sim.West {
		fillRoundRect(dst, cx+ox-s*0.06, cy-s*0.3, s*0.12, s*0.6, s*0.06, fade(colInk))
	} else {
		fillRoundRect(dst, cx-s*0.3, cy+oy-s*0.06, s*0.6, s*0.12, s*0.06, fade(colInk))
	}
	triangle(dst, cx-ox*0.2, cy-oy*0.2, dirAngle(b.Dir), s*0.22, fade(mix(acc, colBG, 0.7)))
	if b.Exit {
		return
	}
	// show the connection when relevant
	if !g.iconMode && (g.tool == sim.KTunnel || g.hover == b.Pos) {
		ex, k := g.w.TunnelExit(b)
		c := rgba(0x5E6B78, 0x99)
		if ex == nil {
			c = rgba(0xE07A5F, 0x99)
			k = sim.TunnelMax + 1
		}
		for i := 1; i < k; i++ {
			p := b.Pos
			for j := 0; j < i; j++ {
				p = p.Step(b.Dir)
			}
			px, py := g.cam.ToScreen(float64(p.X)+0.5, float64(p.Y)+0.5)
			fillCircle(dst, px, py, s*0.07, fade(c))
		}
	}
}

// arrowTo draws a line from (cx,cy) toward direction d with an arrowhead.
func arrowTo(dst *ebiten.Image, cx, cy, s float32, d sim.Dir, out bool, c color.RGBA) {
	ex := cx + float32(d.DX())*s*0.34
	ey := cy + float32(d.DY())*s*0.34
	line(dst, cx, cy, ex, ey, max32(1.2, s*0.07), c)
	a := dirAngle(d)
	if out {
		triangle(dst, ex, ey, a, s*0.2, c)
	} else {
		triangle(dst, (cx+ex)/2, (cy+ey)/2, a+math.Pi, s*0.18, c)
	}
}

func (g *Game) drawRouter(dst *ebiten.Image, b *sim.Building, x, y, s float32, fade func(color.RGBA) color.RGBA) {
	acc := colAccent[b.Kind]
	fillRoundRect(dst, x+s*0.04, y+s*0.04, s*0.92, s*0.92, s*0.14, fade(colBeltBase))
	fillRoundRect(dst, x+s*0.12, y+s*0.15, s*0.76, s*0.76, s*0.16, fade(rgba(0x2E3440, 0x20)))
	fillRoundRect(dst, x+s*0.12, y+s*0.12, s*0.76, s*0.76, s*0.16, fade(colBody))
	strokeRoundRect(dst, x+s*0.12, y+s*0.12, s*0.76, s*0.76, s*0.16, max32(1, s*0.05), fade(mix(acc, colInk, 0.3)))
	cx, cy := x+s/2, y+s/2
	ink := fade(mix(acc, colInk, 0.35))
	switch b.Kind {
	case sim.KSplitter:
		arrowTo(dst, cx, cy, s, b.Dir.CCW(), true, fade(acc))
		arrowTo(dst, cx, cy, s, b.Dir.CW(), true, fade(acc))
		arrowTo(dst, cx, cy, s, b.Dir, true, ink)
		fillCircle(dst, cx, cy, s*0.08, ink)
	case sim.KMerger:
		arrowTo(dst, cx, cy, s, b.Dir.CCW(), false, fade(acc))
		arrowTo(dst, cx, cy, s, b.Dir.CW(), false, fade(acc))
		arrowTo(dst, cx, cy, s, b.Dir.Opp(), false, fade(acc))
		arrowTo(dst, cx, cy, s, b.Dir, true, ink)
	case sim.KSorter:
		for _, d := range []sim.Dir{b.Dir.CCW(), b.Dir.CW()} {
			ex := cx + float32(d.DX())*s*0.3
			ey := cy + float32(d.DY())*s*0.3
			triangle(dst, ex, ey, dirAngle(d), s*0.14, fade(acc))
		}
		ex := cx + float32(b.Dir.DX())*s*0.3
		ey := cy + float32(b.Dir.DY())*s*0.3
		triangle(dst, ex, ey, dirAngle(b.Dir), s*0.16, ink)
		if b.Filter != sim.ItemNone {
			drawItem(dst, b.Filter, cx, cy, s*0.32)
		} else if s >= 16 {
			drawText(dst, "?", float64(cx), float64(cy-s*0.2), float64(s)*0.34, true, ink, alignCenter)
		}
	}
}
