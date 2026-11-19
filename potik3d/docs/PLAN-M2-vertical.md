# M2 — Вертикальне будівництво: помости, сходи, мости

## Goal
Let the player build upward in the 3D settlement game (`potik3d/`): raise walkable
surfaces with **platforms**, put buildings on top of them, connect height differences with
**stairs**, and cross water or gaps with **bridges**. Settlers must path over all of these.

## Current state (recon, with sources)
- Terrain is a height field `World.Height []uint8` (levels 0..`MaxLevel`=8), river bed at
  level 0, a 1-level bank band at level 1, land ≥2 (`internal/world/world.go:generate`).
- One building per tile: `World.owner []int32` (`world.go: mark/Owner/CanPlaceAt`).
- Placement: `CheckPlace` → `CanPlaceAt` requires equal `Height` over the footprint, no owner,
  dry (water ≤0.15) and `h>0` unless `WaterOK` (`build.go`, `world.go`).
- Walking: `WalkHeight(i)` = `Height` (+1 on a finished dam) (`path.go`). A* allows steps with
  |Δ WalkHeight| ≤ 1, diagonals only on level ground; `passable` blocks tiles with water >0.6
  (except dams) and finished buildings except dams/farms/the goal (`path.go`). Idle wandering
  uses the same ±1 rule (`settlers.go:~268`). Settler `Z` follows `WalkHeight` (`settlers.go:78,200`).
- Field workers skip tiles with `owner != 0` (`work.go:105,178`).
- Water: virtual pipes; dams act as a 1-level wall via `wallAt` (`world.go`).
- Rendering: painter's algorithm per tile far→near; buildings drawn at their front tile with
  `base = Height*LevelH` (`internal/view/scene.go:Draw`); `Pick` hits tile tops at `Height`
  (`scene.go:Pick`); ghost uses the same base.
- Saves: gob of `[]*Building` with `SaveVersion=1` (`save.go`); `Kind` values are append-only.

## Decisions
1. **Two layers per tile.** Keep `owner` as the *top* layer (normal buildings, dams, stairs) and
   add a *support* layer `support []int32` for platforms and bridges. A tile may hold at most one
   support and one top building. Rebuilt from buildings in `initCaches` (not saved separately).
2. **New kinds** (appended, so old saves stay valid): `KPlatform`, `KStairs`, `KBridge`, all
   size 1, in a new build category **«Будова»** (`CatBuild`, appended after `CatWater`).
   Costs: Платформа (Помост) 2 колоди + 3 дошки; Сходи 2 колоди + 2 дошки; Міст 3 колоди + 2 дошки.
3. **Building field `Levels int`** (new, gob-compatible; 0 in old saves = not used).
   - Platform: `Levels` 1..3 = how many levels it raises the surface.
   - Bridge: `Levels` 1..3 = deck height above `Height`, chosen automatically at placement.
4. **Surface height.** `Surface(i) = Height[i] + finished support levels`, where a finished
   support contributes `Levels`, a platform under construction contributes `Levels-1` (an
   upgrade in progress keeps the old height; a new site contributes 0). Bridges under
   construction contribute 0. `WalkHeight(i) = Surface(i)` (+1 for a finished dam, unchanged).
5. **Platforms.** Placed on dry land like other buildings (not in water, not at level 0, no props
   kept — props are cleared as today). Placing a platform on a tile whose support is a *finished*
   platform with `Levels<3` and no top building **upgrades** it: `Levels++`, it becomes a site
   again needing the platform cost; refuse at 3 («Помост вже найвищий»). Platforms do not block
   water (no `wallAt`), are passable for walking, and their top can hold buildings.
6. **Buildings on platforms.** Any non-support kind except Farm, Pump and Dam may be placed on
   finished platforms. All footprint tiles must have the same `Surface`, must all be either
   ground or finished platforms (no mixing a platform with bare ground), no top owner, and water
   level `Height+Water` must be below the surface (`< Surface - 0.15` levels for elevated tiles;
   ground tiles keep the existing dry rule). Farms/pumps/dams stay ground-only (moisture and
   water access are ground concepts).
