# M3 — Детальніші low-poly моделі будівель

## Goal
Make the town look richer while keeping the flat-shaded low-poly style and staying fast over RDP:
add windows, doors, chimneys with smoke, porches, signs and props to buildings, and give houses
visible variety, without textures or image files.

## Current state (recon, with sources)
- All models are built from `Batch` primitives `Box`, `Prism`, `Prismoid`, `Pyramid` in
  `internal/view/view.go`; per-kind models in `internal/view/models.go:drawBuilding`.
  Houses are one box + 4-sided pyramid roof, identical for every house.
- Drawing is the painter's algorithm per tile (far→near); within a building, parts draw in code
  order. `Box` emits only camera-facing sides (`Camera.facing`), then the top.
- A building's model is drawn once at its front tile with `base = Surface*LevelH`
  (`scene.go:Draw`); ghosts use a zero-ID `Building` (`scene.go`).
- `World.Residents(id)` gives occupants; `Building.Used()`/`Def().Cap` storage fill;
  `OutN`/`InN`/`Working` production state (`internal/world`).
- No per-frame triangle count exists today.
- Previous milestone's plan/log moved to `docs/` (M2).

## Decisions
1. **No textures.** Detail comes from extra low-poly geometry and flat colour only.
2. **Wall details as decals.** New `Batch.Decal(nx, ny, c, pts...)`: a flat quad lying 0.004 in
   front of a wall with horizontal normal (nx, ny); it is emitted **only if that wall faces the
   camera** (`facing(nx, ny)`). Windows, doors, frames, signs and wall bands use it, so a detail
   on a back wall can never show through the front wall.
3. **Roof-mounted parts** (chimneys, flag poles) are drawn **before** the roof when their
   position is farther from the camera than the roof's centre (rotated depth compare via a new
   `Batch.Farther(x, y, cx, cy) bool`), otherwise after it, so they never pop through the roof.
4. **House variety.** `HouseStyle(id int32) (variant, wall, roof int)` derived from a fixed hash
   of the building ID (stable across save/load; ID 0 = ghost gives variant 0, palette 0). Three
   variants:
   - A cottage: gable roof (Prism), chimney, 1 door + windows;
   - B hipped roof (Pyramid) with a small porch roof over the door on posts, chimney;
   - C narrow two-storey: taller box, upper window row, steep gable roof, chimney.
   Wall palette 4 colours, roof palette 3 colours. Door on the +y side for A/C and −x side for B.
5. **Smoke.** Houses with `Residents > 0` and the hall show 3 small grey cubes rising from the
   chimney, looping every 3 s with phase from the ID; drawn with `Alpha` 0.55, after the roof.
6. **Other buildings** (same footprint, same anchor):
   - Hall: stone plinth band, door, windows on all four sides, flag on a pole waving.
   - Warehouse: big double doors, 0–4 crates beside it in proportion to `Used()/Cap`.
   - Sawmill: open shed roof on posts, log pile showing `InN` (0–3 logs), existing blade.
   - Lumberjack: stump with an axe, small log pile from `OutN`.
   - Forester: two saplings in pots, sign.
   - Gatherer: basket with berries showing `OutN`, sign.
   - Pump: bucket; Tank: two metal hoops; Campfire: two log seats.
   Farm, dam, platform, bridge, stairs and construction sites stay as they are.
7. **Performance budget.** Add `Batch.Tris` (triangles emitted) and `Scene.LastTris`. The main
   program gets `-debug-stats`, printing `tris=N` for the screenshot frame to stderr. Budget: on
   the demo town (`-debug-demo 600 -debug-vertical`, default camera) triangles may grow by at
   most **60 %** over the old models, measured by running the same flags with the counter on the
   old models (counter added first, measured, then models changed).
8. **Selected tint** keeps working: selected buildings mix wall colours with gold as today.

## Non-goals
Textures or image assets, interiors, day/night lighting, settler animation, new building kinds,
changes to simulation or saves.

## Acceptance criteria
A. Houses show at least 2 distinct variants in the demo town, each with door + windows; the
   variant for a given ID is identical across runs and after save/load (pure function of ID).
B. No wall decal, chimney or flag is visible through a wall or roof: screenshots of the demo town
   at yaw 0.55, 2.1, 3.7 and 5.2 show details only on visible faces.
C. Smoke appears only over houses with residents (and the hall).
D. Warehouse crates, sawmill log pile, lumber/gatherer piles reflect state (none when empty).
E. Triangle count on the demo town is ≤ 1.6× the baseline (decision 7).
F. All existing tests pass; new `internal/view` unit tests: `TestHouseStyleStable` (same ID →
   same style, all variants/palettes reachable over IDs 1..100, ID 0 → variant 0) and
   `TestDecalFacing` (a Batch with a fixed camera emits a decal for a facing wall and nothing
   for a back wall; counts vertices only, no drawing).

## Verification
- `cd potik3d && gofmt -l . && go vet ./... && go test ./...`
- `./build.sh` builds `dist/potik3d.exe`.
- Visual: `p3d -debug-shot town_{yaw}.png -debug-demo 600 -debug-vertical -debug-yaw {0.55,2.1,3.7,5.2} -debug-scale 70`
  plus a selected-building shot; inspect each PNG.
- Perf: `-debug-stats` on old vs. new models with identical flags; report both numbers.

## Risks
- Painter's ordering inside a building is code-order based; decisions 2–3 cover walls and roof
  items; small props stand beside buildings and are drawn after the body.
- More triangles may slow software rendering over RDP; decision 7 bounds this.
