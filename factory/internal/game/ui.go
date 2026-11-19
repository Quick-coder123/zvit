package game

import (
	"fmt"
	"image/color"
	"math"
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
	g.drawBuilding(dst, b, 1)
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
	g.drawWaveHUD(dst)
	if g.selected != 0 {
		if b := g.w.Buildings[g.selected]; b != nil {
			g.drawInfo(dst, b)
		} else {
			g.selected = 0
		}
	}

	// hint line
	hint := "ЛКМ — будувати/обрати · ПКМ — знести · P — пауза будівлі · T — дослідження · Ctrl+тягни — копіювати/знести · F1 — довідка"
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
			ty := g.hudBottom + 4*u
			r := rect{W/2 - tw/2, ty, tw, 38 * u}
			fillRoundRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), float32(19*u), rgba(0x2E3440, uint8(230*a)))
			drawText(dst, g.toast, W/2, r.y+9*u, 15*u, true, rgba(0xF4F1EA, uint8(255*a)), alignCenter)
		}
	}

	if g.research {
		g.drawResearch(dst)
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
		if g.stock[it] > 0 {
			items = append(items, it)
		}
	}
	rows := max((len(items)+1)/2, 1)
	r := rect{14 * u, 14 * u, 250 * u, (50 + float64(rows)*26 + 152) * u}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	g.drawTown(dst, r.x+14*u, r.y+10*u)
	r.y += 152 * u
	line(dst, float32(r.x+14*u), float32(r.y), float32(r.x+r.w-14*u), float32(r.y), float32(u), colPanelLine)
	drawText(dst, "Запаси", r.x+14*u, r.y+10*u, 14*u, true, colInk, alignLeft)
	used, total := g.w.StorageTotals()
	uc := colInkSoft
	if used >= total*9/10 {
		uc = colBad
	}
	drawText(dst, fmt.Sprintf("місце %d / %d", used, total), r.x+r.w-14*u, r.y+12*u, 11.5*u, true, uc, alignRight)
	g.capacityBar(dst, r.x+14*u, r.y+32*u, r.w-28*u, used, total)
	for i, it := range items {
		col, row := i%2, i/2
		x := r.x + 14*u + float64(col)*118*u
		y := r.y + 46*u + float64(row)*26*u
		drawItem(dst, it, float32(x+9*u), float32(y+8*u), float32(16*u))
		drawText(dst, fmtCount(g.stock[it]), x+24*u, y, 13*u, false, colInk, alignLeft)
		if (rect{x, y - 4*u, 110 * u, 24 * u}).has(g.mouseX, g.mouseY) {
			g.tooltip(dst, sim.Items[it].Name, "")
		}
	}
	if len(items) == 0 {
		drawText(dst, "порожньо", r.x+14*u, r.y+46*u, 13*u, false, colInkSoft, alignLeft)
	}
}

// capacityBar draws a thin fill bar for storage space.
func (g *Game) capacityBar(dst *ebiten.Image, x, y, w float64, used, cap int) {
	u := g.ui
	f := 0.0
	if cap > 0 {
		f = math.Min(1, float64(used)/float64(cap))
	}
	c := colOK
	switch {
	case f >= 1:
		c = colBad
	case f >= 0.85:
		c = rgb(0xF2A65A)
	}
	fillRoundRect(dst, float32(x), float32(y), float32(w), float32(5*u), float32(2.5*u), rgb(0xE6E0D4))
	fillRoundRect(dst, float32(x), float32(y), float32(w*f), float32(5*u), float32(2.5*u), c)
}

