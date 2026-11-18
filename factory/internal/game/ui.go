package game

import (
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"potik/internal/sim"
)

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h }

type button struct {
	r      rect
	action func()
}

func (g *Game) hitButton(x, y float64) *button {
	for i := len(g.buttons) - 1; i >= 0; i-- {
		if g.buttons[i].r.has(x, y) {
			return &g.buttons[i]
		}
	}
	return nil
}

func (g *Game) overPanel(x, y float64) bool {
	for _, r := range g.panels {
		if r.has(x, y) {
			return true
		}
	}
	return false
}

func (g *Game) addButton(r rect, action func()) { g.buttons = append(g.buttons, button{r, action}) }

func panel(dst *ebiten.Image, r rect, u float64) {
	fillRoundRect(dst, float32(r.x), float32(r.y+2*u), float32(r.w), float32(r.h), float32(10*u), rgba(0x2E3440, 0x18))
	fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(10*u), colPanel)
	strokeRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(10*u), float32(1*u), colPanelLine)
}

// iconBuilding draws a small picture of a building kind in a box at (x,y).
func (g *Game) iconBuilding(dst *ebiten.Image, k sim.Kind, x, y, size float64, d sim.Dir) {
	saved := g.cam
	g.iconMode = true
	defer func() { g.iconMode = false }()
	n := float64(sim.Builds[k].Size)
	g.cam.Scale = size / n
	g.cam.X = (g.cam.W/2 - x) / g.cam.Scale
	g.cam.Y = (g.cam.H/2 - y) / g.cam.Scale
	b := &sim.Building{Kind: k, Dir: d, Recipe: -1}
	if k == sim.KBelt {
		g.drawBelt(dst, b)
	} else {
		g.drawBuilding(dst, b, 1)
	}
	g.cam = saved
}

func fmtCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 10_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

func (g *Game) drawUI(dst *ebiten.Image) {
	g.buttons = g.buttons[:0]
	g.panels = g.panels[:0]
	u := g.ui
	W, H := g.cam.W, g.cam.H

	g.drawStock(dst)
	g.drawToolbar(dst)
	if g.selected != 0 {
		if b := g.w.Buildings[g.selected]; b != nil {
			g.drawInfo(dst, b)
		} else {
			g.selected = 0
		}
	}

	// hint line
	hint := "ЛКМ — будувати/обрати · ПКМ — знести · R — повернути · V — розвернути лінію · Ctrl+тягни — копіювати/знести область · F1 — довідка"
	if g.pasting {
		hint = "Вставка: ЛКМ — поставити · R — повернути · ПКМ або Esc — скасувати"
	}
	hw := textWidth(hint, 12*u, false) + 24*u
	fillRoundRect(dst, float32(W/2-hw/2), float32(H-30*u), float32(hw), float32(22*u), float32(11*u), rgba(0xF4F1EA, 0xD8))
	drawText(dst, hint, W/2, H-27*u, 12*u, false, colInkSoft, alignCenter)

	// toast
	if g.toast != "" {
		age := time.Since(g.toastTime).Seconds()
		if age < 2.8 {
			a := 1.0
			if age > 2.2 {
				a = (2.8 - age) / 0.6
			}
			tw := textWidth(g.toast, 15*u, true) + 32*u
			r := rect{W/2 - tw/2, 18 * u, tw, 38 * u}
			fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(19*u), rgba(0x2E3440, uint8(230*a)))
			drawText(dst, g.toast, W/2, r.y+9*u, 15*u, true, rgba(0xF4F1EA, uint8(255*a)), alignCenter)
		}
	}

	if g.help {
		g.drawHelp(dst)
	}
	if g.menu {
		g.drawMenu(dst)
	}
}

func (g *Game) drawStock(dst *ebiten.Image) {
	u := g.ui
	var items []sim.Item
	for it := sim.Item(1); it < sim.ItemCount; it++ {
		if g.w.Stock[it] > 0 {
			items = append(items, it)
		}
	}
	rows := (len(items) + 1) / 2
	r := rect{14 * u, 14 * u, 250 * u, (44 + float64(rows)*26) * u}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	drawText(dst, "Запас ядра", r.x+14*u, r.y+10*u, 14*u, true, colInk, alignLeft)
	for i, it := range items {
		col, row := i%2, i/2
		x := r.x + 14*u + float64(col)*118*u
		y := r.y + 40*u + float64(row)*26*u
		drawItem(dst, it, float32(x+9*u), float32(y+8*u), float32(16*u))
		drawText(dst, fmtCount(g.w.Stock[it]), x+24*u, y, 13*u, false, colInk, alignLeft)
		if (rect{x, y - 4*u, 110 * u, 24 * u}).has(g.mouseX, g.mouseY) {
			g.tooltip(dst, sim.Items[it].Name, "")
		}
	}
	if len(items) == 0 {
		drawText(dst, "порожньо", r.x+14*u, r.y+40*u, 13*u, false, colInkSoft, alignLeft)
	}
}

