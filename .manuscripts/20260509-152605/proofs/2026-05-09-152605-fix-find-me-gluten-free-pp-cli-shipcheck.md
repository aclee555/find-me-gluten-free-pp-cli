# find-me-gluten-free-pp-cli — Shipcheck Proof

## Summary
Generation + Phase 3 hand-build + 1 fix loop reached PASS on all 6 shipcheck legs. Final verdict: **ship**.

## Final shipcheck output

```
Shipcheck Summary
=================
  LEG               RESULT  EXIT      ELAPSED
  dogfood           PASS    0         1.78s
  verify            PASS    0         3.922s
  workflow-verify   PASS    0         13ms
  verify-skill      PASS    0         100ms
  validate-narrative  PASS    0         196ms
  scorecard         PASS    0         35ms

Verdict: PASS (6/6 legs passed)
```

## Scorecard
- **Total: 81/100 — Grade A**
- Output Modes 10/10, Auth 10/10, Error Handling 10/10, Doctor 10/10, Agent Native 10/10, MCP Quality 10/10, Local Cache 10/10
- README 8/10, Vision 8/10, Agent Workflow 9/10, Terminal UX 9/10
- Gaps (not blockers): mcp_token_efficiency 4/10, mcp_remote_transport 5/10, mcp_tool_design 5/10, insight 2/10, breadth 7/10, workflows 6/10, cache_freshness 5/10
- Type Fidelity 3/5, Dead Code 5/5, Path Validity 10/10, Sync Correctness 10/10

## What was fixed in the loop

Initial run had 2 leg failures:

1. **verify-skill — `places stats --cuisine` flag missing.** The narrative example used `places stats --city austin --cuisine pizza --json` but the command only declared `--city` and `--dedicated`. **Fix:** added `--cuisine` flag to `places stats` plus a `filterByCuisineHint` helper that does a substring match on biz name + OG description (the cuisine taxonomy lives in filter sub-pages we don't always sync, so substring is the cheap fallback).

2. **validate-narrative — three quickstart examples used flags that don't exist.** `sync --city portland`, `places list --city portland --dedicated --min-rating 4`, and `cities diff portland --since last-sync` referenced flags that the actual generated `sync` and `places list` commands don't have, and the original `cities diff` didn't accept `--since`. **Fix:** (a) added `--since` flag to `cities diff` (`last-sync` default, or YYYY-MM-DD), (b) rewrote `narrative.quickstart` and `narrative.recipes` in `research.json` to use the actual command shapes the CLI ships (`cities sync us va richmond`, `places stats --city richmond --json`, `trip compare richmond charlottesville norfolk --json`, etc.).

Plus one bug found during smoke testing (pre-shipcheck):

3. **JSON-LD literal newlines.** The Find Me Gluten Free backend emits review `description` fields with raw `\n\n` characters embedded — strict JSON forbids this, so `json.Unmarshal` returned `invalid character '\n' in string literal`. **Fix:** added `normalizeJSONStringLiterals` in `internal/jsonld/jsonld.go` — a small string-state machine that escapes literal control chars (`\n`, `\r`, `\t`) inside string values before decoding. Test fixture added.

4. **Generator schema bug — `resources.id` is PRIMARY KEY rather than `(resource_type, id)`.** A bookmark for biz_id 6118524171452416 silently overwrote the biz row because `ON CONFLICT(id) DO UPDATE` matched on id alone, without scoping by resource_type. **Workaround in this CLI:** namespace IDs at the store boundary (`storeID(resType, id)` returns `<resType>:<id>`). **Retro candidate** — the generator's resources schema should use `(resource_type, id)` as the primary key so CLIs that legitimately have overlapping natural IDs across resource types don't silently corrupt data.

## What this CLI ships

15 absorbed features (countries, states, cities, places list/filter/get, chains list, ranked cities, cuisines list — all generated from the spec) + 8 transcendence features built by hand:

1. `trip compare <city...>` — multi-city celiac-friendliness comparison
2. `cities diff <city> --since <ts|last-sync>` — snapshot diff
3. `places near --lat --lng --radius-km` — Haversine over local cache
4. `places stats --city --cuisine --dedicated` — SQL aggregate
5. `reviews recent --city --since --max-rating` — city-wide review feed
6. `watch add/list/remove/report` — watchlist with rating drift
7. `cuisines compare <cuisine> <country> <state> <cities...>` — cuisine cross-city
8. `reviews safety <biz_id>` — keyword-filtered safety reports

Plus quality-of-life: `places hydrate`, `places detail`, `places postal`, `cities sync` (foundation for the transcendence features), `bookmark add/list/remove`, `cities save/unsave/saved`.

Foundation packages: `internal/jsonld/` (JSON-LD parser with HTML-entity decode + literal-newline normalizer), `internal/safety/` (curated safety keyword list, deterministic).

## Ship recommendation: `ship`

All ship-threshold conditions met:
- shipcheck exits 0, all 6 legs PASS
- scorecard 81 ≥ 65
- no functional bugs in shipping-scope features (verified via end-to-end live exercise: hydrate, detail, stats, near, bookmark, watch, reviews recent all return correctly shaped data)
- workflow-verify is workflow-pass (no manifest needed for read-only CLI)
- verify-skill clean, validate-narrative clean

Remaining MCP gaps (token efficiency 4/10, remote transport 5/10, tool design 5/10, insight 2/10) are polish-phase work, not shipping blockers. Phase 5.5 will offer to close them.
