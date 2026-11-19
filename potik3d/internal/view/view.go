// Package view draws the world in low-poly 3D with Ebitengine.
//
// Ebitengine has no depth buffer, so the scene is drawn with the painter's
// algorithm: tiles are sorted from far to near for the current camera and
// each tile draws its terrain column, water and objects. For a height-field
// world seen from above this gives correct occlusion.
package view

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"potik3d/internal/world"
)

// LevelH is the height of one terrain level in tile units.
const LevelH = 0.7

// Camera looks at a target point from above.
type Camera struct {
	X, Y, Z float64 // target in tile units (Z up)
	Yaw     float64 // radians, rotation around the vertical axis
	Pitch   float64 // radians above the horizon
	Scale   float64 // pixels per tile
	W, H    float64

	sy, cy, sp, cp float64
}

func (c *Camera) prepare() {
	c.sy, c.cy = math.Sincos(c.Yaw)
	c.sp, c.cp = math.Sincos(c.Pitch)
}

// rot returns the camera-space horizontal coordinates of a world offset.
func (c *Camera) rot(dx, dy float64) (rx, ry float64) {
	return dx*c.cy - dy*c.sy, dx*c.sy + dy*c.cy
}

// Project maps a world point to the screen; depth grows towards the viewer.
func (c *Camera) Project(x, y, z float64) (sx, sy, depth float64) {
	rx, ry := c.rot(x-c.X, y-c.Y)
	dz := z - c.Z
	sx = rx*c.Scale + c.W/2
	sy = (ry*c.sp-dz*c.cp)*c.Scale + c.H/2
	depth = ry*c.cp + dz*c.sp
	return
}

// facing reports whether a face with horizontal normal (nx, ny) faces the viewer.
func (c *Camera) facing(nx, ny float64) bool {
	_, ry := c.rot(nx, ny)
	return ry > 1e-6
}

var light = func() [3]float64 {
	l := [3]float64{-0.45, -0.65, 1}
	n := math.Sqrt(l[0]*l[0] + l[1]*l[1] + l[2]*l[2])
	return [3]float64{l[0] / n, l[1] / n, l[2] / n}
}()

// shade lights a colour for a face normal.
func shade(c color.RGBA, nx, ny, nz float64) color.RGBA {
	n := math.Sqrt(nx*nx + ny*ny + nz*nz)
	d := (nx*light[0] + ny*light[1] + nz*light[2]) / n
	f := 0.6 + 0.4*math.Max(0, d)
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), c.A}
}

func rgb(v uint32) color.RGBA { return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF} }

func rgba(v uint32, a uint8) color.RGBA {
	c := rgb(v)
	f := float64(a) / 255
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), a}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

// --- triangle batch -------------------------------------------------------------

var white = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img
}()

var whiteSub = white.SubImage(white.Bounds().Inset(1)).(*ebiten.Image)

// Batch collects coloured triangles in draw order.
type Batch struct {
	dst   *ebiten.Image
	cam   *Camera
	verts []ebiten.Vertex
	idx   []uint16
}

func (b *Batch) flush() {
	if len(b.idx) > 0 {
		b.dst.DrawTriangles(b.verts, b.idx, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: true})
	}
	b.verts, b.idx = b.verts[:0], b.idx[:0]
}

