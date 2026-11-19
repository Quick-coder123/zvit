package view

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"potik3d/internal/world"
)

// Ghost is a building preview shown under the cursor.
type Ghost struct {
	Kind   world.Kind
	X, Y   int
	OK     bool
	Dir    int // stairs direction
	Levels int // platform/bridge levels, stairs rise
	Base   int // level the building stands on (supports: the ground)
}

// Scene renders a world.
type Scene struct {
	Cam      Camera
	Hover    [2]int // hovered tile, -1 if none
	Ghost    *Ghost
	Selected int32
	Radius   [3]float64 // x, y, r of a highlighted work radius (r=0: none)
	order    []int
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
	colWood   = rgb(0xA5794D)
	colWoodLt = rgb(0xC8A06A)
	colWall   = rgb(0xE8DCC2)
	colRoof   = rgb(0xC9673F)
)

func (s *Scene) groundColor(w *world.World, x, y int) color.RGBA {
	i := world.Idx(x, y)
	h := w.Height[i]
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
		if w.Drought && !w.Moist(i) {
			c = mix(c, rgb(0xC9B06A), 0.55+0.2*hashf(y, x))
		}
	}
	return c
}

// visible returns tile indices around the camera target sorted far to near.
func (s *Scene) visible() []int {
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
	key := func(i int) float64 {
		_, r := c.rot(float64(i%world.Size)+0.5-c.X, float64(i/world.Size)+0.5-c.Y)
		return r
	}
	sort.Slice(s.order, func(a, b int) bool { return key(s.order[a]) < key(s.order[b]) })
	return s.order
}

func heightAt(w *world.World, x, y int) float64 {
	if !world.In(x, y) {
		return -1
	}
	return float64(w.Height[world.Idx(x, y)])
}

