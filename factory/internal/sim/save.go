package sim

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const SaveVersion = 1

type saveFile struct {
	Version   int
	Seed      int64
	Tick      uint64
	NextID    int32
	Buildings []*Building
	Stock     []int
}

// Encode writes the world in the save format.
func (w *World) Encode(out io.Writer) error {
	zw := gzip.NewWriter(out)
	sf := saveFile{Version: SaveVersion, Seed: w.Seed, Tick: w.Tick, NextID: w.NextID, Stock: w.Stock[:]}
	for _, id := range w.Order() {
		sf.Buildings = append(sf.Buildings, w.Buildings[id])
	}
	if err := gob.NewEncoder(zw).Encode(&sf); err != nil {
		return err
	}
	return zw.Close()
}

// Decode reads a world from the save format.
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
	w := &World{Seed: sf.Seed, Tick: sf.Tick, NextID: sf.NextID, Buildings: map[int32]*Building{}}
	copy(w.Stock[:], sf.Stock)
	for _, b := range sf.Buildings {
		if b.Kind == KNone || b.Kind >= KindCount {
			continue
		}
		w.Buildings[b.ID] = b
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

// LoadFile reads a world; returns os.ErrNotExist wrapped if missing.
func LoadFile(path string) (*World, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

// IsNotExist reports a missing save.
func IsNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }
