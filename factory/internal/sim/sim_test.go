package sim

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// emptyWorld has a town hall, everything researched, no enemies, plenty of
// stock and n settlers.
func emptyWorld(n int) *World {
	w := &World{Seed: 1, Buildings: map[int32]*Building{}, Researched: map[string]bool{}, Peaceful: true}
	w.initCaches()
	w.UnlockAll()
	c := w.add(&Building{Kind: KCore, Pos: Pos{-1, -1}, Recipe: -1})
	for i := range w.Stock {
		w.Stock[i] = 15 // stays well within the town hall's capacity
	}
	for i := 0; i < n; i++ {
		w.spawnSettler(c)
	}
	return w
}

func mustPlace(t *testing.T, w *World, k Kind, p Pos) *Building {
	t.Helper()
	b, err := w.Place(k, p, East, 0)
	if err != nil {
		t.Fatalf("place %v at %v: %v", Builds[k].Name, p, err)
	}
	return b
}

// built places a finished building (skips construction).
func built(t *testing.T, w *World, k Kind, p Pos) *Building {
	t.Helper()
	b := mustPlace(t, w, k, p)
	b.Site, b.Need = false, nil
	return b
}

func run(w *World, seconds float64) {
	for i := 0; i < int(seconds*TPS); i++ {
		w.Update(nil)
	}
}

func roundTrip(t *testing.T, w *World) *World {
	t.Helper()
	var buf bytes.Buffer
	if err := w.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	w2, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	return w2
}

func ironSpot(t *testing.T, w *World) Pos {
	t.Helper()
	for y := -12; y < 12; y++ {
		for x := 6; x < 25; x++ {
			if w.OreAt(Pos{x, y}) == IronOre && w.At(Pos{x, y}) == nil {
				return Pos{x, y}
			}
		}
	}
	t.Fatal("no iron near start")
	return Pos{}
}

func TestPlacementRules(t *testing.T) {
	w := NewWorld(3)
	if _, err := w.Place(KHouse, Pos{0, 0}, East, 0); err != ErrOccupied {
		t.Fatalf("placing on the town hall: %v", err)
	}
	if _, err := w.Place(KMiner, Pos{0, 4}, East, 0); err != ErrNoOre {
		t.Fatalf("miner off ore: %v", err)
	}
	if _, err := w.Place(KFarm, ironSpot(t, w), East, 0); err != ErrOnOre {
		t.Fatalf("farm on ore: %v", err)
	}
	if _, err := w.Place(KFurnace, Pos{5, 5}, East, 0); err != ErrLocked {
		t.Fatalf("furnace locked at start: %v", err)
	}
	if _, err := w.Place(KBelt, Pos{5, 5}, East, 0); err != ErrLegacy {
		t.Fatalf("belts are gone: %v", err)
	}
	// placing creates a site that needs materials, no instant payment
	b := mustPlace(t, w, KChest, Pos{4, 4})
	if !b.Site || len(b.Need) != 2 || b.Need[0] != (Stack{Wood, 8}) || w.Stock[Wood] != 30 {
		t.Fatalf("site: %+v stone=%d", b.Need, w.Stock[Stone])
	}
}

func TestStarterSettlement(t *testing.T) {
	w := NewWorld(42)
	w.Peaceful = true
	if w.Population() != 4 || w.Housing() != 4 {
		t.Fatalf("start pop %d housing %d", w.Population(), w.Housing())
	}
	m := mustPlace(t, w, KMiner, ironSpot(t, w))
	f := mustPlace(t, w, KFarm, moistSpot(w, Pos{-4, 4}))
	h := mustPlace(t, w, KHouse, Pos{3, 4})
	if len(m.Need)+len(f.Need)+len(h.Need) != 0 {
		t.Fatal("starter buildings should be free")
	}
	ore := w.Stock[IronOre]
	run(w, 120)
	if m.Site || f.Site || h.Site {
		t.Fatal("starter buildings not built")
	}
	if !m.Running() || !f.Running() {
		t.Fatalf("miner staffed=%d farm staffed=%d", m.Inside(), f.Inside())
	}
	if w.Stock[IronOre] <= ore {
		t.Fatal("no ore hauled to the town hall")
	}
	if w.Housing() != 7 {
		t.Fatalf("housing %d", w.Housing())
	}
}