func (g *Game) drawToolbar(dst *ebiten.Image) {
	u := g.ui
	slot, gap := 64*u, 8*u
	n := float64(len(toolbar))
	tw := n*slot + (n-1)*gap + 20*u
	r := rect{g.cam.W/2 - tw/2, g.cam.H - slot - 60*u, tw, slot + 20*u}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	for i, k := range toolbar {
		k := k
		sr := rect{r.x + 10*u + float64(i)*(slot+gap), r.y + 10*u, slot, slot}
		bg := rgb(0xF7F4EE)
		if g.tool == k {
			bg = mix(colBody, colOK, 0.25)
		}
		fillRoundRect(dst, float32(sr.x), float32(sr.y), float32(sr.w), float32(sr.h), float32(8*u), bg)
		if g.tool == k {
			strokeRoundRect(dst, float32(sr.x), float32(sr.y), float32(sr.w), float32(sr.h), float32(8*u), float32(2*u), colOK)
		}
		g.iconBuilding(dst, k, sr.x+12*u, sr.y+10*u, slot-24*u, sim.East)
		afford := g.w.CanAfford(k)
		nc := colInkSoft
		if !afford {
			nc = colBad
		}
		drawText(dst, fmt.Sprint(i+1), sr.x+5*u, sr.y+2*u, 11*u, true, nc, alignLeft)
		g.addButton(sr, func() { g.selectTool(k) })
		if sr.has(g.mouseX, g.mouseY) {
			g.buildTooltip(dst, k)
		}
	}
}

func (g *Game) costLine(k sim.Kind) string {
	s := ""
	for i, c := range sim.Builds[k].Cost {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%s ×%d", sim.Items[c.Item].Name, c.N)
	}
	return s
}

func (g *Game) buildTooltip(dst *ebiten.Image, k sim.Kind) {
	def := sim.Builds[k]
	body := def.Desc + "\nЦіна: " + g.costLine(k)
	if !g.w.CanAfford(k) {
		body += "\nБракує ресурсів у ядрі"
	}
	g.tooltip(dst, def.Name, body)
}

