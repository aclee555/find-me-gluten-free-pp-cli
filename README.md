# Find Me Gluten Free CLI

**A celiac trip-planner's terminal companion: every Find Me Gluten Free city, cached locally with cross-city compare, change tracking, and review safety filters the consumer app doesn't have.**

The Find Me Gluten Free iOS/Android app is great for 'what's near me right now.' This CLI is for the celiac who plans a trip three weeks out, the parent who returns to the same six cities every year, and the AI agent helping a celiac client weigh dining options. Sync any city once, then run trip compare across cities, cities diff to see what changed since last visit, places near for lat/lng radius search, and reviews recent --max-rating 2 to surface fresh safety reports.

Learn more at [Find Me Gluten Free](https://www.findmeglutenfree.com).

Printed by [@aclee555](https://github.com/aclee555) (Tony Lee).

> ## ⚠️ Personal use only — please respect Find Me Gluten Free
>
> Find Me Gluten Free is an independent, community-built site. Their [Terms of
> Service](https://www.findmeglutenfree.com/terms) prohibit crawling, scraping,
> and spidering, and their `robots.txt` adds an explicit disallow for AI bots
> on `/biz`, `/posts`, and `/postal`. **This CLI is intended for personal use
> only — a single human running point lookups while planning a trip.** It is
> not a crawler. It ships with these defaults so it stays on the right side of
> that line:
>
> - **Honest `User-Agent`** identifying the tool as a personal CLI (no Chrome
>   impersonation).
> - **2 req/sec default rate limit** (lower it with `--rate-limit 0.5` if the
>   site is busy).
> - **No bulk-export commands.** Sync one city at a time when you actually
>   need it; don't try to mirror the whole site.
> - **Read-only.** No commands write back to Find Me Gluten Free — bookmarks,
>   watches, and saved cities are local-only and never sent upstream.
>
> If you're using this CLI to plan a trip or look up celiac-safe restaurants
> for yourself or your family, that's exactly what it's for. If you find
> yourself wanting to mirror the whole catalog, build a competing site, or
> feed an aggregator, **please don't** — that's the use case the ToS forbids,
> and Find Me Gluten Free is a small team running a useful service for a
> community that needs it. If you have a legitimate commercial use case,
> contact them directly via the contact link on their site.
>
> By using this CLI you take responsibility for using it within Find Me
> Gluten Free's terms. The maintainer of this CLI is not affiliated with
> Find Me Gluten Free and provides no warranty.

## Install

This CLI is built with the [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press) but published from a personal repo rather than the public Printing Press library catalog. Install via Go:

```bash
go install github.com/aclee555/find-me-gluten-free-pp-cli/cmd/find-me-gluten-free-pp-cli@latest
```

Confirm it's on your `PATH`:

```bash
find-me-gluten-free-pp-cli --version
find-me-gluten-free-pp-cli doctor
```

If `find-me-gluten-free-pp-cli` is "command not found" after `go install`, add `~/go/bin` to your `PATH` (in `~/.zshrc` or `~/.bash_profile`):

```bash
export PATH="$HOME/go/bin:$PATH"
```

### Install the agent skill (optional)

If you use Claude Code or OpenClaw, the bundled `SKILL.md` tells the agent how to drive the CLI on your behalf — trigger phrases, recipes, when not to reach for it.

**Claude Code:**

```bash
mkdir -p ~/.claude/skills/pp-find-me-gluten-free
curl -sL https://raw.githubusercontent.com/aclee555/find-me-gluten-free-pp-cli/main/SKILL.md \
  -o ~/.claude/skills/pp-find-me-gluten-free/SKILL.md
```

Restart Claude Code so it picks up the new skill.

**OpenClaw:**

```bash
mkdir -p ~/.openclaw/skills/pp-find-me-gluten-free
curl -sL https://raw.githubusercontent.com/aclee555/find-me-gluten-free-pp-cli/main/SKILL.md \
  -o ~/.openclaw/skills/pp-find-me-gluten-free/SKILL.md
```

Or just tell your OpenClaw agent in chat:

```
Install the pp-find-me-gluten-free skill from https://github.com/aclee555/find-me-gluten-free-pp-cli. The SKILL.md at the repo root tells you how to use the CLI.
```

## Authentication

No authentication required. The CLI reads only public city, business, and sitemap pages via standard HTTP. See the **Personal use only** disclaimer above for the full ToS / etiquette posture this CLI ships with.

## Quick Start

```bash
# Foundation: fetch every place in Richmond, parse schema.org JSON-LD, persist into the local SQLite cache so all later commands are offline-fast
find-me-gluten-free-pp-cli cities sync us va richmond


# City-wide aggregate (count, avg rating, p50/p90, dedicated %, GF-menu %) computed locally — Find Me Gluten Free has no equivalent endpoint
find-me-gluten-free-pp-cli places stats --city richmond --json


# The differentiator: side-by-side per-city celiac-friendliness — this question simply isn't answerable in the consumer app
find-me-gluten-free-pp-cli trip compare richmond charlottesville norfolk --json


# City-wide review feed sorted newest-first; --max-rating 2 surfaces the safety-report-style 1-2 star reviews most worth reading before a trip
find-me-gluten-free-pp-cli reviews recent --city richmond --max-rating 2 --json


# After a second 'cities sync' run, this surfaces only what changed (added/removed places, count delta) — re-reading from scratch every visit is the consumer-app failure mode
find-me-gluten-free-pp-cli cities diff richmond --since last-sync --json

```

## Unique Features

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

## Usage

Run `find-me-gluten-free-pp-cli --help` for the full command reference and flag list.

## Commands

### chains

Chain restaurant index

- **`find-me-gluten-free-pp-cli chains list`** - List chain restaurants with gluten-free options.

### cities

City directory and rankings

- **`find-me-gluten-free-pp-cli cities list`** - List cities in a state or region (e.g., /us/va lists Virginia cities).
- **`find-me-gluten-free-pp-cli cities ranked`** - Find Me Gluten Free's published ranking of the most gluten-free-friendly cities.

### countries

Country directory

- **`find-me-gluten-free-pp-cli countries list`** - List every country with gluten-free coverage on Find Me Gluten Free.

### places

Restaurants and the places-by-city / places-by-cuisine views

- **`find-me-gluten-free-pp-cli places filter`** - List restaurants in a city filtered by cuisine or feature. The filter slug can be 'dedicated-facilities', 'gluten-free-menu', or any of 200+ cuisine slugs (burgers, pizza, bakeries, ...).
- **`find-me-gluten-free-pp-cli places get`** - Get a single restaurant page. Use 'places hydrate' for the parsed schema.org JSON-LD (rating, reviews, lat/lng) — this raw command returns the title/description/canonical metadata only.
- **`find-me-gluten-free-pp-cli places list`** - List restaurants in a city (e.g., /us/va/richmond).

### states

State / region directory under one country

- **`find-me-gluten-free-pp-cli states list`** - List states or regions in a country (e.g., /us, /gb, /fr).


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
find-me-gluten-free-pp-cli chains

# JSON for scripting and agents
find-me-gluten-free-pp-cli chains --json

# Filter to specific fields
find-me-gluten-free-pp-cli chains --json --select id,name,status

# Dry run — show the request without sending
find-me-gluten-free-pp-cli chains --dry-run

# Agent mode — JSON + compact + no prompts in one flag
find-me-gluten-free-pp-cli chains --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-find-me-gluten-free -g
```

Then invoke `/pp-find-me-gluten-free <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Then register it:

```bash
claude mcp add find-me-gluten-free find-me-gluten-free-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/find-me-gluten-free-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "find-me-gluten-free": {
      "command": "find-me-gluten-free-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
find-me-gluten-free-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/find-me-gluten-free-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **HTTP 429 from findmeglutenfree.com** — Lower the rate via the global flag: --rate-limit 0.5 (one request every two seconds).
- **places stats --city <slug> returns total: 0** — City has not been hydrated yet. Run: find-me-gluten-free-pp-cli cities sync <country> <state> <city>
- **places near returns []** — Haversine search runs over the local cache. Sync at least one city near the lat/lng first via 'cities sync', then retry.
- **cities diff says 'need 2+ snapshots'** — Run 'cities sync <country> <state> <city>' a second time; diff compares the two most recent snapshots.
- **places hydrate fails with 'no application/ld+json block found'** — A small number of newly listed pages render without schema.org data. The biz exists; the rich detail will appear after the next site index pass. Re-try in 24 hours.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
