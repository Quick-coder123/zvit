package game

import (
	"image"

	"potik/internal/sim"

	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// Debug screenshot support: when ShotPath is set the game renders
// ShotFrames frames, writes a PNG and exits. Used for visual checks.
type shotState struct {
	path   string
	frames int
	n      int
	setup  func(g *Game)
}

// SetScreenshot enables the debug screenshot mode.
func (g *Game) SetScreenshot(path string, frames int, setup func(g *Game)) {
	g.shot = &shotState{path: path, frames: frames, setup: setup}
}

// Debug helpers for scripted screenshots.
func (g *Game) DebugCamera(x, y, scale float64) { g.cam.X, g.cam.Y, g.cam.Scale = x, y, scale*g.ui }
func (g *Game) DebugSelect(id int32)            { g.selected = id }
func (g *Game) DebugMouse(x, y float64)         { g.fakeMouse = true; g.mouseX, g.mouseY = x*g.ui, y*g.ui }
func (g *Game) DebugHelp(on bool)               { g.help = on }

func (g *Game) shotDraw(screen *ebiten.Image) {
	s := g.shot
	s.n++
	if s.n == 2 && s.setup != nil {
		s.setup(g)
	}
	if s.n < s.frames {
		return
	}
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	f, err := os.Create(s.path)
	if err == nil {
		png.Encode(f, img.SubImage(b))
		f.Close()
	}
	g.quit = true
	g.savePath = ""
}

// DebugCopy copies a tile rectangle into the clipboard and enters paste mode.
func (g *Game) DebugCopy(x0, y0, x1, y1 int) {
	g.selStart = sim.Pos{X: x0, Y: y0}
	g.hover = sim.Pos{X: x1, Y: y1}
	g.finishSelection()
}

// DebugTool selects a build tool.
func (g *Game) DebugTool(k sim.Kind, d sim.Dir) { g.tool, g.dir = k, d }

// DebugResearch opens the research panel.
func (g *Game) DebugResearch(on bool) { g.research = on }

// DebugMenu opens the pause menu, optionally with the update button and the
// start-over dialog.
func (g *Game) DebugMenu(update, confirm bool) {
	g.menu, g.updateReady, g.confirmNew = true, update, confirm
}