func (g *Game) tooltip(dst *ebiten.Image, title, body string) {
	u := g.ui
	w := textWidth(title, 14*u, true)
	lines := 0
	if body != "" {
		w = max64(w, textWidth(body, 12.5*u, false))
		for _, ch := range body {
			if ch == '\n' {
				lines++
			}
		}
		lines++
	}
	h := 30*u + float64(lines)*17*u
	x, y := g.mouseX+16*u, g.mouseY-h-8*u
	if x+w+24*u > g.cam.W {
		x = g.cam.W - w - 24*u
	}
	if y < 4*u {
		y = g.mouseY + 24*u
	}
	r := rect{x, y, w + 24*u, h}
	fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(8*u), rgba(0x2E3440, 0xF0))
	drawText(dst, title, r.x+12*u, r.y+7*u, 14*u, true, colBG, alignLeft)
	if body != "" {
		drawText(dst, body, r.x+12*u, r.y+28*u, 12.5*u, false, rgb(0xD8DEE9), alignLeft)
	}
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (g *Game) drawInfo(dst *ebiten.Image, b *sim.Building) {
	u := g.ui
	def := sim.Builds[b.Kind]
	W := 300 * u
	r := rect{g.cam.W - W - 14*u, 14 * u, W, 0}
	// measure content height first by laying out in a closure
	type rowFn func(y float64) float64
	var rows []rowFn
	pad := 16 * u
	add := func(f rowFn) { rows = append(rows, f) }

	add(func(y float64) float64 {
		drawText(dst, def.Name, r.x+pad, y, 17*u, true, colInk, alignLeft)
		return 30 * u
	})
	add(func(y float64) float64 {
		drawText(dst, wrap(def.Desc, 38), r.x+pad, y, 12.5*u, false, colInkSoft, alignLeft)
		return float64(1+countLines(wrap(def.Desc, 38))) * 17 * u
	})

	switch b.Kind {
	case sim.KMiner:
		add(func(y float64) float64 {
			if b.MinerItem == sim.ItemNone {
				drawText(dst, "Під буром немає покладу", r.x+pad, y, 13*u, false, colBad, alignLeft)
				return 24 * u
			}
			drawItem(dst, b.MinerItem, float32(r.x+pad+9*u), float32(y+9*u), float32(18*u))
			rate := 60 / (sim.MinerTime * 4 / float64(b.MinerTiles))
			drawText(dst, fmt.Sprintf("%s · %.0f/хв", sim.Items[b.MinerItem].Name, rate), r.x+pad+26*u, y, 13*u, false, colInk, alignLeft)
			return 28 * u
		})
	case sim.KFurnace, sim.KAssembler:
		add(func(y float64) float64 {
			drawText(dst, "Рецепт", r.x+pad, y, 13*u, true, colInk, alignLeft)
			return 22 * u
		})
		for _, ri := range sim.RecipesFor(b.Kind) {
			ri := ri
			add(func(y float64) float64 {
				rr := rect{r.x + pad, y, W - 2*pad, 30 * u}
				bg := rgb(0xF4F0E8)
				if b.Recipe == ri {
					bg = mix(colBody, colOK, 0.28)
				} else if rr.has(g.mouseX, g.mouseY) {
					bg = rgb(0xECE6DA)
				}
				fillRoundRect(dst, float32(rr.x), float32(rr.y), float32(rr.w), float32(rr.h), float32(6*u), bg)
				g.recipeRow(dst, sim.Recipes[ri], rr.x+8*u, rr.y+15*u)
				g.addButton(rr, func() { g.w.SetRecipe(b.ID, ri) })
				return 34 * u
			})
		}
		if b.Kind == sim.KFurnace {
			add(func(y float64) float64 {
				msg := "Обирається сама за першою рудою"
				if !b.AutoRecipe {
					msg = "Рецепт закріплено вручну"
				}
				drawText(dst, msg, r.x+pad, y, 11.5*u, false, colInkSoft, alignLeft)
				return 20 * u
			})
		}
		add(func(y float64) float64 {
			if b.Recipe < 0 {
				return 0
			}
			rc := sim.Recipes[b.Recipe]
			x := r.x + pad
			for i, in := range rc.In {
				drawItem(dst, in.Item, float32(x+9*u), float32(y+10*u), float32(16*u))
				drawText(dst, fmt.Sprint(b.InBuf[i].N), x+22*u, y+2*u, 13*u, false, colInk, alignLeft)
				x += 60 * u
			}
			drawText(dst, "→", x, y+1*u, 14*u, true, colInkSoft, alignLeft)
			x += 24 * u
			drawItem(dst, rc.Out.Item, float32(x+9*u), float32(y+10*u), float32(16*u))
			drawText(dst, fmt.Sprint(b.OutN), x+22*u, y+2*u, 13*u, false, colInk, alignLeft)
			// progress
			py := y + 28*u
			fillRoundRect(dst, float32(r.x+pad), float32(py), float32(W-2*pad), float32(6*u), float32(3*u), rgb(0xE6E0D4))
			if b.Progress > 0 {
				fillRoundRect(dst, float32(r.x+pad), float32(py), float32((W-2*pad)*float64(b.Progress)), float32(6*u), float32(3*u), colAccent[b.Kind])
			}
			return 42 * u
		})
	case sim.KSorter:
		add(func(y float64) float64 {
			drawText(dst, "Фільтр (іде вбік)", r.x+pad, y, 13*u, true, colInk, alignLeft)
			return 24 * u
		})
		add(func(y float64) float64 {
			cell := 34 * u
			cols := int((W - 2*pad) / cell)
			opts := []sim.Item{sim.ItemNone}
			for it := sim.Item(1); it < sim.ItemCount; it++ {
				opts = append(opts, it)
			}
			for i, it := range opts {
				it := it
				cr := rect{r.x + pad + float64(i%cols)*cell, y + float64(i/cols)*cell, cell - 4*u, cell - 4*u}
				bg := rgb(0xF4F0E8)
				if b.Filter == it {
					bg = mix(colBody, colOK, 0.35)
				} else if cr.has(g.mouseX, g.mouseY) {
					bg = rgb(0xE9E2D4)
				}
				fillRoundRect(dst, float32(cr.x), float32(cr.y), float32(cr.w), float32(cr.h), float32(6*u), bg)
				if it == sim.ItemNone {
					drawText(dst, "—", cr.x+cr.w/2, cr.y+cr.h/2-9*u, 14*u, true, colInkSoft, alignCenter)
				} else {
					drawItem(dst, it, float32(cr.x+cr.w/2), float32(cr.y+cr.h/2), float32(18*u))
				}
				if cr.has(g.mouseX, g.mouseY) {
					name := "Без фільтра: все йде прямо"
					if it != sim.ItemNone {
						name = sim.Items[it].Name
					}
					defer g.tooltip(dst, name, "")
				}
				g.addButton(cr, func() { g.w.SetFilter(b.ID, it) })
			}
			rows := (len(opts) + cols - 1) / cols
			return float64(rows)*cell + 6*u
		})
	case sim.KTunnel:
		add(func(y float64) float64 {
			msg := "Вихід тунелю."
			c := colInk
			if !b.Exit {
				if ex, k := g.w.TunnelExit(b); ex != nil {
					msg = fmt.Sprintf("Вхід. З'єднаний з виходом через %d клітинок.", k-1)
				} else {
					msg = "Вхід без виходу. Постав ще один тунель\nу тому ж напрямку не далі ніж за 5 клітинок."
					c = colBad
				}
			}
			drawText(dst, msg, r.x+pad, y, 12.5*u, false, c, alignLeft)
			return float64(countLines(msg))*18*u + 6*u
		})
	case sim.KChest:
		add(func(y float64) float64 {
			drawText(dst, fmt.Sprintf("Заповнено: %d / %d", b.StoreCount(), sim.ChestCap), r.x+pad, y, 13*u, false, colInk, alignLeft)
			return 22 * u
		})
	case sim.KCore:
		add(func(y float64) float64 {
			drawText(dst, "Привозь сюди пластини й деталі —\nз них будуються нові машини.", r.x+pad, y, 12.5*u, false, colInk, alignLeft)
			return 40 * u
		})
	}

	if b.Kind != sim.KCore {
		add(func(y float64) float64 {
			bw := (W - 2*pad - 8*u) / 2
			b1 := rect{r.x + pad, y + 4*u, bw, 30 * u}
			b2 := rect{b1.x + bw + 8*u, y + 4*u, bw, 30 * u}
			g.smallButton(dst, b1, "Повернути (R)", func() { g.w.Rotate(b.ID) })
			g.smallButton(dst, b2, "Знести", func() {
				g.w.Remove(b.ID)
				g.selected = 0
			})
			return 42 * u
		})
	}

	if g.infoFor != b.ID {
		g.infoFor, g.infoHeight = b.ID, 0
	}
	r.h = g.infoHeight
	if r.h == 0 {
		r.h = 200 * u
	}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	y := r.y + pad
	for _, f := range rows {
		y += f(y)
	}
	g.infoHeight = y - r.y + pad/2
}

