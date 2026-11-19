package game

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"potik/internal/sim"
)

// toolbar groups; the hotkey for each slot is listed in toolKeys.
var toolGroups = [][]sim.Kind{
	{sim.KHouse, sim.KFarm, sim.KChest, sim.KStation, sim.KLab},
	{sim.KLumber, sim.KForester, sim.KGatherer, sim.KSawmill},
	{sim.KPump, sim.KDam, sim.KLevee},
	{sim.KMiner, sim.KFurnace, sim.KAssembler},
	{sim.KWall, sim.KFlamer, sim.KTurret},
}

var toolKeys = []ebiten.Key{
	ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5,
	ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	ebiten.Key0, ebiten.KeyH, ebiten.KeyJ,
	ebiten.KeyM, ebiten.KeyF, ebiten.KeyG,
	ebiten.KeyZ, ebiten.KeyX, ebiten.KeyC,
}

var toolKeyLabels = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "H", "J", "M", "F", "G", "Z", "X", "C"}

func toolList() []sim.Kind {
	var out []sim.Kind
	for _, g := range toolGroups {
		out = append(out, g...)
	}
	return out
}

func lockGlyph(dst *ebiten.Image, cx, cy, s float32, c color.RGBA) {
	strokeCircle(dst, cx, cy-s*0.22, s*0.22, max32(1, s*0.09), c)
	fillRoundRect(dst, cx-s*0.36, cy-s*0.12, s*0.72, s*0.56, s*0.1, c)
}

func (g *Game) drawToolbar(dst *ebiten.Image) {
	u := g.ui
	slot, gap, groupGap := 46*u, 4*u, 12*u
	n := 0
	for _, gr := range toolGroups {
		n += len(gr)
	}
	tw := float64(n)*slot + float64(n-len(toolGroups))*gap + float64(len(toolGroups)-1)*groupGap + 20*u
	r := rect{g.cam.W/2 - tw/2, g.cam.H - slot - 60*u, tw, slot + 20*u}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	x := r.x + 10*u
	idx := 0
	var hover sim.Kind
	for gi, gr := range toolGroups {
		if gi > 0 {
			x += groupGap - gap
			line(dst, float32(x-groupGap/2+gap/2), float32(r.y+16*u), float32(x-groupGap/2+gap/2), float32(r.y+r.h-16*u), float32(1*u), colPanelLine)
		}
		for _, k := range gr {
			k := k
			sr := rect{x, r.y + 10*u, slot, slot}
			unlocked := g.w.Unlocked(k)
			active := g.tool != sim.KNone && g.tool.Base() == k
			shown := k
			if active {
				shown = g.tool
			}
			bg := rgb(0xF7F4EE)
			if active {
				bg = mix(colBody, colOK, 0.25)
			}
			fillRoundRect(dst, float32(sr.x), float32(sr.y), float32(sr.w), float32(sr.h), float32(8*u), bg)
			if active {
				strokeRoundRect(dst, float32(sr.x), float32(sr.y), float32(sr.w), float32(sr.h), float32(8*u), float32(2*u), colOK)
			}
			if big := k.Big(); big != sim.KNone && g.w.Unlocked(big) {
				label := "2×2"
				lc := colInkSoft
				if shown == big {
					lc = colOK
				}
				drawText(dst, label, sr.x+sr.w-5*u, sr.y+2*u, 10*u, true, lc, alignRight)
			}
			if unlocked {
				g.iconBuilding(dst, shown, sr.x+7*u, sr.y+8*u, slot-14*u, sim.East)
			} else {
				g.iconBuilding(dst, k, sr.x+7*u, sr.y+8*u, slot-14*u, sim.East)
				fillRoundRect(dst, float32(sr.x), float32(sr.y), float32(sr.w), float32(sr.h), float32(8*u), rgba(0xF7F4EE, 0xC8))
				lockGlyph(dst, float32(sr.x+sr.w/2), float32(sr.y+sr.h/2+2*u), float32(18*u), rgb(0x9AA0AC))
			}
			nc := colInkSoft
			if unlocked && !g.w.CanAfford(shown) {
				nc = colBad
			}
			drawText(dst, toolKeyLabels[idx], sr.x+5*u, sr.y+2*u, 11*u, true, nc, alignLeft)
			if f := g.w.Free[k]; f > 0 && unlocked {
				label := fmt.Sprintf("×%d", f)
				bw := textWidth(label, 10.5*u, true) + 10*u
				fillRoundRect(dst, float32(sr.x+sr.w-bw-2*u), float32(sr.y+sr.h-18*u), float32(bw), float32(16*u), float32(8*u), colOK)
				drawText(dst, label, sr.x+sr.w-bw/2-2*u, sr.y+sr.h-17*u, 10.5*u, true, colBody, alignCenter)
			}
			g.addButton(sr, func() { g.selectTool(k) })
			if sr.has(g.mouseX, g.mouseY) {
				hover = shown
			}
			x += slot + gap
			idx++
		}
	}
	// research button at the right of the toolbar
	rb := rect{r.x + r.w + 10*u, r.y, 112 * u, r.h}
	avail := len(g.w.ResearchQueue)
	panel(dst, rb, u)
	g.panels = append(g.panels, rb)
	bg := rgb(0xF7F4EE)
	if g.research || rb.has(g.mouseX, g.mouseY) {
		bg = mix(colBody, colOK, 0.25)
	}
	inner := rect{rb.x + 8*u, rb.y + 8*u, rb.w - 16*u, rb.h - 16*u}
	fillRoundRect(dst, float32(inner.x), float32(inner.y), float32(inner.w), float32(inner.h), float32(8*u), bg)
	flask(dst, float32(inner.x+inner.w/2), float32(inner.y+22*u), float32(26*u))
	drawText(dst, "Дослідження", inner.x+inner.w/2, inner.y+inner.h-24*u, 11.5*u, true, colInk, alignCenter)
	drawText(dst, "T", inner.x+5*u, inner.y+2*u, 11*u, true, colInkSoft, alignLeft)
	if avail > 0 {
		fillCircle(dst, float32(inner.x+inner.w-10*u), float32(inner.y+10*u), float32(9*u), colOK)
		drawText(dst, fmt.Sprint(avail), inner.x+inner.w-10*u, inner.y+2*u, 11*u, true, colBody, alignCenter)
	}
	g.addButton(rb, func() { g.research = !g.research })

	if hover != sim.KNone {
		g.buildTooltip(dst, hover)
	}
}

