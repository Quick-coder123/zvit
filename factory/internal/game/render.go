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

	if b.Kind == sim.KBelt {
		fillRoundRect(dst, bx, by, bw, bw, s*0.12, fade(colBeltBase))
		triangle(dst, x+w/2, y+w/2, dirAngle(b.Dir), s*0.4, fade(colChevron))
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

	// output product badge for crafters
	if (b.Kind == sim.KFurnace || b.Kind == sim.KAssembler) && b.Recipe >= 0 && s >= 12 {
		drawItem(dst, sim.Recipes[b.Recipe].Out.Item, bx+bw*0.16, by+bw*0.16, s*0.34)
	}

	// output arrow on the front face
	if b.Kind != sim.KCore {
		a := dirAngle(b.Dir)
		ax := cx + float32(b.Dir.DX())*(bw/2-s*0.02)
		ay := cy + float32(b.Dir.DY())*(bw/2-s*0.02)
		triangle(dst, ax, ay, a, s*0.28, fade(mix(acc, colInk, 0.35)))
	}
}
