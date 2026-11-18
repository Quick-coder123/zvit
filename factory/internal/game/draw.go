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
		sim.KBelt:      rgb(0xA79E8E),
		sim.KMiner:     rgb(0x4FA3A5),
		sim.KFurnace:   rgb(0xF2A65A),
		sim.KAssembler: rgb(0x6C8EBF),
		sim.KChest:     rgb(0xC8A97E),
		sim.KCore:      rgb(0x2E3440),
		sim.KTunnel:    rgb(0x5E6B78),
		sim.KSplitter:  rgb(0x8E7CC3),
		sim.KMerger:    rgb(0x6FAE7B),
		sim.KSorter:    rgb(0xD4A373),
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