func (g *Game) affordable(cost []sim.Stack) bool {
	for _, c := range cost {
		if g.stock[c.Item] < c.N {
			return false
		}
	}
	return true
}

func flask(dst *ebiten.Image, cx, cy, s float32) {
	c := colInk
	path.MoveTo(cx-s*0.12, cy-s*0.45)
	path.LineTo(cx-s*0.12, cy-s*0.1)
	path.LineTo(cx-s*0.4, cy+s*0.38)
	path.LineTo(cx+s*0.4, cy+s*0.38)
	path.LineTo(cx+s*0.12, cy-s*0.1)
	path.LineTo(cx+s*0.12, cy-s*0.45)
	strokePath(dst, max32(1.2, s*0.08), c)
	path.MoveTo(cx-s*0.27, cy+s*0.16)
	path.LineTo(cx-s*0.36, cy+s*0.33)
	path.LineTo(cx+s*0.36, cy+s*0.33)
	path.LineTo(cx+s*0.27, cy+s*0.16)
	path.Close()
	fillPath(dst, colOK)
}

// --- research panel ---------------------------------------------------------

func (g *Game) drawResearch(dst *ebiten.Image) {
	u := g.ui
	cols := 5
	cw, ch := 214*u, 104*u
	if g.cam.W < float64(cols)*cw+80*u {
		cols = 4
	}
	gap := 10 * u
	W := float64(cols)*cw + float64(cols-1)*gap + 40*u
	rows := (len(sim.Techs) + cols - 1) / cols
	H := float64(rows)*ch + float64(rows-1)*gap + 96*u
	if H > g.cam.H-130*u {
		// shrink cards on small screens
		ch = (g.cam.H - 130*u - 96*u - float64(rows-1)*gap) / float64(rows)
		H = g.cam.H - 130*u
	}
	r := rect{g.cam.W/2 - W/2, (g.cam.H-110*u)/2 - H/2, W, H}
	if r.y < 8*u {
		r.y = 8 * u
	}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	drawText(dst, "Дослідження", r.x+20*u, r.y+16*u, 20*u, true, colInk, alignLeft)
	sub := "Обери дослідження — жителі знесуть матеріали на дослідницьку станцію, а дослідник його завершить."
	subC := colInkSoft
	if !g.w.HasLab() {
		sub, subC = "Немає дослідницької станції! Збудуй її (5) — без неї дослідження не просуваються.", colBad
	}
	drawText(dst, sub, r.x+20*u, r.y+46*u, 12.5*u, true, subC, alignLeft)
	g.smallButton(dst, rect{r.x + W - 110*u, r.y + 14*u, 90 * u, 28 * u}, "Закрити", func() { g.research = false })

	const (
		stActive = iota
		stQueued
		stAvail
		stLocked
		stDone
	)
	order := make([]int, len(sim.Techs))
	for i := range order {
		order[i] = i
	}
	rank := func(i int) int {
		t := &sim.Techs[i]
		switch q := g.w.Queued(t.ID); {
		case g.w.Researched[t.ID]:
			return stDone
		case q == 0:
			return stActive
		case q > 0:
			return stQueued
		}
		for _, r := range t.Req {
			if !g.w.Researched[r] && g.w.Queued(r) < 0 {
				return stLocked
			}
		}
		return stAvail
	}
	sort.SliceStable(order, func(a, b int) bool {
		ra, rb := rank(order[a]), rank(order[b])
		if ra != rb {
			return ra < rb
		}
		if ra == stQueued {
			return g.w.Queued(sim.Techs[order[a]].ID) < g.w.Queued(sim.Techs[order[b]].ID)
		}
		return false
	})

	var tip *sim.Tech
	for n, i := range order {
		t := &sim.Techs[i]
		cr := rect{r.x + 20*u + float64(n%cols)*(cw+gap), r.y + 76*u + float64(n/cols)*(ch+gap), cw, ch}
		state := rank(i)
		bg := rgb(0xF7F4EE)
		border := colPanelLine
		switch state {
		case stActive:
			bg = mix(colBody, colOK, 0.22)
			border = colOK
		case stQueued:
			bg = mix(colBody, colOK, 0.1)
			border = colOK
		case stDone:
			bg = rgb(0xEDE8DE)
		}
		hover := cr.has(g.mouseX, g.mouseY)
		if hover && state <= stAvail {
			bg = mix(bg, colOK, 0.15)
		}
		fillRoundRect(dst, float32(cr.x), float32(cr.y), float32(cr.w), float32(cr.h), float32(8*u), bg)
		strokeRoundRect(dst, float32(cr.x), float32(cr.y), float32(cr.w), float32(cr.h), float32(8*u), float32(1.2*u), border)
		nameC := colInk
		if state >= stLocked {
			nameC = colInkSoft
		}
		drawText(dst, t.Name, cr.x+12*u, cr.y+8*u, 14*u, true, nameC, alignLeft)
		switch state {
		case stDone:
			drawText(dst, "готово", cr.x+cr.w-12*u, cr.y+9*u, 12*u, true, colOK, alignRight)
		case stActive:
			drawText(dst, "зараз", cr.x+cr.w-12*u, cr.y+9*u, 12*u, true, colOK, alignRight)
		case stQueued:
			drawText(dst, fmt.Sprintf("черга %d", g.w.Queued(t.ID)), cr.x+cr.w-12*u, cr.y+9*u, 12*u, true, colOK, alignRight)
		}
		if ch >= 90*u {
			desc := wrap(t.Desc, int(cw/(7*u)))
			if state == stActive || state == stLocked {
				desc = firstLines(desc, 2)
			}
			drawText(dst, desc, cr.x+12*u, cr.y+30*u, 11.5*u, false, colInkSoft, alignLeft)
		}
		// cost / requirement line
		y := cr.y + cr.h - 26*u
		switch state {
		case stLocked:
			names := ""
			for j, req := range t.Req {
				if j > 0 {
					names += ", "
				}
				names += sim.TechByID(req).Name
			}
			req := wrap("Потрібно: "+names, int(cw/(6.2*u)))
			drawText(dst, req, cr.x+12*u, y+3*u-float64(countLines(req)-1)*15*u, 11*u, false, colInkSoft, alignLeft)
		case stDone:
		case stActive:
			d, wk := g.w.ResearchProgress()
			label := fmt.Sprintf("матеріали %d%%", int(d*100))
			f := d * 0.5
			if d >= 1 {
				label = fmt.Sprintf("дослідник працює: %d%%", int(wk*100))
				f = 0.5 + wk*0.5
			}
			drawText(dst, label, cr.x+12*u, y, 11.5*u, true, colInk, alignLeft)
			bw := cr.w - 24*u
			fillRoundRect(dst, float32(cr.x+12*u), float32(y+17*u), float32(bw), float32(5*u), float32(2.5*u), rgb(0xE6E0D4))
			fillRoundRect(dst, float32(cr.x+12*u), float32(y+17*u), float32(bw*f), float32(5*u), float32(2.5*u), colOK)
		default:
			x := cr.x + 12*u
			for _, c := range t.Cost {
				have := g.stock[c.Item] >= c.N
				drawItem(dst, c.Item, float32(x+8*u), float32(y+10*u), float32(15*u))
				tc := colInk
				if !have {
					tc = colBad
				}
				label := fmt.Sprint(c.N)
				drawText(dst, label, x+18*u, y+2*u, 12*u, true, tc, alignLeft)
				x += 26*u + textWidth(label, 12*u, true)
			}
		}
		id := t.ID
		switch state {
		case stAvail:
			g.addButton(cr, func() {
				if err := g.w.Research(id); err != nil {
					g.say("%v", err)
				} else if g.w.Queued(id) == 0 {
					g.say("Досліджуємо «%s»: жителі зносять матеріали", sim.TechByID(id).Name)
				} else {
					g.say("«%s» додано в чергу", sim.TechByID(id).Name)
				}
			})
		case stActive, stQueued:
			g.addButton(cr, func() {
				g.w.CancelResearch(id)
				g.say("Дослідження «%s» скасовано", sim.TechByID(id).Name)
			})
			if hover {
				tip = &sim.Tech{Name: "Клацни, щоб скасувати", Desc: "Принесені матеріали повернуться в ратушу. Залежні дослідження в черзі теж скасуються."}
			}
		}
		if hover && ch < 90*u {
			tip = t
		}
	}
	if tip != nil {
		g.tooltip(dst, tip.Name, wrap(tip.Desc, 50))
	}
}

