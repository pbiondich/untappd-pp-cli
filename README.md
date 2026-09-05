# untappd-pp-cli

Readable Untappd beer intelligence for agents — public beer and brewery pages plus search, without the closed official developer API.

Untappd isn't just a check-in social network. It's a public beer intelligence layer: global ratings, style, ABV, IBU, and check-in volume that agents can query without the closed official API.

This CLI is a Printing Press public-page reader (Food and Dining). It does **not** use the official Untappd developer API (closed to new apps) and is not published into `mvanhorn/printing-press-library` this pass. It reads public HTML and Untappd's published frontend Algolia search index only. Private/authenticated data is never requested. Ratings are never invented: if Untappd has no public score, `rating` is `null` and `rating_present` is `false`.

## Install

Requires Go 1.26.6 or newer. Preferred install after this repo is merged:

```bash
go install github.com/pbiondich/untappd-pp-cli/cmd/untappd-pp-cli@latest
```

That writes the binary to `$GOPATH/bin` (default `$HOME/go/bin`). Homelab installs the binary onto the shared agent computer after merge — **do not assume `untappd-pp-cli` is already on PATH**. Verify with an absolute path or `command -v` before calling it.

```bash
command -v untappd-pp-cli || go install github.com/pbiondich/untappd-pp-cli/cmd/untappd-pp-cli@latest
untappd-pp-cli doctor
```

From a checkout:

```bash
go build -o bin/untappd-pp-cli ./cmd/untappd-pp-cli
```

Optional MCP binary: `go install github.com/pbiondich/untappd-pp-cli/cmd/untappd-pp-mcp@latest`.

Identify the client politely. Override the contact token in the User-Agent with `UNTAPPD_CONTACT` (defaults to `contact@example.com`).

## MCP (optional)

```bash
go install github.com/pbiondich/untappd-pp-cli/cmd/untappd-pp-mcp@latest
```

```json
{
  "mcpServers": {
    "untappd": {
      "command": "untappd-pp-mcp"
    }
  }
}
```

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Verify Setup

```bash
untappd-pp-cli doctor
```

This checks your configuration.

### 3. Try Your First Command

```bash
untappd-pp-cli search beer "Cool Bay Hop Butcher" --agent
untappd-pp-cli beer 4384886 --agent
untappd-pp-cli nearby --near "Elliot Park Hotel, Minneapolis" --agent
```

## Usage

Run `untappd-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `UNTAPPD_CONFIG_DIR`, `UNTAPPD_DATA_DIR`, `UNTAPPD_STATE_DIR`, or `UNTAPPD_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `UNTAPPD_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export UNTAPPD_HOME=/srv/untappd
untappd-pp-cli doctor
```

Under `UNTAPPD_HOME=/srv/untappd`, the four dirs resolve to `/srv/untappd/config`, `/srv/untappd/data`, `/srv/untappd/state`, and `/srv/untappd/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "untappd": {
      "command": "untappd-pp-mcp",
      "env": {
        "UNTAPPD_HOME": "/srv/untappd"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `UNTAPPD_DATA_DIR` overrides an explicit `--home` for that kind. Use `UNTAPPD_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `UNTAPPD_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `untappd-pp-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### beer

Public beer pages and search — name, brewery, style, ABV, IBU, global rating, rating count, description, check-in volume.

- **`untappd-pp-cli search beer "Hop Butcher Put On the Glasses"`** — matches with brewery, ABV, rating when present, URL/id
- **`untappd-pp-cli beer search --query "..."`** — same search via the generated verb
- **`untappd-pp-cli beer 4384886`** / **`untappd-pp-cli beer get 4384886`** — name, brewery, style, ABV, IBU, global rating, rating count, description

### brewery

Public brewery search and beer lists with ratings when Untappd publishes them.

- **`untappd-pp-cli brewery search "Hop Butcher"`** — brewery matches
- **`untappd-pp-cli brewery HopButcher beers`** / **`untappd-pp-cli brewery beers HopButcher`** — beer list with ratings when available
- **`untappd-pp-cli brewery 23570`** / **`untappd-pp-cli brewery get 23570`** — brewery page

### venue / nearby

Public venues via Untappd's published Algolia `venue` index (includes `_geoloc`). Place strings are resolved first as an Untappd venue, then via OSM Photon. Nominatim is not required. Nearby ranks by Untappd **popularity** (check-in volume), not an invented star rating — `rating` stays null unless a beer menu publishes a global beer score.

```bash
untappd-pp-cli venue search "Elliot Park" --agent
untappd-pp-cli venue search --near "Elliot Park Hotel, Minneapolis" --agent
untappd-pp-cli venue 8255451 --agent
untappd-pp-cli venue 2714 top-beers --agent
untappd-pp-cli nearby --near "Elliot Park Hotel, Minneapolis" --radius-mi 2 --agent
untappd-pp-cli nearby --lat 44.972332 --lng -93.266396 --radius-mi 2 --sort recent --agent
```

`--beer-only` defaults to true on `nearby` so the list is places Untappd marks as having beer.

### lookup

Look up several tap-list names in one polite sequential pass:

```bash
untappd-pp-cli lookup "Put On the Glasses" "Saturation Above Replacement" "Lord Octopus" --brewery "Hop Butcher" --agent
```


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`untappd-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`untappd-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`untappd-pp-cli learnings list`** - Inspect taught rows
- **`untappd-pp-cli learnings forget <query>`** - Undo a teach
- **`untappd-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`untappd-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`untappd-pp-cli teach-pattern`** - Install a query/resource template up front
- **`untappd-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `UNTAPPD_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `untappd-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
untappd-pp-cli beer 4384886

# JSON for scripting and agents
untappd-pp-cli beer 4384886 --json
# Filter to specific fields
untappd-pp-cli beer 4384886 --json --select id,slug,name,rating

# Dry run — show the request without sending
untappd-pp-cli beer 4384886 --dry-run

# Agent mode — JSON + compact + no prompts in one flag
untappd-pp-cli beer 4384886 --agent
untappd-pp-cli search beer "Cool Bay Hop Butcher" --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
untappd-pp-cli doctor
```

Verifies configuration and that public Untappd pages are reachable (not the closed official developer API).

## Configuration

Run `untappd-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/untappd-pp-cli/config.toml`; `--home`, `UNTAPPD_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the beer, brewery, or venue id/slug
- Search first: `untappd-pp-cli search beer "…" --agent` or `untappd-pp-cli venue search "…" --agent`
- For a place name, use `nearby --near "…" --agent` rather than guessing a venue id

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
