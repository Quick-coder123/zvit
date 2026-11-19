// Package app is the playable game: input, camera, building and the
// interface around the 3D view.
package app

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"potik3d/internal/view"
	"potik3d/internal/world"
)

// Version is set at build time.
var Version = "dev"

type rect struct{ x, y, w, h float64 }

func (r rect) has(x, y float64) bool { return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h }

type button struct {
	r  rect
	fn func()
}

// Game is the running game.
type Game struct {
	w        *world.World
	scene    view.Scene
	savePath string
	ui       float64

	yawTo        float64
	dragX, dragY float64
	rotating     bool
	panning      bool
	mx, my       float64
	hoverX       int
	hoverY       int

	tool     world.Kind
	cat      world.Category
	selected int32
	paused   bool
	speed    int

	menu, help, confirmNew bool
	toast                  string
	toastAt                time.Time
	lastSave               time.Time
	quit                   bool

	buttons []button
	panels  []rect

	checkUpdate func() bool
	restart     func() error
	updateReady bool

	shot       string
	debugHover bool
	shotFrames int
}

// New starts the game around a world.
func New(w *world.World, savePath string) *Game {
	g := &Game{w: w, savePath: savePath, speed: 1, lastSave: time.Now(), help: w.Tick == 0}
	g.centerOnHall()
	return g
}

func (g *Game) centerOnHall() {
	c := &g.scene.Cam
	c.Yaw, c.Pitch, c.Scale = 0.55, 0.85, 44
	g.yawTo = c.Yaw
	for _, id := range g.w.Order() {
		if b := g.w.Buildings[id]; b.Kind == world.KHall {
			c.X, c.Y = b.Center()
			c.Z = float64(g.w.Height[world.Idx(b.X, b.Y)]) * view.LevelH
		}
	}
}

// SetUpdater wires the "new version available" check.
func (g *Game) SetUpdater(check func() bool, restart func() error) {
	g.checkUpdate, g.restart = check, restart
}

// SetScreenshot makes the game save a PNG after a few frames and exit.
func (g *Game) SetScreenshot(path string, frames int) { g.shot, g.shotFrames = path, frames }

func (g *Game) say(f string, a ...any) {
	g.toast = fmt.Sprintf(f, a...)
	g.toastAt = time.Now()
}

func (g *Game) Save() {
	if g.savePath == "" {
		return
	}
	if err := g.w.SaveFile(g.savePath); err != nil {
		g.say("Не вдалося зберегти: %v", err)
		return
	}
	g.lastSave = time.Now()
}

func (g *Game) Layout(ow, oh int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	g.ui = s
	g.scene.Cam.W, g.scene.Cam.H = math.Ceil(float64(ow)*s), math.Ceil(float64(oh)*s)
	return int(g.scene.Cam.W), int(g.scene.Cam.H)
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() || g.quit {
		g.Save()
		return ebiten.Termination
	}
	g.input()
	if !g.menu && !g.paused {
		for i := 0; i < g.speed; i++ {
			g.w.Update()
		}
	}
	if n := len(g.w.Notices); n > 0 {
		g.say("%s", g.w.Notices[n-1])
		g.w.Notices = g.w.Notices[:0]
	}
	if time.Since(g.lastSave) > 30*time.Second {
		g.Save()
	}
	if !g.updateReady && g.checkUpdate != nil && g.checkUpdate() {
		g.updateReady = true
		g.say("Вийшла нова версія! Esc → «Перезапустити з оновленням»")
	}
	return nil
}

func (g *Game) hitButton() *button {
	for i := len(g.buttons) - 1; i >= 0; i-- {
		if g.buttons[i].r.has(g.mx, g.my) {
			return &g.buttons[i]
		}
	}
	return nil
}

func (g *Game) overUI() bool {
	if g.hitButton() != nil {
		return true
	}
	for _, r := range g.panels {
		if r.has(g.mx, g.my) {
			return true
		}
	}
	return false
}

// footprint is the top-left tile for placing kind k under the cursor.
func (g *Game) footprint(k world.Kind) (int, int) {
	s := world.Builds[k].Size
	return g.hoverX - (s-1)/2, g.hoverY - (s-1)/2
}