// --- wave HUD -----------------------------------------------------------------

func fmtTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	s := int(sec)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func (g *Game) drawWaveHUD(dst *ebiten.Image) {
	u := g.ui
	w := g.w
	var title string
	tc := colInk
	switch {
	case w.Drought:
		title = fmt.Sprintf("ПОСУХА · ще %s", fmtTime(w.SeasonLeft))
		tc = rgb(0xC77B2E)
	case w.Peaceful:
		title = fmt.Sprintf("Сезон дощів · посуха через %s", fmtTime(w.SeasonLeft))
		if w.SeasonLeft < 60 {
			tc = rgb(0xC77B2E)
		}
	case len(w.Enemies) > 0:
		title = fmt.Sprintf("Атака! Ворогів: %d", len(w.Enemies))
		tc = colBad
	default:
		title = fmt.Sprintf("Хвиля %d через %s", w.Wave+1, fmtTime(w.NextWave))
		if w.NextWave < 30 {
			tc = colBad
		}
	}
	W := 300 * u
	H := 64 * u
	r := rect{g.cam.W/2 - W/2, 14 * u, W, H}
	if g.selected != 0 && g.cam.W < 1200*u {
		r.x = 280 * u
	}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	drawText(dst, title, r.x+r.w/2, r.y+8*u, 15*u, true, tc, alignCenter)
	// core health
	c := w.Core()
	if c != nil {
		hp := 1 - float64(c.Dmg/w.MaxHP(c))
		bx, by, bw := r.x+16*u, r.y+36*u, r.w-32*u
		fillRoundRect(dst, float32(bx), float32(by), float32(bw), float32(8*u), float32(4*u), rgb(0xE6E0D4))
		hc := mix(colBad, colOK, hp)
		fillRoundRect(dst, float32(bx), float32(by), float32(bw*hp), float32(8*u), float32(4*u), hc)
		drawText(dst, "ратуша", bx, by+10*u, 10.5*u, false, colInkSoft, alignLeft)
		right := fmt.Sprintf("цикл %d", w.Cycle+1)
		if !w.Peaceful {
			right = fmt.Sprintf("хвиля %d через %s", w.Wave+1, fmtTime(w.NextWave))
			if len(w.Enemies) > 0 {
				right = fmt.Sprintf("атака: %d ворогів", len(w.Enemies))
			}
		}
		drawText(dst, right, bx+bw, by+10*u, 10.5*u, false, colInkSoft, alignRight)
	}
	g.drawSpeed(dst, r)
	next := r.y + r.h + 8*u
	// research progress
	if t := w.ActiveResearch(); t != nil {
		rr := rect{r.x, next, r.w, 30 * u}
		panel(dst, rr, u)
		g.panels = append(g.panels, rr)
		d, wk := w.ResearchProgress()
		label := fmt.Sprintf("%s · матеріали %d%%", t.Name, int(d*100))
		f := d * 0.5
		if d >= 1 {
			label, f = fmt.Sprintf("%s · досліджується %d%%", t.Name, int(wk*100)), 0.5+wk*0.5
		}
		c := colInk
		if !w.HasLab() {
			label, c = "Немає дослідницької станції (5)!", colBad
		}
		flask(dst, float32(rr.x+16*u), float32(rr.y+15*u), float32(16*u))
		drawText(dst, label, rr.x+30*u, rr.y+6*u, 12*u, true, c, alignLeft)
		fillRoundRect(dst, float32(rr.x+30*u), float32(rr.y+23*u), float32(rr.w-46*u), float32(3*u), float32(1.5*u), rgb(0xE6E0D4))
		fillRoundRect(dst, float32(rr.x+30*u), float32(rr.y+23*u), float32((rr.w-46*u)*f), float32(3*u), float32(1.5*u), colOK)
		g.addButton(rr, func() { g.research = true })
		next += 38 * u
	}
	// rebuild button
	if n := len(w.Ruins); n > 0 {
		br := rect{r.x, next, r.w, 32 * u}
		g.smallButton(dst, br, fmt.Sprintf("Відбудувати зруйноване: %d (B)", n), g.rebuild)
		next += 40 * u
	}
	g.hudBottom = next
	if g.paused && !g.menu {
		label := "ПАУЗА"
		tw := textWidth(label, 18*u, true) + 40*u
		pr := rect{g.cam.W/2 - tw/2, next + 4*u, tw, 34 * u}
		fillRoundRect(dst, float32(pr.x), float32(pr.y), float32(pr.w), float32(pr.h), float32(17*u), rgba(0x2E3440, 0xE6))
		drawText(dst, label, g.cam.W/2, pr.y+6*u, 18*u, true, colBG, alignCenter)
		g.hudBottom = pr.y + pr.h + 8*u
	}
}

