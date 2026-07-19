# Command Ledger

**Purpose (Wireframe_Session_Plan.md §1.1):** every actuating control introduced in any wireframe gets a row here — control → its future chat-command mapping (verb + args, Guix-simple), or a `RETURN-SESSION` flag when the mapping needs real design work later. No control ships in a wireframe without a row. The future chat engine + LLM router consume this ledger; it is the "document the builds for a return trip" artifact.

Verb grammar is provisional (Guix-simple: `domain.verb key=value`), refined when the chat engine is designed. A `RETURN-SESSION` flag means the mapping is non-trivial and deferred, not that the control is exempt.

---

## Session A — Grid & Spatial System (shell + M1 + Home + quick-dash)

| # | Control | Surface | Command mapping (verb + args) | Notes |
|---|---|---|---|---|
| A1 | Nav module select | Nav rail [MODULE SECTOR] | `nav.goto module=<home\|assets\|pulse\|txn\|trade\|control>` | 6 locked modules |
| A2 | League switch | Nav rail [PORTFOLIO SECTOR] | `league.switch id=<leagueId>` | recency-sorted at scale |
| A3 | Portfolio overview select | Nav rail portfolio crest | `league.portfolio` | `RETURN-SESSION` — Phase-3 cross-league surface not designed |
| A4 | Density tier set | Workspace (curiosity trigger) | `view.density tier=<narrative\|tactical\|matrix>` | CSS-var swap only |
| A5 | M1 view toggle | M1 header band | `assets.view mode=<global\|franchise\|capeff>` | 3 client views over persisted rows |
| A6 | M1 position filter | M1 sticky sub-header | `assets.filter pos=<QB\|RB\|WR\|TE\|…\|all>` | chip toggle |
| A7 | M1 column sort | M1 table header | `assets.sort col=<adjusted\|base\|salary\|rank> dir=<asc\|desc>` | tabular columns |
| A8 | Entity select → inspector | Any row/entity click | `inspect entity=<playerId>` | populates inspector, no navigation |
| A9 | Inspector collapse/expand | Inspector zone | `ui.toggle target=inspector` | `transform: translateX` overlay, not width-trade |
| A10 | Comms summon/collapse | Right-edge 48px strip (any screen) | `ui.summon target=comms` / `ui.collapse target=comms` | quick-dash, transform overlay |
| A11 | Calendar summon/collapse | Top-right 32px seam (any screen) | `ui.summon target=calendar` / `ui.collapse target=calendar` | quick-dash, generous overlay; full anatomy = Session D |
