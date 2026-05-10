# Novel Features Brainstorm — find-me-gluten-free-pp-cli

## Customer model

**Persona 1: Maya, the celiac trip-planner**
*Today (without this CLI):* Maya has Celiac disease and is planning a 9-day trip across Lisbon, Porto, and Madrid. She has 14 browser tabs open: Find Me Gluten Free city pages for each stop, three Google Maps tabs, the FMGF biz page for every restaurant she's vetting, and a half-finished Google Doc where she pastes restaurant names + addresses + "dedicated?" + a star rating she copies by hand. She cannot easily answer "which of my three cities has the most dedicated-GF places within walking distance of the old town?" without clicking through each city, sorting by hand, and counting.
*Weekly ritual:* Every Sunday evening for the month before a trip, she rebuilds her shortlist as new reviews come in and as her itinerary shifts. She also re-checks places she'd already saved in case a 1-star "got glutened here" review just dropped.
*Frustration:* Comparing cities side-by-side is impossible in the consumer app — it's built for "near me right now," not for an itinerary three weeks out.

**Persona 2: Devin, the AI travel-agent operator**
*Today (without this CLI):* Devin runs a Claude-based travel-planning agent for celiac clients. Today the agent has to scrape FMGF biz pages live, hit Cloudflare/ToS friction, and parse JSON-LD on each call. There's no structured "give me dedicated GF Italian in Edinburgh sorted by rating" query — the agent ends up doing 30 sequential page fetches per itinerary stop and times out.
*Weekly ritual:* Every client trip kicks off the same shaped query: 2-4 cities, filter by dedicated + cuisine, rank by rating + review count, surface anything new since last sync. Devin runs this 3-5 times per week.
*Frustration:* No agent-native, structured, rate-safe surface exists. Every agent run is brittle and slow.

**Persona 3: Priya, the parent of a celiac kid**
*Today (without this CLI):* Priya's 8-year-old has Celiac. She has a personal allowlist of ~40 restaurants across 6 cities they visit family in. When they re-visit Grandma in Pittsburgh, she manually re-opens every Pittsburgh bookmark in FMGF to see if anything changed — closures, new bad reviews, a place going from "GF menu" to "dedicated."
*Weekly ritual:* Before any family trip (~monthly), she revisits her saved set for that city and reads the most recent reviews to make sure her son is still safe there.
*Frustration:* The app doesn't tell her *what changed* since she last looked. She re-reads everything from scratch.

## Candidates (pre-cut)

| # | Name | Command | Description | Persona | Source | Inline verdict |
|---|------|---------|-------------|---------|--------|----------------|
| C1 | Multi-city compare | `trip compare <city1> <city2> [<cityN>]` | Side-by-side counts of dedicated, has-GF-menu, avg rating, top-rated biz per city | Maya | (a)(b) | KEEP |
| C2 | City diff since last sync | `cities diff <city> --since <date\|last-sync>` | What's new, removed, or changed since stored snapshot | Priya, Maya | (b)(c) | KEEP |
| C3 | Near-here Haversine | `places near --lat <x> --lng <y>` | Lat/lng radius search over local cache | Maya | (b)(c) | KEEP |
| C4 | Itinerary builder | `trip plan --city <c> --days <n>` | Pick top-rated dedicated/menu places per day | Maya | (a) | RISK |
| C5 | Aggregate rollup | `places stats --city <c>` | count, avg rating, p50/p90, dedicated %, GF-menu % | Devin | (c) | KEEP |
| C6 | Recent-review feed | `reviews recent --city <c>` | City-wide newest reviews sorted by datePublished | Priya, Maya | (b)(c) | KEEP |
| C7 | Watchlist + change report | `watch add/report` | Per-biz monitor for rating drift, new low-stars | Priya, Devin | (a)(b) | KEEP |
| C8 | Cuisine cross-city | `cuisines compare <cuisine> <c1> <c2>` | Counts and top biz per city for one cuisine | Maya | (b)(c) | KEEP |
| C9 | Bulk hydrate | `places hydrate --ids-file` | Batch fetch + cache for input list | Devin | (a) | RISK |
| C10 | Safety-report extractor | `reviews safety <biz-id>` | Filter to safety-report keywords (no LLM) | Maya, Priya | (b) | KEEP |
| C11 | Walkability cluster | `places cluster --city <c>` | Geo clusters within radius | Maya | (c) | KILL — re-presents C3 |
| C12 | Top-N most-improved cities | `cities trending` | Largest dedicated-GF count delta between snapshots | Devin | (b)(c) | KILL — needs many snapshots; weekly users have 1-2 |
| C13 | Photo URL extractor | `places photos <biz-id>` | Print all `image` URLs from JSON-LD | Maya | (b) | KILL — wrapper |
| C14 | Auto-suggest cuisines | `cuisines suggest --city <c>` | List cuisines actually present in city's filter pages | Devin | (b) | KILL — duplicates absorbed cuisines list |
| C15 | GPX/KML export | `trip export <plan> --format gpx` | Emit GPX/KML for import into a map app | Maya | (a) | RISK |
| C16 | Dedicated-density rank | `cities rank --by dedicated-density` | Within country, rank by dedicated-GF count per total biz | Maya | (c) | KEEP |