// drawSpeed shows pause and speed buttons next to the wave panel.
func (g *Game) drawSpeed(dst *ebiten.Image, wave rect) {
	u := g.ui
	bs := 30 * u
	r := rect{wave.x + wave.w + 8*u, wave.y, bs*4 + 3*4*u + 16*u, wave.h}
	panel(dst, r, u)
	g.panels = append(g.panels, r)
	for i := 0; i < 4; i++ {
		i := i
		br := rect{r.x + 8*u + float64(i)*(bs+4*u), r.y + (r.h-bs)/2 - 6*u, bs, bs}
		active := (i == 0 && g.paused) || (i > 0 && !g.paused && g.speed == i)
		bg := rgb(0xF0EBE1)
		if active {
			bg = mix(colBody, colOK, 0.45)
		} else if br.has(g.mouseX, g.mouseY) {
			bg = rgb(0xE5DED0)
		}
		fillRoundRect(dst, float32(br.x), float32(br.y), float32(br.w), float32(br.h), float32(6*u), bg)
		cx, cy := float32(br.x+br.w/2), float32(br.y+br.h/2)
		ic := colInk
		if i == 0 {
			fillRect(dst, cx-6*float32(u), cy-7*float32(u), 4*float32(u), 14*float32(u), ic)
			fillRect(dst, cx+2*float32(u), cy-7*float32(u), 4*float32(u), 14*float32(u), ic)
			g.addButton(br, g.togglePause)
		} else {
			sz := 10 * float32(u)
			start := cx - float32(i-1)*sz*0.45
			for k := 0; k < i; k++ {
				triangle(dst, start+float32(k)*sz*0.9, cy, 0, sz, ic)
			}
			g.addButton(br, func() { g.setSpeed(i) })
		}
	}
	drawText(dst, "пробіл · , .", r.x+r.w/2, r.y+r.h-17*u, 10*u, false, colInkSoft, alignCenter)
}

