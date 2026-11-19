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
	Alpha float64 // < 1 draws translucent (placement previews)
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
	if b.Alpha > 0 && b.Alpha < 1 {
		a := b.Alpha
		c = color.RGBA{uint8(float64(c.R) * a), uint8(float64(c.G) * a), uint8(float64(c.B) * a), uint8(float64(c.A) * a)}
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
