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
| ✅ **Free Agency §6 (v1)** | Live free-agent pool + record-a-signing, with §12 buyout lockout & UFA promotion on rollover |

### ⚡ Go-Powered, Local-First Performance
Zero cloud latency. Absolute privacy. Near-instant processing from a clean, compiled backend. The rankings board **re-ranks the instant you turn a calibration knob** in the admin console — the live tuning loop already works. Native desktop app; your league never leaves your machine.

---

## 🏛️ 4 · The War Room Architecture

A multi-block system design where the boundaries aren't conventions — they're **compiler-enforced law**. A violation is a *build failure*, not a code-review note.

```
   ┌──────────────────────────────────────────────────────────────────────────┐
   │                          THE DESKTOP INTERFACE                            │
   │              Wails v2  ·  React + Tailwind + Zustand                      │
   │        Asset Rankings board · Admin console · Transactions panel          │
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

## 🛠️ 5 · Getting Started

**Prerequisites:** Go `1.26+` · Node + `pnpm` · the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation) · a WebKit runtime (`webkit2gtk` on Linux) · a MyFantasyLeague league id + API credentials.

```bash
# 1 · Clone the war room
git clone https://github.com/secureprospective/TheWarRoom.git
cd TheWarRoom

# 2 · Pull the Go module graph + the frontend deps
go mod download
cd frontend && pnpm install && cd ..

# 3 · Configure ingestion (league id, MFL API host, season)
#     nothing leaves your machine — this is a local-first system of record
#     see docs/ for the ingestion parameters

# 4 · Fire it up in dev (hot-reload UI + Go backend)
wails dev

# 5 · …or build the native desktop binary
wails build
./build/bin/thewarroom
```

**Verify the backend without the GUI** — the headless startup probe runs the real store-floor init chain (db.Open → params → rulebook → state → transactions → output), each step timed and timeout-bounded, so a hang or corruption surfaces instantly:

```bash
./build/bin/thewarroom -probe
```

**Quality gates** (every commit passes these — no `--no-verify`, ever):

```bash
GOMEMLIMIT=3000MiB GOGC=40 make lint     # golangci-lint + ifaceguard + filelen(400)
go test -race ./...                       # -race is non-negotiable
```

---

## 🗺️ 6 · The Draft Board — Roadmap

The engine is the hard part, and the engine is **done**. What's ahead is the payoff — each capability a new *view* onto a score and a ledger that already exist.

**On the clock**
- 🧮 **First-class commissioner UFA calendar** — a signing window that closes at Super Bowl kickoff and reopens on commissioner command (the seam is already wired).
- 📊 **The war-room modules** — Power Rankings, Matchup Predictions, Trade Analyzer, Free-Agency Intel, Rookie Draft board, Commissioner Dashboard.
- 🕳️ **The shadow ledger** — dry-run any transaction against a *forked* cap before you commit it.

**Later rounds — the three horizons**
- 🗣️ **Horizon 1 · The Capologist** — a chat interface on a small **local** model. *"What's my dead cap in 2028 if I cut him?"* answered from the actual ledger, **with receipts**. Numbers never come from the model; the ledger disposes; you decide.
- 💼 **Horizon 2 · The Portfolio Desk** — one engine valuing the same player under *five* leagues' rules simultaneously. Asset management for dynasty football.
- 🎖️ **Horizon 3 · The January War Room** — **fork your franchise** and branch offseason futures like a developer branches code, each run through the *real* transaction engine — then your winning plan becomes the season's execution script.

🔭 The full vision, on the record: **[`docs/roadmap/Vision_2026.md`](docs/roadmap/Vision_2026.md)**

---

## 🤝 How It Gets Built — The Council

One human holds the vision and the veto. A rotating council of AIs does the rest, each in the seat it's best in: **Claude** builds and drives the reviews; **GLM** is the standing blind code reviewer; **Gemini** and **DeepSeek** are pulled in for the hardest architectural calls; **Ornith** runs local for the work that stays home. No model gets rubber-stamped and no model gets to guess — every finding is a *lead*, triaged against the source. The machine proposes; the human and the rulebook decide.

---

<div align="center">

*Built by Christopher Campbell with Claude (Anthropic) — reviewed, challenged, and sharpened by a council of GLM, Gemini, DeepSeek, and Ornith.*

**MFL gives you the league. TheWarRoom helps you win it.**
**And one day, the league gets played here.**

</div>