func (g *Game) rebuild() {
	built, left := g.w.RebuildRuins()
	switch {
	case built == 0 && left > 0:
		g.say("Ці будівлі ще закриті в дослідженнях")
	case left > 0:
		g.say("Відбудовано %d, ще %d чекають на ресурси", built, left)
	default:
		g.say("Будмайданчиків на місці руїн: %d", built)
	}
}

// --- combat rendering ------------------------------------------------------------

func (g *Game) drawRuins(dst *ebiten.Image) {
	s := float32(g.cam.Scale)
	for _, sp := range g.w.Ruins {
		x, y := g.cam.ToScreen(float64(sp.Pos.X), float64(sp.Pos.Y))
		w := s * float32(sim.Builds[sp.Kind].Size)
		if x+w < 0 || y+w < 0 || float64(x) > g.cam.W || float64(y) > g.cam.H {
			continue
		}
		fillRoundRect(dst, x+s*0.08, y+s*0.08, w-s*0.16, w-s*0.16, s*0.16, rgba(0xE07A5F, 0x26))
		strokeRoundRect(dst, x+s*0.08, y+s*0.08, w-s*0.16, w-s*0.16, s*0.16, max32(1, s*0.04), rgba(0xE07A5F, 0x99))
		c := rgba(0xE07A5F, 0x99)
		m := w * 0.18
		line(dst, x+w/2-m, y+w/2-m, x+w/2+m, y+w/2+m, max32(1, s*0.05), c)
		line(dst, x+w/2+m, y+w/2-m, x+w/2-m, y+w/2+m, max32(1, s*0.05), c)
	}
}

