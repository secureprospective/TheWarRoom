# TheWarRoom — Left Rail as Funnel Tree (design, not render)

## 1. Funnel model

**Branching principle: intent → object → move.** L0 = which mode of mind are you in (six locked nodes). L1 = which object of that mind (branch; only exists if a node holds ≥2 distinct objects). L2 = the move (leaf; a workspace you land in or a drawer that slides). **Max 3 rail levels — nothing is 4 deep.** Depth is decided by three rules:

- **Frequency:** anything a default-audience GM touches daily (lineup, cap, FA, offers, live) sits at depth ≤2 and within 2 clicks. Rare/admin surfaces (rules, jobs, backup) may sit at L2 under a collapsed branch — they are phase-narrowed anyway.
- **Phase-activeness:** the active phase's branch/leaves float to the top of their level and go full-opacity; dormant leaves dim to 40% under a hairline divider. Nothing hides; nothing becomes "More."
- **Entity rule:** anything ultimately about one player/team routes its *result* to the Inspector, not to a deeper leaf.

## 2. Rail tree — every surface placed exactly once

Types: **W** = workspace (fills centre) · **SD** = side drawer · **BD** = bottom drawer · **ND** = nested drawer (over another drawer) · **P** = peek. Phases: OFF / DRF / REG / PO.

| Node › Branch › Leaf | Surface | Type | Role gate | Phase |
|---|---|---|---|---|
| **HOME** (dashboard) | Seasonal card (phase hero) | W module | all | all |
| | Alert tray | BD + P strip | all | all |
| | Digests | BD | all | all |
| **WAR ROOM** (flat — no branches) | Asset board | W | GM read / DOT read | always |
| | Free-agent pool | W | GM | DRF?/REG (window) |
| | Bid plans | SD over pool | GM | REG |
| | Market reading | W | GM | always |
| | Watch list | SD / **Inspector tab** | GM | always |
| | Trade bait | SD over asset board | GM | always |
| **FRANCHISE HQ › Lineup & roster** | Lineup editor (plan toggle embedded) | W | GM | REG/PO |
| | Matrix roster | W | GM | always |
| **FRANCHISE HQ › Contracts & cap** | Contracts | W | GM | OFF/DRF |
| | What-if cap | SD over contracts | GM | OFF |
| | Release plans | ND (confirm G2) | GM | OFF/REG |
| | Acquisitions log | SD | GM | REG |
| **FRANCHISE HQ › My moves** | Move receipts | W | GM; DOT read | always |
| **TRADE FLOOR › Trade desk** | Builder | W | GM | always |
| | Plan (impact/cap math) | SD over builder | GM | always |
| | Offers (inbox; review = SD) | W | GM; **DOT review** | REG |
| **TRADE FLOOR › Draft room** | Board | W | GM | DRF |
| | Clock | BD strip | GM | DRF |
| | Your pick | P → W handoff | GM | DRF |
| | Tray (queued targets) | SD | GM | DRF |
| | Outcome | ND summary | GM | DRF |
| **LEAGUE PULSE** (flat) | Now (live game) | W | all | REG/PO |
| | Race (standings) | W | all | always, PO peak |
| | Reports — **every** report incl. draft + market reports | BD index → per-report SD | all | always |
| **CONTROL ROOM › App** | Settings | SD | all | all |
| | Account | SD | all | all |
| | What's new | BD | all | all |
| | Data sources | SD | Admin | all |
| **CONTROL ROOM › League** | Rules | W | Comm | OFF (edit window) |
| | People & roles | W | Comm/Admin | all |
| | Ledger | W | Comm; DOT read | all |
| | Ops (approval queue) | W (+SD per item) | Comm | always |
| | Jobs | BD | Comm | all |
| | Backup | ND (**G3 hold**) | Admin | all |
| | Undo | ND (**G3 hold**) | Comm | all |
| | Seed & start | ND wizard | Comm/Admin | OFF |
| | Hosted league | W | Comm/Admin | all |

Surfaces "Elsewhere" resolved: **draft reports + market reports = report types inside the League Pulse report drawer index** (one report home, per the brief). **Research** leaves the app — excluded.

## 3. Rail behaviour (terminal, rubber-grip, no glow)

