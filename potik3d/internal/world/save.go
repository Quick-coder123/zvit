package world

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const SaveVersion = 1

type saveFile struct {
	Version     int
	Seed        int64
	Tick        uint64
	Water       []float32
	Props       []Prop
	Planted     map[int]uint64
	Picked      map[int]uint64
	Buildings   []*Building
	NextID      int32
	Settlers    []*Settler
	NextSettler int32
	Drought     bool
	SeasonLeft  float64
	Cycle       int
	Day         float64
	Deaths      int
}

// Encode writes the world.
func (w *World) Encode(out io.Writer) error {
	zw := gzip.NewWriter(out)
	sf := saveFile{Version: SaveVersion, Seed: w.Seed, Tick: w.Tick, Water: w.Water, Props: w.Props,
		Planted: w.Planted, Picked: w.Picked, NextID: w.NextID, Settlers: w.Settlers, NextSettler: w.NextSettler,
		Drought: w.Drought, SeasonLeft: w.SeasonLeft, Cycle: w.Cycle, Day: w.Day, Deaths: w.Deaths}
	for _, id := range w.Order() {
		sf.Buildings = append(sf.Buildings, w.Buildings[id])
	}
	if err := gob.NewEncoder(zw).Encode(&sf); err != nil {
		return err
	}
	return zw.Close()
}

// Decode reads a world.
func Decode(in io.Reader) (*World, error) {
	zr, err := gzip.NewReader(in)
	if err != nil {
		return nil, err
	}
	var sf saveFile
	if err := gob.NewDecoder(zr).Decode(&sf); err != nil {
		return nil, err
	}
	if sf.Version > SaveVersion {
		return nil, fmt.Errorf("сейв новішої версії (%d)", sf.Version)
	}
	w := &World{Seed: sf.Seed, Tick: sf.Tick, Planted: sf.Planted, Picked: sf.Picked,
		NextID: sf.NextID, Settlers: sf.Settlers, NextSettler: sf.NextSettler, Drought: sf.Drought,
		SeasonLeft: sf.SeasonLeft, Cycle: sf.Cycle, Day: sf.Day, Deaths: sf.Deaths, Peaceful: true,
		Buildings: map[int32]*Building{}}
	w.generate()
	if len(sf.Props) == len(w.Props) {
		w.Props = sf.Props
	}
	w.Water = sf.Water
	if len(w.Water) != Size*Size {
		w.Water = make([]float32, Size*Size)
	}
	for _, b := range sf.Buildings {
		if b.Kind > KNone && b.Kind < KindCount {
			w.Buildings[b.ID] = b
		}
	}
	w.initCaches()
	return w, nil
}

// SaveFile writes atomically (temp file + rename).
func (w *World) SaveFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := w.Encode(f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadFile reads a saved world.
func LoadFile(path string) (*World, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

// BackupSave renames an existing save to "<path>.<time>.bak", keeping the
// newest few backups.
func BackupSave(path string, keep int) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	if err := os.Rename(path, path+"."+time.Now().Format("2006-01-02_15-04-05")+".bak"); err != nil {
		return err
	}
	old, _ := filepath.Glob(path + ".*.bak")
	sort.Strings(old)
	for len(old) > keep {
		os.Remove(old[0])
		old = old[1:]
	}
	return nil
}