func (g *Game) drawHPBar(dst *ebiten.Image, b *sim.Building) {
	if b.Dmg <= 0 {
		return
	}
	s := float32(g.cam.Scale)
	x, y := g.cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
	w := s * float32(b.Size())
	hp := 1 - b.Dmg/g.w.MaxHP(b)
	bw := w * 0.8
	bx, by := x+w*0.1, y-s*0.12
	fillRoundRect(dst, bx, by, bw, max32(3, s*0.1), s*0.05, rgba(0x2E3440, 0x88))
	fillRoundRect(dst, bx, by, bw*hp, max32(3, s*0.1), s*0.05, mix(colBad, colOK, float64(hp)))
}

func (g *Game) drawEnemies(dst *ebiten.Image) {
	s := float32(g.cam.Scale)
	t := float64(g.w.Tick) / sim.TPS
	for _, e := range g.w.Enemies {
		def := &sim.EnemyDefs[e.Kind]
		x, y := g.cam.ToScreen(e.X, e.Y)
		r := float32(def.Radius) * s
		if float64(x) < -40 || float64(y) < -40 || float64(x) > g.cam.W+40 || float64(y) > g.cam.H+40 {
			continue
		}
		c := rgb(def.Color)
		dark := mix(c, colInk, 0.5)
		wob := float32(math.Sin(t*8+float64(e.Phase))) * 0.08
		// legs
		for k := 0; k < 3; k++ {
			a := float64(k)*math.Pi/3 + math.Pi/6 + float64(wob)*3
			for _, sgn := range []float64{1, -1} {
				lx := x + float32(math.Cos(a*sgn+math.Pi/2*(1-sgn)))*r*1.35
				ly := y + float32(math.Sin(a*sgn+math.Pi/2*(1-sgn)))*r*1.35
				line(dst, x, y, lx, ly, max32(1, r*0.18), dark)
			}
		}
		fillCircle(dst, x, y+r*0.12, r*(1.02+wob), rgba(0x2E3440, 0x30))
		fillCircle(dst, x, y, r*(1+wob), c)
		strokeCircle(dst, x, y, r*(1+wob), max32(1, r*0.14), dark)
		if e.Kind == sim.Brute {
			strokeCircle(dst, x, y, r*0.6, max32(1, r*0.16), dark)
		}
		// eyes toward the target
		ang := 0.0
		if tb := g.w.Buildings[e.Target]; tb != nil {
			bx := float64(tb.Pos.X) + float64(tb.Size())/2
			by := float64(tb.Pos.Y) + float64(tb.Size())/2
			ang = math.Atan2(by-e.Y, bx-e.X)
		}
		for _, off := range []float64{-0.45, 0.45} {
			ex := x + float32(math.Cos(ang+off))*r*0.5
			ey := y + float32(math.Sin(ang+off))*r*0.5
			fillCircle(dst, ex, ey, max32(1, r*0.2), colBody)
			fillCircle(dst, ex+float32(math.Cos(ang))*r*0.07, ey+float32(math.Sin(ang))*r*0.07, max32(0.8, r*0.1), colInk)
		}
		if e.Burn > 0 {
			strokeCircle(dst, x, y, r*1.25, max32(1, r*0.2), rgba(0xF2A65A, 0xAA))
		}
		if e.HP < def.HP {
			hp := e.HP / def.HP
			bw := max32(r*2.2, 10)
			fillRect(dst, x-bw/2, y-r*1.6, bw, max32(2, s*0.06), rgba(0x2E3440, 0x88))
			fillRect(dst, x-bw/2, y-r*1.6, bw*hp, max32(2, s*0.06), colBad)
		}
	}
}