7. **Stairs.** Top-layer, size 1, placed on ground or a finished platform. On placement the
   direction `Dir` (0..3 = +x,−x,+y,−y) is chosen automatically: the orthogonal neighbour whose
   `WalkHeight` is `Surface+2` (preferred) or `Surface+1`; ties broken in the order +x,−x,+y,−y.
   If none, refuse («Сходи мають вести вгору: поставте біля уступу»). Stairs are passable.
   **Rule:** a finished stair lets a settler move orthogonally between the stair tile and the tile
   it faces when that tile is up to **2** levels higher (in both directions). Everywhere else the
   existing ±1 rule applies, so natural terrain play is unchanged and 2-level cliffs/platform
   stacks need stairs.
   **One transition rule everywhere:** a single `World.CanStep(from, to int, goal int32) bool`
   that is the *complete* current A* edge rule plus stairs, for neighbouring tiles only:
   - orthogonal: |ΔWalkHeight| ≤ 1, or ≤ 2 when one tile is a finished stair facing the other;
   - diagonal (unchanged from today): both heights equal **and** both side tiles passable for
     `goal` with the same WalkHeight (no corner cutting); stairs never enable diagonal moves;
   - the destination must be passable for `goal` (or be the goal).
   It is used by A* in `FindPath`,
   by per-step movement along cached paths (decision 10), and by idle wandering in
   `settlers.go` (which today checks `absi(WalkHeight diff) <= 1`; it calls `CanStep(cur, next, 0)`
   whenever its next position lies in another tile). No other movement code may
   compare heights directly.
8. **Bridges.** Support-layer, size 1, allowed on any tile without a top owner or support
   (including water and level 0; also dry gaps). Deck height is chosen at placement:
   `target = max WalkHeight over orthogonal neighbours that are not water-blocked`, then
   `Levels = clamp(target − Height, 1, 3)`. Refuse if no valid neighbour exists («Міст має
   спиратися на берег або інший міст»). A finished bridge ignores water for `passable` and does
   not affect water flow. Nothing can be built on a bridge (except nothing — top layer stays empty).
   **Field-work occupancy:** add `World.occupied(i) = owner[i] != 0 || support[i] != 0` and use
   it in *both* field-target selection (`work.go` `fieldTarget`, today `w.owner[i] != 0`) and the
   final work action (`work.go` forester planting check, today `w.owner[i] == 0`), so no tree is
   planted or targeted on platform or bridge tiles. Test: a forester next to a bridge/platform
   never targets those tiles.
9. **Demolition.** Removing a support with a top building on it is refused («Спочатку знесіть
   те, що стоїть зверху»). Removing a platform refunds only the delivered materials of the
   current site plus `Levels-1` full platform costs if it was upgraded (simple: refund
   `cost × finished levels` + delivered part of an in-progress level).
10. **Settler safety / path invalidation.** A world `pathEpoch` (not saved) is bumped on **every
   change that alters passability or legal transitions**: any building site finishing (incl.
   stairs, platforms, bridges, dams and ordinary buildings, which become blocking), any building
   or site being removed, and any platform upgrade starting. `walk` stores the epoch with a cached
   path and re-plans when it differs. In addition, the per-step movement re-checks the transition
   to the next path tile with `CanStep` and re-plans if
   it is no longer legal. Settlers standing on a changed tile follow the new `WalkHeight`.
11. **Rendering.**
    - Platform: wooden posts at corners from ground to surface, plank deck as a thin box on top
      with edge trim; under construction: posts + partial deck proportional to progress.
    - Bridge: deck at surface (thin), two side rails, two posts down to `Height`.
    - Stairs: `Surface`-based steps rising toward `Dir` over the full rise to the facing tile.
    - Buildings, ghost, settlers, props and hover/pick use `Surface` instead of `Height`.
      `Pick` tests each tile's top at `Surface` (bridges/platforms are clickable via their tile;
      clicking selects the top building first, otherwise the support).