## Survivors and kills

### Survivors

| # | Feature | Command | Score | How It Works | Evidence |
|---|---------|---------|-------|--------------|----------|
| 1 | Multi-city trip compare | `trip compare <city1> <city2> [<cityN>]` | 10/10 | Local SQLite join; per-city dedicated/GF-menu/total counts, avg rating, top-rated biz | Brief Workflow #4; FMGF/Atly/Gluten Dude lack multi-city compare; OSS scraper datatodavid targets the same need |
| 2 | City diff since last sync | `cities diff <city> --since <date\|last-sync>` | 10/10 | Compares current snapshot vs prior snapshot using sitemap `<lastmod>` per biz | Brief Workflow #5 ("entirely missing from consumer apps"); brief Data Layer documents sitemap `<lastmod>` |
| 3 | Lat/lng radius search | `places near --lat <x> --lng <y> [--radius-km <r>] [--dedicated]` | 9/10 | Haversine over `location.geo` parsed from JSON-LD; combines with all filters | Brief Workflow #1; Data Layer documents `location.geo` |
| 4 | City stats rollup | `places stats --city <c> [--cuisine <c>] [--dedicated]` | 8/10 | SQL aggregate: count, avg rating, p50/p90, dedicated %, GF-menu % | Brief Build Priorities lists "aggregate ratings rollup"; FMGF has no stats |
| 5 | Recent reviews feed | `reviews recent --city <c> [--since <date>] [--max-rating <r>]` | 10/10 | Sorts cached reviews for all biz in city by `datePublished`; `--max-rating` surfaces safety-report-style 1-2 star reviews | Brief Workflow #3; Data Layer documents `review[].datePublished` |
| 6 | Watchlist with change report | `watch add <biz-id>` / `watch list` / `watch report` | 9/10 | Per-biz baseline (rating, review count, sitemap `<lastmod>`); report re-fetches each watched biz, reports rating drift + new low reviews | Brief Workflow #5 + Priya gap |
| 7 | Cuisine cross-city compare | `cuisines compare <cuisine> <c1> <c2> [<cN>]` | 9/10 | Joins 214-filter taxonomy against `business`; per city emits count, dedicated %, top-rated for cuisine | Brief 214-filter taxonomy + Workflows #2 and #4 |
| 8 | Safety-report review filter | `reviews safety <biz-id>` | 8/10 | Filters reviews against curated keyword set ("glutened", "cross-contam", etc.) marked `// pp:novel-static-reference`; deterministic, no LLM | Brief Workflow #3 explicit "safety reports often called out" |

### Killed candidates

| Feature | Kill reason | Closest surviving sibling |
|---------|-------------|---------------------------|
| C4 Itinerary builder | No oracle for "good itinerary" — verifiability fails; mechanical version collapses to chained `places list --sort rating` | C1 multi-city compare |
| C9 Bulk biz hydrate | Thin wrapper around `places get`; flirts with bulk-export ToS line the brief explicitly forbids; agents can shell-loop | (none — wrapper) |
| C11 Walkability cluster | Re-presentation of C3; CLIs aren't UIs | C3 lat/lng radius search |
| C12 Top-N most-improved cities | Needs many snapshots; weekly users have 1-2 | C2 city diff |
| C13 Photo URL extractor | Thin wrapper around one JSON-LD field | absorbed `places get` |
| C14 Auto-suggest cuisines | Duplicates absorbed `cuisines list --city` | absorbed manifest #15 |
| C15 GPX/KML export | Once-per-trip, not weekly; user can pipe C3's JSON | C3 lat/lng radius search |
| C16 Dedicated-GF density rank | Overlaps C8 and C1 — weakest weekly-use of comparative trio | C8 cuisine cross-city compare |
