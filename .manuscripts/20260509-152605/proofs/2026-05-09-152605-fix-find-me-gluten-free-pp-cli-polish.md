# find-me-gluten-free-pp-cli — Polish Pass Result

## Polish delta

```
                       Before    After
  Scorecard:           81/100    82/100  (+1)
  Verify:              100%      100%
  Dogfood:             PASS      PASS
  go vet:              clean     clean
  Tools-audit:         3 pending 0 pending  (-3)
  Workflow-verify:     pass      pass
  Verify-skill:        0 errors  0 errors
  Publish-validate:    FAIL      FAIL  (env-blocked)
```

## Fixes applied by polish

- Rewrote the thin Cobra `Short` on `bookmark remove` for agent-grade specificity ("Remove a bookmarked place by biz_id from the local bookmark store")
- Rewrote the thin Cobra `Short` on `cities saved` similarly ("List cities saved locally for quick recall, with their slug, save time, and optional notes")
- Ran `printing-press mcp-sync` to regenerate the `tools-manifest.json` that was missing
- Accepted the `missing-read-only` finding on `watch report` with a per-item rationale: the command body calls `hydrateOne` (HTTP fetch) and `persistBiz` (local store write), so `mcp:read-only: false` is correct — the heuristic flagged on the read-shaped name `report`, but the explicit `false` annotation overrides

## Skipped findings (out of scope)

- **Scorecard MCP dimensions** (`mcp_token_efficiency` 4/10, `mcp_remote_transport` 5/10, `mcp_tool_design` 5/10): require spec edits (`mcp.transport: [stdio, http]`, `mcp.intents`, etc.) and a regen — out of scope for in-flight working-copy polish. With only 8 typed endpoint tools, the >50-tool thresholds for endpoint_tools/orchestration don't apply; only `transport: [stdio, http]` would be a zero-cost win and the user can add it via a follow-up regen.
- **Scorecard `insight` 2/10, `cache_freshness` 5/10, `workflows` 6/10**: structural feature gaps. Adding a workflow manifest, freshness probes, or insight commands is feature work, not polish.
- **Scorecard `--live-check` returned `unable: true`** and the printing-press-output-review sub-skill SKIPped: `research.json` lives at the run-dir root, not at the CLI-dir root that live-check inspects. Environmental — main pipeline knows where research lives. Both are retro candidates.

## Remaining issues that block publish (NOT shipping quality)

These are the two issues the polish skill cited as reasons to downgrade ship_recommendation to `hold`. Neither indicates a CLI defect:

1. **Manifest missing `printer` and `printer_name`.** The user's git config has neither `user.name` nor `github.user` set, so the generator emitted the CLI without printer attribution. Per AGENTS.md the publish step refuses empty/sentinel printer values. Fix:
   ```bash
   git config --global user.name "<Your Name>"
   git config --global github.user "<your-handle>"
   # then regenerate:
   printing-press generate --spec ... --output ./working/find-me-gluten-free-pp-cli --force
   ```
2. **`phase5-acceptance.json` path mismatch (RESOLVED post-polish).** Polish ran `publish validate` before the parent pipeline copied `phase5-acceptance.json` into `<CLI_DIR>/.manuscripts/<run-id>/proofs/`. After the parent pipeline staged the manuscripts there, this gate now PASSES. Retro candidate: the polish skill's publish-validate run order should account for the parent's manuscript-archiving step.

## Verdict (per SKILL.md verdict-override rule)

Polish set `ship_recommendation: hold` because publish-validate failed. The SKILL's Phase 5.5 says: "If the polish skill's ship_recommendation is hold and the Phase 4 verdict was ship, downgrade to hold."

Final verdict: **hold**.

**This does NOT mean the CLI is broken.** The CLI passed all 6 shipcheck legs, hit 105/105 on full live dogfood, and works end-to-end against the live site. The hold reason is purely about publishing readiness (missing git identity in the local environment).

The user has two paths from here, both presented in the Phase 6 hold-path menu:
- **Run retro** — capture the systemic findings (resources-table PK, polish/parent pipeline ordering, scorecard live-check pathing) for the Printing Press maintainers
- **Continue manually** — set git config, re-run `printing-press generate`, run `printing-press lock promote` to copy to library, then `printing-press publish` if desired
