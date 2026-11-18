// Package game is the Ebitengine front-end: input, camera, rendering and UI.
package game

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"potik/internal/sim"
)

const autosaveEvery = 30 * time.Second

// Toolbar order; index+1 is the hotkey.
var toolbar = []sim.Kind{sim.KBelt, sim.KMiner, sim.KFurnace, sim.KAssembler, sim.KChest}

type Game struct {
	w        *sim.World
	savePath string
	cam      Camera
	ui       float64 // UI scale (device scale factor)

	tool     sim.Kind
	dir      sim.Dir
	hover    sim.Pos
	mouseX   float64
	mouseY   float64
	overUI   bool
	dragging bool
	dragLast sim.Pos
	erasing  bool
	panning  bool
	panFromX float64
	panFromY float64

	selected  int32
	menu      bool
	help      bool
	toast     string
	toastTime time.Time
	lastSave  time.Time
	quit      bool

	buttons    []button // rebuilt every frame by the UI pass
	panels     []rect
	infoFor    int32
	infoHeight float64

	shot      *shotState
	fakeMouse bool
}

// New creates the game around a world; savePath is used for autosave.
func New(w *sim.World, savePath string) *Game {
	g := &Game{w: w, savePath: savePath, lastSave: time.Now(), help: w.Tick == 0}
	g.cam.Scale = 40
	return g
}

func (g *Game) say(format string, a ...any) {
	g.toast = fmt.Sprintf(format, a...)
	g.toastTime = time.Now()
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
	w, h := g.LayoutF(float64(ow), float64(oh))
	return int(w), int(h)
}

func (g *Game) LayoutF(ow, oh float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	g.ui = s
	g.cam.W, g.cam.H = math.Ceil(ow*s), math.Ceil(oh*s)
	return g.cam.W, g.cam.H
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() || g.quit {
		g.Save()
		return ebiten.Termination
	}
	g.handleInput()
	if !g.menu {
		g.w.Update(nil)
	}
	if time.Since(g.lastSave) > autosaveEvery {
		g.Save()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawWorld(screen)
	g.drawGhost(screen)
	g.drawUI(screen)
	if g.shot != nil {
		g.shotDraw(screen)
	}
}

// footprint returns the top-left tile for placing kind k under the mouse.
func (g *Game) footprint(k sim.Kind) sim.Pos {
	wx, wy := g.cam.ToWorld(g.mouseX, g.mouseY)
	n := float64(sim.Builds[k].Size)
	return sim.Pos{X: int(math.Floor(wx - n/2 + 0.5)), Y: int(math.Floor(wy - n/2 + 0.5))}
}

func (g *Game) handleInput() {
	if !g.fakeMouse {
		mx, my := ebiten.CursorPosition()
		g.mouseX, g.mouseY = float64(mx), float64(my)
	}
	wx, wy := g.cam.ToWorld(g.mouseX, g.mouseY)
	g.hover = sim.Pos{X: int(math.Floor(wx)), Y: int(math.Floor(wy))}
	g.overUI = g.hitButton(g.mouseX, g.mouseY) != nil || g.overPanel(g.mouseX, g.mouseY)

	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch {
		case g.help:
			g.help = false
		case g.tool != sim.KNone:
			g.tool = sim.KNone
		case g.selected != 0:
			g.selected = 0
		default:
			g.menu = !g.menu
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.help = !g.help
	}
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.Save()
		g.say("Збережено")
	}

	// UI clicks take priority over the world
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if b := g.hitButton(g.mouseX, g.mouseY); b != nil {
			b.action()
			return
		}
		if g.overPanel(g.mouseX, g.mouseY) {
			return
		}
	}
	if g.menu {
		return
	}

	// camera
	pan := 14.0 / g.cam.Scale * 40 / sim.TPS
	if shift {
		pan *= 2.5
	}
	if !ctrl {
		if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			g.cam.X -= pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			g.cam.X += pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			g.cam.Y -= pan
		}
		if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			g.cam.Y += pan
		}
	}
	if _, wy := ebiten.Wheel(); wy != 0 && !g.overUI {
		before := g.cam.Scale
		g.cam.Scale = math.Max(8*g.ui, math.Min(96*g.ui, g.cam.Scale*math.Pow(1.15, wy)))
		// zoom around the cursor
		k := 1/before - 1/g.cam.Scale
		g.cam.X += (g.mouseX - g.cam.W/2) * k
		g.cam.Y += (g.mouseY - g.cam.H/2) * k
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		g.panning, g.panFromX, g.panFromY = true, g.mouseX, g.mouseY
	}
	if g.panning {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			g.panning = false
		} else {
			g.cam.X -= (g.mouseX - g.panFromX) / g.cam.Scale
			g.cam.Y -= (g.mouseY - g.panFromY) / g.cam.Scale
			g.panFromX, g.panFromY = g.mouseX, g.mouseY
		}
	}

	// hotkeys
	for i, k := range toolbar {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			g.selectTool(k)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.Key0) {
		g.tool = sim.KNone
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if g.tool != sim.KNone {
			if shift {
				g.dir = g.dir.CCW()
			} else {
				g.dir = g.dir.CW()
			}
		} else if b := g.w.At(g.hover); b != nil {
			g.w.Rotate(b.ID)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		if b := g.w.At(g.hover); b != nil && sim.Builds[b.Kind].Buildable {
			g.tool, g.dir = b.Kind, b.Dir
		} else {
			g.tool = sim.KNone
		}
	}

	if g.overUI && !g.dragging && !g.erasing {
		return
	}

	// building
	if g.tool != sim.KNone {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.dragging = true
			p := g.footprint(g.tool)
			g.dragLast = p
			g.tryPlace(p, g.dir, true)
		} else if g.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			p := g.footprint(g.tool)
			if p != g.dragLast {
				if g.tool == sim.KBelt {
					g.dragBelts(p)
				} else {
					g.tryPlace(p, g.dir, false)
				}
				g.dragLast = p
			}
		}
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if b := g.w.At(g.hover); b != nil && b.Kind != sim.KBelt {
			g.selected = b.ID
		} else {
			g.selected = 0
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.dragging = false
	}

	// removing
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.erasing = true
		if g.tool != sim.KNone && g.w.At(g.hover) == nil {
			g.tool = sim.KNone
			g.erasing = false
		}
	}
	if g.erasing {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			g.erasing = false
		} else if b := g.w.At(g.hover); b != nil {
			if err := g.w.Remove(b.ID); err != nil {
				g.say("%v", err)
				g.erasing = false
			} else if b.ID == g.selected {
				g.selected = 0
			}
		}
	}
}