func (g *Game) recipeRow(dst *ebiten.Image, rc sim.Recipe, x, cy float64) {
	u := g.ui
	for i, in := range rc.In {
		if i > 0 {
			drawText(dst, "+", x, cy-9*u, 13*u, false, colInkSoft, alignLeft)
			x += 14 * u
		}
		drawItem(dst, in.Item, float32(x+8*u), float32(cy), float32(15*u))
		drawText(dst, fmt.Sprint(in.N), x+18*u, cy-8*u, 12*u, false, colInk, alignLeft)
		x += 32 * u
	}
	drawText(dst, "→", x, cy-9*u, 13*u, true, colInkSoft, alignLeft)
	x += 20 * u
	drawItem(dst, rc.Out.Item, float32(x+8*u), float32(cy), float32(15*u))
	drawText(dst, fmt.Sprintf("%d  %s · %.1fс", rc.Out.N, sim.Items[rc.Out.Item].Name, rc.Time), x+18*u, cy-8*u, 12*u, false, colInk, alignLeft)
}

func (g *Game) smallButton(dst *ebiten.Image, r rect, label string, action func()) {
	u := g.ui
	bg := rgb(0xF0EBE1)
	if r.has(g.mouseX, g.mouseY) {
		bg = rgb(0xE5DED0)
	}
	fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(6*u), bg)
	drawText(dst, label, r.x+r.w/2, r.y+r.h/2-8*u, 12.5*u, true, colInk, alignCenter)
	g.addButton(r, action)
}

func (g *Game) bigButton(dst *ebiten.Image, r rect, label string, action func()) {
	u := g.ui
	bg := rgb(0xF0EBE1)
	if r.has(g.mouseX, g.mouseY) {
		bg = mix(colBody, colOK, 0.3)
	}
	fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(8*u), bg)
	drawText(dst, label, r.x+r.w/2, r.y+r.h/2-10*u, 15*u, true, colInk, alignCenter)
	g.addButton(r, action)
}

func (g *Game) dim(dst *ebiten.Image) {
	fillRect(dst, 0, 0, float32(g.cam.W), float32(g.cam.H), rgba(0x2E3440, 0x55))
	g.panels = append(g.panels, rect{0, 0, g.cam.W, g.cam.H})
}

