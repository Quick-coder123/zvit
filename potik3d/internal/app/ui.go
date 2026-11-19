package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"potik3d/internal/world"
)

var (
	fontRegular, fontBold *text.GoTextFaceSource
	faces                 = map[[2]int]*text.GoTextFace{}
)

func init() {
	var err error
	if fontRegular, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		panic(err)
	}
	if fontBold, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		panic(err)
	}
}

func (g *Game) face(size float64, bold bool) *text.GoTextFace {
	px := int(math.Round(size * g.ui))
	key := [2]int{px, 0}
	src := fontRegular
	if bold {
		key[1], src = 1, fontBold
	}
	f := faces[key]
	if f == nil {
		f = &text.GoTextFace{Source: src, Size: float64(px)}
		faces[key] = f
	}
	return f
}

var (
	colPanel  = color.RGBA{0x20, 0x26, 0x2B, 0xE6}
	colPanel2 = color.RGBA{0x2C, 0x34, 0x3A, 0xF0}
	colBtn    = color.RGBA{0x3A, 0x45, 0x4D, 0xFF}
	colBtnHi  = color.RGBA{0x4F, 0x5E, 0x68, 0xFF}
	colBtnOn  = color.RGBA{0x3F, 0x8F, 0x86, 0xFF}
	colText   = color.RGBA{0xEE, 0xEA, 0xE0, 0xFF}
	colDim    = color.RGBA{0xA8, 0xB0, 0xB4, 0xFF}
	colGood   = color.RGBA{0x8F, 0xD1, 0x7A, 0xFF}
	colBad    = color.RGBA{0xF0, 0x7A, 0x60, 0xFF}
	colWarn   = color.RGBA{0xF2, 0xC1, 0x4E, 0xFF}
	colBlue   = color.RGBA{0x7C, 0xC0, 0xF0, 0xFF}
)

func fill(dst *ebiten.Image, r rect, c color.Color) {
	vector.FillRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), c, false)
}

func (g *Game) txt(dst *ebiten.Image, s string, x, y, size float64, c color.Color, bold bool) float64 {
	f := g.face(size, bold)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = f.Size * 1.3
	text.Draw(dst, s, f, op)
	w, _ := text.Measure(s, f, op.LineSpacing)
	return w
}

func (g *Game) measure(s string, size float64, bold bool) float64 {
	f := g.face(size, bold)
	w, _ := text.Measure(s, f, f.Size*1.3)
	return w
}

// wrap breaks s into lines no wider than width.
func (g *Game) wrap(s string, size, width float64) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			try := word
			if line != "" {
				try = line + " " + word
			}
			if line != "" && g.measure(try, size, false) > width {
				out = append(out, line)
				line = word
			} else {
				line = try
			}
		}
		out = append(out, line)
	}
	return out
}

func (g *Game) btn(dst *ebiten.Image, r rect, label string, on bool, fn func()) {
	c := colBtn
	if on {
		c = colBtnOn
	} else if r.has(g.mx, g.my) {
		c = colBtnHi
	}
	fill(dst, r, c)
	tw := g.measure(label, 13, false)
	g.txt(dst, label, r.x+(r.w-tw)/2, r.y+(r.h-13*g.ui*1.15)/2, 13, colText, false)
	g.buttons = append(g.buttons, button{r, fn})
}

func (g *Game) drawUI(dst *ebiten.Image) {
	g.buttons, g.panels = g.buttons[:0], g.panels[:0]
	u := g.ui
	W, H := g.scene.Cam.W, g.scene.Cam.H

	g.drawTopBar(dst, W, u)
	g.drawBuildBar(dst, W, H, u)
	if b := g.w.Buildings[g.selected]; b != nil {
		g.drawSelection(dst, b, W, H, u)
	} else {
		g.selected = 0
	}
	if g.tool != world.KNone {
		g.drawToolHint(dst, W, H, u)
	}
	if g.toast != "" && time.Since(g.toastAt) < 6*time.Second {
		tw := g.measure(g.toast, 14, false)
		r := rect{(W-tw)/2 - 14*u, 52 * u, tw + 28*u, 30 * u}
		fill(dst, r, colPanel2)
		g.txt(dst, g.toast, r.x+14*u, r.y+7*u, 14, colText, false)
	}
	if g.help {
		g.drawHelp(dst, W, H, u)
	}
	if g.menu {
		g.drawMenu(dst, W, H, u)
	}
}

