package sim

import "testing"

func TestResearchAtStation(t *testing.T) {
	w := emptyWorld(4)
	w.Researched = map[string]bool{}
	w.Stock[IronOre], w.Stock[Stone] = 40, 40
	if err := w.Research("assembler"); err != ErrPrereq {
		t.Fatalf("assembler with nothing before it: %v", err)
	}
	if err := w.Research("furnace"); err != nil {
		t.Fatal(err)
	}
	// queuing a follow-up is allowed once its prerequisite is queued
	if err := w.Research("assembler"); err != nil {
		t.Fatalf("queue assembler after furnace: %v", err)
	}
	if err := w.Research("furnace"); err != ErrQueued {
		t.Fatalf("double queue: %v", err)
	}
	run(w, 60)
	if w.Researched["furnace"] {
		t.Fatal("research must not finish without a research station")
	}
	built(t, w, KLab, Pos{4, 3})
	run(w, 30)
	if w.Researched["furnace"] {
		t.Fatal("research needs time after materials arrive")
	}
	run(w, 120)
	if !w.Researched["furnace"] || !w.Unlocked(KFurnace) {
		d, wk := w.ResearchProgress()
		t.Fatalf("furnace not researched: delivered %.2f work %.2f need %v", d, wk, w.ResearchNeed)
	}
	if w.Stock[IronOre] != 10 || w.Stock[Stone] != 20 {
		t.Fatalf("materials not consumed: ore %d stone %d", w.Stock[IronOre], w.Stock[Stone])
	}
	if a := w.ActiveResearch(); a == nil || a.ID != "assembler" {
		t.Fatal("queue should move on to the assembler")
	}
}

func TestCancelResearchRefunds(t *testing.T) {
	w := emptyWorld(3)
	w.Researched = map[string]bool{}
	w.Stock[IronOre], w.Stock[Stone] = 40, 40
	built(t, w, KLab, Pos{4, 3})
	w.Research("furnace")
	w.Research("assembler")
	run(w, 20) // some materials delivered
	w.CancelResearch("furnace")
	run(w, 10) // let carriers drop anything in hand
	if len(w.ResearchQueue) != 0 {
		t.Fatalf("dependent research should be cancelled too: %v", w.ResearchQueue)
	}
	if got := w.TotalStock(IronOre) + w.TotalStock(Stone); got != 80 {
		t.Fatalf("materials not refunded: %d", got)
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
	w.Peaceful = false
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

func TestWaveSchedule(t *testing.T) {
	w := NewWorld(1)
	if !w.Peaceful {
		t.Fatal("new worlds start peaceful")
	}
	if w.NextWave != 20*60 {
		t.Fatalf("first wave in %.0fs, want 20 min", w.NextWave)
	}
	if WaveInterval(1) < 7*60 || WaveInterval(30) != 180 {
		t.Fatalf("intervals %.0f .. %.0f", WaveInterval(1), WaveInterval(30))
	}
	for n := 1; n < 20; n++ {
		if WaveInterval(n+1) > WaveInterval(n) {
			t.Fatal("intervals should not grow")
		}
	}
}
