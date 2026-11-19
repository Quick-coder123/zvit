// Command potik3d is a calm offline settlement game in low-poly 3D.
package main

import (
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"potik3d/internal/app"
	"potik3d/internal/launch"
	"potik3d/internal/world"
)

func defaultSavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "Potik3D", "world.sav")
}

func main() {
	// Run from a per-user copy so the exe in the game folder can be replaced
	// with a new version at any time.
	if launch.Relaunch() {
		return
	}
	savePath := flag.String("save", defaultSavePath(), "шлях до файлу збереження")
	fresh := flag.Bool("new", false, "почати нову долину (старий світ піде в резервну копію)")
	shot := flag.String("debug-shot", "", "")
	frames := flag.Int("debug-frames", 30, "")
	demo := flag.Int("debug-demo", 0, "секунд симуляції демо-містечка")
	help := flag.Bool("debug-help", false, "")
	tool := flag.Int("debug-tool", 0, "")
	sel := flag.Bool("debug-select", false, "")
	yaw := flag.Float64("debug-yaw", 0.55, "")
	scale := flag.Float64("debug-scale", 44, "")
	flag.Parse()
	if *shot != "" {
		*savePath, *fresh = "", true
	}

	var w *world.World
	if !*fresh {
		var err error
		w, err = world.LoadFile(*savePath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Printf("не вдалося прочитати сейв: %v — починаю новий світ", err)
			os.Rename(*savePath, *savePath+".broken")
			w = nil
		}
	} else if *savePath != "" {
		world.BackupSave(*savePath, 5)
	}
	if w == nil {
		seed := time.Now().UnixNano() % 100000
		if *shot != "" {
			seed = 7
		}
		w = world.New(seed)
	}

	g := app.New(w, *savePath)
	if launch.Shadowed() {
		origin := launch.Origin()
		watcher := launch.NewWatcher(origin)
		g.SetUpdater(watcher.Changed, func() error { return launch.Restart(origin) })
	}
	if *shot != "" {
		if *demo > 0 {
			world.BuildDemo(w, *demo)
		}
		g.Debug(*help, world.Kind(*tool), *sel, *yaw, *scale)
		g.SetScreenshot(*shot, *frames)
	}
	ebiten.SetWindowTitle("Потік 3D " + app.Version)
	ebiten.SetWindowSize(1366, 820)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetTPS(world.TPS)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