func (g *Game) drawMenu(dst *ebiten.Image) {
	u := g.ui
	g.dim(dst)
	W, H := 320*u, 290*u
	r := rect{g.cam.W/2 - W/2, g.cam.H/2 - H/2, W, H}
	panel(dst, r, u)
	drawText(dst, "Потік", r.x+W/2, r.y+20*u, 26*u, true, colInk, alignCenter)
	drawText(dst, "пауза", r.x+W/2, r.y+56*u, 13*u, false, colInkSoft, alignCenter)
	bw := W - 60*u
	y := r.y + 90*u
	g.bigButton(dst, rect{r.x + 30*u, y, bw, 42 * u}, "Продовжити", func() { g.menu = false })
	g.bigButton(dst, rect{r.x + 30*u, y + 52*u, bw, 42 * u}, "Зберегти", func() { g.Save(); g.say("Збережено") })
	g.bigButton(dst, rect{r.x + 30*u, y + 104*u, bw, 42 * u}, "Зберегти й вийти", func() { g.quit = true })
	ago := time.Since(g.lastSave).Round(time.Second)
	drawText(dst, fmt.Sprintf("Автозбереження: %v тому", ago), r.x+W/2, r.y+H-30*u, 11.5*u, false, colInkSoft, alignCenter)
}

const helpText = `Мета зараз: налагодити виробництво і наповнити ядро.

1. Постав бур (6) на поклад руди. Він видає руду вперед — туди, куди дивиться стрілка.
2. Протягни конвеєр (1) від бура до печі (7). Піч сама зрозуміє, що плавити.
3. Від печі веди конвеєр у ядро. Пластини потрапляють у запас, з якого будується все інше.
4. Збирачу (8) обери рецепт (клацни по ньому): шестерні й дріт теж потрібні для будівництва.

Логістика
Тунель (2) — вхід, потім вихід у тому ж напрямку; між ними до 5 клітинок під чим завгодно.
Розгалужувач (3) ділить потік уліво/вперед/вправо, Об'єднувач (4) чесно зливає до трьох потоків.
Сортер (5) відправляє обраний предмет убік, решту — прямо.

Поклади: сіро-блакитний — залізо, мідний — мідь, темний — вугілля,
бежевий — камінь, жовтий — кварцовий пісок.

Керування
ЛКМ — поставити (тягни конвеєр мишею) · ПКМ — знести (можна тягнути)
R / Shift+R — повернути · Q — взяти будівлю під курсором · 0 — без інструмента
V над конвеєром — розвернути всю лінію
Ctrl + тягни ЛКМ — скопіювати область · Ctrl + тягни ПКМ — знести область · Ctrl+V — вставити ще раз
WASD або середня кнопка — камера · колесо — масштаб · Shift — швидше
Ctrl+S — зберегти (є й автозбереження) · Esc — меню`

func (g *Game) drawHelp(dst *ebiten.Image) {
	u := g.ui
	g.dim(dst)
	W := 640 * u
	H := (float64(countLines(helpText))*18 + 120) * u
	r := rect{g.cam.W/2 - W/2, g.cam.H/2 - H/2, W, H}
	panel(dst, r, u)
	drawText(dst, "Як грати", r.x+24*u, r.y+20*u, 20*u, true, colInk, alignLeft)
	drawText(dst, helpText, r.x+24*u, r.y+58*u, 13*u, false, colInk, alignLeft)
	g.bigButton(dst, rect{r.x + W - 164*u, r.y + H - 54*u, 140 * u, 38 * u}, "Зрозуміло", func() { g.help = false })
}

func countLines(s string) int {
	n := 1
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}

// wrap breaks text into lines of at most n runes on word boundaries.
func wrap(s string, n int) string {
	out := []rune{}
	lineLen := 0
	word := []rune{}
	flush := func() {
		if lineLen > 0 && lineLen+1+len(word) > n {
			out = append(out, '\n')
			lineLen = 0
		} else if lineLen > 0 {
			out = append(out, ' ')
			lineLen++
		}
		out = append(out, word...)
		lineLen += len(word)
		word = word[:0]
	}
	for _, c := range s {
		if c == ' ' || c == '\n' {
			if len(word) > 0 {
				flush()
			}
			if c == '\n' {
				out = append(out, '\n')
				lineLen = 0
			}
			continue
		}
		word = append(word, c)
	}
	if len(word) > 0 {
		flush()
	}
	return string(out)
}

var _ color.Color = colInk