// drawTown shows population, jobs and food.
func (g *Game) drawTown(dst *ebiten.Image, x, y float64) {
	u := g.ui
	w := g.w
	free, workers, carriers, starving := w.RoleCounts()
	drawText(dst, "Поселення", x, y, 14*u, true, colInk, alignLeft)
	pc := colInk
	if w.Population() >= w.Housing() {
		pc = colInkSoft
	}
	personGlyph(dst, float32(x+8*u), float32(y+38*u), float32(14*u), rgb(0x4FA3A5))
	drawText(dst, fmt.Sprintf("%d / %d жителів", w.Population(), w.Housing()), x+22*u, y+28*u, 13*u, true, pc, alignLeft)
	drawText(dst, fmt.Sprintf("вільних %d · працюють %d · носіїв %d", free, workers, carriers), x, y+48*u, 11.5*u, false, colInkSoft, alignLeft)
	drawItem(dst, sim.Food, float32(x+8*u), float32(y+77*u), float32(16*u))
	mins := w.FoodMinutes()
	fc := colInk
	msg := fmt.Sprintf("їжа %d · на %s", w.FoodAvailable(), fmtMinutes(mins))
	switch {
	case starving > 0:
		fc, msg = colBad, fmt.Sprintf("Голод/спрага: %d! Їжа й вода!", starving)
	case mins < 3:
		fc = colBad
	}
	drawText(dst, msg, x+22*u, y+68*u, 12.5*u, true, fc, alignLeft)
	drawItem(dst, sim.Water, float32(x+8*u), float32(y+99*u), float32(16*u))
	wm := w.WaterMinutes()
	wc := colInk
	if wm < 3 {
		wc = colBad
	}
	drawText(dst, fmt.Sprintf("вода %d · на %s", w.WaterAvailable(), fmtMinutes(wm)), x+22*u, y+90*u, 12.5*u, true, wc, alignLeft)
	// wellbeing
	wb := w.Wellbeing()
	sx, sy := float32(x+8*u), float32(y+121*u)
	fillCircle(dst, sx, sy, float32(7*u), mix(colBad, colOK, wb/5))
	fillCircle(dst, sx-float32(2.5*u), sy-float32(2*u), float32(1.2*u), colBody)
	fillCircle(dst, sx+float32(2.5*u), sy-float32(2*u), float32(1.2*u), colBody)
	drawText(dst, fmt.Sprintf("добробут %.1f/5 · робота ×%.2f", wb, w.Productivity()), x+22*u, y+112*u, 12.5*u, true, colInk, alignLeft)
	if (rect{x, y + 108*u, 222 * u, 24 * u}).has(g.mouseX, g.mouseY) {
		body := ""
		for n := sim.Need(0); n < sim.NeedCount; n++ {
			if n > 0 {
				body += "\n"
			}
			body += fmt.Sprintf("%s: %d%% жителів", sim.NeedNames[n], int(w.NeedShare(n)*100+0.5))
		}
		body += "\n\nРізноманітна їжа: овочі, ягоди, хліб по черзі.\nВідпочинок: вогнище біля дому. Краса: квітники, статуї."
		defer g.tooltip(dst, "Потреби жителів", body)
	}
}

