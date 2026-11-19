# Plan review log — M3 detailed building models

- Host/coordinator: Claude Code. Plan reviewer: Codex. Builder: Claude (host). Final inspector: fresh Codex session.
- Models: CLI defaults. Codex CLI codex-cli 0.170.0.
- Authorization: user asked to plan and implement ("Давай"); commit/push per session branch rules.
- Limits: MAX_ROUNDS=5, MAX_FIX_ROUNDS=2, MAX_INSPECTION_ROUNDS=2. Inspection on.

## Round 1 — Codex, REVISE. Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex3/review1/claudex-nnlyutve/result.json
- M3-1 (medium) ghost smoke via Residents(0) → ACCEPTED (decision 5, criterion C, TestSmokeRule).
- M3-2 (medium) roof-centre depth compare insufficient → ACCEPTED (decision 3: ridge-mounted parts above roof max height, always after roof; extra pitch shots).

## Round 2 — Codex, APPROVED. Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex3/review2/claudex-hth40jgx/result.json

## Build
- Pre-build commit 461cb9db5ca03f0e193c6ba52ff3c5625bda0d24. Builder: Claude (host), all code by Claude.
- Proof: gofmt/vet clean; go test ./... pass (new view tests TestHouseStyleStable, TestSmokeRule, TestDecalFacing); build.sh ok.
- Perf (demo town, -debug-demo 600 -debug-vertical): default camera tris 18262 → 18568 (+1.7%); scale 70: 9229 → 9535. Budget ≤ +60% met.
- Visual: yaw 0.55/2.1/3.7/5.2 and pitch 0.5/1.2 screenshots inspected; no decal/roof-part see-through observed.
- Refinement beyond plan text: props beside a building (porch, crates, log piles, stump, pots, basket, bucket, shed, seats) are drawn before the body when their side faces away from the camera (propFirst), after it otherwise, so they never draw over the body from behind.
- Hall smoke rises from the tower spire top (the hall's roof-axis part), not a separate chimney.

## Inspection 1 — fresh Codex, REVISE. Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex3/inspect1/claudex-j29mr9tk/result.json
- M3-R1 (low) withdrawn by the reviewer in its own evidence ("not a defect; omit").
- M3-R2 (medium) sawmill log pile covered by body → REJECTED: in models.go the log boxes are emitted after the body Box (body loop precedes `for k := 0; k < min(3, bd.InN)`), and they lie on the body's top face (z base+0.35..0.45 above body top base+0.35), so the body cannot cover them. No code change.

## Inspection 2 — fresh Codex, APPROVED (same snapshot, no code changes since inspection 1). Result: /tmp/claude-0/-home-user-zvit/8be25a58-6c00-516c-93d1-58bd5ed4e097/scratchpad/claudex3/inspect2/claudex-i54thg49/result.json