- **Geometry:** rest = 72px grip strip: glyph + 3-char code (`HOM · WAR · FRQ · TRD · LPS · CTL`) stacked vertically on a textured high-contrast background — like a safety edge, hard 2px border, zero gradient. Hover expands the rail to 240px revealing that node's subtree only. Selected node stays expanded pinned (rail remains 240px); workspace never loses more than it must.
- **Trail furniture (the whole "cinematic terminal" language):** branches are marked with fence glyphs `├` / `└` at 2-space indents; the live leaf carries a solid `▮` cursor. Carets rotate 90° transform-only on expand (**<80ms, no spring overshoot, no glow** — a 1px brightening of the border is the hover ack).
- **Rest:** collapsed strip; nodes with an active-phase child show a 2px phase-mark in the meaning colour (amber pulse = draft clock running, green = live games). **Hover:** subtree slides in, dormant rows dimmed under a hairline. **Selected:** branch header holds, leaf cursor solid, selected glyph fills (not outlines).
- **Phase behaviour:** the six top nodes **never reorder** (locked). Within a node, leaf order re-sorts by phase (e.g. Draft room jumps to top of TRD during DRF; Ops jumps to top of CTL at deadline) and out-of-phase leaves sink, dimmed. Home's hero and the phase-marks make the season state legible without reading a word.
- **Keyboard:** `/` → command bar (the true 0-click funnel, `trade.accept id=X`); `1–6` jump nodes; `j/k` walk rows, `l`/`Enter` open leaf or expand branch, `h`/`Backspace` pop level; `Esc` closes the top drawer; `f` toggles phase-focus filtering between Narrative/Tactical/Matrix densities (Matrix shows dormant rows full-strength; Narrative hides fence-glyph noise and shows only pinned moves).

## 4. Drawer grammar

| Type | Size | Use when | Stack position |
|---|---|---|---|
| **Side** | 400–480px right | A single task flowing out of the current workspace: bid form, release plan, what-if, offer review, reports. | Level 1 |
| **Bottom** | ≤40% height, with 56px peek | "Now" data: clock, live tick, alert tray, digest, jobs, outcome band. Levels never block centre attention longer than needed. | Level 1 (one at a time) |
| **Nested** | over its parent | Confirm/decide steps: release confirm, bid confirm, accept trade, backup/undo (**G3 600ms hold**), seed wizard. | **Max stack depth = 2** (side+nested, or bottom+nested). Never 3. |
| **Peek** | 48–64px strip | Preview without commitment: hovering a trade plan shows the cap delta bar; the clock shows as a strip pre-draft; tray shows queued count. | Any, no interaction cost |

**Dismiss:** Esc peels one level, `Shift+Esc` peels all; click-away closes peek+bottom, never a nested confirm (explicit ✓/✕); swipe/throw closes side; destructive actions additionally require the G-level hold. **Inspector relation:** the Inspector (locked 320px/480px) is the only *entity* surface; side drawers dock **left** of it and a drawer's entity deep-links open the Inspector — the two never swap roles. A drawer may never "become" the Inspector.

**Scroll vs drawer rule:** a workspace may scroll **only inside a fixed-height module** (asset board columns, draft board lanes, ledger table) — never as a page. If content would push the page taller than one viewport and it is about *one task you came to do and leave* → side drawer. If it is about *one entity* → Inspector. If it is Federation-wide "right now" data → bottom drawer. Comparing two entities is a drawer+Inspector pair, not scroll. Anything else exists as a workspace rebuilt at the current density.

## 5. Two-click check

| Job | Path | Clicks |
|---|---|---|
| Set lineup | FRQ → Lineup editor (`1` nodes: 1 click; or `g f` + `Enter` = 0) | **1–2** |
| Accept trade | Alert tray opens Offer SD directly on HOME (1 click); or TRD → Offers → accept in review SD | **1–2** |
| Place FA bid | WAR → pool (1) → Bid SD with confirm ND (2) | **2** |
| Check cap room | Status strip always shows cap = **0**; detail FRQ → Contracts | **0–2** |
| Commissioner approve | Alert tray Ops SD (1 click) from home, or CTL → Ops | **1–2** |
| See who's live now | League Pulse Now via phase-mark on the grip strip; status strip shows live count (click = 1) | **1** |

All ≤2; command bar reduces each to 0 (`lineup.set`, `trade.accept`, `bid.place`, `cap.show`, `ops.approve`).

## 6. Risks / RIPCORD items

1. **RIPCORD — Control Room League = 9 leaves.** No "More" is enforceable only because the command bar is the true funnel for admin. If commish work ever becomes daily (deadline weeks), consider splitting "Commissioner ops" out — that means unlocking the six-node lock. Flag for the owner.
2. **RIPCORD — "one report drawer for every report."** An index drawer holding draft + market + league reports is scroll-hiding in a drawer. Designed around (index BD → per-report SD), but if report types exceed ~8 the breadth belongs on the Trade Floor/Draft room as phase-local surfaces.
3. **Personal lists needed a home without a new node.** Watch list = Inspector tab + WAR facet; trade bait = WAR facet of asset board. Feels forced; if these become daily, that is pressure on the "never a new node" lock — flag now, don't bend later.
4. **Countdown urgency vs "no glow":** draft clock pulses by opacity tick only, coloured amber, frame thickened to G2 — compliant, but reviewers will misread "pulse" as glow; keep it a solid-colour flicker.
5. **Weak GPU + drawer stacks:** no blur allowed anywhere in drawer grammar; depth cue comes from border thickness (G-levels) and a 1-step offset, not shadows. Nested confirm on G3 hold must not animate over 200ms.
6. **Rail expand-on-hover + 2-click ceiling:** a walker could count node-click + leaf-click as 2 with 0 result. Mitigated because a node click also lands you in its phase-default workspace — so a node click alone always yields a ready screen, and every leaf is reachable in ≤1 more.