func (g *Game) drawTopBar(dst *ebiten.Image, W, u float64) {
	bar := rect{0, 0, W, 40 * u}
	fill(dst, bar, colPanel)
	g.panels = append(g.panels, bar)
	free, working, suffering := g.w.Roles()
	pop := len(g.w.Settlers)
	x := 14 * u
	item := func(label, val string, c color.Color) {
		x += g.txt(dst, label, x, 11*u, 13, colDim, false) + 5*u
		x += g.txt(dst, val, x, 10*u, 14, c, true) + 18*u
	}
	popC := color.Color(colText)
	if suffering > 0 {
		popC = colBad
	}
	item("Жителі", fmt.Sprintf("%d/%d", pop, g.w.Housing()), popC)
	item("вільні", fmt.Sprintf("%d", free), colText)
	item("працюють", fmt.Sprintf("%d", working), colText)
	_ = working
	food := g.w.Total(world.Berries) + g.w.Total(world.Crop)
	water := g.w.Total(world.Water)
	need := func(n int) color.Color {
		switch {
		case n == 0:
			return colBad
		case n < pop*2:
			return colWarn
		}
		return colText
	}
	item("Їжа", fmt.Sprint(food), need(food))
	item("Вода", fmt.Sprint(water), need(water))
	item("Колоди", fmt.Sprint(g.w.Total(world.Log)), colText)
	item("Дошки", fmt.Sprint(g.w.Total(world.Plank)), colText)
	used, total := g.w.Space()
	sc := color.Color(colText)
	if used >= total*9/10 {
		sc = colWarn
	}
	item("Склади", fmt.Sprintf("%d/%d", used, total), sc)

	// season and speed at the right
	bx := W - 4*(34*u) - 12*u
	labels := []string{"II", "x1", "x2", "x3"}
	for i, l := range labels {
		i := i
		on := (i == 0 && g.paused) || (i > 0 && !g.paused && g.speed == i)
		g.btn(dst, rect{bx + float64(i)*34*u, 6 * u, 30 * u, 28 * u}, l, on, func() {
			if i == 0 {
				g.paused = !g.paused
			} else {
				g.speed, g.paused = i, false
			}
		})
	}
	season, sc2 := "Сезон дощів", colBlue
	if g.w.Drought {
		season, sc2 = "ПОСУХА", colWarn
	}
	left := int(g.w.SeasonLeft)
	s := fmt.Sprintf("%s · %d:%02d · день %d", season, left/60, left%60, int(g.w.Day)+1)
	tw := g.measure(s, 14, true)
	g.txt(dst, s, bx-tw-18*u, 10*u, 14, sc2, true)
}

func costString(cost []world.Stack) string {
	var parts []string
	for _, c := range cost {
		parts = append(parts, fmt.Sprintf("%d %s", c.N, strings.ToLower(world.Items[c.Item].Name)))
	}
	if len(parts) == 0 {
		return "безкоштовно"
	}
	return strings.Join(parts, ", ")
}

func (g *Game) drawBuildBar(dst *ebiten.Image, W, H, u float64) {
	bh := 94 * u
	bar := rect{0, H - bh, W, bh}
	fill(dst, bar, colPanel)
	g.panels = append(g.panels, bar)
	x := 12 * u
	for c := world.Category(0); c < world.CatCount; c++ {
		c := c
		w := g.measure(world.CategoryNames[c], 13, false) + 22*u
		g.btn(dst, rect{x, H - bh + 8*u, w, 26 * u}, world.CategoryNames[c], g.cat == c, func() { g.cat = c })
		x += w + 6*u
	}
	g.txt(dst, "[ ]  категорії   1–9  будівлі   ПКМ/Esc  скасувати   F1  довідка", x+14*u, H-bh+13*u, 12, colDim, false)

	x = 12 * u
	for i, k := range buildKinds(g.cat) {
		k := k
		d := world.Builds[k]
		label := fmt.Sprintf("%d  %s", i+1, d.Name)
		w := math.Max(g.measure(label, 13, false), g.measure(costString(d.Cost), 11, false)) + 24*u
		r := rect{x, H - bh + 40*u, w, 46 * u}
		on := g.tool == k
		c := colBtn
		if on {
			c = colBtnOn
		} else if r.has(g.mx, g.my) {
			c = colBtnHi
		}
		fill(dst, r, c)
		g.txt(dst, label, r.x+12*u, r.y+6*u, 13, colText, true)
		cc := colDim
		for _, s := range d.Cost {
			if g.w.Total(s.Item) < s.N {
				cc = colWarn
			}
		}
		g.txt(dst, costString(d.Cost), r.x+12*u, r.y+26*u, 11, cc, false)
		g.buttons = append(g.buttons, button{r, func() { g.selectTool(k) }})
		x += w + 6*u
	}
}

