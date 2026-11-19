package view

import (
	"testing"

	"potik3d/internal/world"
)

func TestHouseStyleStable(t *testing.T) {
	if v, w, r := HouseStyle(0); v != 0 || w != 0 || r != 0 {
		t.Fatal("ghost (ID 0) must use the plain style")
	}
	var variants [3]bool
	var walls [4]bool
	var roofs [3]bool
	for id := int32(1); id <= 100; id++ {
		v, w, r := HouseStyle(id)
		if v2, w2, r2 := HouseStyle(id); v2 != v || w2 != w || r2 != r {
			t.Fatalf("style of %d not stable", id)
		}
		variants[v], walls[w], roofs[r] = true, true, true
	}
	for _, ok := range append(append(variants[:], walls[:]...), roofs[:]...) {
		if !ok {
			t.Fatalf("not every variant/colour is used: %v %v %v", variants, walls, roofs)
		}
	}
}

func TestSmokeRule(t *testing.T) {
	cases := []struct {
		id        int32
		site      bool
		kind      world.Kind
		residents int
		want      bool
	}{
		{0, false, world.KHouse, 5, false},   // ghost, even with homeless settlers
		{3, true, world.KHouse, 2, false},    // construction site
		{3, false, world.KHouse, 0, false},   // empty house
		{3, false, world.KHouse, 1, true},    // lived-in house
		{1, false, world.KHall, 0, true},     // the hall always
		{4, false, world.KSawmill, 3, false}, // no chimney
	}
	for _, c := range cases {
		if got := wantsSmoke(c.id, c.site, c.kind, c.residents); got != c.want {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}

func TestDecalFacing(t *testing.T) {
	cam := &Camera{X: 0, Y: 0, Yaw: 0, Pitch: 0.8, Scale: 40, W: 800, H: 600}
	cam.prepare()
	b := &Batch{cam: cam}
	quad := [][3]float64{{0, 1, 0}, {1, 1, 0}, {1, 1, 1}, {0, 1, 1}}
	// with yaw 0 the viewer looks along +y: walls facing +y are visible
	if !b.Decal(0, 1, rgb(0xFFFFFF), quad...) || len(b.verts) != 4 {
		t.Fatalf("facing decal not drawn: %d verts", len(b.verts))
	}
	if b.Decal(0, -1, rgb(0xFFFFFF), quad...) || len(b.verts) != 4 {
		t.Fatalf("back-wall decal drawn: %d verts", len(b.verts))
	}
	// windows on the hidden side add nothing either
	n := len(b.verts)
	b.window(0, 0, 1, 1, sideYM, 0.5, 0.2, 0.2, 0.2)
	if len(b.verts) != n {
		t.Fatal("window on a hidden wall was drawn")
	}
	b.window(0, 0, 1, 1, sideYP, 0.5, 0.2, 0.2, 0.2)
	if len(b.verts) != n+8 {
		t.Fatalf("visible window: %d new verts", len(b.verts)-n)
	}
}
