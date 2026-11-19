// Command potik3d is the 3D prototype of the settlement game.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"

	"potik3d/internal/view"
	"potik3d/internal/world"
)

var font *text.GoTextFaceSource

func init() {
	var err error
	font, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
}

type game struct {
	w      *world.World
	scene  view.Scene
	ui     float64
	yawTo  float64
	dragX  float64
	dragY  float64
	rot    bool
	pan    bool
	shot   string
	frames int
	mock   string
}

func (g *game) Layout(ow, oh int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	g.ui = s
	g.scene.Cam.W, g.scene.Cam.H = float64(ow)*s, float64(oh)*s
	return int(g.scene.Cam.W), int(g.scene.Cam.H)
}

func (g *game) Update() error {
	c := &g.scene.Cam
	speed := 18 / c.Scale * 40 / 60
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		speed *= 2.5
	}
	move := func(fx, fy float64) {
		// move relative to the current view direction
		s, co := math.Sincos(c.Yaw)
		c.X += (fx*co + fy*s) * speed
		c.Y += (-fx*s + fy*co) * speed
	}
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
	mx, my := ebiten.CursorPosition()
	fx, fy := float64(mx), float64(my)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.rot, g.dragX, g.dragY = true, fx, fy
	}
	if g.rot {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			g.rot = false
		} else {
			g.yawTo += (fx - g.dragX) * 0.008
			c.Pitch = math.Max(0.45, math.Min(1.35, c.Pitch-(fy-g.dragY)*0.006))
			g.dragX, g.dragY = fx, fy
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		g.pan, g.dragX, g.dragY = true, fx, fy
	}
	if g.pan {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
			g.pan = false
		} else {
			s, co := math.Sincos(c.Yaw)
			dx, dy := -(fx-g.dragX)/c.Scale, -(fy-g.dragY)/c.Scale/math.Max(0.3, math.Sin(c.Pitch))
			c.X += dx*co + dy*s
			c.Y += -dx*s + dy*co
			g.dragX, g.dragY = fx, fy
		}
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		c.Scale = math.Max(12*g.ui, math.Min(140*g.ui, c.Scale*math.Pow(1.12, wy)))
	}
	c.Yaw += (g.yawTo - c.Yaw) * 0.18
	c.X = math.Max(0, math.Min(world.Size, c.X))
	c.Y = math.Max(0, math.Min(world.Size, c.Y))
	// keep the camera target on the ground
	tx, ty := int(c.X), int(c.Y)
	if world.In(tx, ty) {
		gz := float64(g.w.Height[world.Idx(tx, ty)]) * view.LevelH
		c.Z += (gz - c.Z) * 0.1
	}
	if x, y, ok := g.scene.Pick(g.w, fx, fy); ok {
		g.scene.Hover = [2]int{x, y}
	} else {
		g.scene.Hover = [2]int{-1, -1}
	}
	g.w.Update()
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	g.scene.Draw(screen, g.w)
	u := g.ui
	if g.mock != "" {
		drawMockUI(screen, u, g.mock)
		g.saveShot(screen)
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(14*u, 12*u)
	op.ColorScale.ScaleWithColor(color.RGBA{0x2E, 0x34, 0x40, 0xFF})
	op.LineSpacing = 20 * u
	hover := ""
	if h := g.scene.Hover; h[0] >= 0 {
		i := world.Idx(h[0], h[1])
		hover = fmt.Sprintf("\nклітинка %d,%d · рівень %d · вода %.2f", h[0], h[1], g.w.Height[i], g.w.Water[i])
	}
	msg := fmt.Sprintf("Потік 3D — прототип · %.0f FPS\nWASD — рух · Q/E — поворот · R/F або ПКМ — нахил і поворот · колесо — зум%s", ebiten.ActualFPS(), hover)
	text.Draw(screen, msg, &text.GoTextFace{Source: font, Size: 14 * u}, op)
	g.saveShot(screen)
}

func (g *game) saveShot(screen *ebiten.Image) {
	if g.shot != "" {
		g.frames--
		if g.frames == 0 {
			b := screen.Bounds()
			img := image.NewRGBA(b)
			screen.ReadPixels(img.Pix)
			f, err := os.Create(g.shot)
			if err == nil {
				png.Encode(f, img)
				f.Close()
			}
			os.Exit(0)
		}
	}
}

func main() {
	shot := flag.String("debug-shot", "", "")
	yaw := flag.Float64("debug-yaw", 0.5, "")
	pitch := flag.Float64("debug-pitch", 0.95, "")
	scale := flag.Float64("debug-scale", 34, "")
	scene := flag.String("debug-scene", "", "")
	flag.Parse()

	w := world.New(7)
	g := &game{w: w, shot: *shot, frames: 4, mock: *scene}
	cx, cy := int(w.RiverX(48))+8, 48
	if *scene != "" {
		buildTown(w, cx, cy)
		damRiver(w, cy+10)
		switch *scene {
		case "drought":
			drought(w, cy+10)
		}
		tx, ty := float64(cx)+1, float64(cy)+1
		if *scene == "levels" {
			lx, ly := buildLevels(w, int(w.RiverX(float64(cy+12)))+5, cy+12)
			tx, ty = float64(lx)-1, float64(ly)+2
		}
		c := &g.scene.Cam
		c.X, c.Y, c.Scale, c.Yaw, c.Pitch = tx, ty, *scale, *yaw, *pitch
		c.Z = float64(w.Height[world.Idx(int(tx), int(ty))]) * view.LevelH
		g.yawTo = c.Yaw
		ebiten.SetWindowSize(1280, 800)
		if err := ebiten.RunGame(g); err != nil {
			log.Fatal(err)
		}
		return
	}
	// a few sample buildings next to the river
	w.Place(world.KHall, cx, cy-1, 2)
	w.Place(world.KHouse, cx+3, cy-2, 1)
	w.Place(world.KHouse, cx+3, cy, 1)
	w.Place(world.KHouse, cx+4, cy+2, 1)
	w.Place(world.KLumber, cx+5, cy-4, 1)
	for x := cx - 8; x < cx; x++ {
		if w.Height[world.Idx(x, cy+2)] <= 1 && w.Water[world.Idx(x, cy+2)] > 0.2 {
			w.Place(world.KPump, x+1, cy+3, 1)
			break
		}
	}
	for x := 0; x < world.Size; x++ {
		if w.Height[world.Idx(x, cy+8)] == 0 {
			w.Place(world.KDam, x, cy+8, 1)
		}
	}
	c := &g.scene.Cam
	c.X, c.Y, c.Scale, c.Yaw, c.Pitch = float64(cx), float64(cy), *scale, *yaw, *pitch
	g.yawTo = c.Yaw
	ebiten.SetWindowTitle("Потік 3D — прототип")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
