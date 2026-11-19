package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"potik3d/internal/view"
	"potik3d/internal/world"
)

// Concept mock-up scenes: hand-placed buildings and people to show the
// intended look before the mechanics exist.

func groundAt(w *world.World, x, y int) float64 { return float64(w.Height[world.Idx(x, y)]) }

func flatFree(w *world.World, x, y, size int) bool {
	if !world.In(x, y) || !world.In(x+size-1, y+size-1) {
		return false
	}
	h := w.Height[world.Idx(x, y)]
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			i := world.Idx(x+dx, y+dy)
			if w.Owner[i] != 0 || w.Height[i] != h || w.Water[i] > 0.1 || h < 2 {
				return false
			}
		}
	}
	return true
}

// put places a building at the free flat spot nearest to (x, y).
func put(w *world.World, k world.Kind, x, y, size int) *world.Building {
	for r := 0; r < 12; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if flatFree(w, x+dx, y+dy, size) {
					b := &world.Building{Kind: k, X: x + dx, Y: y + dy, Size: size}
					w.PlaceAt(b)
					return b
				}
			}
		}
	}
	return nil
}

func person(w *world.World, x, y float64, carry int) {
	tx, ty := int(x), int(y)
	z := 0.0
	if world.In(tx, ty) {
		z = groundAt(w, tx, ty)
	}
	w.Settlers = append(w.Settlers, &world.Settler{X: x, Y: y, Z: z, Carry: carry, Phase: x*3 + y})
}

func buildTown(w *world.World, cx, cy int) {
	put(w, world.KHall, cx, cy-1, 2)
	for k, p := range [][2]int{{3, -3}, {4, -1}, {3, 2}, {5, 1}, {6, -2}, {1, 3}, {5, 3}} {
		_ = k
		put(w, world.KHouse, cx+p[0], cy+p[1], 1)
	}
	put(w, world.KWarehouse, cx-1, cy+3, 2)
	put(w, world.KWaterTank, cx+1, cy-3, 1)
	s := put(w, world.KSite, cx+7, cy+1, 1)
	if s != nil {
		s.Prog = 0.45
	}
	put(w, world.KLumber, cx+7, cy-5, 1)
	// farms near the river bank
	rx := int(w.RiverX(float64(cy))) + 4
	for r := 0; r < 3; r++ {
		for k := 0; k < 3; k++ {
			put(w, world.KFarm, rx+k, cy-2+r, 1)
		}
	}
	for x := cx - 10; x < cx; x++ {
		if world.In(x, cy+1) && w.Water[world.Idx(x, cy+1)] < 0.1 && w.Height[world.Idx(x, cy+1)] >= 1 && w.Water[world.Idx(x-1, cy+1)] > 0.3 {
			w.PlaceAt(&world.Building{Kind: world.KPump, X: x, Y: cy + 1, Size: 1})
			break
		}
	}
	// people at work
	person(w, float64(cx)+4.5, float64(cy)-4.2, 1)
	person(w, float64(cx)+5.6, float64(cy)-3.6, 1)
	person(w, float64(cx)+2.4, float64(cy)+1.6, 2)
	person(w, float64(cx)+6.4, float64(cy)+1.3, 2)
	person(w, float64(cx)+7.7, float64(cy)+0.6, 0)
	person(w, float64(cx)-1.5, float64(cy)+1.8, 3)
	person(w, float64(cx)-3.3, float64(cy)+1.4, 3)
	person(w, float64(rx)+1.5, float64(cy)+1.7, 4)
	person(w, float64(rx)+2.6, float64(cy)-2.3, 4)
	person(w, float64(cx)+0.5, float64(cy)+5.6, 0)
	person(w, float64(cx)+3.6, float64(cy)+4.4, 1)
	person(w, float64(cx)-0.6, float64(cy)-2.4, 0)
}

// damRiver blocks the river with a dam at row y.
func damRiver(w *world.World, y int) {
	for x := 0; x < world.Size; x++ {
		if w.Height[world.Idx(x, y)] <= 1 && w.Owner[world.Idx(x, y)] == 0 {
			w.PlaceAt(&world.Building{Kind: world.KDam, X: x, Y: y, Size: 1})
		}
	}
}

