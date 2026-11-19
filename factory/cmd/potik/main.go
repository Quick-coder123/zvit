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
	"potik/internal/launch"
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
	// Run from a per-user copy so potik.exe in the game folder can be
	// replaced with a new version at any time.
	if launch.Relaunch() {
		return
	}
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
	} else if *savePath != "" {
		sim.BackupSave(*savePath, 5)
	}
	if w == nil {
		w = sim.NewWorld(time.Now().UnixNano())
	}

	g := game.New(w, *savePath)
	if launch.Shadowed() {
		origin := launch.Origin()
		watcher := launch.NewWatcher(origin)
		g.SetUpdater(watcher.Changed, func() error { return launch.Restart(origin) })
	}
	if *shot != "" {
		w.Seed = 7
		if *shotScene != "start" {
			w.UnlockAll()
			w.Peaceful = true
			sim.BuildDemo(w)
		}
		switch *shotScene {
		case "combat":
			sim.BuildCombatDemo(w)
		case "research":
			w.Researched = map[string]bool{"furnace": true, "walls": true, "assembler": true, "chest": true}
			w.Research("station")
			w.Research("flamer")
			w.Research("belt2")
		}
		g.SetScreenshot(*shot, *shotFrames, func(g *game.Game) { debugScene(g, w, *shotScene) })
	}
	ebiten.SetWindowTitle("Потік " + game.Version)
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
		g.DebugCamera(1, 0, 30)
	case "river":
		g.DebugCamera(-9, 8, 34)
	case "nature":
		g.DebugCamera(2, 8, 44)
		if b := w.At(sim.Pos{X: 5, Y: 10}); b != nil {
			g.DebugSelect(b.ID)
		}
	case "town":
		g.DebugCamera(0, 4, 52)
		if b := w.At(sim.Pos{X: 0, Y: 4}); b != nil {
			g.DebugSelect(b.ID)
		}
	case "close":
		g.DebugCamera(6, -4, 64)
	case "copper":
		g.DebugCamera(-8, -3, 56)
		if b := w.At(sim.Pos{X: -6, Y: -2}); b != nil {
			g.DebugSelect(b.ID)
		}
	case "help":
		g.DebugHelp(true)
	case "combat":
		g.DebugCamera(17, -5, 40)
	case "start", "research":
		g.DebugCamera(0, 0, 40)
		if scene == "research" {
			g.DebugResearch(true)
		}
	case "menu":
		g.DebugMenu(true, false)
	case "confirm":
		g.DebugMenu(false, true)
	case "far":
		g.DebugCamera(0, 0, 9)
	}
}