func (g *Game) drawShots(dst *ebiten.Image) {
	s := float32(g.cam.Scale)
	for _, sh := range g.w.Shots {
		x0, y0 := g.cam.ToScreen(float64(sh.X0), float64(sh.Y0))
		x1, y1 := g.cam.ToScreen(float64(sh.X1), float64(sh.Y1))
		if sh.Flame {
			a := uint8(200 * sh.TTL / 0.12)
			line(dst, x0, y0, x1, y1, max32(2, s*0.28), rgba(0xF2A65A, a/2))
			line(dst, x0, y0, x1, y1, max32(1, s*0.12), rgba(0xF7D774, a))
			fillCircle(dst, x1, y1, s*0.5, rgba(0xF2A65A, a/3))
			continue
		}
		line(dst, x0, y0, x1, y1, max32(1, s*0.05), rgba(0xC9A227, 0xDD))
		fillCircle(dst, x1, y1, max32(1.5, s*0.08), rgba(0xF7D774, 0xEE))
	}
}

// drawThreats marks enemies and the coming wave when they are off-screen.
func (g *Game) drawThreats(dst *ebiten.Image) {
	u := g.ui
	W, H := g.cam.W, g.cam.H
	margin := 24 * u
	edge := func(wx, wy float64, c color.RGBA, size float32) {
		sx, sy := g.cam.ToScreen(wx, wy)
		if float64(sx) >= 0 && float64(sy) >= 0 && float64(sx) <= W && float64(sy) <= H {
			return
		}
		cx, cy := W/2, H/2
		dx, dy := float64(sx)-cx, float64(sy)-cy
		k := math.Min((W/2-margin)/math.Abs(dx+1e-9), (H/2-margin)/math.Abs(dy+1e-9))
		px, py := cx+dx*k, cy+dy*k
		triangle(dst, float32(px), float32(py), float32(math.Atan2(dy, dx)), size, c)
	}
	n := 0
	for _, e := range g.w.Enemies {
		if n > 60 {
			break
		}
		edge(e.X, e.Y, rgba(0xE07A5F, 0xCC), float32(14*u))
		n++
	}
	if !g.w.Peaceful && len(g.w.Enemies) == 0 && g.w.NextWave < 60 {
		wx, wy := g.w.WaveSpawnPoint()
		pulse := 0.5 + 0.5*math.Sin(float64(g.w.Tick)/sim.TPS*5)
		sx, sy := g.cam.ToScreen(wx, wy)
		if float64(sx) >= 0 && float64(sy) >= 0 && float64(sx) <= W && float64(sy) <= H {
			s := float32(g.cam.Scale)
			strokeCircle(dst, sx, sy, s*(3+float32(pulse)), max32(2, s*0.1), rgba(0xE07A5F, 0xAA))
			drawText(dst, "звідси прийде хвиля", float64(sx), float64(sy)+float64(s)*4.2, 12*u, true, colBad, alignCenter)
		} else {
			edge(wx, wy, rgba(0xE07A5F, uint8(150+100*pulse)), float32(26*u))
		}
	}
}

