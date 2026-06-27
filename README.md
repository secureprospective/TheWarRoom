<div align="center">

# 🏈 TheWarRoom

### A dynasty fantasy football intelligence engine that scores what real players do in real games — and out-thinks the stock platforms doing it.

[![Status](https://img.shields.io/badge/status-🔥%20ENGINE%20LIVE%20·%20scoring%20players-brightgreen)](docs/build-handoffs/Build_Tracker.md)
[![Progress](https://img.shields.io/badge/build-20%20of%2038%20sessions%20·%2053%25%20·%20past%20halfway-success)](docs/build-handoffs/Build_Tracker.md)
[![Models](https://img.shields.io/badge/position%20models-7%20of%2010%20scoring-blue)](#-pillar-1--the-scoring-engine)
[![Verified](https://img.shields.io/badge/live%20data-1217%20players%20·%2032%20teams%20·%20verified-blueviolet)](#-its-alive--from-raw-data-to-a-score)
[![Phase](https://img.shields.io/badge/phase-1%20·%20personal%20tool-orange)](#️-the-road)

[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails%20v2-DF0000?logo=wails&logoColor=white)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB)](https://react.dev)
[![Tailwind](https://img.shields.io/badge/Tailwind-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com)
[![SQLite](https://img.shields.io/badge/SQLite%20WAL-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)

*32 teams. Six scoring layers. Ten position models. Twenty-one scouting sources. **One number per player.***

</div>

---

## ⚡ The Pitch

MyFantasyLeague gives you a roster and a box score. **TheWarRoom gives you an edge.**

It ingests live league data from MFL, fuses it with twenty-one professional scouting sources, and runs every rostered player through a **six-layer valuation engine** built on this league's exact ruleset. The output is a single **Adjusted Score** per player — and a war room full of tools that turn that score into decisions: who to bid on, what a trade is really worth, where your roster is thin, and what a contract costs you three years from now.

It also replaces the part nobody enjoys: the bid math, the dead-cap arithmetic, the snipe clock, the trade-deadline policing. The machine handles the mechanical. The humans keep the judgment.

> **The engine's core trick:** it separates *"is this player good?"* from *"how long does he have?"* — scoring talent and aging as two different jobs, the way a real scout does. An elite receiver in his prime and a declining veteran can post the same box score; TheWarRoom prices them apart.

---

## 🔥 It's Alive — From Raw Data to a Score

Six months ago this was a plan. **Today it pulls a real league, scores real players, and ranks them on screen.** The whole spine — data in, engine in the middle, a number out, on a desktop board you can drive — is built, merged, and verified end-to-end.

```
   real MFL league ──► transport ──► ingestion ──► normalize ──► ENGINE ──► Adjusted Score ──► ranked board
      (live API)       rate-limited   boundary-     raw→domain    6 layers    one number        on screen,
                       host-routed    guarded       talent≠age    7 models    per player        live-tunable

   ✅ DATA   1,217 rostered players · 32 franchises · zero loss · dead-cap reconciled to the penny
   ✅ ENGINE pure-function pipeline · fail-loud on bad input · nothing hardcoded · compiler-enforced layers
   ✅ MODELS QB · RB · WR · TE · DT · DE · LB  scoring live  (CB · S · K next)
   ✅ PROOF  every build: green tests + blind AI review + used live on real hardware before it counts as "done"
```

This isn't a mock or a fixture. It's the **actual Legacy NFL league**, pulled live, normalized into a locked type system, run through the scoring pipeline, and rendered on a board that re-ranks the instant you change a calibration knob. The hard parts of any data product — clean ingestion, a trustworthy engine, a working UI loop — are **done and proven.**

---

## 📈 Build Progress — Past the Halfway Line

```
Tier 0  Scaffold            [██████████]  1/1    ✅ DONE
Tier 1  Data Pipeline       [██████████]  7/7    ✅ DONE   — live, end-to-end
Tier 2  Logic Engine        [██████████]  5/5    ✅ DONE   — stores + pipeline + harness
Tier 2  Position Models     [███████░░░]  7/10   🔥 LIVE   — QB RB WR TE DT DE LB · CB S K next
Tier 2  Output Store        [░░░░░░░░░░]  0/1    ▸ next
Tier 3  Modules + Trades    [░░░░░░░░░░]  0/14   ▸ ahead

         OVERALL            [████████████████░░░░░░░░░░░░░░]   20 / 38   ·   53%
```

| Milestone | Status |
|---|---|
| 🏗️ **Foundation** — Go · Wails · React · SQLite WAL, compiler-enforced architecture | ✅ Shipped |
| 🔌 **Data pipeline** — MFL transport → ingestion → normalization, live against the real league | ✅ Shipped |
| 📋 **Scouting schema** — all 10 positions, 21 sources wired and fetching | ✅ Shipped |
| 🗄️ **Logic stores** — rulebook, league state, admin parameters (versioned, single-writer) | ✅ Shipped |
| ⚙️ **The engine** — six-layer pure-function scoring pipeline | ✅ Shipped |
| 🧪 **Testing harness** — three-state validation board, live admin tuning loop | ✅ Shipped |
| 🧠 **Position models** — 7 of 10 calibrated and scoring | 🔥 In flight |
| 💾 **Output store → Modules → Transactions** | ▸ The back half |

---

## 🧠 Pillar 1 — The Scoring Engine

The valuation brain. Six layers, run in order, every value MFL-sourced or admin-tunable — nothing hardcoded.

```
              MFL API  +  21 scouting sources (PFF · RAS · NGS · Madden · film)
                                      │
                                      ▼
   ┌──────────────────────────────────────────────────────────────┐
   │  L1  Data Hygiene        clean inputs, enforce contract floors │  ◄── LIVE
   │  L2  Rulebook Scoring    this league's exact scoring matrix    │  ◄── store live
   │  L3  Age Decay           position-specific aging curves        │  ◄── LIVE
   │  L4  Scouting Layer      Film × RAS × Breakout (Madden-checked)│  ◄── 7/10 LIVE
   │  L5  Cap Efficiency      value per dollar, by cap tier         │  ◄── LIVE
   │  L6  Tiebreaker          tenure → RAS → positional scarcity    │  ◄── LIVE
   └──────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                       ⭐ Adjusted Score  (one number per player)
```

**Ten position models, each individually calibrated** — because a shutdown corner and a power back aren't graded on the same curve. Seven are live and scoring:

| Position | Model | What makes it its own thing |
|:--|:--:|:--|
| 🎯 **QB** | ✅ live | RAS held neutral — the position where athleticism lies least |
| 🏃 **RB** | ✅ live | Medium-tier athletic weighting; aggressive early-career breakout curve |
| 🙌 **WR** | ✅ live | Breakout age is gold — the highest breakout weighting of any position |
| 🧤 **TE** | ✅ live | First athletic-longevity modulator; steepest athletic curve |
| 🧱 **DT** | ✅ live | Film compression + a late-career "Cushion Guard" for elite athletes |
| 💨 **DE** | ✅ live | College pass-rush production share — the cleanest pre-NFL signal |
| 🛡️ **LB** | ✅ live | Most scheme-dependent position; college production weighted highest |
| 🔒 **CB** | ▸ next | NGS coverage metrics anchor the grade |
| 🦅 **S** | ▸ next | NGS-anchored; free vs. box safety roles |
| 🦵 **K** | ▸ next | Madden-driven; its own scoring path entirely |

Under the hood: athletic testing (RAS) is weighted *per position* — gold at receiver, neutral at quarterback. NGS coverage metrics will anchor the cornerback and safety models. Madden ratings *regulate* subjective scouting takes so hype doesn't leak into the score. A late-career "Cushion Guard" keeps elite athletes from falling off a cliff on paper before they do on the field. And every model reuses the same proven mechanics — the architecture **composes**, so each new position is a calibration, not a rebuild.

---

## 🔧 Pillar 2 — The Transaction Engine

Nine transaction types, fully validated, with the math done for you:

`UFA bidding` · `RFA offer sheets` · `Waivers` · `Trades` · `Trade block` · `Franchise tag` · `Extensions` · `Restructures` · `Buyouts`

- ✅ **Dead-cap auto-calculator** — no more manual `salary × 35% × years` on every waiver
- ✅ **Bid validation** — wrong term, bad increment, over-cap, over-roster: rejected before it's ever posted
- ⏱️ **24-hour clock + snipe detection** *(Phase 2)* — the 20-hour rule, enforced
- 🗳️ **DOT vote tracking** *(Phase 2)* — closes at 3-0 either way
- 🚫 **Trade-deadline hard block** — a 7-minute miss is caught by a timestamp, not a human

> It removes the friction. It never replaces the judgment. The trade analyzer surfaces value — the DOT still decides what's good for the league.

---

## 📊 Pillar 3 — The Modules

The war room itself. Eight views into the engine, plus two admin surfaces:

| Module | What it does |
|---|---|
| **M1 · Asset Rankings** | All 32 rosters ranked into one league-wide board, with per-team and cap-efficiency views |
| **M2 · Power Rankings** | Weekly team strength — roster value, trend, and strength of schedule, not just record |
| **M3 · Matchup Predictions** | Projected scores with floor/ceiling and boom-bust flags |
| **M4 · Transaction UI** | The Pillar 2 interface — every transaction type in-app |
| **M5 · Free Agency Intel** | The FA pool ranked, with team-need overlays and live bid tracking |
| **M6 · Rookie Draft Intel** | Prospect boards with the scouting layer applied, live draft board |
| **M7 · Trade Analyzer** | Side-by-side value, cap impact, and historical comps — for both teams and the DOT |
| **M8 · Commissioner Dashboard** | League health, compliance flags, and the pending-decision queue at a glance |

---

## 🎛️ Pillar 4 — The Admin Console

The engine is only as good as its calibration. Every parameter that *should* be tuned against real results — scouting weights, S-curve shapes, age curves, cap tiers, the scarcity matrix — is exposed in the admin console. **This isn't a future promise: the live tuning loop already works** — change a parameter in the sidebar today and the validation board re-ranks in front of you. Tune the engine through the UI; the structural mechanics stay code-locked. That's also what makes Phase 3 possible: every league installs and tunes its own instance.

---

## 🛡️ Built to Last

This isn't a weekend hack. The architecture is governed by hard rules so it stays maintainable as it grows — and those rules are **already enforced in the code that's shipping today:**

- **A three-layer law** — real-football data is read-only; the app owns its logic; users mutate state *only* through validated transactions. No layer bleeds into another.
- **One writer to league state** — a single coordinator is the only thing that can change who-owns-what. Everything else reads. A planted test *proves* the read-only handle can't be cast back into a writer.
- **A pure engine** — the entire scoring pipeline imports no database, no network, no clock. Inputs come in as parameters; a score comes out. A custom linter fails the build if anything dirties it.
- **Historical scores are immutable** — every score is stamped with its scoring config. Change the engine, and last season stays exactly as it was scored. The record is the record.
- **Enforced by the compiler, not by hope** — architectural rules are wired into the type system and custom linters, so a violation is a *build failure*, not a code-review note. (Player IDs literally cannot be forged — the bypass doesn't compile.)
- **Fail loud, never silent** — every fetcher already caught a real bug a linter couldn't: MFL collapses single-element arrays, returns HTTP 200 with an error body, and omits commissioner-created players. Each one would have silently corrupted league data. Each one is now guarded and tested.
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

## 📍 Status — Engine Lit, Models Filling In

**The pipeline is live, the engine scores, and 7 of 10 position models are calibrated. The next three close out scoring; then the output store and the modules turn scores into decisions.**

| ✓ Done & verified | ▸ Up next |
|---|---|
| **B0–B3** — Pipeline live (1,217 players · 32 teams · type system locked) | **B5b-CB / S / K** — the last 3 position models |
| **B2b** — Scouting schema + 21 sources, all 10 positions | **B6** — Per-season output store (immutable, config-stamped) |
| **B3b/c/B4** — Rulebook · league-state · parameter stores | **M1** — Asset Rankings: the first real-data board |
| **B5a** — Six-layer engine pipeline (pure, fail-loud) | **B7** — Transaction engine + dead-cap coordinator |
| **Harness** — three-state validation board + live tuning loop | **M2–M9** — the rest of the war room |
| **B5b** — 7 of 10 position models scoring (QB·RB·WR·TE·DT·DE·LB) | |

```
[████████████████░░░░░░░░░░░░░░]  20 / 38 sessions   ·   data ✓   engine ✓   models 7/10   →   output + modules
```

Every layer ships only after a build passes clean, an **independent blind AI review** (GLM 5.2) signs off, and it's verified against **real league data** on real hardware. No layer is "done" until it's been *used*, not just compiled.

---

<div align="center">

*Built by Christopher Campbell with Claude (Anthropic).*
**MFL gives you the league. TheWarRoom helps you win it.**

</div>