var sideDirs = [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

func sideQuad(fx, fy, z0, z1 float64, d [2]int) [4][3]float64 {
	switch {
	case d[1] == -1:
		return [4][3]float64{{fx, fy, z0}, {fx + 1, fy, z0}, {fx + 1, fy, z1}, {fx, fy, z1}}
	case d[1] == 1:
		return [4][3]float64{{fx, fy + 1, z0}, {fx + 1, fy + 1, z0}, {fx + 1, fy + 1, z1}, {fx, fy + 1, z1}}
	case d[0] == -1:
		return [4][3]float64{{fx, fy, z0}, {fx, fy + 1, z0}, {fx, fy + 1, z1}, {fx, fy, z1}}
	}
	return [4][3]float64{{fx + 1, fy, z0}, {fx + 1, fy + 1, z0}, {fx + 1, fy + 1, z1}, {fx + 1, fy, z1}}
}

// Draw renders the world.
func (s *Scene) Draw(dst *ebiten.Image, w *world.World) {
	c := &s.Cam
	c.prepare()
	dst.Fill(colSky)
	b := &Batch{dst: dst, cam: c}
	t := float64(w.Tick) / world.TPS

	front := map[int][]*world.Building{}
	for _, id := range w.Order() {
		bd := w.Buildings[id]
		if bd.Kind.IsSupport() {
			continue // drawn with their tile, below anything on top
		}
		fi := s.frontTile(bd.X, bd.Y, bd.Size())
		front[fi] = append(front[fi], bd)
	}
	var ghostTile = -1
	if s.Ghost != nil && world.In(s.Ghost.X, s.Ghost.Y) {
		ghostTile = s.frontTile(s.Ghost.X, s.Ghost.Y, world.Builds[s.Ghost.Kind].Size)
	}
	people := map[int][]*world.Settler{}
	for _, st := range w.Settlers {
		if st.Inside {
			continue
		}
		x, y := int(st.X), int(st.Y)
		if world.In(x, y) {
			k := world.Idx(x, y)
			people[k] = append(people[k], st)
		}
	}

	for _, i := range s.visible() {
		x, y := i%world.Size, i/world.Size
		fx, fy := float64(x), float64(y)
		h := float64(w.Height[i])
		top := h * LevelH

		for _, d := range sideDirs {
			nh := heightAt(w, x+d[0], y+d[1])
			if nh >= h || !c.facing(float64(d[0]), float64(d[1])) {
				continue
			}
			for lv := h - 1; lv >= math.Max(nh, -1); lv-- {
				col := colDirt
				if int(lv)%2 == 1 {
					col = colDirt2
				}
				if lv == h-1 {
					col = mix(col, s.groundColor(w, x, y), 0.25)
				}
				q := sideQuad(fx, fy, lv*LevelH, (lv+1)*LevelH, d)
				b.Poly(shade(col, float64(d[0]), float64(d[1]), 0), q[0], q[1], q[2], q[3])
			}
		}
		gc := shade(s.groundColor(w, x, y), 0, 0, 1)
		for _, d := range sideDirs {
			if heightAt(w, x+d[0], y+d[1]) > h {
				gc = mix(gc, rgb(0x2E3440), 0.08)
			}
		}
		if s.Hover[0] == x && s.Hover[1] == y && s.Ghost == nil {
			gc = mix(gc, rgb(0xFFFFFF), 0.3)
		}
		if s.Radius[2] > 0 && math.Hypot(fx+0.5-s.Radius[0], fy+0.5-s.Radius[1]) <= s.Radius[2] {
			gc = mix(gc, rgb(0xF2D16B), 0.22)
		}
		b.Poly(gc, [3]float64{fx, fy, top}, [3]float64{fx + 1, fy, top}, [3]float64{fx + 1, fy + 1, top}, [3]float64{fx, fy + 1, top})

		// water
		if dep := float64(w.Water[i]); dep > 0.03 {
			wz := (h + dep) * LevelH
			k := math.Min(1, dep/1.5)
			wc := mix(rgb(0x7CC0E4), colWater, k)
			wc = mix(wc, rgb(0xFFFFFF), math.Max(0, 0.05*math.Sin(t*1.5+fx*0.9+fy*0.6)))
			a := uint8(150 + 80*k)
			wc = rgba(uint32(wc.R)<<16|uint32(wc.G)<<8|uint32(wc.B), a)
			for _, d := range sideDirs {
				nx, ny := x+d[0], y+d[1]
				nl := -1.0
				if world.In(nx, ny) {
					j := world.Idx(nx, ny)
					nl = float64(w.Height[j]) + float64(w.Water[j])
				}
				if nl*LevelH >= wz-0.02 || !c.facing(float64(d[0]), float64(d[1])) {
					continue
				}
				q := sideQuad(fx, fy, math.Max(nl*LevelH, top), wz, d)
				b.Poly(shade(wc, float64(d[0]), float64(d[1]), 0.3), q[0], q[1], q[2], q[3])
			}
			b.Poly(wc, [3]float64{fx, fy, wz}, [3]float64{fx + 1, fy, wz}, [3]float64{fx + 1, fy + 1, wz}, [3]float64{fx, fy + 1, wz})
		}

		if sp := w.Support(i); sp != nil {
			hov := s.Hover[0] == x && s.Hover[1] == y && s.Ghost == nil
			drawSupport(b, w, sp, top, t, sp.ID == s.Selected || hov, sp.Levels)
		}
		for _, bd := range front[i] {
			base := float64(w.Surface(world.Idx(bd.X, bd.Y))) * LevelH
			drawBuilding(b, w, bd, base, t, bd.ID == s.Selected)
		}
		if i == ghostTile {
			g := s.Ghost
			gb := &world.Building{Kind: g.Kind, X: g.X, Y: g.Y, Dir: g.Dir, Levels: g.Levels}
			base := float64(g.Base) * LevelH
			b.Alpha = 0.6
			if g.Kind.IsSupport() {
				drawSupport(b, w, gb, base, t, false, g.Levels)
				base += float64(g.Levels) * LevelH
			} else {
				drawBuilding(b, w, gb, base, t, false)
			}
			col := rgba(0x4FA3A5, 0xC0)
			if !g.OK {
				col = rgba(0xE07A5F, 0xC0)
			}
			sz := float64(gb.Size())
			z := base + 0.02
			b.Alpha = 0
			gx, gy := float64(g.X), float64(g.Y)
			b.Poly(col, [3]float64{gx, gy, z}, [3]float64{gx + sz, gy, z}, [3]float64{gx + sz, gy + 0.08, z}, [3]float64{gx, gy + 0.08, z})
			b.Poly(col, [3]float64{gx, gy + sz - 0.08, z}, [3]float64{gx + sz, gy + sz - 0.08, z}, [3]float64{gx + sz, gy + sz, z}, [3]float64{gx, gy + sz, z})
			b.Poly(col, [3]float64{gx, gy, z}, [3]float64{gx + 0.08, gy, z}, [3]float64{gx + 0.08, gy + sz, z}, [3]float64{gx, gy + sz, z})
			b.Poly(col, [3]float64{gx + sz - 0.08, gy, z}, [3]float64{gx + sz, gy, z}, [3]float64{gx + sz, gy + sz, z}, [3]float64{gx + sz - 0.08, gy + sz, z})
		}
		for _, st := range people[i] {
			drawSettler(b, w, st, t)
		}
		if w.Owner(i) != nil {
			continue
		}
		switch w.Props[i] {
		case world.PropTree:
			drawTree(b, x, y, top, w.TreeGrowth(i))
		case world.PropBush:
			drawBush(b, x, y, top, w.BushRipe(i))
		case world.PropRock:
			b.Pyramid(fx+0.5, fy+0.5, top, 0.32, 0.28, 5, hashf(x, y)*3, colRock)
		}
	}
	b.flush()
}

func drawTree(b *Batch, x, y int, top, grow float64) {
	r := (0.85 + 0.3*hashf(x, y)) * (0.3 + 0.7*grow)
	fx, fy := float64(x), float64(y)
	cx, cy := fx+0.5+0.2*(hashf(x+7, y)-0.5), fy+0.5+0.2*(hashf(x, y+7)-0.5)
	b.Box(cx-0.07*r, cy-0.07*r, top, cx+0.07*r, cy+0.07*r, top+0.35*r, colTrunk, colTrunk)
	lc := colLeaf
	if (x+y)%3 == 0 {
		lc = colLeaf2
	}
	b.Pyramid(cx, cy, top+0.25*r, 0.42*r, 0.7*r, 6, hashf(x, y)*3, lc)
	b.Pyramid(cx, cy, top+0.65*r, 0.32*r, 0.6*r, 6, hashf(y, x)*3, mix(lc, rgb(0xFFFFFF), 0.08))
}

func drawBush(b *Batch, x, y int, top float64, ripe bool) {
	cx, cy := float64(x)+0.5, float64(y)+0.5
	b.Pyramid(cx, cy, top, 0.3, 0.3, 5, hashf(x, y)*3, colLeaf2)
	if ripe {
		for k := 0; k < 3; k++ {
			a := float64(k)*2.1 + hashf(x, y)*6
			bx, by := cx+math.Cos(a)*0.16, cy+math.Sin(a)*0.16
			b.Box(bx-0.04, by-0.04, top+0.12, bx+0.04, by+0.04, top+0.2, rgb(0xC0504D), rgb(0xD9605A))
		}
	}
}

// frontTile is the footprint tile nearest to the viewer.
func (s *Scene) frontTile(x0, y0, size int) int {
	c := &s.Cam
	best, bi := -1e9, 0
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			x, y := x0+dx, y0+dy
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

func hashf(x, y int) float64 {
	h := uint32(x*73856093) ^ uint32(y*19349663)
	h ^= h >> 13
	h *= 0x5bd1e995
	return float64(h&0xFFFF) / 65535
}

// Pick returns the tile whose top face is under the screen point.
func (s *Scene) Pick(w *world.World, mx, my float64) (int, int, bool) {
	c := &s.Cam
	c.prepare()
	order := s.visible()
	for k := len(order) - 1; k >= 0; k-- {
		i := order[k]
		x, y := i%world.Size, i/world.Size
		z := float64(w.Surface(i)) * LevelH
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

// ScreenOf projects a world point for UI overlays.
func (s *Scene) ScreenOf(x, y, level float64) (float64, float64) {
	s.Cam.prepare()
	sx, sy, _ := s.Cam.Project(x, y, level*LevelH)
	return sx, sy
}
