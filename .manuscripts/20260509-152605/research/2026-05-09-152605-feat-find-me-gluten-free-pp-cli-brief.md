# Find Me Gluten Free CLI Brief

## API Identity
- **Domain**: gluten-free / celiac-safe restaurant directory; community-curated
- **Users**: people with celiac disease or gluten sensitivity (~1% of US adults), traveling celiac-safe diners, and parents/companions feeding them
- **Data profile**: business listings with location, ratings, community reviews, dedicated-GF flag, GF-menu flag, GiG (Gluten Intolerance Group) training flag, photos, phone, hours, price range
- **Coverage**: ~50 US states + international (UK, Ireland, France, etc.). 214 cuisine/feature filters per city. Big enough sitemap to fan out to dozens of millions of business pages over time.

## Reachability Risk
- **Probe**: stdlib HTTP returns 200 on home, city, and biz pages. `printing-press probe-reachability` → `mode: standard_http`. No Cloudflare challenge, no bot wall on public surfaces. Runtime is plain stdlib HTTP.
- **ToS / robots.txt etiquette concern (must disclose)**: site ToS prohibits crawling, scraping, or spidering ([tosdr case 150](https://edit.tosdr.org/cases/150)); `robots.txt` adds a specific extra block list for AI agents (ClaudeBot, GPTBot, Google-Extended, etc.) covering `/biz`, `/posts`, `/postal`. The printed CLI is a *user-driven* tool — a single human running point lookups — not a crawler. Defaults must reflect that:
  - Honest `User-Agent: find-me-gluten-free-pp-cli/<version> (+https://github.com/.../find-me-gluten-free)` (no Chrome-impersonation)
  - 1 req/sec default rate limit, configurable via env var
  - No bulk export / state-wide harvest command
  - README "Etiquette" section disclosing ToS + robots.txt explicitly
- **Existing prior art**: only one open-source scraper found (`datatodavid/GF_Restaurant_Scraper`, Scrapy, US-cities-comparison aim). No SDK wrappers, no Claude plugin, no MCP server. Greenfield.

## Top Workflows
1. **"Find dedicated GF places near me"** — given city or zip, return only fully-dedicated facilities (highest celiac-safety bar)
2. **"Find GF [cuisine] in [city]"** — e.g. "GF burgers in Richmond"; the cuisine filter sub-pages cover 214 filters per city
3. **"Read reviews for [restaurant] before going"** — biz detail pulls JSON-LD reviews including author, rating, dated description; safety reports often called out
4. **"Trip planning"** — compare 3 candidate cities, see which has the most/best dedicated GF, build a route
5. **"Track changes"** — return to a saved city later, see what's new since last visit (entirely missing from the consumer apps)

## Table Stakes (Find Me Gluten Free + Gluten Dude + Atly + Gluten Adviser)
- Search restaurants near location or city
- Filter: dedicated-GF / has-GF-menu / cuisine / chain
- Read user reviews + safety reports
- Phone, address, geo coordinates, directions
- Bookmarks / favorites / saved spots
- Photos
- Map view (n/a for CLI; replace with lat/lng output)
- Trip Planner (Gluten Dude has this)
- Coupons (Gluten Dude has this; FMGF doesn't)
- Vetted-protocol flags (Gluten Dude vets restaurants directly)

## Data Layer
- **Primary entities**: `Business`, `Review`, `City`, `Country`, `Filter` (cuisine/feature)
- **Sync cursors**:
  - Sitemap-driven URL discovery via `/sitemapindex` + `/sitemaps/{N}` (each carries `<lastmod>`)
  - Per-biz page refresh on demand
  - Per-city page refresh on demand (city's own `<lastmod>` from sitemap)
- **FTS**: business name, address, city, country, review text (combined index), feature flags
- **Bookmarks**: local SQLite-only; never sent upstream

## Codebase Intelligence
- **Source**: site is undocumented. No public API. No SDK.
- **Auth**: none for read; site session cookies (JSESSIONID) drive logged-in features (saved spots, profile) — out of scope for v1
- **Data shape evidence**:
  - Biz pages serve `<script type="application/ld+json">` with full schema.org Restaurant: `name`, `image`, `priceRange`, `telephone`, `location.geo.{latitude,longitude}`, `aggregateRating.{ratingValue,ratingCount}`, `review[].{reviewRating.ratingValue,datePublished,description,author.name}`
  - City pages serve plain HTML lists of `/biz/{slug}/{id}` links
  - Filter pages share the city-page shape with a narrowed list
  - `/postal/{zip}` 301-redirects to the appropriate city page
- **Architecture**: Java/Jetty backend (HTTP `Server: cloudflare`, `JSESSIONID` cookie style); SSR; Cloudflare CDN

## User Vision
- Briefing: "Let's go" — no specific feature requests; user is letting the absorb manifest set scope. They picked "the website itself" knowing this is a website-derived CLI.

## Product Thesis
- **Name**: `find-me-gluten-free-pp-cli`
- **Display name**: Find Me Gluten Free
- **Why it should exist**: the iOS/Android app is great for "what's near me right now" but useless for *trip planning, multi-city comparison, change-tracking, or feeding GF restaurant data to an AI agent*. A locally-cached gluten-free restaurant database with offline FTS + agent-native filters is a structurally different product from a consumer app — it serves the celiac traveler who plans an itinerary days or weeks before the trip, and the AI agent helping a celiac plan a vacation.

## Build Priorities
1. **Foundation**: Business + City + Review + Filter entities; sync from sitemap + city pages + biz pages; SQLite FTS5; standard HTTP transport with polite rate-limiter and honest User-Agent
2. **Absorb (parity with FMGF + GD + Atly)**: list countries, list states, list cities, list restaurants in city, filter by cuisine/feature, get biz detail (incl. reviews + lat/lng + rating), read user reviews, postal-code lookup, dedicated-only filter, has-GF-menu filter, bookmarks (local), saved cities (local)
3. **Transcend**: trip planner (multi-city itinerary comparison), city-diff (what changed since last sync), aggregate ratings rollup ("dedicated GF places with avg ≥4.5 in `<city>`"), agent-native query DSL via SQL/search, "near here" by lat/lng with Haversine, batch-bookmark from a list of restaurant IDs, change feed from sitemap `<lastmod>`
