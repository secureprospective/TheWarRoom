<div align="center">

# 🏈 TheWarRoom

### A dynasty fantasy football intelligence engine that scores what real players do in real games — and out-thinks the stock platforms doing it.

[![Status](https://img.shields.io/badge/status-pre--build%20·%20planning%20complete-blue)](docs/build-handoffs/Build_Tracker.md)
[![Build Plan](https://img.shields.io/badge/build-38%20sessions%20sequenced-success)](docs/build-handoffs/Build_Tracker.md)
[![Phase](https://img.shields.io/badge/phase-1%20·%20personal%20tool-orange)](#-the-road)

[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails%20v2-DF0000?logo=wails&logoColor=white)](https://wails.io)
[![React](https://img.shields.io/badge/React-20232A?logo=react&logoColor=61DAFB)](https://react.dev)
[![Tailwind](https://img.shields.io/badge/Tailwind-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com)
[![SQLite](https://img.shields.io/badge/SQLite%20WAL-003B57?logo=sqlite&logoColor=white)](https://sqlite.org)

*32 teams. Six scoring layers. Ten position models. Twenty-one scouting sources. One number per player.*

</div>

---

## ⚡ The Pitch

MyFantasyLeague gives you a roster and a box score. **TheWarRoom gives you an edge.**

It ingests live league data from MFL, fuses it with twenty-one professional scouting sources, and runs every rostered player through a **six-layer valuation engine** built on this league's exact ruleset. The output is a single **Adjusted Score** per player — and a war room full of tools that turn that score into decisions: who to bid on, what a trade is really worth, where your roster is thin, and what a contract costs you three years from now.

It also replaces the part nobody enjoys: the bid math, the dead-cap arithmetic, the snipe clock, the trade-deadline policing. The machine handles the mechanical. The humans keep the judgment.

> **The engine's core trick:** it separates *"is this player good?"* from *"how long does he have?"* — scoring talent and aging as two different jobs, the way a real scout does. An elite receiver in his prime and a declining veteran can post the same box score; TheWarRoom prices them 22% apart.

---

## 🧠 Pillar 1 — The Scoring Engine

The valuation brain. Six layers, run in order, every value MFL-sourced or admin-tunable — nothing hardcoded.

```
              MFL API  +  21 scouting sources (PFF · RAS · NGS · Madden · film)
                                      │
                                      ▼
   ┌──────────────────────────────────────────────────────────────┐
   │  L1  Data Hygiene        clean inputs, enforce contract floors │
   │  L2  Rulebook Scoring    this league's exact scoring matrix    │
   │  L3  Age Decay           position-specific aging curves        │
   │  L4  Scouting Layer      Film × RAS × Breakout (Madden-checked)│
   │  L5  Cap Efficiency      value per dollar, by cap tier         │
   │  L6  Tiebreaker          tenure → RAS → positional scarcity    │
   └──────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                       ⭐ Adjusted Score  (one number per player)
```

**Ten position models, each individually calibrated** — because a shutdown corner and a power back aren't graded on the same curve:

| | | |
|---|---|---|
| 🎯 **QB** | 🏃 **RB** | 🙌 **WR** |
| 🧤 **TE** | 💨 **DE** | 🛡️ **LB** |
| 🔒 **CB** | 🦅 **S** | 🧱 **DT** |
| 🦵 **K** | | |

Under the hood: NGS coverage metrics anchor the cornerback and safety models. Madden ratings *regulate* subjective scouting takes so hype doesn't leak into the score. A late-career "Cushion Guard" keeps elite athletes from falling off a cliff on paper before they do on the field. This is the depth a 32-team dynasty ruleset deserves.

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

The engine is only as good as its calibration. Every parameter that *should* be tuned against real results — scouting weights, S-curve shapes, age curves, cap tiers, the scarcity matrix — is exposed in the admin console. Tune the engine through the UI; the structural mechanics stay code-locked. That's also what makes Phase 3 possible: every league installs and tunes its own instance.

---

## 🛡️ Built to Last

This isn't a weekend hack. The architecture is governed by hard rules so it stays maintainable as it grows:

- **A three-layer law** — real-football data is read-only; the app owns its logic; users mutate state *only* through validated transactions. No layer bleeds into another.
- **One writer to league state** — a single coordinator is the only thing that can change who-owns-what. Everything else reads.
- **Historical scores are immutable** — every score is stamped with its scoring config. Change the engine, and last season stays exactly as it was scored. The record is the record.
- **Enforced by the compiler, not by hope** — architectural rules are wired into the type system and the linter, so a violation is a build failure, not a code review note.
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

## 📍 Status

**Documentation and planning complete. First code build is next.**

The entire engine, all ten rubrics, the data pipeline, the transaction rules, the UI and backend architecture, and a 38-session build plan are specified and audited. The first build session is the project scaffold; the engine follows.

```
[████████████████████░░░░░░░░░░]  Planning ✓   →   Building (next)
```

---

<div align="center">

*Built by Christopher Campbell with Claude (Anthropic).*
**MFL gives you the league. TheWarRoom helps you win it.**

</div>
