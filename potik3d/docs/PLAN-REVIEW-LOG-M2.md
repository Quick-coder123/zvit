# Plan review log — M2 vertical building

- Host/coordinator: Claude Code (claude). Plan reviewer: Codex. Builder: Claude (host). Final inspector: fresh Codex session.
- Models: CLI defaults (no explicit override). Codex CLI codex-cli 0.170.0.
- Scope: potik3d vertical building (platforms, stairs, bridges). Authorization: user asked to plan and implement ("Так роби"); commits/pushes follow the session's branch rules.
- Limits: MAX_ROUNDS=5, MAX_FIX_ROUNDS=2, MAX_INSPECTION_ROUNDS=2. Inspection: on.
- Diagnostics: outside checkout in scratchpad/claudex.

## Round 1 — Codex (effective model gpt-6-luna, CLI default), verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/review1/claudex-zyrgs74c/result.json
- M2-1 (medium) bridges invisible to field-work occupancy checks → ACCEPTED, decision 8 + test.
- M2-2 (low) UI hint can't show auto Dir/height on success → ACCEPTED, decision 12 PlanPlacement + test.

## Round 2 — Codex resumed session, verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/review2/claudex-hladyizo/result.json
- M2-1, M2-2 confirmed addressed by reviewer.
- M2-3 (medium) stale cached paths over demolished stairs → ACCEPTED, decision 10 rewritten + test.

## Round 3 — Codex resumed session, verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/review3/claudex-oc22rt6b/result.json
- M2-3 confirmed addressed.
- M2-4 (low) idle wandering ignores stair rule → ACCEPTED, shared CanStep rule in decision 7.

## Round 4 — Codex resumed session, verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/review4/claudex-6hak2sz3/result.json
- M2-4 confirmed addressed.
- M2-5 (medium) CanStep dropped diagonal constraints → ACCEPTED, full edge rule + equivalence test.

## Round 5 — Codex resumed session, verdict APPROVED (plan sha256 f578077d…)
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/review5/claudex-b_hkwtqz/result.json
- No findings. Limitations: plan-only review, no tests run by reviewer.

## Build
- Builder: Claude (host). Pre-build commit: see commit after 'approved plan'.
- Pre-build commit: b1dbfe2002e2a84e3c16a79992e45cb1fd3a673c. Builder: Claude (host); all code authored by Claude.
- Host proof: gofmt clean, go vet clean, go test ./... pass (new: TestPlatformHouse, TestPlatformStack, TestStairsPath, TestStairRemovalReplans, TestCanStepMatchesOldRule, TestBridgeCross, TestDemolishSupport, TestFieldWorkSkipsSupports, TestVerticalSaveRoundTrip); build.sh ok; screenshots checked (terrace+house+stairs, bridge, stairs ghost with error).
- Deviation: criterion D "FindPath fails before bridges" — the generated river has fords elsewhere, so the test instead checks deep tiles are impassable before and passable after, and that the bridge path is direct and shorter than the detour.

## Inspection 1 — fresh Codex session, verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/inspect1/claudex-f43qmdw3/result.json
- M2-1 (medium) "dams can be placed on platforms / level-0 via WaterOK" → REJECTED with evidence: elevated placement of Dam/Farm/Pump returns ErrGroundOnly (build.go, GroundOnly switch after the footprint loop; TestPlatformStack asserts it). Dam on level-0 water is the pre-existing intended rule (old CanPlaceAt skipped water/h==0 checks when inWater=true; dams are meant to cross the river). Added TestDamPlacementUnchanged to pin both. No production code changed.

## Inspection 2 — fresh Codex session, verdict REVISE
Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex/inspect2/claudex-yg62vwur/result.json
- M2-1 rejection not contested.
- M2-R1 (medium) PlanPlacement indexes Builds before range check → ACCEPTED, fixed (range check first) + TestPlanPlacementBadKind.
- Inspection budget (2) exhausted: the M2-R1 fix (4 lines in build.go + 1 test) is NOT independently inspected. Host re-ran full proof after the fix: gofmt/vet clean, go test ./... pass, build.sh ok.
