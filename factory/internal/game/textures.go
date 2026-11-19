package game

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	xdraw "golang.org/x/image/draw"

	"potik/assets"
	"potik/internal/sim"
)

const (
	texSize     = 512 // pixels textures are scaled to after loading
	groundTiles = 8   // ground texture spans this many tiles
	oreTiles    = 4   // deposit texture spans this many tiles
)

var (
	texOnce   sync.Once
	texGround *ebiten.Image
	texOre    [sim.ItemCount]*ebiten.Image
)

var oreTexNames = map[sim.Item]string{
	sim.IronOre:   "ore_iron.png",
	sim.CopperOre: "ore_copper.png",
	sim.Coal:      "ore_coal.png",
	sim.Stone:     "ore_stone.png",
	sim.Sand:      "ore_sand.png",
}

// loadTexture decodes an embedded PNG and scales it down; nil if missing.
func loadTexture(name string) *ebiten.Image {
	data, err := assets.FS.ReadFile(name)
	if err != nil {
		return nil
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, texSize, texSize))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return ebiten.NewImageFromImage(dst)
}

func loadTextures() {
	texOnce.Do(func() {
		texGround = loadTexture("ground.png")
		for it, name := range oreTexNames {
			texOre[it] = loadTexture(name)
		}
	})
}

var texVerts []ebiten.Vertex
var texIdx []uint16

// drawTexPoly fills a convex screen-space polygon lying inside one texture
// repeat. (ox, oy) is the screen position of that repeat's top-left corner
// and span its size on screen; the texture is sampled without wrapping.
func drawTexPoly(dst, img *ebiten.Image, pts [][2]float32, ox, oy, span float32) {
	texVerts, texIdx = texVerts[:0], texIdx[:0]
	k := float32(texSize) / span
	clamp := func(v float32) float32 { // stay half a pixel inside to avoid bleeding
		if v < 0.5 {
			return 0.5
		}
		if v > texSize-0.5 {
			return texSize - 0.5
		}
		return v
	}
	for _, p := range pts {
		texVerts = append(texVerts, ebiten.Vertex{
			DstX: p[0], DstY: p[1],
			SrcX: clamp((p[0] - ox) * k), SrcY: clamp((p[1] - oy) * k),
			ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
		})
	}
	for i := 1; i+1 < len(pts); i++ {
		texIdx = append(texIdx, 0, uint16(i), uint16(i+1))
	}
	dst.DrawTriangles(texVerts, texIdx, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// drawGround covers the screen with the tiling ground texture.
func (g *Game) drawGround(dst *ebiten.Image) bool {
	if texGround == nil {
		return false
	}
	x0, y0 := g.cam.ToWorld(0, 0)
	x1, y1 := g.cam.ToWorld(g.cam.W, g.cam.H)
	span := float32(g.cam.Scale * groundTiles)
	for by := floorDiv(int(math.Floor(y0)), groundTiles); by*groundTiles <= int(math.Ceil(y1)); by++ {
		for bx := floorDiv(int(math.Floor(x0)), groundTiles); bx*groundTiles <= int(math.Ceil(x1)); bx++ {
			sx, sy := g.cam.ToScreen(float64(bx*groundTiles), float64(by*groundTiles))
			drawTexPoly(dst, texGround, [][2]float32{{sx, sy}, {sx + span, sy}, {sx + span, sy + span}, {sx, sy + span}}, sx, sy, span)
		}
	}
	return true
}

// drawOreTile draws one deposit tile from its texture, rounding the outer
// corners of the deposit so patches have soft edges. Returns false when the
// deposit has no texture.
func (g *Game) drawOreTile(dst *ebiten.Image, it sim.Item, x, y int) bool {
	img := texOre[it]
	if img == nil {
		return false
	}
	same := func(dx, dy int) bool { return g.w.OreAt(sim.Pos{X: x + dx, Y: y + dy}) == it }
	up, down, left, right := same(0, -1), same(0, 1), same(-1, 0), same(1, 0)
	sx, sy := g.cam.ToScreen(float64(x), float64(y))
	s := float32(g.cam.Scale)
	r := s * 0.35
	pts := make([][2]float32, 0, 24)
	// corner adds either the sharp corner point or a quarter arc. (cx, cy)
	// is the arc centre; the arc runs from angle a0 over 90 degrees.
	corner := func(px, py, cx, cy float32, round bool, a0 float64) {
		if !round {
			pts = append(pts, [2]float32{px, py})
			return
		}
		for i := 0; i <= 4; i++ {
			a := a0 + float64(i)*math.Pi/8
			pts = append(pts, [2]float32{cx + r*float32(math.Cos(a)), cy + r*float32(math.Sin(a))})
		}
	}
	// clockwise on screen (y down): top-left, top-right, bottom-right, bottom-left
	corner(sx, sy, sx+r, sy+r, !up && !left, math.Pi)
	corner(sx+s, sy, sx+s-r, sy+r, !up && !right, -math.Pi/2)
	corner(sx+s, sy+s, sx+s-r, sy+s-r, !down && !right, 0)
	corner(sx, sy+s, sx+r, sy+s-r, !down && !left, math.Pi/2)
	// screen position of the texture repeat this tile belongs to
	ox, oy := g.cam.ToScreen(float64(floorDiv(x, oreTiles)*oreTiles), float64(floorDiv(y, oreTiles)*oreTiles))
	drawTexPoly(dst, img, pts, ox, oy, s*oreTiles)
	return true
}
