package sim

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const SaveVersion = 4

type saveFile struct {
	Version   int
	Seed      int64
	Tick      uint64
	NextID    int32
	Buildings []*Building
	Stock     []int

	// v2
	Free       []int
	Researched map[string]bool
	Enemies    []*Enemy
	Wave       int
	NextWave   float64
	WaveAngle  float64
	Peaceful   bool
	Ruins      []Spec
	Kills      int

	// v4
	Settlers    []*Settler
	NextSettler int32
	WanderTimer float32
	Deaths      int
}

// Encode writes the world in the save format.
func (w *World) Encode(out io.Writer) error {
	zw := gzip.NewWriter(out)
	sf := saveFile{Version: SaveVersion, Seed: w.Seed, Tick: w.Tick, NextID: w.NextID, Stock: w.Stock[:],
		Free: w.Free[:], Researched: w.Researched, Enemies: w.Enemies, Wave: w.Wave, NextWave: w.NextWave,
		WaveAngle: w.WaveAngle, Peaceful: w.Peaceful, Ruins: w.Ruins, Kills: w.Kills,
		Settlers: w.Settlers, NextSettler: w.NextSettler, WanderTimer: w.WanderTimer, Deaths: w.Deaths}
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
	w := &World{Seed: sf.Seed, Tick: sf.Tick, NextID: sf.NextID, Buildings: map[int32]*Building{},
		Researched: sf.Researched, Enemies: sf.Enemies, Wave: sf.Wave, NextWave: sf.NextWave,
		WaveAngle: sf.WaveAngle, Peaceful: sf.Peaceful, Ruins: sf.Ruins, Kills: sf.Kills,
		Settlers: sf.Settlers, NextSettler: sf.NextSettler, WanderTimer: sf.WanderTimer, Deaths: sf.Deaths}
	copy(w.Stock[:], sf.Stock)
	copy(w.Free[:], sf.Free)
	if w.Researched == nil {
		w.Researched = map[string]bool{}
	}
	for _, b := range sf.Buildings {
		if b.Kind == KNone || b.Kind >= KindCount {
			continue
		}
		w.Buildings[b.ID] = b
	}
	if sf.Version < 2 {
		// Worlds from before research: keep what was built usable.
		for _, b := range w.Buildings {
			if id := TechFor(b.Kind); id != "" {
				w.Researched[id] = true
				for _, r := range TechByID(id).Req {
					w.Researched[r] = true
				}
			}
		}
		w.NextWave = FirstWaveDelay
		w.WaveAngle = w.rollWaveAngle()
	}
	if sf.Version < 3 {
		// Machines used to be 2×2. Keep old layouts working by turning them
		// into the 2×2 upgraded variants (and unlocking those).
		for _, b := range w.Buildings {
			if big := b.Kind.Big(); big != KNone {
				b.Kind = big
				for _, t := range Techs {
					for _, k := range t.Unlocks {
						if k == big {
							w.Researched[t.ID] = true
						}
					}
				}
			}
		}
		for i := range w.Ruins {
			if big := w.Ruins[i].Kind.Big(); big != KNone {
				w.Ruins[i].Kind = big
			}
		}
	}
	if sf.Version < 4 {
		// Conveyor logistics were replaced by settlers: refund the belts and
		// friends, and bring people to an existing town.
		for id, b := range w.Buildings {
			if b.Kind.Legacy() {
				for _, c := range Builds[b.Kind].Cost {
					w.Stock[c.Item] += c.N
				}
				delete(w.Buildings, id)
			}
		}
		var ruins []Spec
		for _, r := range w.Ruins {
			if !r.Kind.Legacy() {
				ruins = append(ruins, r)
			}
		}
		w.Ruins = ruins
		w.Free[KHouse] += 3
		w.Free[KFarm]++
		w.Stock[Food] += 60
		if w.Researched["chest"] || w.Researched["assembler"] {
			w.Researched["station"] = true
		}
	}
	w.initCaches()
	if sf.Version < 4 {
		if c := w.Core(); c != nil {
			for i := 0; i < 6; i++ {
				s := w.spawnSettler(c)
				s.X += float64(i%3) - 1
			}
		}
	}
	w.rebuildBookkeeping()
	for _, b := range w.Buildings {
		if b.Kind.Base() == KMiner {
			b.MinerItem, b.MinerTiles = w.minerOre(b.Pos, b.Size())
		}
	}
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

// BackupSave renames an existing save to "<path>.<time>.bak" and keeps only
// the newest `keep` backups. Returns the backup path ("" if there was no save).
func BackupSave(path string, keep int) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	bak := path + "." + time.Now().Format("2006-01-02_15-04-05") + ".bak"
	if err := os.Rename(path, bak); err != nil {
		return "", err
	}
	old, _ := filepath.Glob(path + ".*.bak")
	sort.Strings(old) // timestamps sort chronologically
	for len(old) > keep {
		os.Remove(old[0])
		old = old[1:]
	}
	return bak, nil
}