func vtx(x, y float64, c color.RGBA) ebiten.Vertex {
	return ebiten.Vertex{DstX: float32(x), DstY: float32(y), SrcX: 1, SrcY: 1,
		ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
}

// Poly adds a convex polygon given in world coordinates.
func (b *Batch) Poly(c color.RGBA, pts ...[3]float64) {
	if len(b.verts)+len(pts) > 60000 {
		b.flush()
	}
	base := uint16(len(b.verts))
	for _, p := range pts {
		sx, sy, _ := b.cam.Project(p[0], p[1], p[2])
		b.verts = append(b.verts, vtx(sx, sy, c))
	}
	for k := 1; k+1 < len(pts); k++ {
		b.idx = append(b.idx, base, base+uint16(k), base+uint16(k+1))
	}
}

// Box draws an axis-aligned box: visible sides first, then the top.
func (b *Batch) Box(x0, y0, z0, x1, y1, z1 float64, side, top color.RGBA) {
	cam := b.cam
	if cam.facing(0, -1) {
		b.Poly(shade(side, 0, -1, 0), [3]float64{x0, y0, z0}, [3]float64{x1, y0, z0}, [3]float64{x1, y0, z1}, [3]float64{x0, y0, z1})
	}
	if cam.facing(0, 1) {
		b.Poly(shade(side, 0, 1, 0), [3]float64{x0, y1, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z1})
	}
	if cam.facing(-1, 0) {
		b.Poly(shade(side, -1, 0, 0), [3]float64{x0, y0, z0}, [3]float64{x0, y1, z0}, [3]float64{x0, y1, z1}, [3]float64{x0, y0, z1})
	}
	if cam.facing(1, 0) {
		b.Poly(shade(side, 1, 0, 0), [3]float64{x1, y0, z0}, [3]float64{x1, y1, z0}, [3]float64{x1, y1, z1}, [3]float64{x1, y0, z1})
	}
	b.Poly(shade(top, 0, 0, 1), [3]float64{x0, y0, z1}, [3]float64{x1, y0, z1}, [3]float64{x1, y1, z1}, [3]float64{x0, y1, z1})
}

// Prism draws a gable roof running along the x axis.
func (b *Batch) Prism(x0, y0, x1, y1, z, h float64, roof, gable color.RGBA) {
	ym := (y0 + y1) / 2
	cam := b.cam
	if cam.facing(-1, 0) {
		b.Poly(shade(gable, -1, 0, 0), [3]float64{x0, y0, z}, [3]float64{x0, y1, z}, [3]float64{x0, ym, z + h})
	}
	if cam.facing(1, 0) {
		b.Poly(shade(gable, 1, 0, 0), [3]float64{x1, y0, z}, [3]float64{x1, y1, z}, [3]float64{x1, ym, z + h})
	}
	hw := (y1 - y0) / 2
	order := []int{0, 1}
	if !cam.facing(0, -1) {
		order = []int{1, 0}
	}
	for _, k := range order {
		if k == 0 {
			b.Poly(shade(roof, 0, -h, hw), [3]float64{x0, y0, z}, [3]float64{x1, y0, z}, [3]float64{x1, ym, z + h}, [3]float64{x0, ym, z + h})
		} else {
			b.Poly(shade(roof, 0, h, hw), [3]float64{x0, y1, z}, [3]float64{x1, y1, z}, [3]float64{x1, ym, z + h}, [3]float64{x0, ym, z + h})
		}
	}
}

// Prismoid draws an n-sided vertical prism (a round tank).
func (b *Batch) Prismoid(cx, cy, z0, r, h float64, n int, c color.RGBA) {
	for k := 0; k < n; k++ {
		a0 := float64(k) * 2 * math.Pi / float64(n)
		a1 := float64(k+1) * 2 * math.Pi / float64(n)
		am := (a0 + a1) / 2
		if !b.cam.facing(math.Cos(am), math.Sin(am)) {
			continue
		}
		p0x, p0y := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
		p1x, p1y := cx+r*math.Cos(a1), cy+r*math.Sin(a1)
		b.Poly(shade(c, math.Cos(am), math.Sin(am), 0), [3]float64{p0x, p0y, z0}, [3]float64{p1x, p1y, z0}, [3]float64{p1x, p1y, z0 + h}, [3]float64{p0x, p0y, z0 + h})
	}
	pts := make([][3]float64, n)
	for k := range pts {
		a := float64(k) * 2 * math.Pi / float64(n)
		pts[k] = [3]float64{cx + r*math.Cos(a), cy + r*math.Sin(a), z0 + h}
	}
	b.Poly(shade(c, 0, 0, 1), pts...)
}

// Pyramid draws an n-sided pyramid (n=4 roof, n=6..8 tree cone).
func (b *Batch) Pyramid(cx, cy, z0, r, h float64, n int, rot float64, c color.RGBA) {
	type face struct {
		pts   [3][3]float64
		depth float64
		col   color.RGBA
	}
	faces := make([]face, 0, n)
	apex := [3]float64{cx, cy, z0 + h}
	for k := 0; k < n; k++ {
		a0 := rot + float64(k)*2*math.Pi/float64(n)
		a1 := rot + float64(k+1)*2*math.Pi/float64(n)
		p0 := [3]float64{cx + r*math.Cos(a0), cy + r*math.Sin(a0), z0}
		p1 := [3]float64{cx + r*math.Cos(a1), cy + r*math.Sin(a1), z0}
		am := (a0 + a1) / 2
		nx, ny := math.Cos(am)*h, math.Sin(am)*h
		nz := r * math.Cos(math.Pi/float64(n))
		if !b.cam.facing(math.Cos(am), math.Sin(am)) && nz < 0.2*math.Hypot(nx, ny) {
			continue
		}
		_, _, d := b.cam.Project((p0[0]+p1[0]+apex[0])/3, (p0[1]+p1[1]+apex[1])/3, (p0[2]+p1[2]+apex[2])/3)
		faces = append(faces, face{[3][3]float64{p0, p1, apex}, d, shade(c, nx, ny, nz)})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, f := range faces {
		b.Poly(f.col, f.pts[0], f.pts[1], f.pts[2])
	}
}

// --- scene ----------------------------------------------------------------------------

// Scene renders a world.
type Scene struct {
	Cam   Camera
	Hover [2]int // hovered tile, -1 if none
	order []int
}

var (
	colGrass  = rgb(0x8DB86A)
	colGrass2 = rgb(0x7FAE5F)
	colSand   = rgb(0xD9C79A)
	colMud    = rgb(0xB8A27A)
	colDirt   = rgb(0x9C7A55)
	colDirt2  = rgb(0x8A6A48)
	colRock   = rgb(0xA8A39A)
	colWater  = rgb(0x4F9FD0)
	colTrunk  = rgb(0x7A5536)
	colLeaf   = rgb(0x4E8A55)
	colLeaf2  = rgb(0x5E9A62)
	colSky    = rgb(0xCFE3EA)
)

func (s *Scene) groundColor(w *world.World, x, y int) color.RGBA {
	h := w.Height[world.Idx(x, y)]
	var c color.RGBA
	switch {
	case h == 0:
		c = colMud
	case h == 1:
		c = colSand
	case h >= 6:
		c = mix(colGrass, colRock, float64(h-5)/4)
	default:
		c = mix(colGrass, colGrass2, hashf(x, y)*0.8)
		c = mix(c, rgb(0xE8F0C8), float64(h-2)*0.06)
		if w.Drought && !s.nearWater(w, x, y) {
			c = mix(c, rgb(0xC9B06A), 0.6+0.2*hashf(y, x))
		}
	}
	return c
}

// visible returns tile indices around the camera target sorted far to near.
func (s *Scene) visible(w *world.World) []int {
	c := &s.Cam
	reach := math.Hypot(c.W, c.H)/c.Scale/2/math.Max(0.35, c.sp) + 4
	x0 := int(math.Max(0, c.X-reach))
	x1 := int(math.Min(world.Size-1, c.X+reach))
	y0 := int(math.Max(0, c.Y-reach))
	y1 := int(math.Min(world.Size-1, c.Y+reach))
	s.order = s.order[:0]
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			s.order = append(s.order, world.Idx(x, y))
		}
	}
	sort.Slice(s.order, func(a, b int) bool {
		ia, ib := s.order[a], s.order[b]
		_, ra := c.rot(float64(ia%world.Size)+0.5-c.X, float64(ia/world.Size)+0.5-c.Y)
		_, rb := c.rot(float64(ib%world.Size)+0.5-c.X, float64(ib/world.Size)+0.5-c.Y)
		return ra < rb
	})
	return s.order
}

