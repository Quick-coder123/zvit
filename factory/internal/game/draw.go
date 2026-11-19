package game

import (
	"bytes"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"potik/internal/sim"
)

func rgb(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF}
}

func rgba(v uint32, a uint8) color.RGBA {
	c := rgb(v)
	f := float64(a) / 255
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), a}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

var (
	colBG        = rgb(0xF4F1EA)
	colGrid      = rgb(0xE8E2D6)
	colInk       = rgb(0x2E3440)
	colInkSoft   = rgb(0x6B7385)
	colPanel     = rgba(0xFFFDF8, 0xF2)
	colPanelLine = rgb(0xDDD5C6)
	colBeltBase  = rgb(0xD8D1C4)
	colBeltEdge  = rgb(0xC2B9A9)
	colChevron   = rgb(0xBDB3A2)
	colBody      = rgb(0xFBF9F4)
	colOK        = rgb(0x4FA3A5)
	colBad       = rgb(0xE07A5F)
	colAccent    = map[sim.Kind]color.RGBA{
		sim.KMiner:     rgb(0x4FA3A5),
		sim.KFurnace:   rgb(0xF2A65A),
		sim.KAssembler: rgb(0x6C8EBF),
		sim.KChest:     rgb(0xC8A97E),
		sim.KCore:      rgb(0x2E3440),
		sim.KHouse:     rgb(0xC97B4A),
		sim.KFarm:      rgb(0x8DB360),
		sim.KStation:   rgb(0xD4A373),
		sim.KLab:       rgb(0x8E7CC3),
		sim.KLumber:    rgb(0x9C6B45),
		sim.KForester:  rgb(0x6FA06A),
		sim.KGatherer:  rgb(0xC0504D),
		sim.KSawmill:   rgb(0xB08A5A),
	}
)

// --- primitive shapes -------------------------------------------------------

var path vector.Path

func fillPath(dst *ebiten.Image, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, &path, nil, op)
	path.Reset()
}

func strokePath(dst *ebiten.Image, w float32, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst, &path, &vector.StrokeOptions{Width: w, LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}, op)
	path.Reset()
}

func roundRectPath(x, y, w, h, r float32) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	path.MoveTo(x+r, y)
	path.LineTo(x+w-r, y)
	path.ArcTo(x+w, y, x+w, y+r, r)
	path.LineTo(x+w, y+h-r)
	path.ArcTo(x+w, y+h, x+w-r, y+h, r)
	path.LineTo(x+r, y+h)
	path.ArcTo(x, y+h, x, y+h-r, r)
	path.LineTo(x, y+r)
	path.ArcTo(x, y, x+r, y, r)
	path.Close()
}

func fillRoundRect(dst *ebiten.Image, x, y, w, h, r float32, c color.Color) {
	roundRectPath(x, y, w, h, r)
	fillPath(dst, c)
}

func strokeRoundRect(dst *ebiten.Image, x, y, w, h, r, sw float32, c color.Color) {
	roundRectPath(x, y, w, h, r)
	strokePath(dst, sw, c)
}

func fillCircle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	vector.FillCircle(dst, x, y, r, c, true)
}

func strokeCircle(dst *ebiten.Image, x, y, r, w float32, c color.Color) {
	vector.StrokeCircle(dst, x, y, r, w, c, true)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float32, c color.Color) {
	vector.StrokeLine(dst, x0, y0, x1, y1, w, c, true)
}

func fillRect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, false)
}

// chevron draws a ">" pointing along angle a.
func chevron(dst *ebiten.Image, cx, cy, a, s, w float32, c color.Color) {
	ca, sa := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	rot := func(x, y float32) (float32, float32) { return cx + x*ca - y*sa, cy + x*sa + y*ca }
	x0, y0 := rot(-s*0.35, -s*0.5)
	x1, y1 := rot(s*0.25, 0)
	x2, y2 := rot(-s*0.35, s*0.5)
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	path.LineTo(x2, y2)
	strokePath(dst, w, c)
}

// triangle pointing along angle a.
func triangle(dst *ebiten.Image, cx, cy, a, s float32, c color.Color) {
	ca, sa := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	rot := func(x, y float32) (float32, float32) { return cx + x*ca - y*sa, cy + x*sa + y*ca }
	x0, y0 := rot(s*0.5, 0)
	x1, y1 := rot(-s*0.4, -s*0.5)
	x2, y2 := rot(-s*0.4, s*0.5)
	path.MoveTo(x0, y0)
	path.LineTo(x1, y1)
	path.LineTo(x2, y2)
	path.Close()
	fillPath(dst, c)
}

func dirAngle(d sim.Dir) float32 { return float32(d) * math.Pi / 2 }

