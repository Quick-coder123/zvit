package view

import (
	"image/color"
	"math"

	"potik3d/internal/world"
)

// Wall sides of an axis-aligned box.
const (
	sideXP = iota // +x
	sideXM        // -x
	sideYP        // +y
	sideYM        // -y
)

var sideNormals = [4][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

// Decal draws a flat polygon lying just in front of a wall with horizontal
// normal (nx, ny). It is skipped when that wall faces away from the camera,
// so details on hidden walls never show through.
func (b *Batch) Decal(nx, ny float64, c color.RGBA, pts ...[3]float64) bool {
	if !b.cam.facing(nx, ny) {
		return false
	}
	const off = 0.004
	q := make([][3]float64, len(pts))
	for i, p := range pts {
		q[i] = [3]float64{p[0] + nx*off, p[1] + ny*off, p[2]}
	}
	b.Poly(shade(c, nx, ny, 0), q...)
	return true
}

// WallRect draws a rectangle on one wall of the box [x0,x1]×[y0,y1]: u0..u1
// is the span along the wall (0..1), z0..z1 the height range.
func (b *Batch) WallRect(x0, y0, x1, y1 float64, side int, u0, u1, z0, z1 float64, c color.RGBA) {
	n := sideNormals[side]
	var p0, p1 [2]float64
	switch side {
	case sideXP:
		p0, p1 = [2]float64{x1, y0 + (y1-y0)*u0}, [2]float64{x1, y0 + (y1-y0)*u1}
	case sideXM:
		p0, p1 = [2]float64{x0, y0 + (y1-y0)*u0}, [2]float64{x0, y0 + (y1-y0)*u1}
	case sideYP:
		p0, p1 = [2]float64{x0 + (x1-x0)*u0, y1}, [2]float64{x0 + (x1-x0)*u1, y1}
	default:
		p0, p1 = [2]float64{x0 + (x1-x0)*u0, y0}, [2]float64{x0 + (x1-x0)*u1, y0}
	}
	b.Decal(n[0], n[1], c, [3]float64{p0[0], p0[1], z0}, [3]float64{p1[0], p1[1], z0},
		[3]float64{p1[0], p1[1], z1}, [3]float64{p0[0], p0[1], z1})
}

var (
	colGlass = rgb(0x3E5568)
	colFrame = rgb(0xF4EEE0)
	colDoor  = rgb(0x6B4A2E)
	colStone = rgb(0x9A958C)
)

// window draws a framed window centred at u along a wall, bottom at z.
func (b *Batch) window(x0, y0, x1, y1 float64, side int, u, z, w, h float64) {
	length := x1 - x0
	if side == sideXP || side == sideXM {
		length = y1 - y0
	}
	hw := w / 2 / length
	fw := 0.025 / length
	b.WallRect(x0, y0, x1, y1, side, u-hw-fw, u+hw+fw, z-0.025, z+h+0.025, colFrame)
	b.WallRect(x0, y0, x1, y1, side, u-hw, u+hw, z, z+h, colGlass)
}

// door draws a door centred at u along a wall from the ground at z.
func (b *Batch) door(x0, y0, x1, y1 float64, side int, u, z, w, h float64) {
	length := x1 - x0
	if side == sideXP || side == sideXM {
		length = y1 - y0
	}
	hw := w / 2 / length
	fw := 0.022 / length
	b.WallRect(x0, y0, x1, y1, side, u-hw-fw, u+hw+fw, z, z+h+0.022, colFrame)
	b.WallRect(x0, y0, x1, y1, side, u-hw, u+hw, z, z+h, colDoor)
}

// Ring draws only the sides of an n-sided prism (a hoop around a tank).
func (b *Batch) Ring(cx, cy, z0, r, h float64, n int, c color.RGBA) {
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
}

// propFirst reports whether a prop standing beside a building in direction
// (nx, ny) must be drawn before the building body (it is behind it).
func (b *Batch) propFirst(nx, ny float64) bool { return !b.cam.facing(nx, ny) }

// HouseStyle picks a stable look for a house from its ID: variant 0..2,
// wall colour 0..3 and roof colour 0..2. ID 0 (a placement ghost) is plain.
func HouseStyle(id int32) (variant, wall, roof int) {
	if id == 0 {
		return 0, 0, 0
	}
	h := uint32(id) * 2654435761
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	return int(h % 3), int((h / 3) % 4), int((h / 12) % 3)
}

var (
	houseWalls = [4]color.RGBA{rgb(0xE8DCC2), rgb(0xD9C7A8), rgb(0xEFE6D6), rgb(0xCDB99A)}
	houseRoofs = [3]color.RGBA{rgb(0xC9673F), rgb(0x8C5A3C), rgb(0x6F7F8C)}
)

// wantsSmoke decides whether a building's chimney smokes: only real,
// finished buildings, houses only when someone lives there.
func wantsSmoke(id int32, site bool, kind world.Kind, residents int) bool {
	if id == 0 || site {
		return false
	}
	switch kind {
	case world.KHall:
		return true
	case world.KHouse:
		return residents > 0
	}
	return false
}

// smoke draws puffs rising from a chimney top.
func smoke(b *Batch, x, y, z, t float64, id int32) {
	a := b.Alpha
	for k := 0; k < 3; k++ {
		p := math.Mod(t/3+float64(id)*0.37+float64(k)/3, 1)
		s := 0.04 + 0.05*p
		px, pz := x+0.12*p, z+0.05+0.6*p
		b.Alpha = 0.55 * (1 - p)
		c := rgb(0xD8DCE0)
		b.Box(px-s, y-s, pz-s, px+s, y+s, pz+s, c, c)
	}
	b.Alpha = a
}

// chimney stands on a roof's highest line from zTop upward and returns the
// height of its top.
func chimney(b *Batch, x, y, zTop float64) float64 {
	c := rgb(0x8C6B5A)
	b.Box(x-0.06, y-0.06, zTop, x+0.06, y+0.06, zTop+0.16, c, rgb(0x5A4636))
	return zTop + 0.16
}

// drawHouse draws one of the house variants.
func drawHouse(b *Batch, w *world.World, bd *world.Building, x0, y0, base, t float64, selected bool) {
	variant, wi, ri := HouseStyle(bd.ID)
	wall, roof := houseWalls[wi], houseRoofs[ri]
	if selected {
		wall = mix(wall, rgb(0xF2D16B), 0.45)
	}
	residents := 0
	if bd.ID != 0 && w != nil {
		residents = w.Residents(bd.ID)
	}
	puff := wantsSmoke(bd.ID, bd.Site, bd.Kind, residents)
	var cx, cy, ctop float64
	switch variant {
	case 0: // cottage with a gable roof
		ax0, ay0, ax1, ay1 := x0+0.15, y0+0.18, x0+0.85, y0+0.82
		top := base + 0.48
		b.Box(ax0, ay0, base, ax1, ay1, top, wall, wall)
		b.door(ax0, ay0, ax1, ay1, sideYP, 0.5, base, 0.15, 0.3)
		for _, u := range []float64{0.2, 0.8} {
			b.window(ax0, ay0, ax1, ay1, sideYP, u, base+0.2, 0.11, 0.13)
			b.window(ax0, ay0, ax1, ay1, sideYM, u, base+0.2, 0.11, 0.13)
		}
		b.window(ax0, ay0, ax1, ay1, sideXP, 0.5, base+0.2, 0.13, 0.13)
		b.window(ax0, ay0, ax1, ay1, sideXM, 0.5, base+0.2, 0.13, 0.13)
		b.Prism(x0+0.08, y0+0.1, x0+0.92, y0+0.9, top, 0.4, roof, wall)
		cx, cy = x0+0.66, y0+0.5
		ctop = chimney(b, cx, cy, top+0.4)
	case 1: // hipped roof and a porch on the -x side
		ax0, ay0, ax1, ay1 := x0+0.2, y0+0.14, x0+0.86, y0+0.86
		top := base + 0.46
		porch := func() {
			b.Box(x0+0.04, y0+0.34, base, x0+0.07, y0+0.37, base+0.36, colWood, colWood)
			b.Box(x0+0.04, y0+0.63, base, x0+0.07, y0+0.66, base+0.36, colWood, colWood)
			b.Box(x0+0.02, y0+0.3, base+0.36, x0+0.2, y0+0.7, base+0.4, roof, roof)
		}
		if b.propFirst(-1, 0) {
			porch()
		}
		b.Box(ax0, ay0, base, ax1, ay1, top, wall, wall)
		b.door(ax0, ay0, ax1, ay1, sideXM, 0.5, base, 0.15, 0.3)
		for _, s := range []int{sideYP, sideYM} {
			b.window(ax0, ay0, ax1, ay1, s, 0.3, base+0.2, 0.11, 0.13)
			b.window(ax0, ay0, ax1, ay1, s, 0.72, base+0.2, 0.11, 0.13)
		}
		b.window(ax0, ay0, ax1, ay1, sideXP, 0.5, base+0.2, 0.14, 0.13)
		if !b.propFirst(-1, 0) {
			porch()
		}
		b.Pyramid((ax0+ax1)/2, (ay0+ay1)/2, top, 0.56, 0.4, 4, math.Pi/4, roof)
		cx, cy = (ax0+ax1)/2, (ay0+ay1)/2
		ctop = chimney(b, cx, cy, top+0.4)
	default: // narrow two-storey house
		ax0, ay0, ax1, ay1 := x0+0.22, y0+0.2, x0+0.78, y0+0.8
		top := base + 0.82
		b.Box(ax0, ay0, base, ax1, ay1, top, wall, wall)
		b.door(ax0, ay0, ax1, ay1, sideYP, 0.5, base, 0.14, 0.28)
		for _, s := range []int{sideYP, sideYM} {
			b.window(ax0, ay0, ax1, ay1, s, 0.27, base+0.52, 0.1, 0.14)
			b.window(ax0, ay0, ax1, ay1, s, 0.73, base+0.52, 0.1, 0.14)
		}
		b.window(ax0, ay0, ax1, ay1, sideYM, 0.5, base+0.16, 0.12, 0.14)
		for _, s := range []int{sideXP, sideXM} {
			b.window(ax0, ay0, ax1, ay1, s, 0.5, base+0.16, 0.12, 0.14)
			b.window(ax0, ay0, ax1, ay1, s, 0.5, base+0.52, 0.12, 0.14)
		}
		b.Prism(x0+0.16, y0+0.14, x0+0.84, y0+0.86, top, 0.46, roof, wall)
		cx, cy = x0+0.36, y0+0.5
		ctop = chimney(b, cx, cy, top+0.46)
	}
	if puff {
		smoke(b, cx, cy, ctop, t, bd.ID)
	}
}