func (s *Scene) heightAt(w *world.World, x, y int) float64 {
	if !world.In(x, y) {
		return -1
	}
	return float64(w.Height[world.Idx(x, y)])
}

// Draw renders the world.
func (s *Scene) Draw(dst *ebiten.Image, w *world.World) {
	c := &s.Cam
	c.prepare()
	dst.Fill(colSky)
	b := &Batch{dst: dst, cam: c}
	t := float64(w.Tick) / 60
	front := map[int][]*world.Building{}
	for _, bd := range w.Builds {
		fi := s.frontTile(bd)
		front[fi] = append(front[fi], bd)
	}
	people := map[int][]*world.Settler{}
	for _, st := range w.Settlers {
		if world.In(int(st.X), int(st.Y)) {
			k := world.Idx(int(st.X), int(st.Y))
			people[k] = append(people[k], st)
		}
	}
	for _, i := range s.visible(w) {
		x, y := i%world.Size, i/world.Size
		fx, fy := float64(x), float64(y)
		h := float64(w.Height[i])
		top := h * LevelH
		// terrain sides down to each lower neighbour (or the map's base)
		dirs := [4][4]int{{0, -1, 0, -1}, {0, 1, 0, 1}, {-1, 0, -1, 0}, {1, 0, 1, 0}}
		for _, d := range dirs {
			nh := s.heightAt(w, x+d[0], y+d[1])
			if nh >= h || !c.facing(float64(d[2]), float64(d[3])) {
				continue
			}
			for lv := h - 1; lv >= math.Max(nh, -1); lv-- {
				z0, z1 := lv*LevelH, (lv+1)*LevelH
				col := colDirt
				if int(lv)%2 == 1 {
					col = colDirt2
				}
				if lv == h-1 {
					col = mix(col, s.groundColor(w, x, y), 0.25)
				}
				col = shade(col, float64(d[2]), float64(d[3]), 0)
				var p [4][3]float64
				switch {
				case d[1] == -1:
					p = [4][3]float64{{fx, fy, z0}, {fx + 1, fy, z0}, {fx + 1, fy, z1}, {fx, fy, z1}}
				case d[1] == 1:
					p = [4][3]float64{{fx, fy + 1, z0}, {fx + 1, fy + 1, z0}, {fx + 1, fy + 1, z1}, {fx, fy + 1, z1}}
				case d[0] == -1:
					p = [4][3]float64{{fx, fy, z0}, {fx, fy + 1, z0}, {fx, fy + 1, z1}, {fx, fy, z1}}
				default:
					p = [4][3]float64{{fx + 1, fy, z0}, {fx + 1, fy + 1, z0}, {fx + 1, fy + 1, z1}, {fx + 1, fy, z1}}
				}
				b.Poly(col, p[0], p[1], p[2], p[3])
			}
		}
		// top, darkened along the foot of higher neighbours
		gc := shade(s.groundColor(w, x, y), 0, 0, 1)
		for _, d := range dirs {
			if s.heightAt(w, x+d[0], y+d[1]) > h {
				gc = mix(gc, rgb(0x2E3440), 0.08)
			}
		}
		if s.Hover[0] == x && s.Hover[1] == y {
			gc = mix(gc, rgb(0xFFFFFF), 0.35)
		}
		b.Poly(gc, [3]float64{fx, fy, top}, [3]float64{fx + 1, fy, top}, [3]float64{fx + 1, fy + 1, top}, [3]float64{fx, fy + 1, top})

		// water
		if d := float64(w.Water[i]); d > 0.03 {
			wz := (h + d) * LevelH
			k := math.Min(1, d/1.5)
			wc := mix(rgb(0x7CC0E4), colWater, k)
			ripple := 0.04 * math.Sin(t*1.5+fx*0.9+fy*0.6)
			wc = mix(wc, rgb(0xFFFFFF), math.Max(0, ripple))
			wc.A = uint8(150 + 80*k)
			wc = rgba(uint32(wc.R)<<16|uint32(wc.G)<<8|uint32(wc.B), wc.A)
			for _, d4 := range dirs {
				nx, ny := x+d4[0], y+d4[1]
				nl := -1.0
				if world.In(nx, ny) {
					j := world.Idx(nx, ny)
					nl = float64(w.Height[j]) + float64(w.Water[j])
				}
				if nl*LevelH >= wz-0.02 || !c.facing(float64(d4[2]), float64(d4[3])) {
					continue
				}
				zb := math.Max(nl*LevelH, top)
				var p [4][3]float64
				switch {
				case d4[1] == -1:
					p = [4][3]float64{{fx, fy, zb}, {fx + 1, fy, zb}, {fx + 1, fy, wz}, {fx, fy, wz}}
				case d4[1] == 1:
					p = [4][3]float64{{fx, fy + 1, zb}, {fx + 1, fy + 1, zb}, {fx + 1, fy + 1, wz}, {fx, fy + 1, wz}}
				case d4[0] == -1:
					p = [4][3]float64{{fx, fy, zb}, {fx, fy + 1, zb}, {fx, fy + 1, wz}, {fx, fy, wz}}
				default:
					p = [4][3]float64{{fx + 1, fy, zb}, {fx + 1, fy + 1, zb}, {fx + 1, fy + 1, wz}, {fx + 1, fy, wz}}
				}
				b.Poly(shade(wc, float64(d4[2]), float64(d4[3]), 0.3), p[0], p[1], p[2], p[3])
			}
			b.Poly(wc, [3]float64{fx, fy, wz}, [3]float64{fx + 1, fy, wz}, [3]float64{fx + 1, fy + 1, wz}, [3]float64{fx, fy + 1, wz})
		}

		// objects
		if bs := front[i]; len(bs) > 0 {
			sort.Slice(bs, func(a, c int) bool { return bs[a].Level < bs[c].Level })
			for _, bd := range bs {
				base := float64(int(w.Height[world.Idx(bd.X, bd.Y)])+bd.Level) * LevelH
				drawBuilding(b, bd, base, t)
			}
		}
		for _, st := range people[i] {
			drawSettler(b, st, t)
		}
		if w.Owner[i] != 0 {
			continue
		}
		switch w.Props[i] {
		case world.PropTree:
			r := 0.85 + 0.3*hashf(x, y)
			cx, cy := fx+0.5+0.2*(hashf(x+7, y)-0.5), fy+0.5+0.2*(hashf(x, y+7)-0.5)
			b.Box(cx-0.07, cy-0.07, top, cx+0.07, cy+0.07, top+0.35*r, colTrunk, colTrunk)
			lc := colLeaf
			if (x+y)%3 == 0 {
				lc = colLeaf2
			}
			b.Pyramid(cx, cy, top+0.25*r, 0.42*r, 0.7*r, 6, hashf(x, y)*3, lc)
			b.Pyramid(cx, cy, top+0.65*r, 0.32*r, 0.6*r, 6, hashf(y, x)*3, mix(lc, rgb(0xFFFFFF), 0.08))
		case world.PropBush:
			cx, cy := fx+0.5, fy+0.5
			b.Pyramid(cx, cy, top, 0.3, 0.3, 5, hashf(x, y)*3, colLeaf2)
			for k := 0; k < 3; k++ {
				a := float64(k)*2.1 + hashf(x, y)*6
				bx, by := cx+math.Cos(a)*0.16, cy+math.Sin(a)*0.16
				b.Box(bx-0.04, by-0.04, top+0.12, bx+0.04, by+0.04, top+0.2, rgb(0xC0504D), rgb(0xD9605A))
			}
		case world.PropRock:
			cx, cy := fx+0.5, fy+0.5
			b.Pyramid(cx, cy, top, 0.32, 0.28, 5, hashf(x, y)*3, colRock)
		}
	}
	b.flush()
}

// frontTile is the footprint tile nearest to the viewer; a building is
// drawn there, after the rest of its footprint.
func (s *Scene) frontTile(bd *world.Building) int {
	c := &s.Cam
	best, bi := -1e9, 0
	for dy := 0; dy < bd.Size; dy++ {
		for dx := 0; dx < bd.Size; dx++ {
			x, y := bd.X+dx, bd.Y+dy
			if !world.In(x, y) {
				continue
			}
			_, r := c.rot(float64(x)+0.5-c.X, float64(y)+0.5-c.Y)
			if r > best {
				best, bi = r, world.Idx(x, y)
			}
		}
	}
	return bi
}

func (s *Scene) nearWater(w *world.World, x, y int) bool {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if world.In(x+dx, y+dy) && w.Water[world.Idx(x+dx, y+dy)] > 0.1 && dx*dx+dy*dy <= 16 {
				return true
			}
		}
	}
	return false
}

func hashf(x, y int) float64 {
	h := uint32(x*73856093) ^ uint32(y*19349663)
	h ^= h >> 13
	h *= 0x5bd1e995
	return float64(h&0xFFFF) / 65535
}

func drawBuilding(b *Batch, bd *world.Building, base float64, t float64) {
	x0, y0 := float64(bd.X), float64(bd.Y)
	s := float64(bd.Size)
	switch bd.Kind {
	case world.KHall:
		b.Box(x0+0.15, y0+0.15, base, x0+s-0.15, y0+s-0.15, base+0.9, rgb(0xE8DCC2), rgb(0xE8DCC2))
		b.Pyramid(x0+s/2, y0+s/2, base+0.9, s*0.62, 0.9, 4, math.Pi/4, rgb(0x4F7A8C))
		b.Box(x0+s/2-0.12, y0+s/2-0.12, base+1.6, x0+s/2+0.12, y0+s/2+0.12, base+2.2, rgb(0x6B4F3A), rgb(0x6B4F3A))
		b.Pyramid(x0+s/2, y0+s/2, base+2.2, 0.35, 0.3, 4, math.Pi/4, rgb(0x4FA3A5))
	case world.KHouse:
		b.Box(x0+0.15, y0+0.15, base, x0+s-0.15, y0+s-0.15, base+0.55, rgb(0xEDE3CF), rgb(0xEDE3CF))
		b.Pyramid(x0+s/2, y0+s/2, base+0.55, s*0.6, 0.5, 4, math.Pi/4, rgb(0xC9673F))
	case world.KLumber:
		b.Box(x0+0.2, y0+0.2, base, x0+s-0.2, y0+s-0.2, base+0.45, rgb(0xB08A5A), rgb(0xB08A5A))
		b.Pyramid(x0+s/2, y0+s/2, base+0.45, s*0.55, 0.35, 4, math.Pi/4, rgb(0x7A5536))
		for k := 0; k < 3; k++ {
			ly := y0 + s - 0.15 - float64(k)*0.12
			b.Box(x0+0.1, ly-0.05, base, x0+s*0.6, ly+0.05, base+0.1, rgb(0x9C6B45), rgb(0xE3C79B))
		}
	case world.KPump:
		b.Box(x0+0.25, y0+0.25, base, x0+s-0.25, y0+s-0.25, base+0.3, rgb(0xA8A39A), rgb(0x4F9FD0))
		b.Box(x0+s/2-0.05, y0+s/2-0.05, base+0.3, x0+s/2+0.05, y0+s/2+0.05, base+0.8, rgb(0x6B4F3A), rgb(0x6B4F3A))
		arm := 0.2 * math.Sin(t*3)
		b.Box(x0+s/2-0.3, y0+s/2-0.04, base+0.8+arm*0.3, x0+s/2+0.3, y0+s/2+0.04, base+0.86+arm*0.3, rgb(0x6B4F3A), rgb(0x7A5536))
	case world.KDam:
		b.Box(x0, y0+0.2, base, x0+s, y0+s-0.2, base+LevelH*1.0, rgb(0x8C6B4A), rgb(0x9C7A55))
	case world.KFarm:
		b.Box(x0+0.04, y0+0.04, base, x0+s-0.04, y0+s-0.04, base+0.04, rgb(0x8A6A48), rgb(0x8A6A48))
		for r := 0; r < 3; r++ {
			for k := 0; k < 3; k++ {
				px, py := x0+0.2+float64(k)*0.3, y0+0.2+float64(r)*0.3
				hgt := 0.12 + 0.03*math.Sin(t*2+px+py)
				col := rgb(0x7FB35A)
				if (r+k)%3 == 0 {
					col = rgb(0xD9B44A)
				}
				b.Pyramid(px, py, base+0.04, 0.09, hgt+0.12, 4, 0.3, col)
			}
		}
	case world.KSite:
		wood := rgb(0xC8A06A)
		for _, p := range [][2]float64{{0.12, 0.12}, {s - 0.12, 0.12}, {0.12, s - 0.12}, {s - 0.12, s - 0.12}} {
			b.Box(x0+p[0]-0.04, y0+p[1]-0.04, base, x0+p[0]+0.04, y0+p[1]+0.04, base+0.8, wood, wood)
		}
		b.Box(x0+0.16, y0+0.16, base, x0+s-0.16, y0+s-0.16, base+0.8*bd.Prog, rgb(0xE8DCC2), rgb(0xD8CCB2))
		b.Box(x0+0.08, y0+0.08, base+0.78, x0+s-0.08, y0+0.16, base+0.84, wood, wood)
		b.Box(x0+0.08, y0+s-0.16, base+0.78, x0+s-0.08, y0+s-0.08, base+0.84, wood, wood)
	case world.KWarehouse:
		b.Box(x0+0.1, y0+0.1, base, x0+s-0.1, y0+s-0.1, base+0.7, rgb(0xB08A5A), rgb(0xB08A5A))
		b.Prism(x0+0.02, y0+0.02, x0+s-0.02, y0+s-0.02, base+0.7, 0.55, rgb(0x7A5536), rgb(0xB08A5A))
	case world.KPlatform:
		wood := rgb(0xA5794D)
		for _, p := range [][2]float64{{0.1, 0.1}, {s - 0.1, 0.1}, {0.1, s - 0.1}, {s - 0.1, s - 0.1}} {
			b.Box(x0+p[0]-0.06, y0+p[1]-0.06, base, x0+p[0]+0.06, y0+p[1]+0.06, base+LevelH-0.08, wood, wood)
		}
		b.Box(x0, y0, base+LevelH-0.08, x0+s, y0+s, base+LevelH, wood, rgb(0xC8A06A))
	case world.KStairs:
		n := 5
		for k := 0; k < n; k++ {
			f0, f1 := float64(k)/float64(n), float64(k+1)/float64(n)
			z := base + LevelH*f1
			switch bd.Dir {
			case 0:
				b.Box(x0+0.15, y0+1-f1, base, x0+0.85, y0+1-f0, z, rgb(0xA5794D), rgb(0xC8A06A))
			case 1:
				b.Box(x0+f0, y0+0.15, base, x0+f1, y0+0.85, z, rgb(0xA5794D), rgb(0xC8A06A))
			case 2:
				b.Box(x0+0.15, y0+f0, base, x0+0.85, y0+f1, z, rgb(0xA5794D), rgb(0xC8A06A))
			default:
				b.Box(x0+1-f1, y0+0.15, base, x0+1-f0, y0+0.85, z, rgb(0xA5794D), rgb(0xC8A06A))
			}
		}
	case world.KBridge:
		wood := rgb(0xA5794D)
		deck := base + LevelH*bd.Prog
		b.Box(x0+0.45, y0+0.4, base-1.4, x0+0.55, y0+0.6, deck, wood, wood)
		b.Box(x0, y0+0.05, deck-0.08, x0+s, y0+s-0.05, deck, wood, rgb(0xC8A06A))
		b.Box(x0, y0+0.05, deck, x0+s, y0+0.1, deck+0.25, wood, wood)
		b.Box(x0, y0+s-0.1, deck, x0+s, y0+s-0.05, deck+0.25, wood, wood)
	case world.KWaterTank:
		b.Prismoid(x0+s/2, y0+s/2, base, 0.38*s, 0.9, 8, rgb(0x9C7A55))
		b.Prismoid(x0+s/2, y0+s/2, base+0.9, 0.33*s, 0.02, 8, rgb(0x4F9FD0))
	}
}