func (g *Game) drawToolHint(dst *ebiten.Image, W, H, u float64) {
	d := world.Builds[g.tool]
	status, c := "Наведіть на місце для будівлі", color.Color(colDim)
	if g.hoverX >= 0 {
		x, yy := g.footprint(g.tool)
		if p, err := g.w.PlanPlacement(g.tool, x, yy); err != nil {
			status, c = err.Error(), colBad
		} else {
			status, c = placementHint(g.tool, p), colGood
		}
	}
	lines := g.wrap(d.Desc, 13, 300*u)
	slines := g.wrap(status, 13, 300*u)
	h := float64(len(lines)+len(slines))*17*u + 56*u
	r := rect{12 * u, H - 94*u - h - 10*u, 324 * u, h}
	fill(dst, r, colPanel2)
	g.panels = append(g.panels, r)
	g.txt(dst, d.Name, r.x+12*u, r.y+10*u, 15, colText, true)
	y := r.y + 34*u
	for _, l := range lines {
		g.txt(dst, l, r.x+12*u, y, 13, colDim, false)
		y += 17 * u
	}
	y += 4 * u
	for _, l := range slines {
		g.txt(dst, l, r.x+12*u, y, 13, c, false)
		y += 17 * u
	}
}

func (g *Game) drawSelection(dst *ebiten.Image, b *world.Building, W, H, u float64) {
	d := b.Def()
	pw := 330 * u
	var lines []struct {
		s string
		c color.Color
	}
	add := func(c color.Color, f string, a ...any) {
		for _, l := range g.wrap(fmt.Sprintf(f, a...), 13, pw-24*u) {
			lines = append(lines, struct {
				s string
				c color.Color
			}{l, c})
		}
	}
	add(colDim, "%s", d.Desc)
	add(colDim, "")
	switch {
	case b.Site:
		if len(b.Need) > 0 {
			var parts []string
			for _, n := range b.Need {
				parts = append(parts, fmt.Sprintf("%d %s", n.N, strings.ToLower(world.Items[n.Item].Name)))
			}
			add(colWarn, "Будується. Ще потрібно: %s", strings.Join(parts, ", "))
		} else {
			add(colWarn, "Будується: %d%%", int(b.Work*100))
		}
	case b.Paused:
		add(colWarn, "На паузі — працівники вільні")
	}
	switch b.Kind {
	case world.KPlatform:
		add(colText, "Рівнів: %d/%d", b.Levels, world.MaxStack)
		if g.w.BuildingAt(b.X, b.Y) != nil {
			add(colDim, "Зверху стоїть будівля — спершу знесіть її, щоб прибрати помост")
		}
	case world.KBridge:
		add(colText, "Висота настилу: %d", b.Levels)
	case world.KStairs:
		add(colText, "Підйом: %d рівн. на %s", b.Levels, dirNames[b.Dir&3])
	}
	if d.Workers > 0 && !b.Site {
		c := color.Color(colText)
		if b.Staff() < d.Workers {
			c = colBad
		}
		add(c, "Працівники: %d/%d (на місці %d)", b.Staff(), d.Workers, b.Inside())
		switch b.Kind {
		case world.KFarm:
			if !g.w.Moist(world.Idx(b.X, b.Y)) {
				add(colBad, "Ґрунт сухий — поле не росте")
			} else {
				add(colText, "Урожай: %d%%", int(b.Progress*100))
			}
		case world.KSawmill:
			add(colText, "Колод на вході: %d · дошок готово: %d", b.InN, b.OutN)
		default:
			if b.OutN > 0 {
				add(colText, "Готово до вивозу: %d %s", b.OutN, strings.ToLower(world.Items[b.Out].Name))
			}
		}
	}
	if d.Housing > 0 && !b.Site {
		add(colText, "Мешканці: %d/%d", g.w.Residents(b.ID), d.Housing)
	}
	if d.Cap > 0 && !b.Site {
		add(colText, "Зайнято: %d/%d", b.Used(), d.Cap)
		for _, s := range b.Store {
			if s.N > 0 {
				add(colDim, "  %s: %d", world.Items[s.Item].Name, s.N)
			}
		}
	}
	h := 48*u + float64(len(lines))*17*u + 46*u
	r := rect{W - pw - 12*u, 52 * u, pw, h}
	fill(dst, r, colPanel2)
	g.panels = append(g.panels, r)
	g.txt(dst, d.Name, r.x+12*u, r.y+12*u, 16, colText, true)
	y := r.y + 42*u
	for _, l := range lines {
		g.txt(dst, l.s, r.x+12*u, y, 13, l.c, false)
		y += 17 * u
	}
	if b.Kind != world.KHall {
		bw := (pw - 36*u) / 2
		if d.Workers > 0 {
			label := "Пауза (P)"
			if b.Paused {
				label = "Увімкнути (P)"
			}
			g.btn(dst, rect{r.x + 12*u, y + 8*u, bw, 28 * u}, label, b.Paused, func() { g.w.SetPaused(b.ID, !b.Paused) })
		}
		label := "Знести (Del)"
		if b.Site {
			label = "Скасувати (Del)"
		}
		g.btn(dst, rect{r.x + 24*u + bw, y + 8*u, bw, 28 * u}, label, false, func() { g.demolish(b.ID) })
	}
}

