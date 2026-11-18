package sim

import "testing"

func TestStarterEconomy(t *testing.T) {
	w := NewWorld(5)
	// furnace is locked at the start
	if err := w.CheckPlace(KFurnace, Pos{5, 5}, East); err != ErrLocked {
		t.Fatalf("furnace should be locked: %v", err)
	}
	// one free miner and ten free belts, nothing else
	if _, err := w.Place(KMiner, Pos{12, -6}, West, 0); err != nil {
		t.Fatalf("free miner: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := w.Place(KBelt, Pos{11 - i, -5}, West, 0); err != nil {
			t.Fatalf("free belt %d: %v", i, err)
		}
	}
	if _, err := w.Place(KBelt, Pos{0, 5}, West, 0); err != ErrNoFunds {
		t.Fatalf("11th belt should cost ore: %v", err)
	}
	// research needs resources and prerequisites
	if err := w.Research("assembler"); err != ErrPrereq {
		t.Fatalf("assembler before furnace: %v", err)
	}
	if err := w.Research("furnace"); err != ErrNoFunds {
		t.Fatalf("furnace without ore: %v", err)
	}
	w.Stock[IronOre], w.Stock[Stone] = 30, 20
	if err := w.Research("furnace"); err != nil || w.Stock[IronOre] != 0 {
		t.Fatalf("research furnace: %v", err)
	}
	if !w.Unlocked(KFurnace) {
		t.Fatal("furnace not unlocked")
	}
	// steel recipe is locked until researched
	if w.autoRecipe(KFurnace, IronPlate) >= 0 {
		t.Fatal("steel should be locked")
	}
}

func TestFreeTokenRefund(t *testing.T) {
	w := NewWorld(5)
	b, _ := w.Place(KBelt, Pos{5, 5}, East, 0)
	w.Remove(b.ID)
	if w.Free[KBelt] != 10 || w.Stock[IronOre] != 0 {
		t.Fatalf("token refund: free=%d ore=%d", w.Free[KBelt], w.Stock[IronOre])
	}
}

func TestWaveScheduleAndCoreDefends(t *testing.T) {
	w := NewWorld(11)
	if w.NextWave != FirstWaveDelay {
		t.Fatal("first wave timer")
	}
	w.NextWave = DT / 2
	w.Update(nil)
	if w.Wave != 1 || len(w.Enemies) == 0 {
		t.Fatalf("wave not spawned: wave=%d enemies=%d", w.Wave, len(w.Enemies))
	}
	n := len(w.Enemies)
	// the core gun alone should handle the first wave
	run(w, 120)
	if len(w.Enemies) != 0 || w.Kills != n {
		t.Fatalf("enemies left %d, kills %d/%d", len(w.Enemies), w.Kills, n)
	}
	if c := w.Core(); c.Dmg >= w.MaxHP(c) {
		t.Fatal("core should survive wave 1")
	}
}

func TestEnemiesDestroyAndRuins(t *testing.T) {
	w := emptyWorld()
	w.Peaceful = false
	w.NextWave = 1e9
	core(w, -1, -1)
	wall := mustPlace(t, w, KWall, Pos{20, 0}, East)
	w.Enemies = []*Enemy{{Kind: Brute, X: 23, Y: 0.5, HP: 1e6}}
	run(w, 60)
	if w.Buildings[wall.ID] != nil {
		t.Fatal("brute should break the wall")
	}
	if len(w.Ruins) != 1 || w.Ruins[0].Kind != KWall {
		t.Fatalf("ruins %v", w.Ruins)
	}
	w.Enemies = nil
	built, left := w.RebuildRuins()
	if built != 1 || left != 0 || w.At(Pos{20, 0}) == nil {
		t.Fatal("rebuild failed")
	}
}

func TestTurretUsesAmmo(t *testing.T) {
	w := emptyWorld()
	w.Peaceful = false
	w.NextWave = 1e9
	core(w, -30, -1)
	tu := mustPlace(t, w, KTurret, Pos{0, 0}, East)
	w.Enemies = []*Enemy{{Kind: Crawler, X: 8, Y: 1, HP: 30}}
	run(w, 1)
	if w.Enemies[0].HP != 30 {
		t.Fatal("turret without ammo must not shoot")
	}
	src := mustPlace(t, w, KChest, Pos{-2, 0}, East)
	src.Store = []Stack{{Ammo, 1}}
	mustPlace(t, w, KBelt, Pos{-1, 0}, East)
	run(w, 3)
	if len(w.Enemies) != 0 {
		t.Fatal("crawler should be dead")
	}
	if tu.Ammo >= AmmoShots || tu.Ammo <= 0 {
		t.Fatalf("ammo left %d", tu.Ammo)
	}
}

func TestFlamerBurnsGroup(t *testing.T) {
	w := emptyWorld()
	w.Peaceful = false
	w.NextWave = 1e9
	core(w, -30, -1)
	f := mustPlace(t, w, KFlamer, Pos{0, 0}, East)
	f.Ammo = 200
	for i := 0; i < 5; i++ {
		w.Enemies = append(w.Enemies, &Enemy{Kind: Swarmer, X: 3 + float64(i)*0.2, Y: 0.5, HP: 12})
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
	w.Research("walls")
	w.Researched["walls"] = true
	w.Enemies = []*Enemy{{Kind: Brute, X: 3, Y: 4, HP: 100}}
	w.Ruins = []Spec{{Kind: KWall, Pos: Pos{9, 9}}}
	w.Wave = 3
	w2 := roundTrip(t, w)
	if !w2.Researched["walls"] || len(w2.Enemies) != 1 || w2.Wave != 3 || len(w2.Ruins) != 1 || w2.Free[KBelt] != 10 {
		t.Fatal("combat state lost")
	}
}