// weapon parts drawn on top of turrets, flamers and the core.
func (g *Game) drawWeapon(dst *ebiten.Image, b *sim.Building, cx, cy, w float32, fade func(color.RGBA) color.RGBA) {
	s := float32(g.cam.Scale)
	a := float64(b.Aim)
	ca, sa := float32(math.Cos(a)), float32(math.Sin(a))
	switch b.Kind.Base() {
	case sim.KTurret:
		fillCircle(dst, cx, cy, w*0.3, fade(rgb(0x4C566A)))
		line(dst, cx, cy, cx+ca*w*0.48, cy+sa*w*0.48, max32(2, s*0.2), fade(rgb(0x3B4252)))
		line(dst, cx+ca*w*0.2, cy+sa*w*0.2, cx+ca*w*0.48, cy+sa*w*0.48, max32(1, s*0.08), fade(rgb(0x6B7385)))
		fillCircle(dst, cx, cy, w*0.16, fade(colAccent[sim.KTurret]))
	case sim.KFlamer:
		fillCircle(dst, cx, cy, w*0.3, fade(rgb(0x4C566A)))
		line(dst, cx, cy, cx+ca*w*0.42, cy+sa*w*0.42, max32(2, s*0.18), fade(rgb(0x3B4252)))
		fillCircle(dst, cx+ca*w*0.44, cy+sa*w*0.44, max32(1.5, s*0.09), fade(colAccent[sim.KFlamer]))
		fillCircle(dst, cx, cy, w*0.14, fade(colAccent[sim.KFlamer]))
	case sim.KCore:
		line(dst, cx+ca*w*0.18, cy+sa*w*0.18, cx+ca*w*0.42, cy+sa*w*0.42, max32(2, s*0.14), fade(rgb(0xD8DEE9)))
	}
	if base := b.Kind.Base(); (base == sim.KTurret || base == sim.KFlamer) && s >= 12 {
		cap := float32(sim.TurretAmmoCap * b.Size())
		if b.Kind == sim.KFlamer {
			cap = sim.FlamerFuelCap
		}
		f := float32(b.Ammo) / cap
		bx, by, bw := cx-w*0.38, cy+w*0.36, w*0.76
		fillRoundRect(dst, bx, by, bw, max32(2, s*0.08), s*0.04, fade(rgb(0xE6E0D4)))
		c := rgb(0xC9A227)
		if b.Ammo == 0 {
			c = colBad
			fillRoundRect(dst, bx, by, bw, max32(2, s*0.08), s*0.04, fade(rgba(0xE07A5F, 0x88)))
		}
		fillRoundRect(dst, bx, by, bw*f, max32(2, s*0.08), s*0.04, fade(c))
	}
}

// firstLines keeps at most n lines of text, marking the cut with an ellipsis.
func firstLines(s string, n int) string {
	lines := 0
	for i, c := range s {
		if c == '\n' {
			lines++
			if lines == n {
				return s[:i] + "…"
			}
		}
	}
	return s
}
