<div align="center">

# 🏈 TheWarRoom

### A dynasty fantasy football intelligence engine that scores what real players do in real games — and out-thinks the stock platforms doing it.

[![Status](https://img.shields.io/badge/status-🔥%20ENGINE%20COMPLETE%20·%20transactions%20live-brightgreen)](docs/build-handoffs/Build_Tracker.md)
[![Progress](https://img.shields.io/badge/build-27%20of%2038%20sessions%20·%2071%25%20·%20final%20third-success)](docs/build-handoffs/Build_Tracker.md)
[![Models](https://img.shields.io/badge/position%20models-10%20of%2010%20·%20ALL%20LIVE-blue)](#-pillar-1--the-scoring-engine--✅-complete)
[![Verified](https://img.shields.io/badge/live%20data-1217%20players%20·%2032%20teams%20·%20verified-blueviolet)](#-its-alive--from-raw-data-to-a-decision)
[![Money](https://img.shields.io/badge/money-int64%20cents%20·%20exact%20to%20the%20penny-9cf)](#-pillar-2--the-transaction-engine--🔥-live)
[![Phase](https://img.shields.io/badge/phase-1%20·%20personal%20tool-orange)](#️-the-road)

[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails%20v2-DF0000?logo=wails&logoColor=white)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB)](https://react.dev)
[![Tailwind](https://img.shields.io/badge/Tailwind-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com)
[![SQLite](https://img.shields.io/badge/SQLite%20WAL-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)

[![Claude](https://img.shields.io/badge/Claude-Anthropic-D97757?logo=anthropic&logoColor=white)](https://anthropic.com)
[![GLM](https://img.shields.io/badge/GLM-Z.ai-6E3AF2?logo=zhipuai&logoColor=white)](https://z.ai)
[![Gemini](https://img.shields.io/badge/Gemini-Google-8E75B2?logo=googlegemini&logoColor=white)](https://gemini.google.com)
[![Ornith](https://img.shields.io/badge/Ornith-local%20model-4B5563)](#-built-to-last)

*32 teams. Six scoring layers. Ten position models. Twenty-one scouting sources. **One number per player.***

</div>

---

## ⚡ The Pitch

MyFantasyLeague gives you a roster and a box score. **TheWarRoom gives you an edge.**

It ingests live league data from MFL, fuses it with twenty-one professional scouting sources, and runs every rostered player through a **six-layer valuation engine** built on this league's exact ruleset. The output is a single **Adjusted Score** per player — and a war room full of tools that turn that score into decisions: who to bid on, what a trade is really worth, where your roster is thin, and what a contract costs you three years from now.

It also replaces the part nobody enjoys: the bid math, the dead-cap arithmetic, the snipe clock, the trade-deadline policing. The machine handles the mechanical. The humans keep the judgment.

> **The engine's core trick:** it separates *"is this player good?"* from *"how long does he have?"* — scoring talent and aging as two different jobs, the way a real scout does. An elite receiver in his prime and a declining veteran can post the same box score; TheWarRoom prices them apart.

---

## 🚀 It's Alive — From Raw Data to a Decision

A year ago this was a plan. **Today it pulls a real league, scores real players, ranks them on screen, and executes real transactions with the cap math done automatically.** The entire scoring spine — data in, engine in the middle, a number out, on a board you can drive, with mutations that move real money — is built, merged, and verified end-to-end on real hardware.

```
  real MFL league ─► transport ─► ingestion ─► normalize ─► ENGINE ─► Adjusted Score ─► ranked board ─► TRANSACTIONS
     (live API)      rate-limited   boundary-    raw→domain   6 layers   one number       live board      trades · cuts
                     host-routed    guarded      talent≠age  10 models   per player        on screen       dead-cap auto

   ✅ DATA     1,217 rostered players · 32 franchises · zero loss · dead cap reconciled to the penny
   ✅ ENGINE   6-layer pure-function pipeline · ALL 10 position models calibrated · nothing hardcoded
   ✅ BOARD    real 32-team rankings on screen · re-ranks the instant you turn a calibration knob
   ✅ MONEY    int64-cents exact · cut a player and watch §8 dead cap hit the cap — verified live
   ✅ PROOF    every build: green tests + blind AI review + used on real hardware before it counts as "done"
```

This isn't a mock or a fixture. It's the **actual Legacy NFL league**, pulled live, normalized into a locked type system, scored through the full pipeline, ranked on a board that re-ranks the moment you change a calibration knob — and now, a coordinator that executes trades atomically and cuts players with the §8 dead-cap penalty computed and applied on the spot. The hard parts of any data product — clean ingestion, a trustworthy engine, a working UI loop, **and safe money-moving transactions** — are **done and proven.**

---

## 📈 Build Progress — Into the Final Third

```
Tier 0  Scaffold            [██████████]  1/1     ✅ DONE
Tier 1  Data Pipeline       [██████████]  7/7     ✅ DONE   — live, end-to-end
Tier 2  Logic Stores        [██████████]  4/4     ✅ DONE   — rulebook · state · params · output
Tier 2  Scoring Engine      [██████████]  1/1     ✅ DONE   — six-layer pure pipeline
Tier 2  Testing Harness     [██████████]  1/1     ✅ DONE   — validation board + live tuning
Tier 2  Position Models     [██████████]  10/10   ✅ DONE   — every position calibrated & scoring
Tier 3  Board + Trades      [██░░░░░░░░]  3/14    🔥 LIVE   — rankings board · trades · dead-cap
Tier 3  The War Room        [░░░░░░░░░░]  0/…     ▸ ahead   — the 8 modules turn scores into calls

         OVERALL            [█████████████████████░░░░░░░░]   27 / 38   ·   71%
```

| Milestone | Status |
|---|---|
| 🏗️ **Foundation** — Go · Wails · React · SQLite WAL, compiler-enforced architecture | ✅ Shipped |
| 🔌 **Data pipeline** — MFL transport → ingestion → normalization, live against the real league | ✅ Shipped |
| 📋 **Scouting schema** — all 10 positions, 21 sources wired and fetching | ✅ Shipped |
| 🗄️ **Logic stores** — rulebook · league state · parameters · **immutable output store** | ✅ Shipped |
| ⚙️ **The engine** — six-layer pure-function scoring pipeline | ✅ Shipped |
| 🧪 **Testing harness** — three-state validation board, live admin tuning loop | ✅ Shipped |
| 🧠 **Position models** — **all 10 calibrated and scoring** | ✅ Shipped |
| 📊 **Asset Rankings board** — all 32 rosters ranked on screen from real data | ✅ Shipped |
| 🔧 **Transaction engine** — atomic trades + waivers with **automated §8 dead cap** | 🔥 In flight |
| 💾 **Contract ops → the 8 modules → live bidding** | ▸ The back half |

---

## 🧠 Pillar 1 — The Scoring Engine · ✅ COMPLETE

The valuation brain. Six layers, run in order, every value MFL-sourced or admin-tunable — nothing hardcoded. **All six layers and all ten position models are built, calibrated, and scoring live.**

```
              MFL API  +  21 scouting sources (PFF · RAS · NGS · Madden · film)
                                      │
                                      ▼
   ┌──────────────────────────────────────────────────────────────┐
   │  L1  Data Hygiene        clean inputs, enforce contract floors │  ◄── LIVE
   │  L2  Rulebook Scoring    this league's exact scoring matrix    │  ◄── store live
   │  L3  Age Decay           position-specific aging curves        │  ◄── LIVE
   │  L4  Scouting Layer      Film × RAS × Breakout (Madden-checked)│  ◄── 10/10 LIVE
   │  L5  Cap Efficiency      value per dollar, by cap tier         │  ◄── LIVE
   │  L6  Tiebreaker          tenure → RAS → positional scarcity    │  ◄── LIVE
   └──────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                       ⭐ Adjusted Score  (one number per player)
```

**Ten position models, each individually calibrated** — because a shutdown corner and a power back aren't graded on the same curve. **Every one is live and scoring:**

| Position | Model | What makes it its own thing |
|:--|:--:|:--|
| 🎯 **QB** | ✅ live | RAS held neutral — the position where athleticism lies least |
| 🏃 **RB** | ✅ live | Medium-tier athletic weighting; aggressive early-career breakout curve |
| 🙌 **WR** | ✅ live | Breakout age is gold — the highest breakout weighting of any position |
| 🧤 **TE** | ✅ live | First athletic-longevity modulator; steepest athletic curve |
| 🧱 **DT** | ✅ live | Film compression + a late-career "Cushion Guard" for elite athletes |
| 💨 **DE** | ✅ live | College pass-rush production share — the cleanest pre-NFL signal |
| 🛡️ **LB** | ✅ live | Most scheme-dependent position; college production weighted highest |
| 🔒 **CB** | ✅ live | NGS coverage metrics anchor the grade — passer rating allowed, not hype |
| 🦅 **S** | ✅ live | NGS-anchored twin of CB; tuned for the box-to-deep athletic spread |
| 🦵 **K** | ✅ live | Madden-driven film on its own scoring path entirely |

Under the hood: athletic testing (RAS) is weighted *per position* — gold at receiver, neutral at quarterback. NGS coverage metrics anchor the cornerback and safety models. Madden ratings *regulate* subjective scouting takes so hype doesn't leak into the score. A late-career "Cushion Guard" keeps elite athletes from falling off a cliff on paper before they do on the field. And every model reuses the same proven mechanics — the architecture **composes**, so each new position was a calibration, not a rebuild. That's why 10 of 10 shipped.

---

## 🔧 Pillar 2 — The Transaction Engine · 🔥 LIVE

The thing every other tool leaves you to do by hand. Nine transaction types, fully validated, with the math done for you — and the first ones **already execute against real league state:**

`UFA bidding` · `RFA offer sheets` · `Waivers` · `Trades` · `Trade block` · `Franchise tag` · `Extensions` · `Restructures` · `Buyouts`

- ✅ **Trades — LIVE & atomic.** A multi-leg swap either lands *entirely* or rolls back *whole*. No half-executed trade can ever exist; a single bad leg reverts everything. Verified on real hardware.
- ✅ **Waiver cut + dead-cap auto-calculator — LIVE.** Cut a player and the §8 penalty (`35% × salary × remaining years`, 50% on a restructured deal) is computed and charged to your cap **automatically** — no more manual arithmetic on every release. Watched move real money on the live league.
- ✅ **Exact money, everywhere.** Every dollar is stored as integer cents — no floating-point drift, reconciled to the penny. The cap is never "about right."
- 🔜 **Franchise tag · Extensions · Restructures · Buyouts** — the contract-ops suite, up next.
- ⏱️ **UFA/RFA bidding · 24-hour clock · snipe detection** *(Phase 2)* — the 20-hour rule, enforced by a timestamp.
- 🗳️ **DOT vote tracking** *(Phase 2)* — closes at 3-0 either way.
- 🚫 **Trade-deadline hard block** — a 7-minute miss is caught by a clock, not a human.

> It removes the friction. It never replaces the judgment. The trade analyzer surfaces value — the DOT still decides what's good for the league.

**How it stays safe:** a *single* transaction coordinator is the only thing in the entire system allowed to change who-owns-what. Every mutation runs inside one atomic database transaction. Everything else can only read. That is the difference between a spreadsheet and a system of record.

---

## 📊 Pillar 3 — The Modules

The war room itself. Eight views into the engine, plus two admin surfaces. **M1 is live — the rest are the road ahead:**

| Module | What it does | |
|---|---|:--:|
| **M1 · Asset Rankings** | All 32 rosters ranked into one league-wide board, with per-team drill-down and cap-efficiency views | ✅ **live** |
| **M2 · Power Rankings** | Weekly team strength — roster value, trend, and strength of schedule, not just record | ▸ ahead |
| **M3 · Matchup Predictions** | Projected scores with floor/ceiling and boom-bust flags | ▸ ahead |
| **M4 · Transaction UI** | The Pillar 2 interface — every transaction type in-app | 🔧 building |
| **M5 · Free Agency Intel** | The FA pool ranked, with team-need overlays and live bid tracking | ▸ ahead |
| **M6 · Rookie Draft Intel** | Prospect boards with the scouting layer applied, live draft board | ▸ ahead |
| **M7 · Trade Analyzer** | Side-by-side value, cap impact, and historical comps — for both teams and the DOT | ▸ ahead |
| **M8 · Commissioner Dashboard** | League health, compliance flags, and the pending-decision queue at a glance | ▸ ahead |

The engine is the hard part, and the engine is done. The modules are the payoff — each one is a **view** onto a score that already exists.

---

## 🎛️ Pillar 4 — The Admin Console

The engine is only as good as its calibration. Every parameter that *should* be tuned against real results — scouting weights, S-curve shapes, age curves, cap tiers, the scarcity matrix — is exposed in the admin console. **This isn't a future promise: the live tuning loop already works** — change a parameter in the sidebar today and the validation board re-ranks in front of you. Tune the engine through the UI; the structural mechanics stay code-locked. That's also what makes Phase 3 possible: every league installs and tunes its own instance.

---

## 🛡️ Built to Last

This isn't a weekend hack. The architecture is governed by hard rules so it stays maintainable as it grows — and those rules are **already enforced in the code that's shipping today:**

- **A three-layer law** — real-football data is read-only; the app owns its logic; users mutate state *only* through validated transactions. No layer bleeds into another.
- **One writer to league state** — a single coordinator is the only thing that can change who-owns-what, and every change is one atomic transaction. Everything else reads. A planted test *proves* the read-only handle can't be cast back into a writer.
- **A pure engine** — the entire scoring pipeline imports no database, no network, no clock. Inputs come in as parameters; a score comes out. A custom linter fails the build if anything dirties it.
- **Money is exact by construction** — every dollar is integer cents, never a float; a value parsed from MFL never round-trips through floating point. The cap can't drift.
- **Historical records are immutable** — every score is stamped with its scoring config; the dead-cap ledger is append-only and the database itself rejects an edit. Change the engine, and last season stays exactly as it was scored. The record is the record.
- **Enforced by the compiler, not by hope** — architectural rules are wired into the type system and custom linters, so a violation is a *build failure*, not a code-review note. (Player IDs literally cannot be forged — the bypass doesn't compile.)
- **Fail loud, never silent** — every fetcher already caught a real bug a linter couldn't: MFL collapses single-element arrays, returns HTTP 200 with an error body, and omits commissioner-created players. Each one would have silently corrupted league data. Each one is now guarded and tested.
- **Reviewed by an adversary** — every build is checked by an **independent blind AI reviewer** whose findings are treated as leads to triage against the source, never rubber-stamped. It has earned its keep with real catches.
- **A 38-session build plan** — every session sized to one context window, every session closes with a ready-to-go handoff. Spaghetti has nowhere to hide.

📋 Full build sequence: **[`docs/build-handoffs/Build_Tracker.md`](docs/build-handoffs/Build_Tracker.md)**

---

## 🛣️ The Road

| Phase | What it is | Who it's for |
|:---:|---|---|
| **1** | Personal tool — all 32 teams, one GM's edge | Christopher |
| **2** | League-wide alpha — multi-user, live bidding, mobile | The Legacy NFL |
| **3** | Self-hosted beta — any MFL league runs its own instance | The world |
| **4** | Long-term — computer-use automation, historical analytics, offseason cap planner | Someday |

*No deadline. No defined endpoint. It grows as the league grows.*

---

## 🧰 Stack

**Go** scoring engine + service layer · **Wails v2** desktop shell · **React + Tailwind + Zustand** frontend · **SQLite** (WAL) storage · **MyFantasyLeague** API integration.

Phase 1 ships as a desktop app. Phase 2 opens it to the league. Phase 3 sets it free.

---

## 📍 Status — Engine Complete, Transactions Live

**The full scoring spine is done: data pipeline, all four logic stores, the six-layer engine, all ten position models, and the live rankings board. The transaction engine is now executing — trades commit atomically, and cutting a player charges §8 dead cap to the cap automatically. From here, it's contract ops and the eight war-room modules that turn scores into decisions.**

| ✓ Done & verified | ▸ Up next |
|---|---|
| **B0–B3** — Pipeline live (1,217 players · 32 teams · type system locked) | **B7c** — Contract ops: tag · extension · restructure · buyout |
| **B2b** — Scouting schema + 21 sources, all 10 positions | **Free agency** — the FA pool + UFA/RFA bidding |
| **B3b/c · B4 · B6** — Rulebook · state · parameter · **immutable output** stores | **M2–M8** — power rankings, matchups, trade analyzer, commissioner |
| **B5a** — Six-layer engine pipeline (pure, fail-loud) | **Phase 2** — multi-user, live clock, DOT voting, mobile |
| **B5b** — **all 10 position models** scoring (QB·RB·WR·TE·DT·DE·LB·CB·S·K) | |
| **M1** — Asset Rankings: the real-data 32-team board, live | |
| **B7a/b** — Transaction coordinator: atomic trades + waiver dead cap | |

```
[█████████████████████░░░░░░░░]  27 / 38 sessions  ·  data ✓  engine ✓  models 10/10 ✓  board ✓  trades ✓  →  contract ops + modules
```

Every layer ships only after a build passes clean, an **independent blind AI review** signs off, and it's verified against **real league data** on real hardware. No layer is "done" until it's been *used*, not just compiled.

---

<div align="center">

*Built by Christopher Campbell with Claude (Anthropic), with contributions from GLM, Gemini, and Ornith.*
**MFL gives you the league. TheWarRoom helps you win it.**

</div>