// drawSettler draws a small low-poly person, bobbing as they walk.
func drawSettler(b *Batch, st *world.Settler, t float64) {
	z := st.Z * LevelH
	bob := 0.03 * math.Abs(math.Sin(t*8+st.Phase))
	x, y := st.X, st.Y
	skin, tunic := rgb(0xF1D2B0), rgb(0x4FA3A5)
	switch st.Carry {
	case 1:
		tunic = rgb(0x7A9A4A)
	case 3:
		tunic = rgb(0x5E86C8)
	case 4:
		tunic = rgb(0xC9673F)
	}
	leg := 0.04 * math.Sin(t*8+st.Phase)
	b.Box(x-0.06+leg, y-0.04, z, x-0.01+leg, y+0.04, z+0.16, rgb(0x5A4636), rgb(0x5A4636))
	b.Box(x+0.01-leg, y-0.04, z, x+0.06-leg, y+0.04, z+0.16, rgb(0x5A4636), rgb(0x5A4636))
	b.Box(x-0.09, y-0.06, z+0.16+bob, x+0.09, y+0.06, z+0.38+bob, tunic, tunic)
	b.Box(x-0.07, y-0.07, z+0.38+bob, x+0.07, y+0.07, z+0.52+bob, skin, rgb(0x6B4A2E))
	switch st.Carry {
	case 1:
		b.Box(x-0.22, y-0.05, z+0.4+bob, x+0.22, y+0.05, z+0.48+bob, rgb(0x9C6B45), rgb(0x9C6B45))
	case 2:
		b.Box(x-0.2, y-0.07, z+0.4+bob, x+0.2, y+0.07, z+0.45+bob, rgb(0xD9B27C), rgb(0xE3C79B))
	case 3:
		b.Box(x+0.08, y-0.04, z+0.12+bob, x+0.16, y+0.04, z+0.24+bob, rgb(0x8C99A6), rgb(0x4F9FD0))
	case 4:
		b.Box(x-0.08, y-0.1, z+0.28+bob, x+0.08, y+0.1, z+0.36+bob, rgb(0xC8A97E), rgb(0xC0504D))
	}
}

// Pick returns the tile whose top face is under the screen point.
func (s *Scene) Pick(w *world.World, mx, my float64) (int, int, bool) {
	c := &s.Cam
	c.prepare()
	order := s.visible(w)
	for k := len(order) - 1; k >= 0; k-- {
		i := order[k]
		x, y := i%world.Size, i/world.Size
		z := float64(w.Height[i]) * LevelH
		var px, py [4]float64
		corners := [4][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
		for n, co := range corners {
			px[n], py[n], _ = c.Project(float64(x)+co[0], float64(y)+co[1], z)
		}
		if inQuad(mx, my, px, py) {
			return x, y, true
		}
	}
	return -1, -1, false
}

func inQuad(x, y float64, px, py [4]float64) bool {
	sign := 0.0
	for k := 0; k < 4; k++ {
		ax, ay := px[k], py[k]
		bx, by := px[(k+1)%4], py[(k+1)%4]
		cr := (bx-ax)*(y-ay) - (by-ay)*(x-ax)
		if cr == 0 {
			continue
		}
		if sign == 0 {
			sign = cr
		} else if (cr > 0) != (sign > 0) {
			return false
		}
	}
	return true
}
