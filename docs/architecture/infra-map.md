# TheWarRoom — infrastructure map: today, should-have, future (CLI + AI first)
Claudebox, 2026-11-22 evening CT. Read from main 52610ec18 + today's rulings. Planning doc; not a build order.
Confidence: "today" is read from repo docs/code (high); live-fact storage per league is NOT yet verified (T693 audit).

## 0. The one-paragraph answer
TWR already has the bones of an AI-first product: ONE command catalog (147 commands, each with permission,
gravity G0-G3, undo rule, which doors may call it), a `twr` CLI, an MCP server (`twr mcp`), and a
preview -> confirm -> execute path where AI can never confirm its own big change. That is the industry pattern
(Algolia, Google Workspace CLI, agent-CLI specs all land on dry-run -> confirm token). Do NOT reinvent it.
What is missing is (a) a clean, mapped data flow for NFL facts and projections, (b) a real sandbox
("copy my league and try it"), (c) an analytics layer for the model/research site, and (d) the CLI/MCP
covering reads and the whole league life cycle, not just 3 tools.

## 1. TODAY (what exists)
| Layer | What it is | Where |
|---|---|---|
| Language/runtime | Go 1.26 server + Wails desktop; React/TS frontend; phone = browser | cmd/twr-server, main.go, frontend/ |
| Storage | SQLite (modernc), 3-way split: accounts.db / reference.db (shared NFL data, players) / one league.db per league | ADR 0004, storage-recovery-identity.md |
| Identity | Opaque canonical player ID minted in reference; each league keeps league_player rows | ADR 0006 |
| League record | Native league = command ledger (every change recorded, revisioned) | internal/commissioner, store/history |
| Ingestion | ~30 feed packages: MFL, nflverse, DynastyProcess, CFBD, ESPN news, live game/stats, injuries, schedule | internal/ingestion/* |
| Raw capture | Every outbound fetch passes internal/archive, body kept in history.db; unknown URLs logged | system-narrative "External services" |
| Scoring | Pure scoring engine; per-league rules (leaguerules) | ADR 0005, fantasyscore |
| Models | D11 history projection range; M2 power rankings; dynasty value | projection, powerrankings, appvalue |
| Commands | commandcatalog: 147 commands (41 G0, 28 G1, 9 G2, 2 G3 tagged), 5 doors | command-spine.md |
| CLI / AI | `twr` (ledger, preview, execute, mcp, league-export); token from commissioner; G2/G3 need human code | cmd/twr/README.md |
| Backups | Offsite to R2-style object store (minio client); nightly season-server backup to CT105 | internal/offsite |
| Trust | TUF + sigstore signed releases | ADR 0008 |
| Environments | Dev (Beelink) / Lab (CT260, wipe any time) / Season server (laptop, Saturday-only) | CLAUDE.md "Environments", PR #463 |
| Guard rails in CI | placement, import boundaries, binding freshness, command parity, MFL fence | SYSTEM_MAP "Rules CI enforces" |

Honest weak spots today:
1. Live stats/box score: believed one-per-server, not proven; projections are points (MFL) or history-only.
2. Cross-file writes (accounts/reference/league) are ordered, not atomic.
3. MCP exposes 3 tools (ledger, preview_change, execute_change); no typed reads (standings, lineup, roster, value).
4. No league sandbox/clone; the Lab is the only "sandbox" and it is a whole server.
5. Analytics/model work has no home: history.db raw bodies + SQLite, no columnar store.
6. ~10+ stale worktrees carry old copies of cmd/twr (Stage 1 cleanup owns this).

## 2. SHOULD HAVE (by end of season / before the Jan campaign gate)
### 2a. Data flow, mapped (one picture, in repo as docs/architecture/data-flow.md)
```
SOURCES (MFL, Sleeper proj, nflverse, ESPN news, live box score, injuries, schedule)
   │  fetched ONCE per server, every body archived (archive/history.db)  ← exists
   ▼
reference.db  = league-NEUTRAL facts: players, IDs, schedule, live stat lines,
                injuries, depth, projected STAT LINES (weekly snapshots)   ← T693 audit, T695
   │  read-only to leagues
   ▼
league.db (one per league) = league RULES + LEDGER + scored results      ← exists
   │  scoring engine applies league rules to neutral stat lines
   ▼
doors: web/phone, desktop, CLI, MCP — all through commandcatalog         ← exists
```
Rule to write down as an ADR: "A fact that does not depend on league rules lives once in reference.db.
A league stores only its rules, its ledger, and what its rules produce." (Ruling 11-22; T693 checks it.)

### 2b. Sandbox (the gap Christopher feels most)
Three sandbox levels, smallest first; reuse what exists:
1. **Preview** (exists): every change previews without recording. Make sure EVERY mutation has one.
2. **League clone** (new, small): copy one league.db (+ pinned reference snapshot) into a throwaway
   "sandbox league" the owner/AI can play forward ("what if we change scoring", "replay last season
   with new rules"). Built from existing pieces: leaguetransfer export/check + stand-up (read-only
   stand-up already exists) + whatif sessions. Never touches the real ledger. Expire after N days.
3. **Server sandboxes** (exist): Dev / Lab / Season server. Keep.
Rule: an AI helper may freely run anything in a sandbox league; in a real league it is preview-only
unless the person confirms (current G-levels).

### 2c. Environments — finish, do not expand
Keep the 3 names. Add: one seeded "known league" fixture per league shape (32 dynasty, 10/12/16 redraft)
so Lab resets are one command; Season server restore drill once (part of GATE part 1 rollback).
Backups: consider Litestream (continuous SQLite copy to the same object store, v0.5 stable, used in
production for years) instead of only nightly copies — per-league files fit it. Decide after T693.

### 2d. Projections pipeline (T695 → T696)
Sleeper stat lines → reference.db weekly snapshot (raw kept) → score per league preset → blend with D11.
Log every projection vs actual (accuracy table) from day one; that table is what makes step 2 provable.

## 3. FUTURE (after the season: CLI and AI first)
### 3a. "Every door is the catalog" — finish the command spine
- Generate CLI verbs AND MCP tools from the catalog (verbs.gen.go already exists for CLI): one typed
  MCP tool per exposed command, reads included. Target: an AI can do everything a commissioner can,
  through preview, with the same permissions. No hand-written tool lists.
- Agent-CLI conventions to adopt (industry, 2026): JSON output by default for agents, --dry-run on
  every write (= preview), confirm token (= twrc1 code, exists), retry-safe (same command twice = no
  double effect; the ledger revision `expected` already gives this), never wait on keyboard input.
- Ask flow = "AI flags assumption risk" ruling: preview output carries "assumes X, cost if wrong Y,
  cheapest check Z".
### 3b. Analytics / model stack (for own model, research site, stock market, power-rank replay)
- Columnar side store, NOT a new database server: Parquet files + DuckDB (queries Parquet directly,
  nflverse already publishes Parquet; parquet-go is already in go.mod). Lives beside reference.db,
  rebuilt from archive; read-only to the app.
- Flow: nflverse/Sleeper/MFL archive → Parquet by season/week → DuckDB model jobs on Hermes →
  results written back into reference.db as projected stat lines / values.
- Public research site reads ONLY our own model + licensed data (never Sleeper) — enforce by a
  source tag on every row (sources.csv already tags sources).
### 3c. Sandbox grows into "replay"
League clone + historical stat lines = replay any past season under new rules (power-ranking tuning,
rule-vote previews "here is how last year would have ended"). Same engine, no new math.
### 3d. Scale (only if the campaign works)
Per-league SQLite scales by adding files, not servers. Reference.db is the shared hot spot: load test it
(GATE part 3) before thinking about Postgres. Do not migrate databases pre-emptively.

## 4. What NOT to build (already exists or not needed)
- A new protocol/API layer for AI — commandcatalog + MCP is it.
- A permission system for agents — helper tokens, scope, G2/G3 human code, revocation exist.
- A data warehouse server — DuckDB over Parquet is files.
- A second backup system — extend offsite (object store) / possibly Litestream.

## 5. Gaps still open (research or audit, owner)
1. Is every live/neutral fact really once per server? → Tom T693.
2. Which commands lack a preview or an undo rule? → small catalog report (new task suggested).
3. Litestream vs nightly copy for league.db files → decide after T693; test on Lab.
4. MCP tool generation from catalog: size the work → Tom, after season.
5. Cross-file ordering risk (accounts/reference/league) under load → GATE part 3 load test.

## Sources
- Repo: SYSTEM_MAP.md, docs/architecture/{command-spine,storage-recovery-identity,system-narrative,desktop-modes}.md,
  docs/adr/0004..0008, cmd/twr/README.md, CLAUDE.md Environments.
- https://www.algolia.com/blog/engineering/we-rewrote-the-algolia-cli-for-ai-agents
- https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents.md
- https://dev.to/tumf/agentic-cli-design-7-principles-for-designing-cli-as-a-protocol-for-ai-agents-2c10
- https://litestream.io/how-it-works/ ; https://simonwillison.net/2025/Oct/3/litestream
- https://docs.ropensci.org/piggyback/articles/cloud_native.html (DuckDB over nflverse Parquet)
- https://pypi.org/project/nfl-mcp/0.1.1/ (nflverse + DuckDB + MCP reference)
