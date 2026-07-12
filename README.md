<div align="center">

# 🏈 TheWarRoom

### Local-First Data Aggregation & Custom Ranking Engine for Elite Dynasty Leagues

**A compiled Go intelligence center that ingests a live 32-team dynasty league, fuses it with 21 pro scouting sources, and prices every player, every contract, and every cap consequence — on your machine, at native speed, with the math done for you.**

<br/>

[![Powered by Go](https://img.shields.io/badge/Powered%20by-Go%201.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Multi-Block Architecture](https://img.shields.io/badge/Architecture-Multi--Block%20·%20Compiler%20Enforced-4B5563)](#-4--the-war-room-architecture)
[![Local-First](https://img.shields.io/badge/Local--First-Zero%20Cloud%20·%20Total%20Privacy-brightgreen)](#-2--the-problem--the-solution)
[![Ledger](https://img.shields.io/badge/Salary%20Ledger-Per--Year%20Cells%20·%20Sole%20Cap%20Truth-gold)](#-3--feature-showcase)
[![Money](https://img.shields.io/badge/Money-int64%20cents%20·%20exact%20to%20the%20penny-9cf)](#-3--feature-showcase)
[![Position Models](https://img.shields.io/badge/Position%20Models-10%20of%2010%20·%20LIVE-blue)](#-3--feature-showcase)

[![Wails v2](https://img.shields.io/badge/Wails%20v2-DF0000?logo=wails&logoColor=white)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB)](https://react.dev)
[![Tailwind](https://img.shields.io/badge/Tailwind-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com)
[![SQLite WAL](https://img.shields.io/badge/SQLite%20WAL-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)
[![MFL](https://img.shields.io/badge/MyFantasyLeague-live%20ingestion-orange)](https://home.myfantasyleague.com)

*32 teams. Six scoring layers. Ten position models. Twenty-one scouting sources. **One number per player.***

</div>

---

## ⚡ 1 · The Hook

> MyFantasyLeague gives you a roster and a box score.
> **TheWarRoom gives you a front office.**

Stock fantasy platforms and fragile spreadsheets fall apart the instant a league gets *serious* — 32 teams, per-year salary caps, multi-year contracts, dead-cap penalties, franchise tags, IDP scoring, and a decade of history that all has to reconcile to the penny. That's not a fantasy app's job anymore. That's a **data system**.

TheWarRoom is that system: a local-first, compiled Go engine that pulls your live league, runs every rostered player through a **six-layer valuation pipeline** built on *your* exact ruleset, and turns the result into a single **Adjusted Score** — then hands you a war room of tools that convert that score into decisions. Who to bid on. What a trade is really worth. What a contract costs you three years from now. And it executes those moves *atomically*, with the cap arithmetic computed and applied automatically.

No cloud. No latency. No data leaving your machine. **Just an edge.**

---

## 🎯 2 · The Problem & The Solution

### The problem the stock tools can't touch

A hyper-complex dynasty league is a **hard data problem wearing a football jersey**:

- **Scale** — 32 franchises, 1,200+ rostered players, deep IDP + offensive scoring across ten distinct position archetypes.
- **Contracts** — multi-year deals, per-year cap hits, franchise tags, restructures, extensions, buyouts, and dead-cap penalties that ripple forward through *future* seasons.
- **Truth** — the cap has to be *exact*. "About right" is a corrupted league. Floating-point drift is not acceptable when real trades hinge on the number.
- **Disparate signal** — MFL box scores, PFF grades, RAS athleticism, NGS coverage metrics, Madden film, breakout ages — twenty-one sources that don't agree on IDs, formats, or units.

Spreadsheets buckle. SaaS platforms hardcode *their* scoring and never expose *yours*. Neither shows its work.

### The solution

**A localized, blazingly fast Go intelligence center** that gives one manager a professional-grade front-office advantage right from their own machine:

```
   DISPARATE, MESSY SIGNAL        ───►     ONE UNIFIED LOCAL STATE     ───►     ONE NUMBER, ONE DECISION
   MFL · PFF · RAS · NGS · film            normalized · type-locked            Adjusted Score → the call
```

The engine separates *"is this player good?"* from *"how long does he have?"* — grading talent and aging as two different jobs, the way a real scout does. An elite receiver in his prime and a declining veteran can post the same box score. **TheWarRoom prices them apart.**

---

## 🚀 3 · Feature Showcase

### 🛰️ Multi-Block Ingestion Engine — *live, end-to-end*
Automated, robust data pipelines that pull the **real MFL league** over a rate-limited, host-routed transport, guard every boundary (MFL silently collapses single-element arrays, returns HTTP 200 with error bodies, and omits commissioner-created players — each trap is caught and tested), and normalize wildly disparate inputs into a single **type-locked domain state**. `1,217 players · 32 franchises · zero loss · reconciled to the penny.`

### ⚙️ Custom Valuation Engine — *6 layers, 10 models, all calibrated*
A mathematically flexible framework that crunches *this league's* exact scoring matrix — nothing hardcoded, every value MFL-sourced or admin-tunable. Six pure-function layers run in order; **ten position models**, each individually calibrated, because a shutdown corner and a power back are not graded on the same curve.

```
   L1 Data Hygiene   → L2 Rulebook Scoring → L3 Age Decay
   L4 Scouting Layer → L5 Cap Efficiency   → L6 Tiebreaker
                          ▼
                 ⭐ Adjusted Score  (one number per player)

   QB · RB · WR · TE · DT · DE · LB · CB · S · K   →  10 / 10 LIVE
```

RAS athleticism is weighted *per position* (gold at receiver, neutral at quarterback). NGS coverage metrics anchor the CB/S models. Madden ratings *regulate* subjective takes so hype can't leak into the score. The architecture **composes** — each new position was a calibration, not a rebuild.

### 📒 The Per-Year Salary Ledger — *the sole source of cap truth*
Every contract is a row of **per-year cells**. Every change is an append-only, dated, immutable audit entry — the database itself rejects an edit to history. The cap is *derived* from the cells (`CapUsed = Σ paid cells + Σ dead cap − Σ cap relief`, floored at 0), never stored as a competing number. **Money is `int64` cents on a flat $10k grid** — no floating-point drift, exact by construction.

### 🔧 Atomic Transaction Engine — *the mechanical work, automated*
A **single transaction coordinator** is the only thing in the entire system allowed to change who-owns-what; everything else can only read. Every mutation runs inside one spanning database transaction — a multi-leg trade lands *entirely* or rolls back *whole*.

| Live & executing | What it does automatically |
|---|---|
| ✅ **Trades** | Atomic multi-leg swaps — no half-executed trade can exist |
| ✅ **Waivers / cuts** | §8 dead cap (`35% × salary × years left`) computed & charged on the spot |
| ✅ **Franchise tag (§9)** | Top-5-by-position pricing, 120%-of-prior floor — UI sends a player id, never a dollar |
| ✅ **Restructure (§11)** | Owner cap moves bounded by the rulebook — a violation is *unrepresentable* |
| ✅ **Extension §10 · Buyout §12 · special situations §13–§14** | Full contract rulebook, phase-gated |
| ✅ **Free Agency §6 (v1)** | Live free-agent pool + record-a-signing, with §12 buyout lockout, min-salary floor & UFA promotion on rollover |
| ✅ **Commissioner UFA calendar (§6)** | A signing window the commissioner opens/closes on top of the phase gate — closed blocks every signing, and it persists until toggled |

**🖥️ The operator workspace** — a subject-centric front office drives it all: pick a franchise by its *real team name* → see its roster by *real player name* → click a player and the panel offers **only the moves that are legal this phase** (an offseason-only buyout simply isn't there mid-season). Priced moves — cut, tag, extend, restructure, buyout, sign — **quote before they commit**: the engine dry-runs the *real* handler and rolls it back, so you see "this will commit" or the authoritative rejection reason *before* anything is written. You never type a player id or a dollar figure — the UI sends the intent, the engine computes the money.

**🔁 The trade builder** — its own surface, because a trade is the only move that spans *multiple* franchises. Browse any team's roster, add players to a cart, set each one's destination, and stage a single **atomic multi-leg swap** — the same quote-before-commit gate confirms the whole trade lands together or rolls back whole.

**🎖️ Commissioner controls** — the league-calendar and off-common-path powers live on their own surface, segregated from the per-player workspace: advance the season phase, roll the season over (§14), open or close the free-agency signing window (§6), and — under a red, irreversible divider — retirement, death, and cap-relief appeals (§13). Every one runs through the same dry-run-then-confirm gate.

*The contract ops (cut · tag · extend · restructure · buyout · sign) are verified end-to-end through the live operator workspace on real hardware — not just unit-tested. The trade builder and commissioner surfaces are built and awaiting the same live-hardware gate.*

### ⚡ Go-Powered, Local-First Performance
Zero cloud latency. Absolute privacy. Near-instant processing from a clean, compiled backend. The rankings board **re-ranks the instant you turn a calibration knob** in the admin console — the live tuning loop already works. Native desktop app; your league never leaves your machine.

---

## 📈 Build Progress — *engine done · contract rulebook done · the operator front office is going in*

A disciplined, session-by-session build — every session sized to a single context window, every session closed only after green tests, a **blind AI code review**, and real-hardware verification. Forty-plus sessions in, the whole logic core is shipped and the operator UI that drives it is well underway. Here's the board:

```
   Foundation & Scaffold      [██████████]  100%   ✅  Go · Wails · React · SQLite WAL · compiler-enforced arch
   Multi-Block Data Pipeline  [██████████]  100%   ✅  live MFL ingest → normalize → type-locked state
   Logic Stores               [██████████]  100%   ✅  rulebook · state · params · immutable output
   Scoring Engine (6 layers)  [██████████]  100%   ✅  pure, fail-loud pipeline
   Position Models            [██████████]  100%   ✅  QB·RB·WR·TE·DT·DE·LB·CB·S·K  — 10 / 10 calibrated
   Asset Rankings Board (M1)  [██████████]  100%   ✅  real 32-team board, live re-rank on tune
   Salary Ledger (cutover)    [██████████]  100%   ✅  per-year cells = sole cap truth, append-only
   Transaction Rulebook       [██████████]  100%   ✅  trades · §8–§14 contract ops · §6 free agency v1
   Operator Workspace (M4)    [████████░░]  ~80%   ▸   phase-legal quote→commit · contract ops · trade builder · commish
   War-Room Modules (M2–M8)   [██░░░░░░░░]  ~15%   ▸   power rankings · matchups · trade analyzer · shadow ledger
   Admin / Calibration UIs    [███░░░░░░░]  ~25%   ▸   engine tuning + governance surfaces

   ────────────────────────────────────────────────────────────────────────────────────────────
   OVERALL   [████████████████████████░░░░]   logic core 100%   ·   operator UI in progress   ·   ~85%
```

| Layer | Milestone | Status |
|:--|:--|:--:|
| 🏗️ | **Foundation** — Go · Wails · React · SQLite WAL, compiler-enforced architecture | ✅ Shipped |
| 🛰️ | **Multi-block pipeline** — MFL transport → ingestion → normalize, live vs the real league | ✅ Shipped |
| 🗄️ | **Logic stores** — rulebook · state · params · **immutable output** | ✅ Shipped |
| ⚙️ | **The engine** — six-layer pure-function scoring pipeline | ✅ Shipped |
| 🧠 | **Position models** — **all 10 calibrated and scoring** | ✅ Shipped |
| 📊 | **Asset Rankings (M1)** — all 32 rosters ranked on screen from real data | ✅ Shipped |
| 📒 | **The salary ledger** — per-year cells, sole cap truth, append-only audit trail | ✅ Shipped |
| 🔧 | **Transaction rulebook** — atomic trades, §8–§14 contract ops, **§6 free agency v1** | ✅ Shipped |
| 🖥️ | **Operator workspace (M4)** — phase-legal quote→commit UI: contract ops **shipped**; trade builder + commissioner controls **built, gating** | ◕ In progress |
| 📊 | **War-room modules (M2–M8)** — the analytical views that turn scores into calls | ▸ The back half |
| 🎛️ | **Admin / calibration UIs** — tune the engine, govern the rules | ▸ Ahead |

📋 Full session-by-session ledger: **[`docs/build-handoffs/Build_Tracker.md`](docs/build-handoffs/Build_Tracker.md)** — *the engine and the entire contract rulebook are done; the operator UI that surfaces them is being built out slice by slice.*

---

## 🏛️ 4 · The War Room Architecture

A multi-block system design where the boundaries aren't conventions — they're **compiler-enforced law**. A violation is a *build failure*, not a code-review note.

```
   ┌──────────────────────────────────────────────────────────────────────────┐
   │                          THE DESKTOP INTERFACE                            │
   │              Wails v2  ·  React + Tailwind + Zustand                      │
   │      Asset Rankings · Operator workspace · Trade builder · Commissioner      │
   └───────────────────────────────┬──────────────────────────────────────────┘
                                    │  IPC (typed, one-way: UI reads / requests)
   ┌───────────────────────────────▼──────────────────────────────────────────┐
   │                       THE TRANSACTION COORDINATOR                          │
   │        the ONLY writer to league state · one atomic tx per op             │
   │        default-deny phase gate · derived cap · append-only ledger         │
   └───────────────┬───────────────────────────────────────┬──────────────────┘
                   │                                         │
   ┌───────────────▼───────────────┐         ┌───────────────▼──────────────────┐
   │      THE CALCULATION ENGINE    │         │       THE STORAGE LAYER           │
   │   6-layer PURE pipeline        │         │   SQLite (WAL) · split R/W pools  │
   │   no db · no net · no clock    │◄────────│   rulebook · state · params ·     │
   │   inputs in → score out        │  params │   immutable output · ledger       │
   └───────────────▲────────────────┘         └───────────────▲──────────────────┘
                   │                                           │
   ┌───────────────┴───────────────────────────────────────────┴────────────────┐
   │                     THE MULTI-BLOCK INGESTION LAYER                          │
   │   MFL transport → ingestion → normalize → playerid resolve → domain types   │
   │        21 scouting sources · boundary-guarded · fail-loud · read-only       │
   └─────────────────────────────────────────────────────────────────────────────┘
```

**The laws that hold it together:**

- **Three-layer separation** — real-football data is read-only; the app owns its logic; users mutate state *only* through validated transactions. No layer bleeds into another.
- **One writer** — a single coordinator changes league state; a planted test *proves* the read-only handle can't be cast back into a writer.
- **A pure engine** — the scoring pipeline imports no database, no network, no clock. A custom linter fails the build if anything dirties it.
- **Immutable history** — every score is stamped with its scoring config; ledgers are append-only and SQLite triggers reject edits. Change the engine, and last season stays exactly as it was scored.
- **Forgery-proof IDs** — player IDs literally cannot be fabricated; the bypass doesn't compile.

📐 Full system map: **[`SYSTEM_MAP.md`](SYSTEM_MAP.md)** · Build sequence: **[`docs/build-handoffs/Build_Tracker.md`](docs/build-handoffs/Build_Tracker.md)**

---

## 🗺️ 5 · The Draft Board — Roadmap

The engine is the hard part, and the engine is **done** — and the front office that operates it is already going in (roster moves, contract ops, trades, and commissioner controls all run live against the real ledger). What's ahead is the payoff on top: each remaining capability a new *analytical view* onto a score and a ledger that already exist.

**On the clock**
- 📊 **The war-room modules** — Power Rankings, Matchup Predictions, Trade Analyzer, Free-Agency Intel, Rookie Draft board, Commissioner Dashboard.
- 🕳️ **The shadow ledger** — dry-run any transaction against a *forked* cap before you commit it.

**Later rounds — the three horizons**
- 🗣️ **Horizon 1 · The Capologist** — a chat interface on a small **local** model. *"What's my dead cap in 2028 if I cut him?"* answered from the actual ledger, **with receipts**. Numbers never come from the model; the ledger disposes; you decide.
- 💼 **Horizon 2 · The Portfolio Desk** — one engine valuing the same player under *five* leagues' rules simultaneously. Asset management for dynasty football.
- 🎖️ **Horizon 3 · The January War Room** — **fork your franchise** and branch offseason futures like a developer branches code, each run through the *real* transaction engine — then your winning plan becomes the season's execution script.

🔭 The full vision, on the record: **[`docs/roadmap/Vision_2026.md`](docs/roadmap/Vision_2026.md)**

---

## 🤝 How It Gets Built — The Council

One human holds the vision and the veto. A council of AIs does the rest, each in the seat it's best in. No model gets rubber-stamped and no model gets to guess — every finding is a *lead*, triaged against the source. The machine proposes; the human and the rulebook decide.

<div align="center">

[![Claude](https://img.shields.io/badge/Claude-The%20Builder%20·%20Head%20Brain-D97757?style=for-the-badge&logo=anthropic&logoColor=white)](https://anthropic.com)
[![GLM](https://img.shields.io/badge/GLM%205.2-The%20Blind%20Reviewer-6E3AF2?style=for-the-badge)](https://z.ai)
[![Gemini](https://img.shields.io/badge/Gemini-The%20Second%20Opinion-8E75B2?style=for-the-badge&logo=googlegemini&logoColor=white)](https://gemini.google.com)
[![DeepSeek](https://img.shields.io/badge/DeepSeek-The%20Reasoner-4D6BFE?style=for-the-badge&logo=deepseek&logoColor=white)](https://deepseek.com)
[![Ornith](https://img.shields.io/badge/Ornith-The%20Heir%20·%20Local-10B981?style=for-the-badge&logo=ollama&logoColor=white)](#)

</div>

| Seat | Model | The job |
|:--:|:--|:--|
| 🛠️ | **Claude** *(Anthropic)* | The builder and head brain — writes the code, drives the reviews, keeps the map. |
| 🔍 | **GLM 5.2** *(Z.ai)* | The standing **blind code reviewer** — reads every build cold and hunts the bug the tests can't see. |
| 🎯 | **Gemini** *(Google)* | The second opinion — pulled in when a problem needs a fresh pair of eyes. |
| 🧩 | **DeepSeek** | The reasoner — brought to the table for the hardest, once-only architectural calls. |
| 🦅 | **Ornith** *(local — the heir)* | Runs on hardware **in the room**. Today it takes the work that stays home; **it is being trained for the job it will one day hold outright — the daily maintainer of this codebase once TheWarRoom goes enterprise.** Every review it shadows, every lesson it logs, is an apprenticeship. The plan is a self-hosted system of record maintained by a model that never leaves the building. |

> **Ornith is not the junior seat — it's the succession plan.** GLM, Gemini, and DeepSeek sharpen the code today; Ornith is learning to *own* it tomorrow. When this goes enterprise, the maintainer is already home.

---

## 📜 License & Ownership

**TheWarRoom is source-available, not open-source — free to run and tinker with, never to sell.**

- 🆓 **Free forever for non-commercial use** — released under the [**PolyForm Noncommercial License 1.0.0**](LICENSE). Run it, fork it, self-host it, study it, share it. Homelabbers and hobbyists: this is yours to play with.
- 🏛️ **Owned, in full, by SecureProspective LLC** (Texas) — all rights reserved. Copyright never leaves the owner.
- 🙏 **Tech Freedom Ministries holds a perpetual, irrevocable, free-use grant** — including commercial use — that **survives any change of ownership.** TFM is the inspiration for this project; its rights are permanent and cannot be altered by any future buyer. See [`docs/licensing/TFM-Grant.md`](docs/licensing/TFM-Grant.md).
- 💼 **Commercial use and sale are reserved to SecureProspective.** Want to use TheWarRoom commercially? [Contact SecureProspective](https://secureprospective.com) for terms.
- 🤝 **Contributions welcome** — by contributing you agree to the [Contributor License Agreement](CLA.md), which keeps ownership consolidated with SecureProspective while you keep the copyright to your own work. Start with [`CONTRIBUTING.md`](CONTRIBUTING.md).

> 📖 The whole licensing picture in plain English: [`docs/licensing/README-license-summary.md`](docs/licensing/README-license-summary.md).

---

<div align="center">

*Built by Christopher Campbell with Claude (Anthropic) — reviewed, challenged, and sharpened by a council of GLM, Gemini, DeepSeek, and Ornith.*

**MFL gives you the league. TheWarRoom helps you win it.**
**And one day, the league gets played here.**

</div>
