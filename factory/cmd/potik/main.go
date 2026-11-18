// Command potik is a calm offline factory-building game.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"potik/internal/game"
	"potik/internal/sim"
)

func defaultSavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "Potik", "solo.sav")
}

func main() {
	savePath := flag.String("save", defaultSavePath(), "шлях до файлу збереження")
	fresh := flag.Bool("new", false, "почати новий світ (старий буде перейменовано)")
	shot := flag.String("debug-shot", "", "")
	shotFrames := flag.Int("debug-frames", 240, "")
	shotScene := flag.String("debug-scene", "overview", "")
	flag.Parse()
	if *shot != "" {
		*savePath = ""
		*fresh = true
	}

	var w *sim.World
	if !*fresh {
		var err error
		w, err = sim.LoadFile(*savePath)
		if err != nil && !sim.IsNotExist(err) {
			log.Printf("не вдалося прочитати сейв: %v — починаю новий світ", err)
			os.Rename(*savePath, *savePath+".broken")
			w = nil
		}
	} else if _, err := os.Stat(*savePath); *savePath != "" && err == nil {
		os.Rename(*savePath, *savePath+"."+time.Now().Format("20060102-150405")+".bak")
	}
	if w == nil {
		w = sim.NewWorld(time.Now().UnixNano())
	}

	g := game.New(w, *savePath)
	if *shot != "" {
		w.Seed = 7
		if *shotScene != "start" {
			w.UnlockAll()
			sim.BuildDemo(w)
			sim.BuildLogisticsDemo(w)
			w.Peaceful = true
		}
		switch *shotScene {
		case "combat":
			sim.BuildCombatDemo(w)
		case "research":
			w.Researched = map[string]bool{"furnace": true, "walls": true, "assembler": true, "chest": true}
		}
		g.SetScreenshot(*shot, *shotFrames, func(g *game.Game) { debugScene(g, w, *shotScene) })
	}
	ebiten.SetWindowTitle("Потік")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetTPS(sim.TPS)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func debugScene(g *game.Game, w *sim.World, scene string) {
	g.DebugHelp(false)
	switch scene {
	case "overview":
		g.DebugCamera(0, -2, 26)
	case "close":
		g.DebugCamera(6, -4, 64)
	case "copper":
		g.DebugCamera(-8, -3, 56)
		if b := w.At(sim.Pos{X: -5, Y: -3}); b != nil {
			g.DebugSelect(b.ID)
		}
	case "help":
		g.DebugHelp(true)
	case "logistics":
		o := sim.LogisticsDemoOrigin
		g.DebugCamera(float64(o.X)+9, float64(o.Y)+0.5, 46)
		if b := w.At(o.Add(3, 0)); b != nil {
			g.DebugSelect(b.ID)
		}
	case "paste":
		o := sim.LogisticsDemoOrigin
		g.DebugCamera(float64(o.X)+8, float64(o.Y)+3, 34)
		g.DebugCopy(o.X+5, o.Y-2, o.X+10, o.Y+2)
		g.DebugMouse(640+34*1, 400+34*4)
	case "tunnel":
		o := sim.LogisticsDemoOrigin
		g.DebugCamera(float64(o.X)+9, float64(o.Y)+0.5, 46)
		g.DebugTool(sim.KTunnel, sim.East)
		g.DebugMouse(640+46*7.5, 400+46*3)
	case "combat":
		g.DebugCamera(17, -5, 40)
	case "start", "research":
		g.DebugCamera(0, 0, 40)
		if scene == "research" {
			g.DebugResearch(true)
		}
	case "far":
		g.DebugCamera(0, 0, 9)
	}
}
