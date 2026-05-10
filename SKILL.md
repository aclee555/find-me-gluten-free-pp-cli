---
name: pp-find-me-gluten-free
description: "A celiac trip-planner's terminal companion: every Find Me Gluten Free city, cached locally with cross-city compare,... Trigger phrases: `find me gluten free in`, `celiac safe restaurants in`, `dedicated gluten free near`, `compare gluten free places between`, `what changed in <city> since I last checked`, `use find-me-gluten-free`, `run find-me-gluten-free`."
author: "Tony Lee"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - find-me-gluten-free-pp-cli
---

# Find Me Gluten Free — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `find-me-gluten-free-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install find-me-gluten-free --cli-only
   ```
2. Verify: `find-me-gluten-free-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

The Find Me Gluten Free iOS/Android app is great for 'what's near me right now.' This CLI is for the celiac who plans a trip three weeks out, the parent who returns to the same six cities every year, and the AI agent helping a celiac client weigh dining options. Sync any city once, then run trip compare across cities, cities diff to see what changed since last visit, places near for lat/lng radius search, and reviews recent --max-rating 2 to surface fresh safety reports.

## When to Use This CLI

Use this CLI when planning a celiac-safe trip, weighing multiple cities, tracking changes to a personal allowlist, or feeding gluten-free restaurant data to an AI agent. It is structurally not the right tool for a celiac who only needs 'what's safe within a half mile of where I'm standing right now' — that's the consumer app's strength. Reach for this when the question is 'where on this trip is GF pizza most available?', 'what changed in Pittsburgh since I was last there?', or 'list every dedicated-GF place in Portland with at least a 4-star average and a recent review.'

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Trip planning that compounds locally
- **`trip compare`** — Compare 2+ cities side-by-side: dedicated-GF count, GF-menu count, total places, average rating, and top-rated picks per city — the question the consumer apps can't answer.

  _Reach for this when the user is choosing between cities for a celiac trip; emits structured per-city counts an agent can rank or filter._

  ```bash
  find-me-gluten-free-pp-cli trip compare lisbon porto madrid --json
  ```
- **`cities diff`** — Show what changed in a city since the last time you synced it: added/removed places, rating drift, new low-star safety reports.

  _Reach for this before a return visit to a city the user has tracked previously — surfaces only what's actually changed._

  ```bash
  find-me-gluten-free-pp-cli cities diff pittsburgh --since last-sync --json
  ```
- **`watch`** — Watch specific places; periodic 'watch report' fetches each and reports rating drift and new low-rated reviews since the baseline.

  _Reach for this on a personal allowlist of trusted places; surfaces only places where something celiac-relevant changed._

  ```bash
  find-me-gluten-free-pp-cli watch report --json
  ```

### Geographic queries the site doesn't expose
- **`places near`** — Find dedicated-GF or GF-menu places within a radius of a lat/lng using Haversine over locally cached schema.org geo data.

  _Reach for this when an agent has coordinates (a hotel, an attraction) and needs nearby celiac-safe options ranked._

  ```bash
  find-me-gluten-free-pp-cli places near --lat 38.9072 --lng -77.0369 --radius-km 2 --dedicated --json
  ```

### Aggregates the site doesn't compute
- **`places stats`** — One-shot SQL aggregate over a city: count, average rating, p50/p90 rating, dedicated-GF percent, GF-menu percent. Filterable by cuisine.

  _Reach for this when an agent needs a one-line answer to 'how celiac-friendly is city X for cuisine Y?' before drilling into specific places._

  ```bash
  find-me-gluten-free-pp-cli places stats --city austin --cuisine pizza --json
  ```
- **`cuisines compare`** — For one cuisine across multiple cities: per-city count, dedicated-GF percent, top-rated place. Useful when planning where on a trip GF pizza (or burgers, or bakery) is most available.

  _Reach for this when cuisine availability matters for the trip and the user wants to weigh cities by what's actually on offer._

  ```bash
  find-me-gluten-free-pp-cli cuisines compare burgers us va richmond charlottesville --json
  ```