12. **Placement preview.** Add `World.PlanPlacement(k, x, y) (Placement, error)` returning
    `Placement{Dir, Levels int; Upgrade bool; Base int}` (the computed stair direction, the
    bridge/platform target levels, whether it upgrades an existing platform, and the surface level
    the building will stand on). `CheckPlace` returns only its error and `Place` applies its result,
    so preview and placement cannot diverge. The ghost is drawn with the computed `Dir`/`Levels`
    at `Base`, and the tool hint shows a success line: stairs «Сходи: вгору на N рівн. (→ напрям)»,
    bridge «Міст на висоті N», platform «Помост: рівень N/3» (or «Надбудова: рівень N/3»).
    **UI.** New category tab «Будова» with the three items (no rotate key in this milestone).
    Selection panel shows «Рівнів: N/3» for platforms and «Висота: N» for bridges.
13. **Save.** `SaveVersion` stays 1 (fields only added; gob ignores missing). `support` cache is
    rebuilt by `initCaches` by marking support kinds into `support`, others into `owner`.

## Non-goals
Walking *under* platforms, multi-level interiors, rotating with a key, platforms in water,
falling/collapse physics, water-blocking platforms, enemy/combat interactions.

## Acceptance criteria (observable)
A. A finished platform raises `WalkHeight` by 1 and a House can be placed and finished on top;
   `BuildingAt` returns the house, `SupportAt` returns the platform.
B. Platform upgrade 1→2→3 works on an empty platform; a 4th placement and an upgrade with a
   house on top are refused with errors; Farm/Pump/Dam refused on a platform.
C. With a platform stack of 2 levels next to ground, `FindPath` from the ground to the platform
   top fails without stairs and succeeds with finished stairs facing it. Existing
   `TestPathAroundTerrain` still passes, and a new `TestCanStepMatchesOldRule` checks that on the
   generated terrain without structures `CanStep` equals the old A* edge rule for every tile and
   all 8 neighbours.
D. A row of bridges across the river (auto height) lets `FindPath` cross from bank to bank where
   it failed before; water simulation stays within 1% total-volume of the no-bridge run after
   N steps (bridges don't block water).
E. Demolishing a platform with a house on top is refused; demolishing the house then the
   platform works and refunds materials to storage.
F. Save/load round trip keeps `Levels`, `Dir`, supports and top buildings (`TestSaveRoundTrip`
   extended).
G. `TestEarlyGame` still passes (no regression in the normal economy).
H. Visual: a screenshot of a demo town with a 2-level platform + house on top, stairs leading to
   it and a bridge across the river renders correctly (no missing faces/obvious sorting errors).

## Verification
- `cd potik3d && gofmt -l . && go vet ./... && go test ./...` → no gofmt output, vet clean, all
  tests pass (new tests: `TestPlatformStack`, `TestStairsPath`, `TestBridgeCross`,
  `TestDemolishSupport`, `TestFieldWorkSkipsSupports`, `TestStairRemovalReplans` (a settler
  walking a route over a stair re-plans and never crosses a 2-level gap after the stair is removed), `TestPlanPlacement` (preview equals the
  placed building's `Dir`/`Levels`), extended `TestSaveRoundTrip`).
- `./build.sh` → `dist/potik3d.exe` builds for Windows.
- Visual: `xvfb-run ... ./p3d -debug-shot vert.png -debug-demo 300 -debug-vertical` (new flag
  that adds the demo structures via `world.BuildVerticalDemo` and completes their sites) and
  inspect the PNG.

## Risks
- Painter's-algorithm artifacts for tall stacks seen from low pitch (accepted; tiles are drawn
  far→near and each tile draws bottom→top).
- Hauling to elevated buildings depends on stairs; a building placed on a platform with no way
  up will never be supplied. Mitigation: the tool hint warns «Немає шляху нагору — поставте
  сходи» is out of scope; the selection panel shows the site's missing materials as usual.