func (g *Game) selectTool(k sim.Kind) {
	if g.tool == k {
		g.tool = sim.KNone
		return
	}
	g.tool = k
	g.selected = 0
}

func (g *Game) tryPlace(p sim.Pos, d sim.Dir, loud bool) bool {
	_, err := g.w.Place(g.tool, p, d, 0)
	if err != nil {
		if loud || err == sim.ErrNoFunds {
			g.say("%s: %v", sim.Builds[g.tool].Name, err)
		}
		return false
	}
	return true
}

// dragBelts lays belts from dragLast to p, turning the line as the mouse moves.
func (g *Game) dragBelts(p sim.Pos) {
	cur := g.dragLast
	for steps := 0; cur != p && steps < 200; steps++ {
		var next sim.Pos
		ax, ay := p.X-cur.X, p.Y-cur.Y
		if abs(ax) >= abs(ay) {
			next = sim.Pos{X: cur.X + sign(ax), Y: cur.Y}
		} else {
			next = sim.Pos{X: cur.X, Y: cur.Y + sign(ay)}
		}
		d, _ := sim.DirBetween(cur, next)
		g.dir = d
		if b := g.w.At(cur); b != nil && b.Kind == sim.KBelt {
			g.w.Place(sim.KBelt, cur, d, 0)
		}
		g.tryPlace(next, d, false)
		cur = next
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func sign(a int) int {
	switch {
	case a > 0:
		return 1
	case a < 0:
		return -1
	}
	return 0
}

// drawGhost previews the building under the cursor.
func (g *Game) drawGhost(dst *ebiten.Image) {
	if g.menu || g.overUI {
		return
	}
	s := float32(g.cam.Scale)
	if g.tool == sim.KNone {
		if b := g.w.At(g.hover); b != nil {
			x, y := g.cam.ToScreen(float64(b.Pos.X), float64(b.Pos.Y))
			w := s * float32(b.Size())
			strokeRoundRect(dst, x+1, y+1, w-2, w-2, s*0.18, max32(1.5, s*0.05), rgba(0x2E3440, 0x55))
		}
		return
	}
	p := g.footprint(g.tool)
	ghost := &sim.Building{Kind: g.tool, Pos: p, Dir: g.dir, Recipe: -1}
	if g.tool == sim.KMiner {
		ghost.MinerItem = g.w.OreAt(p)
	}
	ok := g.w.CheckPlace(g.tool, p, g.dir) == nil
	g.drawBuilding(dst, ghost, 0.6)
	x, y := g.cam.ToScreen(float64(p.X), float64(p.Y))
	w := s * float32(sim.Builds[g.tool].Size)
	c := rgba(0x4FA3A5, 0xCC)
	if !ok {
		c = rgba(0xE07A5F, 0xCC)
	}
	strokeRoundRect(dst, x+1, y+1, w-2, w-2, s*0.18, max32(2, s*0.06), c)
}
