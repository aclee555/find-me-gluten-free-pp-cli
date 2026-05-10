# Find Me Gluten Free CLI Absorb Manifest

## Tools surveyed
- **Find Me Gluten Free** (the source itself) — web + iOS + Android consumer app; canonical feature set
- **Gluten Dude** — competing app with restaurant vetting protocol, Trip Planner, coupons
- **Atly** — community-based GF restaurant finder
- **Gluten Adviser** — Europe-focused GF finder
- **datatodavid/GF_Restaurant_Scraper** (GitHub, Scrapy) — only OSS prior art; goal is US city comparison
- **No SDKs**, **no Claude plugin**, **no MCP server** — greenfield agent surface

## Absorbed (match or beat everything that exists)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---|---|---|---|
| 1 | Search restaurants in a city | Find Me Gluten Free | `places list --city <slug>` | Offline cache, --json, --select, `--country`/`--state` for disambiguation |
| 2 | Filter by cuisine in city | Find Me Gluten Free (214 filter pages per city) | `places list --city <c> --cuisine <c>` | Combinable filters, agent-native output |
| 3 | Dedicated-GF-only filter | Find Me Gluten Free, Gluten Dude | `--dedicated` flag | Combinable with --has-gf-menu, --min-rating |
| 4 | Has-GF-menu filter | Find Me Gluten Free, Gluten Dude | `--has-gf-menu` flag | Combinable |
| 5 | Get business detail (name, address, phone, hours, price, lat/lng, rating) | Find Me Gluten Free | `places get <slug-or-id>` | JSON-LD parsed; cached locally |
| 6 | Read reviews | Find Me Gluten Free, Gluten Dude | `places reviews <id>` | All embedded reviews (not lazy-loaded) |
| 7 | Aggregate rating + review count | Find Me Gluten Free | included in `places get` | Surfaced in --json for agents |
| 8 | Bookmarks (saved restaurants) | Find Me Gluten Free Premium, Gluten Dude | `bookmark add/remove/list` | Local-only — no premium subscription |
| 9 | Saved cities | Find Me Gluten Free Premium | `cities save/unsave/list` | Local |
| 10 | Postal code lookup | Find Me Gluten Free (/postal redirect) | `places near --zip <zip>` | Resolves redirect to city page |
| 11 | Country directory | Find Me Gluten Free (/countries) | `countries list` | Offline |
| 12 | State directory | Find Me Gluten Free (/{country}) | `states list --country <c>` | Offline |
| 13 | Most-GF-friendly cities ranking | Find Me Gluten Free (/most-gluten-free-friendly-cities) | `cities ranked` | Offline cache of the published ranking |
| 14 | Chain restaurant index | Find Me Gluten Free (/chains) | `chains list` | Local FTS |
| 15 | Cuisine filter index | Find Me Gluten Free (214 filters per city) | `cuisines list --city <c>` | Discoverable filter set |

Every row is shipping-scope. No stubs.

## Transcendence (only possible with our approach)

| # | Feature | Command | Score | Why Only We Can Do This |
|---|---|---|---|---|
| 1 | Multi-city trip compare | `trip compare <city1> <city2> [<cityN>]` | 10/10 | Cross-city local SQL join; consumer apps have no compare view |
| 2 | City diff since last sync | `cities diff <city> --since <date\|last-sync>` | 10/10 | Snapshot-vs-snapshot using sitemap `<lastmod>`; consumer apps don't show what changed |
| 3 | Lat/lng radius search | `places near --lat <x> --lng <y> [--radius-km <r>] [--dedicated]` | 9/10 | Haversine over JSON-LD `location.geo`; site is city-bucketed only |
| 4 | City stats rollup | `places stats --city <c> [--cuisine <c>] [--dedicated]` | 8/10 | SQL aggregate (count, avg rating, p50/p90, dedicated %, GF-menu %); FMGF has no stats endpoint |
| 5 | City-wide recent reviews | `reviews recent --city <c> [--since <date>] [--max-rating <r>]` | 10/10 | Cross-biz review join sorted by `datePublished`; FMGF buries reviews per-biz |
| 6 | Watchlist with change report | `watch add <biz-id>` / `watch list` / `watch report` | 9/10 | Per-biz snapshot diff (rating drift, new low reviews) — entirely missing from consumer apps |
| 7 | Cuisine cross-city compare | `cuisines compare <cuisine> <c1> <c2> [<cN>]` | 9/10 | Joins the 214-filter taxonomy across cities — site only renders one cuisine in one city at a time |
| 8 | Safety-report review filter | `reviews safety <biz-id>` | 8/10 | Deterministic keyword filter ("glutened", "cross-contam", etc., curated list, `// pp:novel-static-reference`); no LLM; FMGF has no safety filter |

All 8 ≥ 5/10. All shipping-scope. Audit trail in `2026-05-09-152605-novel-features-brainstorm.md`.

## Total scope

15 absorbed + 8 novel = **23 features** to implement.

## Etiquette defaults (mandatory in shipping CLI)

- `User-Agent: find-me-gluten-free-pp-cli/<version> (+printing-press)` — honest, identifies the tool, no Chrome impersonation
- 1 req/sec default rate limit (`FMGF_RPS=N` to override; never above 5)
- No bulk-state-export command; explicit per-list `places hydrate` not in scope (was killed in cut)
- README "Etiquette" section discloses ToS and the AI-bot robots.txt rule explicitly