func TestConstructionHaulsMaterials(t *testing.T) {
	w := emptyWorld(3)
	stone := w.Stock[Stone]
	f := mustPlace(t, w, KFurnace, Pos{8, 3})
	run(w, 40)
	if f.Site {
		t.Fatalf("furnace still a site, need=%v work=%.2f", f.Need, f.BuildWork)
	}
	if w.Stock[Stone] != stone-12 {
		t.Fatalf("stone used %d, want 12", stone-w.Stock[Stone])
	}
}

func TestNoWorkerNoWork(t *testing.T) {
	w := emptyWorld(0)
	m := built(t, w, KMiner, ironSpot(t, w))
	run(w, 20)
	if m.OutN != 0 || m.Running() {
		t.Fatal("a miner without a worker must not work")
	}
}

func TestPauseFreesWorker(t *testing.T) {
	w := emptyWorld(1)
	m := built(t, w, KMiner, ironSpot(t, w))
	run(w, 10)
	if !m.Running() || w.FreeWorkers() != 0 {
		t.Fatal("worker should be employed at the miner")
	}
	w.SetPaused(m.ID, true)
	run(w, 1)
	if m.Running() || w.FreeWorkers() != 1 {
		t.Fatal("pausing should free the worker")
	}
	w.SetPaused(m.ID, false)
	run(w, 10)
	if !m.Running() {
		t.Fatal("worker should come back")
	}
}

func TestSmeltingChain(t *testing.T) {
	w := emptyWorld(4)
	w.Stock[IronOre], w.Stock[IronPlate] = 0, 0
	built(t, w, KMiner, ironSpot(t, w))
	f := built(t, w, KFurnace, Pos{5, -3})
	w.SetRecipe(f.ID, 0)
	run(w, 150)
	if w.Stock[IronPlate] < 8 {
		t.Fatalf("only %d iron plates reached the town hall", w.Stock[IronPlate])
	}
}

func TestStarvation(t *testing.T) {
	w := emptyWorld(3)
	for _, f := range Foods {
		w.Stock[f] = 0
	}
	for _, s := range w.Settlers {
		s.MealIn = 1
	}
	run(w, StarveDeath+5)
	if w.Deaths != 3 {
		t.Fatalf("deaths %d, want 3", w.Deaths)
	}
}

func TestFarmFeeds(t *testing.T) {
	w := emptyWorld(4)
	w.Stock[Food], w.Stock[Berries], w.Stock[Bread] = 2, 0, 0
	built(t, w, KFarm, moistSpot(w, Pos{-4, 4}))
	built(t, w, KPump, pumpSpot(w, Pos{-4, 0}))
	run(w, 400)
	if w.Deaths != 0 {
		t.Fatalf("%d starved despite a farm", w.Deaths)
	}
}

func TestBirths(t *testing.T) {
	w := emptyWorld(2)
	built(t, w, KHouse, Pos{4, 4})
	built(t, w, KHouse, Pos{6, 4})
	run(w, 5*BirthTime)
	if w.Population() <= 2 || w.Population() > w.Housing() {
		t.Fatalf("population %d (housing %d)", w.Population(), w.Housing())
	}
	for _, f := range Foods {
		w.Stock[f] = 0
	}
	pop := w.Population()
	run(w, 2*BirthTime)
	if w.Population() > pop {
		t.Fatal("no births without food")
	}
}

func TestStationCarriers(t *testing.T) {
	w := emptyWorld(4)
	w.Stock[IronOre] = 0
	spot := ironSpot(t, w)
	built(t, w, KMiner, spot)
	built(t, w, KStation, Pos{spot.X - 4, spot.Y + 1})
	// a storehouse within the station's radius receives the ore even when the
	// town hall is too far for the carriers
	ch := built(t, w, KChest, Pos{spot.X - 3, spot.Y + 3})
	run(w, 120)
	if w.FreeWorkers() != 0 {
		t.Fatalf("free workers %d, want all employed", w.FreeWorkers())
	}
	if got := w.TotalStock(IronOre); got < 15 {
		t.Fatalf("carriers delivered only %d ore (chest %d)", got, ch.StoreCount())
	}
	for _, s := range w.Settlers {
		if s.Carry.N > CarryStation {
			t.Fatal("overloaded")
		}
	}
}