### Reviews surfaced as a feed
- **`reviews recent`** — Newest reviews across every place in a city, sorted by datePublished. Filter to safety-report-style 1-2 star reviews with --max-rating.

  _Reach for this when checking for new safety reports across a city before a trip; the most recent low-stars are the highest-signal warnings._

  ```bash
  find-me-gluten-free-pp-cli reviews recent --city portland --max-rating 2 --since 2026-04-01 --json
  ```
- **`reviews safety`** — Filter cached reviews for one place to those mentioning got-glutened / cross-contamination keywords from a curated, deterministic list — no LLM.

  _Reach for this before vetting a specific place; surfaces just the reviews that report a celiac reaction or cross-contamination._

  ```bash
  find-me-gluten-free-pp-cli reviews safety 6118524171452416 --json
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**chains** — Chain restaurant index

- `find-me-gluten-free-pp-cli chains` — List chain restaurants with gluten-free options.

**cities** — City directory and rankings

- `find-me-gluten-free-pp-cli cities list` — List cities in a state or region (e.g., /us/va lists Virginia cities).
- `find-me-gluten-free-pp-cli cities ranked` — Find Me Gluten Free's published ranking of the most gluten-free-friendly cities.

**countries** — Country directory

- `find-me-gluten-free-pp-cli countries` — List every country with gluten-free coverage on Find Me Gluten Free.

**places** — Restaurants and the places-by-city / places-by-cuisine views

- `find-me-gluten-free-pp-cli places filter` — List restaurants in a city filtered by cuisine or feature. The filter slug can be 'dedicated-facilities',...
- `find-me-gluten-free-pp-cli places get` — Get a single restaurant page. Use 'places hydrate' for the parsed schema.org JSON-LD (rating, reviews, lat/lng) —...
- `find-me-gluten-free-pp-cli places list` — List restaurants in a city (e.g., /us/va/richmond).

**states** — State / region directory under one country

- `find-me-gluten-free-pp-cli states <country>` — List states or regions in a country (e.g., /us, /gb, /fr).


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
find-me-gluten-free-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Plan a 3-city trip by GF density

```bash
find-me-gluten-free-pp-cli trip compare richmond charlottesville norfolk --json --select dedicated_count,avg_rating,highest_rated
```

After 'cities sync' for each, this asks which has the most dedicated places and the highest average rating; --select narrows the per-city JSON to the comparison fields an agent needs.

### Find safe lunch within walking distance of your hotel

```bash
find-me-gluten-free-pp-cli places near --lat 41.1496 --lng -8.6109 --radius-km 1 --dedicated --json
```

Haversine over the local cache returns dedicated-GF places within 1 km of the hotel lat/lng; useful when an agent already has coordinates.

### Surface fresh safety reports before a trip

```bash
find-me-gluten-free-pp-cli reviews recent --city richmond --max-rating 2 --json --select biz_name,rating,date_published,description
```

City-wide low-rated reviews sorted newest-first; --select keeps the response under a few KB.

### Watch your personal allowlist for safety drift

```bash
find-me-gluten-free-pp-cli watch report --json
```

After running 'watch add <biz_id>' for places you trust, watch report re-fetches each (rate-limited) and surfaces rating drift or new low-rated reviews.

### Compare a cuisine across trip stops

```bash
find-me-gluten-free-pp-cli cuisines compare burgers us va richmond charlottesville --json
```

The 214-cuisine taxonomy joined across cities; cuisine slug first, then country and state, then 2+ city slugs.

## Auth Setup

No authentication required.

Run `find-me-gluten-free-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  find-me-gluten-free-pp-cli chains --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
find-me-gluten-free-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
find-me-gluten-free-pp-cli feedback --stdin < notes.txt
find-me-gluten-free-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.find-me-gluten-free-pp-cli/feedback.jsonl`. They are never POSTed unless `FIND_ME_GLUTEN_FREE_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `FIND_ME_GLUTEN_FREE_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
find-me-gluten-free-pp-cli profile save briefing --json
find-me-gluten-free-pp-cli --profile briefing chains
find-me-gluten-free-pp-cli profile list --json
find-me-gluten-free-pp-cli profile show briefing
find-me-gluten-free-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `find-me-gluten-free-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add find-me-gluten-free-pp-mcp -- find-me-gluten-free-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which find-me-gluten-free-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   find-me-gluten-free-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `find-me-gluten-free-pp-cli <command> --help`.