func (g *Game) overlay(dst *ebiten.Image, W, H float64) {
	r := rect{0, 0, W, H}
	fill(dst, r, color.RGBA{0, 0, 0, 0x90})
	g.panels = append(g.panels, r)
}

var helpText = `Ви — громада переселенців у долині з річкою. Розбудовуйте поселення, запасайте їжу й воду: кожні кілька хвилин настає ПОСУХА — річка міліє, ґрунт сохне, кущі й поля не родять.

Жителі самі беруть роботу в будівлях, а вільні — носять вантажі й будують. Будівлі ставляться як будмайданчики: жителі зносять туди матеріали й будують.

Почніть так: Збирач ягід і Насос біля води, Лісоруб у лісі, Будинки для нових жителів. Далі — Лісник, Поле на вологій землі, Бак для води, Тартак і Склади. Греблею перегородіть річку — вода триматиметься довше.

Камера: WASD/стрілки — рух (Shift швидше), Q/E — поворот, ПКМ + рух миші — оберт і нахил, R/F — нахил, коліщатко — масштаб, СКМ — тягнути.
Гра: Пробіл — пауза, , . — швидкість, ЛКМ — вибрати будівлю, P — пауза будівлі, Del — знести, Ctrl+S — зберегти, Esc — меню.
Гра зберігається сама кожні 30 секунд і при виході.`

func (g *Game) drawHelp(dst *ebiten.Image, W, H, u float64) {
	g.overlay(dst, W, H)
	pw := math.Min(W-40*u, 720*u)
	lines := g.wrap(helpText, 14, pw-48*u)
	ph := float64(len(lines))*19*u + 120*u
	r := rect{(W - pw) / 2, math.Max(20*u, (H-ph)/2), pw, ph}
	fill(dst, r, colPanel2)
	g.txt(dst, "Потік — поселення в долині", r.x+24*u, r.y+18*u, 20, colText, true)
	y := r.y + 58*u
	for _, l := range lines {
		g.txt(dst, l, r.x+24*u, y, 14, colText, false)
		y += 19 * u
	}
	g.btn(dst, rect{r.x + pw - 184*u, r.y + ph - 44*u, 160 * u, 30 * u}, "Грати (F1)", true, func() { g.help = false })
}

func (g *Game) drawMenu(dst *ebiten.Image, W, H, u float64) {
	g.overlay(dst, W, H)
	pw, ph := 340*u, 300*u
	r := rect{(W - pw) / 2, (H - ph) / 2, pw, ph}
	fill(dst, r, colPanel2)
	g.txt(dst, "Пауза", r.x+24*u, r.y+18*u, 20, colText, true)
	g.txt(dst, "версія "+Version, r.x+pw-24*u-g.measure("версія "+Version, 11, false), r.y+24*u, 11, colDim, false)
	y := r.y + 64*u
	row := func(label string, on bool, fn func()) {
		g.btn(dst, rect{r.x + 24*u, y, pw - 48*u, 34 * u}, label, on, fn)
		y += 42 * u
	}
	if g.confirmNew {
		lines := g.wrap("Почати нову долину? Поточний світ буде збережено в резервну копію поруч із сейвом.", 13, pw-48*u)
		for _, l := range lines {
			g.txt(dst, l, r.x+24*u, y, 13, colWarn, false)
			y += 17 * u
		}
		y += 10 * u
		row("Так, почати заново", true, g.newGame)
		row("Ні", false, func() { g.confirmNew = false })
		return
	}
	row("Продовжити", true, func() { g.menu = false })
	row("Зберегти", false, func() { g.Save(); g.say("Збережено"); g.menu = false })
	row("Почати заново", false, func() { g.confirmNew = true })
	if g.updateReady && g.restart != nil {
		row("Перезапустити з оновленням", true, func() {
			g.Save()
			if err := g.restart(); err != nil {
				g.say("Не вдалося перезапустити: %v", err)
				return
			}
			g.quit = true
		})
	}
	row("Зберегти й вийти", false, func() { g.quit = true })
}

func (g *Game) saveShot(screen *ebiten.Image) {
	if g.stats {
		fmt.Fprintf(os.Stderr, "tris=%d\n", g.scene.LastTris)
	}
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(g.shot)
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
	g.quit = true
}