func fmtMinutes(m float64) string {
	switch {
	case m > 600:
		return "довго"
	case m >= 1:
		return fmt.Sprintf("%.0f хв", m)
	}
	return "< 1 хв"
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
	body := wrap(def.Desc, 52) + "\nЦіна: " + g.costLine(k)
	switch {
	case !g.w.Unlocked(k):
		body += "\nЗакрито. Відкрий у дослідженні «" + sim.TechByID(sim.TechFor(k)).Name + "» (T)"
	case g.w.Free[k] > 0:
		body += fmt.Sprintf("\nБезкоштовно ще: %d", g.w.Free[k])
	case !g.w.CanAfford(k):
		body += "\nМатеріалів поки немає — майданчик чекатиме на них"
	}
	if n := sim.Builds[k].Workers; n > 0 {
		if k == sim.KStation {
			body += fmt.Sprintf("\nЗаймає %d жителів як носіїв", n)
		} else {
			body += fmt.Sprintf("\nПотрібно працівників: %d", n)
		}
	}
	if big := k.Big(); big != sim.KNone && g.w.Unlocked(big) {
		body += "\nTab — перемкнути на «" + sim.Builds[big].Name + "» (2×2)"
	} else if base := k.Base(); base != k {
		body += "\nTab — назад до «" + sim.Builds[base].Name + "» (1×1)"
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

	if b.Site {
		add(func(y float64) float64 {
			if len(b.Need) == 0 {
				drawText(dst, fmt.Sprintf("Будується: %d%%", int(b.BuildWork*100)), r.x+pad, y, 13*u, true, colInk, alignLeft)
				return 24 * u
			}
			drawText(dst, "Будмайданчик чекає матеріалів:", r.x+pad, y, 13*u, true, colInk, alignLeft)
			x := r.x + pad
			for _, n := range b.Need {
				drawItem(dst, n.Item, float32(x+8*u), float32(y+32*u), float32(15*u))
				label := fmt.Sprint(n.N)
				c := colInk
				if g.stock[n.Item] < n.N {
					c = colBad
				}
				drawText(dst, label, x+18*u, y+24*u, 12.5*u, true, c, alignLeft)
				x += 30*u + textWidth(label, 12.5*u, true)
			}
			return 50 * u
		})
	} else if n := sim.Builds[b.Kind].Workers; n > 0 {
		add(func(y float64) float64 {
			var msg string
			c := colInk
			switch {
			case b.Paused:
				msg, c = "На паузі — працівники вільні", colInkSoft
			case b.Kind == sim.KStation:
				msg = fmt.Sprintf("Носіїв: %d / %d · радіус %d клітинок", b.Staff(), n, int(sim.StationRadius))
			case b.Inside() >= n:
				msg = fmt.Sprintf("Працівників: %d / %d", b.Inside(), n)
			case b.Staff() > b.Inside():
				msg = fmt.Sprintf("Працівник іде на роботу (%d / %d)", b.Inside(), n)
			default:
				msg, c = fmt.Sprintf("Бракує працівників: %d / %d", b.Inside(), n), colBad
			}
			drawText(dst, msg, r.x+pad, y, 13*u, true, c, alignLeft)
			if c == colBad {
				drawText(dst, "Збудуй будинки або постав іншу\nбудівлю на паузу (P).", r.x+pad, y+20*u, 11.5*u, false, colInkSoft, alignLeft)
				return 58 * u
			}
			return 24 * u
		})
	}
	if h := sim.Builds[b.Kind].Housing; h > 0 && !b.Site {
		add(func(y float64) float64 {
			drawText(dst, fmt.Sprintf("Мешканців: %d / %d", g.w.Residents(b.ID), h), r.x+pad, y, 13*u, true, colInk, alignLeft)
			lines := g.w.HomeNeeds(b.ID)
			if lines == "" {
				return 24 * u
			}
			drawText(dst, lines, r.x+pad, y+22*u, 12*u, false, colInkSoft, alignLeft)
			return 30*u + float64(countLines(lines))*16*u
		})
	}
	if b.IsFlooded() {
		add(func(y float64) float64 {
			drawText(dst, "Затоплено! Будівля не працює,\nпоки вода не спаде.", r.x+pad, y, 12.5*u, true, colBad, alignLeft)
			return 42 * u
		})
	}
	if b.Dmg > 0 {
		add(func(y float64) float64 {
			max := g.w.MaxHP(b)
			drawText(dst, fmt.Sprintf("Міцність: %.0f / %.0f", max-b.Dmg, max), r.x+pad, y, 12.5*u, false, colBad, alignLeft)
			return 22 * u
		})
	}
	switch b.Kind.Base() {
	case sim.KTurret:
		add(func(y float64) float64 {
			msg := fmt.Sprintf("Набоїв: %d пострілів (макс. %d)", b.Ammo, sim.TurretAmmoCap*b.Size())
			c := colInk
			if b.Ammo == 0 {
				msg, c = "Без набоїв! Жителі принесуть «Набої» зі складу.", colBad
			}
			drawText(dst, msg, r.x+pad, y, 12.5*u, false, c, alignLeft)
			return 22 * u
		})
	case sim.KFlamer:
		add(func(y float64) float64 {
			msg := fmt.Sprintf("Паливо: %d%%", int(b.Ammo)*100/sim.FlamerFuelCap)
			c := colInk
			if b.Ammo == 0 {
				msg, c = "Без палива! Жителі принесуть вугілля зі складу.", colBad
			}
			drawText(dst, msg, r.x+pad, y, 12.5*u, false, c, alignLeft)
			return 22 * u
		})
	case sim.KPump:
		add(func(y float64) float64 {
			msg := fmt.Sprintf("Качає воду: 1 за %.0f с.\nЖитель п'є раз на хвилину.", sim.PumpTime)
			c := colInk
			if b.Site == false && !b.Working && b.Running() {
				msg, c = "Немає води поруч —\nрічка обміліла або висохла.", colBad
			}
			drawText(dst, msg, r.x+pad, y, 12.5*u, false, c, alignLeft)
			return 42 * u
		})
	case sim.KLumber, sim.KForester, sim.KGatherer:
		add(func(y float64) float64 {
			var msg string
			switch b.Kind {
			case sim.KLumber:
				msg = fmt.Sprintf("Рубає дорослі дерева в радіусі %d.\nДерево — %d деревини.", sim.FieldRadius, sim.WoodPerTree)
			case sim.KForester:
				msg = fmt.Sprintf("Садить дерева в радіусі %d.\nПоруч із лісорубом ліс не закінчиться.", sim.FieldRadius)
			default:
				msg = fmt.Sprintf("Збирає ягоди в радіусі %d.\nКущ — %d їжі, відростає за 1.5 хв.", sim.FieldRadius, sim.FoodPerBush)
			}
			drawText(dst, msg, r.x+pad, y, 12.5*u, false, colInk, alignLeft)
			return 42 * u
		})
	case sim.KLab:
		add(func(y float64) float64 {
			t := g.w.ActiveResearch()
			if t == nil {
				drawText(dst, "Нічого не досліджується.\nОбери дослідження у вікні (T).", r.x+pad, y, 12.5*u, false, colInkSoft, alignLeft)
				return 40 * u
			}
			d, wk := g.w.ResearchProgress()
			drawText(dst, "Зараз: "+t.Name, r.x+pad, y, 13*u, true, colInk, alignLeft)
			msg := fmt.Sprintf("Матеріали: %d%%", int(d*100))
			if d >= 1 {
				msg = fmt.Sprintf("Матеріали на місці · робота %d%%", int(wk*100))
			}
			drawText(dst, msg, r.x+pad, y+20*u, 12.5*u, false, colInk, alignLeft)
			x := r.x + pad
			for _, n := range g.w.ResearchNeed {
				drawItem(dst, n.Item, float32(x+8*u), float32(y+50*u), float32(15*u))
				label := fmt.Sprint(n.N)
				drawText(dst, label, x+18*u, y+42*u, 12.5*u, true, colInk, alignLeft)
				x += 30*u + textWidth(label, 12.5*u, true)
			}
			if len(g.w.ResearchNeed) > 0 {
				return 66 * u
			}
			return 44 * u
		})
	case sim.KFarm:
		add(func(y float64) float64 {
			if g.w.MoistAt(b.Pos) {
				drawText(dst, "Ґрунт вологий — поле росте.", r.x+pad, y, 12.5*u, true, colOK, alignLeft)
			} else {
				drawText(dst, "Сухий ґрунт — поле не росте.\nЧекай дощів або тримай воду греблею.", r.x+pad, y, 12.5*u, true, colBad, alignLeft)
				return 42 * u
			}
			return 24 * u
		})
		add(func(y float64) float64 {
			drawItem(dst, sim.Food, float32(r.x+pad+9*u), float32(y+9*u), float32(18*u))
			drawText(dst, fmt.Sprintf("Їжа · %.1f/хв · годує ~%d жителів", 60/sim.FarmTime, int(math.Floor(sim.MealInterval/sim.FarmTime))), r.x+pad+26*u, y, 13*u, false, colInk, alignLeft)
			return 28 * u
		})
	case sim.KMiner:
		add(func(y float64) float64 {
			if b.MinerItem == sim.ItemNone {
				drawText(dst, "Під буром немає покладу", r.x+pad, y, 13*u, false, colBad, alignLeft)
				return 24 * u
			}
			drawItem(dst, b.MinerItem, float32(r.x+pad+9*u), float32(y+9*u), float32(18*u))
			rate := 60 * sim.MinerRate(b) * g.w.MineMul()
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
				if !g.w.RecipeUnlocked(ri) {
					fillRoundRect(dst, float32(rr.x), float32(rr.y), float32(rr.w), float32(rr.h), float32(6*u), rgba(0xF4F0E8, 0xD0))
					lockGlyph(dst, float32(rr.x+rr.w-16*u), float32(rr.y+rr.h/2), float32(14*u), rgb(0x9AA0AC))
					if rr.has(g.mouseX, g.mouseY) {
						defer g.tooltip(dst, "Рецепт закрито", "Відкривається дослідженням «"+sim.TechByID(sim.Recipes[ri].Tech).Name+"»")
					}
					return 34 * u
				}
				g.addButton(rr, func() { g.w.SetRecipe(b.ID, ri) })
				return 34 * u
			})
		}
		if b.Kind.Base() == sim.KFurnace {
			add(func(y float64) float64 {
				msg := "Обирається сама за наявною рудою"
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
				fillRoundRect(dst, float32(r.x+pad), float32(py), float32((W-2*pad)*float64(b.Progress)), float32(6*u), float32(3*u), accentOf(b.Kind))
			}
			return 42 * u
		})
	case sim.KChest, sim.KCore:
		add(func(y float64) float64 {
			used, cap := g.w.StorageUsed(b), g.w.StorageCap(b)
			c := colInk
			if used >= cap {
				c = colBad
			}
			drawText(dst, fmt.Sprintf("Заповнено: %d / %d", used, cap), r.x+pad, y, 13*u, true, c, alignLeft)
			g.capacityBar(dst, r.x+pad, y+22*u, W-2*pad, used, cap)
			return 36 * u
		})
	}
	switch b.Kind {
	case sim.KCore:
		add(func(y float64) float64 {
			drawText(dst, "Сюди й на склади жителі зносять\nусе вироблене. Місце обмежене —\nставте склади біля виробництва.", r.x+pad, y, 12.5*u, false, colInk, alignLeft)
			return 58 * u
		})
	}

	if b.Kind != sim.KCore {
		add(func(y float64) float64 {
			bw := (W - 2*pad - 8*u) / 2
			b1 := rect{r.x + pad, y + 4*u, bw, 30 * u}
			b2 := rect{b1.x + bw + 8*u, y + 4*u, bw, 30 * u}
			if sim.Builds[b.Kind].Workers > 0 && !b.Site {
				label := "Пауза (P)"
				if b.Paused {
					label = "Відновити (P)"
				}
				g.smallButton(dst, b1, label, func() { g.w.SetPaused(b.ID, !b.Paused) })
			} else {
				g.smallButton(dst, b1, "Повернути (R)", func() { g.w.Rotate(b.ID) })
			}
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
	firstButton := len(g.buttons)
	type item struct {
		label  string
		action func()
		accent bool
	}
	peace := "Вороги: увімкнені"
	if g.w.Peaceful {
		peace = "Вороги: вимкнені"
	}
	items := []item{
		{"Продовжити", func() { g.menu = false }, false},
		{"Зберегти", func() { g.Save(); g.say("Збережено") }, false},
		{peace, func() {
			g.w.Peaceful = !g.w.Peaceful
			if g.w.Peaceful {
				g.say("Мирний режим: хвилі не приходять")
			} else {
				g.say("Хвилі знову приходитимуть")
			}
		}, false},
		{"Почати заново", func() { g.confirmNew = true }, false},
	}
	if g.updateReady {
		items = append(items, item{"Перезапустити з оновленням", g.restartForUpdate, true})
	}
	items = append(items, item{"Зберегти й вийти", func() { g.quit = true }, false})

	W := 340 * u
	H := (110 + float64(len(items))*52 + 40) * u
	r := rect{g.cam.W/2 - W/2, g.cam.H/2 - H/2, W, H}
	panel(dst, r, u)
	drawText(dst, "Потік", r.x+W/2, r.y+20*u, 26*u, true, colInk, alignCenter)
	drawText(dst, "пауза", r.x+W/2, r.y+56*u, 13*u, false, colInkSoft, alignCenter)
	bw := W - 60*u
	y := r.y + 90*u
	for _, it := range items {
		br := rect{r.x + 30*u, y, bw, 42 * u}
		if it.accent {
			fillRoundRect(dst, float32(br.x-3*u), float32(br.y-3*u), float32(br.w+6*u), float32(br.h+6*u), float32(10*u), colOK)
		}
		g.bigButton(dst, br, it.label, it.action)
		y += 52 * u
	}
	ago := time.Since(g.lastSave).Round(time.Second)
	drawText(dst, fmt.Sprintf("Автозбереження: %v тому · версія %s", ago, Version), r.x+W/2, r.y+H-34*u, 11.5*u, false, colInkSoft, alignCenter)

	if g.confirmNew {
		g.buttons = g.buttons[:firstButton] // only the dialog is clickable
		g.drawConfirmNew(dst)
	}
}

func (g *Game) drawConfirmNew(dst *ebiten.Image) {
	u := g.ui
	fillRect(dst, 0, 0, float32(g.cam.W), float32(g.cam.H), rgba(0x2E3440, 0x55))
	W, H := 460*u, 230*u
	r := rect{g.cam.W/2 - W/2, g.cam.H/2 - H/2, W, H}
	panel(dst, r, u)
	drawText(dst, "Почати заново?", r.x+24*u, r.y+20*u, 20*u, true, colInk, alignLeft)
	msg := "Твій світ, запаси й дослідження скинуться до початку.\n" +
		"Поточний світ збережеться в резервну копію (зберігаються 5 останніх).\n" +
		"Це стосується тільки тебе: у інших гравців власні збереження."
	drawText(dst, wrap(msg, 64), r.x+24*u, r.y+60*u, 13*u, false, colInk, alignLeft)
	bw := (W - 60*u) / 2
	by := r.y + H - 62*u
	g.bigButton(dst, rect{r.x + 24*u, by, bw, 42 * u}, "Скасувати", func() { g.confirmNew = false })
	yes := rect{r.x + 36*u + bw, by, bw, 42 * u}
	fillRoundRect(dst, float32(yes.x-3*u), float32(yes.y-3*u), float32(yes.w+6*u), float32(yes.h+6*u), float32(10*u), colBad)
	g.bigButton(dst, yes, "Так, почати заново", g.newGame)
}

const helpText = `Розбудуй поселення біля річки, переживи посухи й зроби жителів щасливими.

Жителі працюють у будівлях і переносять вантажі (кілька завжди вільні).
Бракує рук — постав будівлю на паузу (P). Їдять раз на 1.5 хв, п'ють раз на хвилину.

Вода й сезони
· Сезон дощів змінюється посухою (таймер угорі): річка міліє, ґрунт висихає.
· Насос біля води дає питну воду — запасайте її до посухи.
· Гребля тримає воду на рівень вище: загатіть річку. Насип воду не пропускає.
· Ферми ростуть лише на вологому ґрунті біля води.

Добробут (0–5) — їжа, вода, різноманітна їжа, відпочинок, краса
· Овочі (ферма), ягоди (збирач), хліб (пекарня) — їжте по черзі.
· Вогнище біля домів — відпочинок; квітники й статуї — краса.
· Вищий добробут — швидша робота (до ×1.3) і більше народжень.

Панель будівництва: вкладки-категорії, [ ] — перемкнути, 1–9 — будівля у вкладці.
Дослідження (T): матеріали зносять на станцію, потім дослідник працює.
Вороги вимкнені — їх можна ввімкнути в меню (Esc).

ЛКМ — поставити / обрати · ПКМ — знести · P — пауза будівлі · Tab — версія 2×2
Пробіл — пауза гри · , / . — швидкість · Ctrl + тягни — копіювати / знести область`

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