// drought drains the river except for the reservoir upstream of the dam.
func drought(w *world.World, damY int) {
	w.Drought = true
	for y := 0; y < world.Size; y++ {
		for x := 0; x < world.Size; x++ {
			i := world.Idx(x, y)
			h := w.Height[i]
			switch {
			case y < damY && y > damY-14 && h == 0:
				w.Water[i] = 1.55
			case y < damY && y > damY-14 && h == 1:
				w.Water[i] = 0.55
			case y < damY-14 && h == 0:
				w.Water[i] = float32(0.25 * math.Max(0, 1-float64(damY-14-y)/10))
			default:
				w.Water[i] = 0
			}
		}
	}
}

// buildLevels shows stacking: platforms with houses on top, stairs and a bridge.
func buildLevels(w *world.World, cx, cy int) (int, int) {
	// find a terrace step near (cx, cy): a tile whose east neighbour is one level higher
	sx, sy := cx, cy
	found := false
	for r := 0; r < 14 && !found; r++ {
		for dy := -r; dy <= r && !found; dy++ {
			for dx := -r; dx <= r && !found; dx++ {
				x, y := cx+dx, cy+dy
				if world.In(x+1, y) && world.In(x, y+4) && w.Height[world.Idx(x, y)] >= 2 &&
					w.Height[world.Idx(x+1, y)] == w.Height[world.Idx(x, y)]+1 &&
					w.Owner[world.Idx(x, y)] == 0 && w.Water[world.Idx(x, y)] < 0.1 {
					sx, sy, found = x, y, true
				}
			}
		}
	}
	w.PlaceAt(&world.Building{Kind: world.KStairs, X: sx, Y: sy, Size: 1, Dir: 1})
	w.PlaceAt(&world.Building{Kind: world.KHouse, X: sx + 1, Y: sy, Size: 1})
	w.PlaceAt(&world.Building{Kind: world.KHouse, X: sx + 1, Y: sy + 1, Size: 1})
	for k := 1; k <= 3; k++ {
		w.PlaceAt(&world.Building{Kind: world.KPlatform, X: sx - k, Y: sy, Size: 1})
		w.PlaceAt(&world.Building{Kind: world.KHouse, X: sx - k, Y: sy, Size: 1, Level: 1})
		w.PlaceAt(&world.Building{Kind: world.KHouse, X: sx - k, Y: sy + 1, Size: 1})
	}
	w.PlaceAt(&world.Building{Kind: world.KPlatform, X: sx - 1, Y: sy - 1, Size: 1})
	w.PlaceAt(&world.Building{Kind: world.KWaterTank, X: sx - 1, Y: sy - 1, Size: 1, Level: 1})
	person(w, float64(sx)+0.5, float64(sy)+0.5, 2)
	w.Settlers[len(w.Settlers)-1].Z += 0.5
	person(w, float64(sx)-1.5, float64(sy)+0.5, 0)
	w.Settlers[len(w.Settlers)-1].Z += 1
	// bridge across the river near the town
	by := sy + 3
	bank := 2.0
	for x := 0; x < world.Size; x++ {
		i := world.Idx(x, by)
		if w.Height[i] <= 1 && math.Abs(float64(x)-w.RiverX(float64(by)+0.5)) < 4 {
			w.PlaceAt(&world.Building{Kind: world.KBridge, X: x, Y: by, Size: 1, Prog: bank - float64(w.Height[i])})
		}
	}
	person(w, w.RiverX(float64(by)+0.5)-0.6, float64(by)+0.5, 1)
	w.Settlers[len(w.Settlers)-1].Z = bank
	person(w, w.RiverX(float64(by)+0.5)+1.2, float64(by)+0.5, 0)
	w.Settlers[len(w.Settlers)-1].Z = bank
	return sx, sy
}

// --- mock interface ------------------------------------------------------------