func (g *Game) input() {
	mx, my := ebiten.CursorPosition()
	g.mx, g.my = float64(mx), float64(my)
	if g.debugHover {
		g.mx, g.my = g.scene.Cam.W*0.42, g.scene.Cam.H*0.52
	}
	c := &g.scene.Cam
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch {
		case g.confirmNew:
			g.confirmNew = false
		case g.help:
			g.help = false
		case g.tool != world.KNone:
			g.tool = world.KNone
		case g.selected != 0:
			g.selected = 0
		default:
			g.menu = !g.menu
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.help = !g.help
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if b := g.hitButton(); b != nil {
			b.fn()
			return
		}
		if g.overUI() {
			return
		}
	}
	if g.menu || g.help {
		return
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.Save()
		g.say("Збережено")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) || inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.speed, g.paused = min(3, g.speed+1), false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyComma) || inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.speed, g.paused = max(1, g.speed-1), false
	}

	// camera
	speed := 18 / c.Scale * 40 / 60
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		speed *= 2.5
	}
	move := func(fx, fy float64) {
		s, co := math.Sincos(c.Yaw)
		c.X += (fx*co + fy*s) * speed
		c.Y += (-fx*s + fy*co) * speed
	}
	if !ctrl {
		if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			move(0, -1)
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			move(0, 1)
		}
		if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			move(-1, 0)
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			move(1, 0)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.yawTo -= math.Pi / 4
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.yawTo += math.Pi / 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		c.Pitch = math.Min(1.35, c.Pitch+0.02)
	}
	if ebiten.IsKeyPressed(ebiten.KeyF) {
		c.Pitch = math.Max(0.45, c.Pitch-0.02)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if g.tool != world.KNone {
			g.tool = world.KNone
		} else {
			g.rotating, g.dragX, g.dragY = true, g.mx, g.my
		}
	}
	if g.rotating {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			g.rotating = false
		} else {
			g.yawTo += (g.mx - g.dragX) * 0.008
			c.Pitch = math.Max(0.45, math.Min(1.35, c.Pitch-(g.my-g.dragY)*0.006))
			g.dragX, g.dragY = g.mx, g.my
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		g.panning, g.dragX, g.dragY = true, g.mx, g.my
	}
	if g.panning {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			g.panning = false
		} else {
			s, co := math.Sincos(c.Yaw)
			dx, dy := -(g.mx-g.dragX)/c.Scale, -(g.my-g.dragY)/c.Scale/math.Max(0.3, math.Sin(c.Pitch))
			c.X += dx*co + dy*s
			c.Y += -dx*s + dy*co
			g.dragX, g.dragY = g.mx, g.my
		}
	}
	if _, wy := ebiten.Wheel(); wy != 0 && !g.overUI() {
		c.Scale = math.Max(14*g.ui, math.Min(140*g.ui, c.Scale*math.Pow(1.12, wy)))
	}
	c.Yaw += (g.yawTo - c.Yaw) * 0.18
	c.X = math.Max(0, math.Min(world.Size, c.X))
	c.Y = math.Max(0, math.Min(world.Size, c.Y))
	if tx, ty := int(c.X), int(c.Y); world.In(tx, ty) {
		c.Z += (float64(g.w.Height[world.Idx(tx, ty)])*view.LevelH - c.Z) * 0.1
	}

	// hovered tile
	g.hoverX, g.hoverY = -1, -1
	if !g.overUI() {
		if x, y, ok := g.scene.Pick(g.w, g.mx, g.my); ok {
			g.hoverX, g.hoverY = x, y
		}
	}

	// build hotkeys: 1-9 inside the current category, [ ] switch category
	kinds := buildKinds(g.cat)
	for i, k := range kinds {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			g.selectTool(k)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.cat = (g.cat + 1) % world.CatCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		g.cat = (g.cat + world.CatCount - 1) % world.CatCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) && g.selected != 0 {
		if b := g.w.Buildings[g.selected]; b != nil {
			g.w.SetPaused(b.ID, !b.Paused)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) && g.selected != 0 {
		g.demolish(g.selected)
	}

	if g.hoverX < 0 {
		return
	}
	if g.tool != world.KNone {
		x, y := g.footprint(g.tool)
		place := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		// dams and fields can be painted by dragging
		if !place && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && (g.tool == world.KDam || g.tool == world.KFarm || g.tool == world.KBridge) {
			place = g.w.CheckPlace(g.tool, x, y) == nil
		}
		if place {
			if _, err := g.w.Place(g.tool, x, y); err != nil {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					g.say("%v", err)
				}
			}
		}
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		b := g.w.BuildingAt(g.hoverX, g.hoverY)
		if b == nil {
			b = g.w.SupportAt(g.hoverX, g.hoverY)
		}
		if b != nil {
			g.selected = b.ID
		} else {
			g.selected = 0
		}
	}
}

func (g *Game) selectTool(k world.Kind) {
	if g.tool == k {
		g.tool = world.KNone
		return
	}
	g.tool, g.selected = k, 0
}

