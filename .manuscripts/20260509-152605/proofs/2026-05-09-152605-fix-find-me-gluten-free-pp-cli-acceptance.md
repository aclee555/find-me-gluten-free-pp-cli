# find-me-gluten-free-pp-cli — Live Dogfood Acceptance

## Summary
- **Level**: Full Dogfood (binary-owned matrix)
- **Status**: PASS
- **Matrix size**: 105 tests
- **Passed**: 105
- **Failed**: 0
- **Auth context**: none required (read-only public CLI)

## Live behavior verified

Beyond the matrix, the agent ran an end-to-end live exercise during Phase 3:
- `places hydrate pinkys 6118524171452416` returned a parsed JSON-LD payload with rating 5/5, 78 reviews, lat 37.568386, lng -77.469025, has_gf_menu true
- `places detail 6118524171452416` loaded that record from the local SQLite cache
- `places stats --city richmond` produced the full aggregate (count, dedicated %, avg, p50, p90, total reviews, highest rated)
- `places near --lat 37.568 --lng -77.469 --radius-km 10` returned Pinky's at 0.04 km
- `bookmark add` + `bookmark list` round-tripped (after fixing the resource-id namespacing bug)
- `watch add` + `watch list` round-tripped
- `reviews recent --city richmond --max-rating 4` returned three real 4-star reviews
- `reviews safety <bizID>` ran the deterministic keyword filter

## Fixes applied during the dogfood loop

The first matrix run failed 11/91. The second loop reduced this to 5/91. The third loop hit 0/105.

1. **Missing `Example:` fields on 10 novel sub-commands** (bookmark add/list/remove, watch add/list/remove/report, cities save/saved/unsave). Each got a realistic invocation with concrete biz_ids and city slugs.
2. **`places postal __printing_press_invalid__` did not error.** Added `validZip` pre-check (3-12 chars, alnum + `-` + space) plus a redirect-to-`/` rejection so invalid postals exit non-zero with a clear message.
3. **`bookmark add` / `watch add` accepted any string as biz_id.** Added `validBizID` (10+ digits) so invalid IDs error before hitting the store.
4. **`cities save` accepted any string as a city slug.** Added `validCitySlug` (lowercase letters, digits, hyphens; 2-80 chars).
5. **`bookmark remove` / `watch remove` / `cities unsave` silently succeeded when no row matched.** Each now checks `RowsAffected()` and returns `notFoundErr` when nothing was deleted.

## What this validates

- **Foundation holds.** JSON-LD parsing (with the literal-newline normalizer and HTML-entity decode) works against the live site's real biz pages.
- **Storage namespacing fix is correct.** Bookmarks, watches, saved cities, and snapshots all coexist with biz rows on the same biz_id without collision.
- **All 8 transcendence commands are wired and operational.** trip compare, cities diff, places near, places stats, reviews recent, watch report, cuisines compare, reviews safety.
- **Etiquette defaults are honored.** Honest User-Agent on every request, default rate limit 2 RPS (lowerable), no bulk-export commands.
- **Error paths are well-shaped.** Invalid input → exit 2 with usage error; missing data → exit 3 with not-found error; rate-limit → exit 7. The dogfood matrix's error_path tests all returned the expected non-zero exits with helpful messages.

## Gate: PASS

All ship-threshold conditions met. Proceeding to Phase 5.5 polish.