func fill(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func label(dst *ebiten.Image, s string, x, y, size float64, c color.Color, center bool) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	if center {
		op.PrimaryAlign = text.AlignCenter
	}
	op.LineSpacing = size * 1.3
	text.Draw(dst, s, &text.GoTextFace{Source: font, Size: size}, op)
}

var (
	uiPanel = color.RGBA{0x2E, 0x34, 0x40, 0xD8}
	uiText  = color.RGBA{0xF4, 0xF1, 0xEA, 0xFF}
	uiSoft  = color.RGBA{0xB8, 0xC0, 0xCC, 0xFF}
	uiWarn  = color.RGBA{0xF2, 0xA6, 0x5A, 0xFF}
	uiOK    = color.RGBA{0x4F, 0xA3, 0xA5, 0xFF}
)

func drawMockUI(dst *ebiten.Image, u float64, scene string) {
	W, H := float64(dst.Bounds().Dx()), float64(dst.Bounds().Dy())
	// top bar: resources and season
	fill(dst, 0, 0, W, 40*u, uiPanel)
	res := []struct{ n, v string }{{"Жителі", "14 / 18"}, {"Деревина", "126"}, {"Дошки", "38"}, {"Їжа", "84"}, {"Вода", "142"}, {"Добробут", "3.6"}}
	if scene == "drought" {
		res[4].v = "96 · ↓"
	}
	x := 16 * u
	for _, r := range res {
		label(dst, r.n, x, 6*u, 11*u, uiSoft, false)
		label(dst, r.v, x, 20*u, 14*u, uiText, false)
		x += 104 * u
	}
	season, sc := "Сезон дощів · посуха через 3:20", uiText
	if scene == "drought" {
		season, sc = "ПОСУХА · ще 2:10 · водосховище 78%", uiWarn
	}
	label(dst, season, W-16*u-textW(season, 14*u), 12*u, 14*u, sc, false)
	label(dst, "День 12", W-16*u-textW("День 12", 11*u), 28*u, 11*u, uiSoft, false)

	// bottom build bar
	cats := []string{"Поселення", "Їжа", "Вода", "Ліс", "Виробництво", "Добробут", "Яруси"}
	bw := 104 * u
	bx := W/2 - float64(len(cats))*bw/2
	fill(dst, bx-8*u, H-58*u, float64(len(cats))*bw+16*u, 50*u, uiPanel)
	for i, c := range cats {
		cc := uiSoft
		if (scene == "levels" && i == 6) || (scene == "drought" && i == 2) || (scene == "town" && i == 0) {
			fill(dst, bx+float64(i)*bw+4*u, H-52*u, bw-8*u, 38*u, color.RGBA{0x4F, 0xA3, 0xA5, 0xC0})
			cc = uiText
		}
		label(dst, c, bx+float64(i)*bw+bw/2, H-42*u, 13*u, cc, true)
	}

	// side panel with a selected building
	var title, body string
	switch scene {
	case "town":
		title, body = "Склад", "Деревина 64 · Дошки 38 · Їжа 40\nМісце: 142 / 200\n\nНосії: 2 / 2"
	case "drought":
		title, body = "Гребля", "Тримає воду на 1 рівень.\nВодосховище: ~1 300 одиниць\nВистачить на ≈ 6 хв посухи.\n\nПоля в радіусі 4 від води\nлишаються зеленими."
	case "levels":
		title, body = "Платформа", "Будуй на висоті: будинки,\nбаки й майстерні стають\nодне на одне.\n\nСходи з'єднують рівні,\nмости — береги річки."
	}
	pw := 250 * u
	fill(dst, W-pw-16*u, 56*u, pw, 150*u, uiPanel)
	label(dst, title, W-pw-4*u, 66*u, 16*u, uiText, false)
	label(dst, body, W-pw-4*u, 92*u, 12*u, uiSoft, false)
	_ = fmt.Sprint
	_ = view.LevelH
}

func textW(s string, size float64) float64 {
	w, _ := text.Measure(s, &text.GoTextFace{Source: font, Size: size}, size*1.3)
	return w
}
