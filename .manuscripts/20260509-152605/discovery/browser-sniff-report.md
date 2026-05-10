# Browser-Sniff Discovery Report — find-me-gluten-free

## Goal
Primary user goal: "Find dedicated GF places near me, plus filter by cuisine in a city."
Secondary: "Read reviews and aggregate ratings before going."

## Capture method
The site is fully server-rendered HTML. `printing-press probe-reachability` returned `mode: standard_http` (confidence 0.95) on home, city (`/us/va/richmond`), and biz (`/biz/pinkys/6118524171452416`). Direct `curl` returns full HTML on every page tested, including schema.org JSON-LD on biz pages — there are no client-rendered XHR endpoints required for the primary user goal.

Capture method was therefore "manual HAR via curl" (browser-sniff-capture.md fallback option), augmented by sitemap inspection. No browser automation was needed; spawning browser-use/agent-browser/Chrome would not have surfaced additional endpoints.

## Endpoint surface (replayable HTTP, all GET)

### `/sitemapindex`
- Returns XML index pointing to ~16 sitemaps under `/sitemaps/{N}`.
- `<lastmod>` per sitemap, refreshed nightly.
- Use: discover the URL space; drive change detection.

### `/sitemaps/{N}`
- Returns XML with `<url><loc>` + `<lastmod>` per page (city, biz, etc.).
- Use: enumerate cities and businesses; per-URL `<lastmod>` powers the change-tracking transcendence features.

### `/{country}` (e.g., `/us`)
- Returns HTML; embeds `<a href="/{country}/{state}">` links.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/{country}/"]`.

### `/{country}/{state}` (e.g., `/us/va`)
- Returns HTML; embeds `<a href="/{country}/{state}/{city}">` links.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/{country}/{state}/"]`.

### `/{country}/{state}/{city}` (e.g., `/us/va/richmond`)
- Returns HTML; lists `/biz/{slug}/{id}` hrefs and 214 cuisine/feature filter sub-page hrefs.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/biz/"]` for restaurant list.

### `/{country}/{state}/{city}/{filter}` (e.g., `/us/va/richmond/dedicated-facilities`, `/us/va/richmond/burgers`)
- Returns HTML; same shape as city page, narrowed by filter.
- 214 filters per city: cuisine (e.g., `burgers`, `apple-pie`, `bagels`) plus `gluten-free-menu` and `dedicated-facilities`.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/biz/"]`.

### `/biz/{slug}/{id}`
- Returns HTML; embeds `<script type="application/ld+json">` with schema.org Restaurant:
  - `name`, `image`, `priceRange`, `telephone`
  - `location.geo.{latitude, longitude}`
  - `aggregateRating.{ratingValue, ratingCount}`
  - `review[].{reviewRating.ratingValue, datePublished, description, author.name}`
- Mode: `html_extract.mode: page` with JSON-LD parser. Optionally `html_extract.mode: embedded-json` with `script_selector: "script[type='application/ld+json']"` if the runtime supports it; otherwise parse the JSON-LD blob in command code.

### `/postal/{zip}`
- 301 redirect to the appropriate city page.
- Use: zip-code lookup as a convenience entry point.

### `/chains`
- Returns HTML index of chain restaurants.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/biz/"]`.

### `/most-gluten-free-friendly-cities`
- Returns HTML with the published ranking.
- Mode: `html_extract.mode: page` (extract the ranked list of city links).

### `/countries`
- Returns HTML index of countries.
- Mode: `html_extract.mode: links`, `link_prefixes: ["/"]` (filtered to two-letter country codes).

## Reachability runtime

`mode: standard_http`. Plain stdlib `net/http` clients are sufficient. No Surf, no clearance cookie, no resident browser sidecar. JSESSIONID cookies are returned by the server but are not required for any read-only endpoint above.

## Out of scope (login-walled, not in v1)

`/profile`, `/users/*`, `/edit-business-listing`, `/recent-activity`, `/review/*` (review submit), `/suggest`, `/redir`, `/reset-password/*`, `/register`, `/login`. The robots.txt also disallows `/search` and `/map` for all bots. v1 sticks to the public, no-auth surfaces above.

## Etiquette flags

The ToS prohibits crawling/scraping/spidering, and `robots.txt` adds an extra-disallow set for AI bots covering `/biz`, `/posts`, `/postal`. The printed CLI is a personal user-driven tool, not a crawler — it must default to:
- Honest `User-Agent` identifying the tool (no Chrome impersonation).
- 1 req/sec default rate limit, env-var-overridable (capped at 5 RPS).
- No bulk-state-export commands.
- README "Etiquette" section disclosing the ToS and AI-bot robots.txt block list.

## No traffic capture artifacts saved

Because the discovery was direct curl (no browser session), there is no `browser-sniff-capture.har` to archive. The replayable URL set above is the discovery output. No session state was captured (no cookies needed).