// drawItem draws an item icon centred at (cx,cy) fitting in size s.
func drawItem(dst *ebiten.Image, it sim.Item, cx, cy, s float32) {
	if it == sim.ItemNone || it >= sim.ItemCount {
		return
	}
	def := sim.Items[it]
	c := rgb(def.Color)
	dark := mix(c, colInk, 0.45)
	h := s / 2
	switch def.Shape {
	case sim.ShapeOre:
		path.MoveTo(cx-h*0.8, cy-h*0.2)
		path.LineTo(cx-h*0.3, cy-h*0.85)
		path.LineTo(cx+h*0.55, cy-h*0.7)
		path.LineTo(cx+h*0.9, cy+h*0.1)
		path.LineTo(cx+h*0.4, cy+h*0.8)
		path.LineTo(cx-h*0.5, cy+h*0.75)
		path.Close()
		fillPath(dst, c)
		path.MoveTo(cx-h*0.8, cy-h*0.2)
		path.LineTo(cx-h*0.3, cy-h*0.85)
		path.LineTo(cx+h*0.55, cy-h*0.7)
		path.LineTo(cx+h*0.9, cy+h*0.1)
		path.LineTo(cx+h*0.4, cy+h*0.8)
		path.LineTo(cx-h*0.5, cy+h*0.75)
		path.Close()
		strokePath(dst, max32(1, s*0.07), dark)
	case sim.ShapePlate:
		fillRoundRect(dst, cx-h*0.8, cy-h*0.8, h*1.6, h*1.6, h*0.25, c)
		strokeRoundRect(dst, cx-h*0.8, cy-h*0.8, h*1.6, h*1.6, h*0.25, max32(1, s*0.07), dark)
	case sim.ShapeBar:
		fillRoundRect(dst, cx-h*0.9, cy-h*0.5, h*1.8, h, h*0.2, c)
		strokeRoundRect(dst, cx-h*0.9, cy-h*0.5, h*1.8, h, h*0.2, max32(1, s*0.07), dark)
	case sim.ShapeGem:
		path.MoveTo(cx, cy-h*0.9)
		path.LineTo(cx+h*0.8, cy)
		path.LineTo(cx, cy+h*0.9)
		path.LineTo(cx-h*0.8, cy)
		path.Close()
		fillPath(dst, c)
		path.MoveTo(cx, cy-h*0.9)
		path.LineTo(cx+h*0.8, cy)
		path.LineTo(cx, cy+h*0.9)
		path.LineTo(cx-h*0.8, cy)
		path.Close()
		strokePath(dst, max32(1, s*0.07), dark)
	case sim.ShapeGear:
		fillCircle(dst, cx, cy, h*0.85, dark)
		fillCircle(dst, cx, cy, h*0.7, c)
		fillCircle(dst, cx, cy, h*0.28, dark)
	case sim.ShapeAmmo:
		for _, off := range []float32{-0.35, 0.35} {
			bx := cx + h*off
			fillRoundRect(dst, bx-h*0.24, cy-h*0.35, h*0.48, h*1.15, h*0.1, c)
			path.MoveTo(bx-h*0.24, cy-h*0.35)
			path.LineTo(bx, cy-h*0.95)
			path.LineTo(bx+h*0.24, cy-h*0.35)
			path.Close()
			fillPath(dst, rgb(0xB87333))
			strokeRoundRect(dst, bx-h*0.24, cy-h*0.35, h*0.48, h*1.15, h*0.1, max32(1, s*0.05), dark)
		}
	case sim.ShapeFood:
		fillRoundRect(dst, cx-h*0.6, cy-h*0.45, h*1.2, h*1.25, h*0.45, c)
		strokeRoundRect(dst, cx-h*0.6, cy-h*0.45, h*1.2, h*1.25, h*0.45, max32(1, s*0.06), dark)
		line(dst, cx-h*0.35, cy-h*0.45, cx+h*0.35, cy-h*0.45, max32(1, s*0.08), dark)
		line(dst, cx, cy-h*0.45, cx-h*0.2, cy-h*0.85, max32(1, s*0.06), rgb(0x8DB360))
		line(dst, cx, cy-h*0.45, cx+h*0.25, cy-h*0.8, max32(1, s*0.06), rgb(0x8DB360))
	case sim.ShapeLog:
		fillRoundRect(dst, cx-h*0.9, cy-h*0.4, h*1.8, h*0.8, h*0.4, c)
		strokeRoundRect(dst, cx-h*0.9, cy-h*0.4, h*1.8, h*0.8, h*0.4, max32(1, s*0.06), dark)
		fillCircle(dst, cx+h*0.62, cy, h*0.3, rgb(0xE3C79B))
		strokeCircle(dst, cx+h*0.62, cy, h*0.15, max32(1, s*0.04), dark)
	case sim.ShapeBoard:
		for k := -1; k <= 1; k++ {
			y := cy + float32(k)*h*0.42
			fillRoundRect(dst, cx-h*0.85, y-h*0.17, h*1.7, h*0.34, h*0.08, c)
			strokeRoundRect(dst, cx-h*0.85, y-h*0.17, h*1.7, h*0.34, h*0.08, max32(1, s*0.04), dark)
		}
	case sim.ShapeCoil:
		fillCircle(dst, cx, cy, h*0.8, c)
		strokeCircle(dst, cx, cy, h*0.55, max32(1, s*0.08), dark)
		strokeCircle(dst, cx, cy, h*0.28, max32(1, s*0.08), dark)
	}
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// --- text -------------------------------------------------------------------

var (
	fontRegular *text.GoTextFaceSource
	fontBold    *text.GoTextFaceSource
	faces       = map[[2]int]*text.GoTextFace{}
)

func init() {
	var err error
	fontRegular, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	fontBold, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
}

func face(size float64, bold bool) *text.GoTextFace {
	key := [2]int{int(size * 4), 0}
	if bold {
		key[1] = 1
	}
	if f, ok := faces[key]; ok {
		return f
	}
	src := fontRegular
	if bold {
		src = fontBold
	}
	f := &text.GoTextFace{Source: src, Size: float64(key[0]) / 4}
	faces[key] = f
	return f
}

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

func drawText(dst *ebiten.Image, s string, x, y float64, size float64, bold bool, c color.Color, al align) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	switch al {
	case alignCenter:
		op.PrimaryAlign = text.AlignCenter
	case alignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	op.LineSpacing = size * 1.35
	text.Draw(dst, s, face(size, bold), op)
}

func textWidth(s string, size float64, bold bool) float64 {
	w, _ := text.Measure(s, face(size, bold), size*1.35)
	return w
}

// accentOf is the accent colour of a building kind (upgrades share the base colour).
func accentOf(k sim.Kind) color.RGBA { return colAccent[k.Base()] }