func TestSaveRoundTrip(t *testing.T) {
	w := NewWorld(9)
	mustPlace(t, w, KHouse, Pos{4, 4})
	run(w, 3)
	w2 := roundTrip(t, w)
	if len(w2.Buildings) != len(w.Buildings) || w2.Stock != w.Stock || len(w2.Settlers) != len(w.Settlers) {
		t.Fatal("mismatch after load")
	}
	if w2.OreAt(Pos{13, -5}) != IronOre {
		t.Fatal("ore not regenerated from seed")
	}
	run(w2, 60)
	if b := w2.At(Pos{4, 4}); b == nil || b.Site {
		t.Fatal("construction should continue after loading")
	}
}

func TestOldBeltSaveMigrates(t *testing.T) {
	w := emptyWorld(0)
	w.Stock = [ItemCount]int{}
	w.add(&Building{Kind: KBelt, Pos: Pos{5, 5}, Recipe: -1})
	w.add(&Building{Kind: KSplitter, Pos: Pos{6, 5}, Recipe: -1})
	w.add(&Building{Kind: KFurnace, Pos: Pos{7, 5}, Recipe: -1})
	var buf bytes.Buffer
	sf := saveFile{Version: 3, Seed: w.Seed, NextID: w.NextID, Stock: w.Stock[:], Researched: map[string]bool{}}
	for _, id := range w.Order() {
		sf.Buildings = append(sf.Buildings, w.Buildings[id])
	}
	zw := gzip.NewWriter(&buf)
	gob.NewEncoder(zw).Encode(&sf)
	zw.Close()
	w2, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if w2.At(Pos{5, 5}) != nil || w2.At(Pos{6, 5}) != nil || w2.At(Pos{7, 5}) == nil {
		t.Fatal("belts should be removed, machines kept")
	}
	if w2.Stock[IronOre] != 1 || w2.Stock[IronPlate] != 4 || w2.Population() == 0 || w2.Stock[Food] == 0 {
		t.Fatalf("refund/settlers: ore=%d plate=%d pop=%d", w2.Stock[IronOre], w2.Stock[IronPlate], w2.Population())
	}
}

func TestBackupRotation(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/solo.sav"
	if b, err := BackupSave(path, 2); err != nil || b != "" {
		t.Fatalf("no save: %q %v", b, err)
	}
	for i := 0; i < 4; i++ {
		os.WriteFile(path, []byte{byte(i)}, 0o644)
		os.WriteFile(path+".0000-0"+string(rune('0'+i))+".bak", nil, 0o644)
	}
	if _, err := BackupSave(path, 2); err != nil {
		t.Fatal(err)
	}
	left, _ := filepath.Glob(path + ".*.bak")
	if len(left) != 2 {
		t.Fatalf("kept %d backups", len(left))
	}
}

func TestStorageIsLimited(t *testing.T) {
	w := emptyWorld(3)
	w.Researched["chest"] = false
	core := w.Core()
	if cap := w.StorageCap(core); cap != CoreCap {
		t.Fatalf("town hall capacity %d", cap)
	}
	// fill the town hall almost to the brim
	w.Stock[Stone] += CoreCap - w.StorageUsed(core) - 2
	m := built(t, w, KMiner, ironSpot(t, w))
	run(w, 90)
	if used := w.StorageUsed(core); used > CoreCap {
		t.Fatalf("town hall overfilled: %d", used)
	}
	if m.OutN < OutCap-1 {
		t.Fatalf("miner should back up when storage is full, out=%d", m.OutN)
	}
	full := false
	for _, n := range w.Notices {
		if strings.Contains(n, "Склади заповнені") {
			full = true
		}
	}
	if !full {
		t.Fatalf("expected a storage-full warning, got %v", w.Notices)
	}
	// a storehouse unblocks production
	ch := built(t, w, KChest, Pos{4, 4})
	run(w, 120)
	if ch.StoreCount() == 0 {
		t.Fatal("goods should go to the new storehouse")
	}
	if ch.StoreCount() > ChestCap {
		t.Fatalf("storehouse overfilled: %d", ch.StoreCount())
	}
}
