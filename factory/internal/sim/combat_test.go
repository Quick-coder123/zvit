package sim

import "testing"

func TestResearch(t *testing.T) {
	w := NewWorld(5)
	w.Stock = [ItemCount]int{}
	if err := w.Research("assembler"); err != ErrPrereq {
		t.Fatalf("assembler before furnace: %v", err)
	}
	if err := w.Research("furnace"); err != ErrNoFunds {
		t.Fatalf("furnace without ore: %v", err)
	}
	w.Stock[IronOre], w.Stock[Stone] = 30, 20
	if err := w.Research("furnace"); err != nil || w.Stock[IronOre] != 0 || !w.Unlocked(KFurnace) {
		t.Fatalf("research furnace: %v", err)
	}
	if w.autoRecipe(KFurnace, IronPlate) >= 0 {
		t.Fatal("steel should be locked")
	}
	// research can be paid from storehouses too
	ch := built(t, w, KChest, Pos{5, 5})
	ch.Store = []Stack{{Stone, 30}, {IronOre, 30}}
	if err := w.Research("station"); err != nil || ch.StoreCount() != 0 {
		t.Fatalf("pay from storehouse: %v", err)
	}
}

func TestFreeTokenRefund(t *testing.T) {
	w := NewWorld(5)
	b, _ := w.Place(KHouse, Pos{5, 5}, East, 0)
	w.Remove(b.ID)
	if w.Free[KHouse] != 1 {
		t.Fatalf("token refund: %d", w.Free[KHouse])
	}
}

func TestWaveScheduleAndCoreDefends(t *testing.T) {
	w := NewWorld(11)
	w.NextWave = DT / 2
	w.Update(nil)
	if w.Wave != 1 || len(w.Enemies) == 0 {
		t.Fatalf("wave not spawned: wave=%d enemies=%d", w.Wave, len(w.Enemies))
	}
	n := len(w.Enemies)
	run(w, 120)
	if len(w.Enemies) != 0 || w.Kills != n {
		t.Fatalf("enemies left %d, kills %d/%d", len(w.Enemies), w.Kills, n)
	}
	if c := w.Core(); c.Dmg >= w.MaxHP(c) {
		t.Fatal("town hall should survive wave 1")
	}
}

func TestEnemiesDestroyAndRuins(t *testing.T) {
	w := emptyWorld(2)
	w.Peaceful = false
	w.NextWave = 1e9
	w.Buildings[1].Pos = Pos{-40, -40} // move the town hall's gun out of range
	w.initCaches()
	wall := built(t, w, KWall, Pos{20, 0})
	w.Enemies = []*Enemy{{Kind: Brute, X: 23, Y: 0.5, HP: 1e6}}
	run(w, 60)
	if w.Buildings[wall.ID] != nil {
		t.Fatal("brute should break the wall")
	}
	if len(w.Ruins) != 1 || w.Ruins[0].Kind != KWall {
		t.Fatalf("ruins %v", w.Ruins)
	}
	w.Enemies = nil
	if built, _ := w.RebuildRuins(); built != 1 {
		t.Fatal("rebuild failed")
	}
	if b := w.At(Pos{20, 0}); b == nil || !b.Site {
		t.Fatal("rebuild should place a construction site")
	}
}

func TestTurretNeedsWorkerAndAmmo(t *testing.T) {
	w := emptyWorld(0)
	w.Peaceful = false
	w.NextWave = 1e9
	w.Buildings[1].Pos = Pos{-40, -40}
	w.initCaches()
	tu := built(t, w, KTurret, Pos{0, 0})
	tu.Ammo = 100
	w.Enemies = []*Enemy{{Kind: Crawler, X: 6, Y: 0.5, HP: 1e6}}
	run(w, 2)
	if w.Enemies[0].HP != 1e6 {
		t.Fatal("turret without a worker must not shoot")
	}
	// a worker arrives; ammo is hauled from the town hall
	tu.Ammo = 0
	w.spawnSettler(w.Buildings[1])
	w.spawnSettler(w.Buildings[1])
	w.Enemies[0].HP = 30
	run(w, 60)
	if len(w.Enemies) != 0 {
		t.Fatal("crawler should be dead")
	}
	if tu.Ammo <= 0 {
		t.Fatal("ammo should have been delivered")
	}
}

func TestFlamerBurnsGroup(t *testing.T) {
	w := emptyWorld(1)
	w.Peaceful = false
	w.NextWave = 1e9
	w.Buildings[1].Pos = Pos{-40, -40}
	w.initCaches()
	f := built(t, w, KFlamer, Pos{0, 3})
	f.Ammo = 200
	run(w, 3) // worker walks in
	for i := 0; i < 5; i++ {
		w.Enemies = append(w.Enemies, &Enemy{Kind: Swarmer, X: 3 + float64(i)*0.2, Y: 3.5, HP: 12})
	}
	run(w, 2)
	if len(w.Enemies) != 0 {
		t.Fatalf("%d swarmers survived the flamer", len(w.Enemies))
	}
}

func TestCoreFallPenalty(t *testing.T) {
	w := NewWorld(2)
	w.Stock[IronPlate] = 100
	w.Enemies = []*Enemy{{Kind: Crawler, X: 50, Y: 50, HP: 30}}
	c := w.Core()
	w.Damage(c, w.MaxHP(c))
	if w.Stock[IronPlate] != 75 || len(w.Enemies) != 0 || c.Dmg != 0 {
		t.Fatalf("penalty: stock=%d enemies=%d dmg=%v", w.Stock[IronPlate], len(w.Enemies), c.Dmg)
	}
}

func TestSaveKeepsCombatState(t *testing.T) {
	w := NewWorld(4)
	w.Researched["walls"] = true
	w.Enemies = []*Enemy{{Kind: Brute, X: 3, Y: 4, HP: 100}}
	w.Ruins = []Spec{{Kind: KWall, Pos: Pos{9, 9}}}
	w.Wave = 3
	w2 := roundTrip(t, w)
	if !w2.Researched["walls"] || len(w2.Enemies) != 1 || w2.Wave != 3 || len(w2.Ruins) != 1 || w2.Free[KMiner] != 1 {
		t.Fatal("combat state lost")
	}
}

func TestMachinesAreOneTile(t *testing.T) {
	for _, k := range []Kind{KMiner, KFurnace, KAssembler, KTurret, KHouse, KFarm, KStation, KChest} {
		if Builds[k].Size != 1 {
			t.Fatalf("%s should be 1×1", Builds[k].Name)
		}
	}
	for _, k := range []Kind{KMiner, KFurnace, KAssembler, KTurret} {
		if big := k.Big(); Builds[big].Size != 2 || big.Base() != k {
			t.Fatalf("%s needs a 2×2 upgrade", Builds[k].Name)
		}
	}
	if Builds[KCore].Size != 3 {
		t.Fatal("town hall stays 3×3")
	}
}

func TestBigAssemblerIsFaster(t *testing.T) {
	w := emptyWorld(2)
	a := built(t, w, KAssemblerBig, Pos{0, 6})
	w.SetRecipe(a.ID, 5) // gears
	run(w, 5)            // two workers walk in
	if !a.Running() {
		t.Fatal("big assembler needs two workers")
	}
	a.InBuf[0].N, a.OutN, a.Progress, a.Working = 2, 0, 0, false
	w.Researched["craft1"], w.Researched["craft2"] = false, false
	before := a.OutN
	for i := 0; i < int((2.0/3+0.05)*TPS); i++ {
		w.Update(nil)
	}
	if a.OutN-before < 1 {
		t.Fatalf("a 2s gear should take 2/3s, out=%d", a.OutN)
	}
}