func (g *Game) demolish(id int32) {
	if err := g.w.Remove(id); err != nil {
		g.say("%v", err)
		return
	}
	g.selected = 0
}

func (g *Game) newGame() {
	g.confirmNew, g.menu = false, false
	if g.savePath != "" {
		world.BackupSave(g.savePath, 5)
	}
	g.w = world.New(time.Now().UnixNano() % 100000)
	g.tool, g.selected, g.help = world.KNone, 0, true
	g.centerOnHall()
	g.Save()
	g.say("Нова долина! Попередній світ збережено в резервну копію.")
}

// buildKinds lists the buildings of a menu category.
func buildKinds(c world.Category) []world.Kind {
	var out []world.Kind
	for k := world.Kind(1); k < world.KindCount; k++ {
		d := &world.Builds[k]
		if !d.Hidden && d.Cat == c {
			out = append(out, k)
		}
	}
	return out
}

func (g *Game) Draw(screen *ebiten.Image) {
	s := &g.scene
	s.Hover = [2]int{g.hoverX, g.hoverY}
	s.Ghost, s.Radius, s.Selected = nil, [3]float64{}, g.selected
	if g.tool != world.KNone && g.hoverX >= 0 {
		x, y := g.footprint(g.tool)
		s.Ghost = g.ghost(x, y)
		if r := world.Builds[g.tool].Radius; r > 0 {
			sz := float64(world.Builds[g.tool].Size) / 2
			s.Radius = [3]float64{float64(x) + sz, float64(y) + sz, float64(r) + 0.5}
		}
	} else if b := g.w.Buildings[g.selected]; b != nil && b.Def().Radius > 0 {
		cx, cy := b.Center()
		s.Radius = [3]float64{cx, cy, float64(b.Def().Radius) + 0.5}
	}
	s.Draw(screen, g.w)
	g.drawUI(screen)
	if g.shot != "" {
		g.shotFrames--
		if g.shotFrames == 0 {
			g.saveShot(screen)
		}
	}
}

// Debug sets up a view for automated screenshots.
func (g *Game) Debug(help bool, tool world.Kind, selectHall bool, yaw, scale float64) {
	g.help = help
	g.centerOnHall()
	g.scene.Cam.Yaw, g.yawTo, g.scene.Cam.Scale = yaw, yaw, scale
	g.speed = 1
	if tool != world.KNone {
		g.tool, g.cat = tool, world.Builds[tool].Cat
		g.debugHover = true
	}
	if selectHall {
		for _, id := range g.w.Order() {
			if b := g.w.Buildings[id]; b.Kind == world.KSawmill || (g.selected == 0 && b.Kind == world.KHall) {
				g.selected = id
			}
		}
	}
}

// ghost previews the current tool at a spot with its automatic choices.
func (g *Game) ghost(x, y int) *view.Ghost {
	p, err := g.w.PlanPlacement(g.tool, x, y)
	gh := &view.Ghost{Kind: g.tool, X: x, Y: y, OK: err == nil, Dir: p.Dir, Levels: p.Levels, Base: p.Base}
	if err != nil && world.In(x, y) {
		i := world.Idx(x, y)
		gh.Base, gh.Levels = g.w.Surface(i), 1
		if g.tool.IsSupport() {
			gh.Base = int(g.w.Height[i])
			if sp := g.w.Support(i); sp != nil {
				gh.Levels = sp.Levels
			}
		}
	}
	return gh
}

var dirNames = [4]string{"схід", "захід", "південь", "північ"}

// placementHint describes what a successful placement will do.
func placementHint(k world.Kind, p world.Placement) string {
	switch k {
	case world.KStairs:
		return fmt.Sprintf("Сходи: вгору на %d рівн. (%s) — ЛКМ поставити", p.Levels, dirNames[p.Dir&3])
	case world.KBridge:
		return fmt.Sprintf("Міст на висоті %d — ЛКМ поставити", p.Levels)
	case world.KPlatform:
		if p.Upgrade {
			return fmt.Sprintf("Надбудова: рівень %d/%d — ЛКМ", p.Levels, world.MaxStack)
		}
		return fmt.Sprintf("Помост: рівень 1/%d — ЛКМ поставити", world.MaxStack)
	}
	if p.Base > 0 {
		return fmt.Sprintf("Стоятиме на рівні %d — ЛКМ поставити", p.Base)
	}
	return "ЛКМ — поставити"
}

// DebugPitch sets the camera pitch for screenshots.
func (g *Game) DebugPitch(p float64) { g.scene.Cam.Pitch = p }
