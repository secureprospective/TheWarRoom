# THE WAR ROOM — SESSION E SYNTHESIS: MOBILE HANDOFF
**Architecture:** Go + Wails Desktop (React + Tailwind, Dark-Only)
**Thesis:** R-A. The phone is an *Inbox that opens onto an Oracle*. Reactive at the front door (push-first); proactive once inside (`/`-pivot query).

## 1. SURVIVAL TABLE (RULING R-B)
*Desktop surfaces mapped to mobile fates. The reactive push-spine solves Run 2's pull-only RIPCORD.*

| Desktop Surface | Mobile Fate | Escalation / Push Behavior |
| :--- | :--- | :--- |
| **M2 Power Rankings** | CHAT-ANSWERABLE | None (Query: "where do I rank?") |
| **Home Dashboard** | DEAD | None (Merged into Chat Shell) |
| **M1 Matchup** | CHAT-ANSWERABLE | None (Query: "M1 vs X") |
| **Inspector** | CHAT-ANSWERABLE | None (Query: "breakdown Y") |
| **Comms** | LIVE CARD | The Thread Shell itself |
| **Feed / Pulse** | LIVE CARD | Inline updates within Thread |
| **Cross-league** | PUSH | Surfaces as escalated push event |
| **Calendar** | CHAT-ANSWERABLE | **PUSH:** Deadline escalations (<1hr / roster lock) |
| **M4 Transaction WS** | DEAD (as builder) | **PUSH:** Escalated trade/bid -> `/offer` Control Card |

---

## 2. THE CHAT-FIRST SHELL & FLOWS

### The One Flawless Flow (Reactive / Under-Clock)
Zero-typing, push-deep-linked (R-C).
1. **PUSH:** M4 Transaction WS detects an escalated bid. OS push notification fires.
2. **DEEP-LINK:** Operator taps push. App opens directly to the Comms thread.
3. **FORCE-PERSIST (R-D):** The `/offer` Control Card is *already expanded inline*. Action chrome is visible; no tap-to-expand required.
4. **HOLD-TO-FIRE (R-E):** Operator presses and holds the action button (≤600ms). Ink fills the button/card with `--c-success`. 
5. **RECEDE:** On release, engine commits via `txn.commit action=`. Text reads `[COMMITTED — 23:58]`. Spine recedes to `--edge-freshness-stale`, row fades to 50% opacity.

### The Proactive Query Path (Everyday / Calm)
Noise-tolerant NL + Voice (R-F).
1. **INPUT:** Operator taps mic and speaks *"Trade CMC for Tyreek"* or thumb-types *"trde cmach for cheetah"*.
2. **NORMALIZE-THEN-ACT:** The LLM router intercepts the fuzzy input.
   `↳ SYS` resolves fuzzy input to canonical Command-Ledger verb: `/trade`.
   *(Never echoes the garbled token, never mis-parses).*
3. **RENDER:** System returns an actuation-bearing `/trade` Control Card.
4. **VAV (R-D):** Resting cards show spine + timestamp + subject/predicate. Operator taps the card to slide-down the action drawer. Swipe-to-dismiss reserved for acknowledging the thread.

---

## 3. ASCII WIREFRAME: THE FLAWLESS FLOW

```text
┌─────────────────────────────────┐
│ 23:58 ┃ THE WAR ROOM         ⚙️ │
├─────────────────────────────────┤
│ ▌ LIVE FEED                     │
│ ──────────────────────────────┐ │
│ │ 23:57 │ SYS: Bid Escalated  │ │
│ │       │ (M4 WS)             │ │
│ │ ┌─────────────────────────┐ │ │
│ │ │ /offer (Control Card)   │ │ │
│ │ │ Give: C. McCaffrey      │ │ │
│ │ │ Get:  T. Hill           │ │
│ │ │ ────────────────────────┤ │ │
│ │ │ ▓▓▓▓▓▓▓▓▓░░░░ HOLD ▓▓▓▓▓ │ │ │  <-- Force-persisted,
│ │ └─────────────────────────┘ │ │      Ink-filling inline
│ └─────────────────────────────┘ │
│                                 │
│ 23:45 ┃ PRIOR THREAD (Faded 50%)│
│                                 │
├─────────────────────────────────┤
│ [ 🎙️ ] "trde cmach..."  [ / ]   │  <-- NL / Voice Prompt
│ ─────────────────────────────── │
│ [ /rank ] [ /m1 ] [ /inspect ]  │  <-- Quick-Verb Accelerator
└─────────────────────────────────┘
```

---

## 4. LESSONS LEDGER (RULING R-G)

### BINDS (Sacred)
*   **Color/Restraint Doctrine:** Optical elevation via shadows/borders. Color reserved strictly for actuation and semantic states (`--c-success`, `--edge-freshness-stale`).
*   **One-Thread Event Grammar:** A single append-only timeline. Append-Only Truth scales perfectly to mobile (Run 2).
*   **Hold-to-Fire Actuation:** The 480px modal dies; the gesture survives. ≤600ms fill, release-cancels, engine-rejects.
*   **Token Typography:** Strict adherence to `--t-section`, `--t-data`, `--t-mute` for hierarchy.
*   **Command-Ledger Verbs:** `/offer`, `/trade`, `/rank`, `/m1`, `/inspect`, `/ir` (A1-D12).
*   **LLM Router Input Normalization:** Fuzzy voice/typo input MUST be resolved to canonical verbs before rendering. *(New Addendum: `sys.normalize`)*.

### BREAKS (Mobile May Break)
*   **4-Column Shell:** BREAKS. Desktop quadrant is dead. Mobile is strictly single-column. 
    * *Why:* 360px width cannot support optical elevation across 4 panes. Single column allows focus.
*   **Hover-VAV:** BREAKS. Replaced by Tap-to-Expand (slide-down drawer). Swipe is exclusively reserved for ack/dismiss (R-D).
    * *Why:* Capacitive touch lacks hover state to preview chrome.
*   **480px ConfirmModal:** BREAKS. Replaced by inline Hold-to-Fire on the Control Card itself (R-E).
    * *Why:* Modal intercepts pull-down notifications and feels heavy under time-pressure.
*   **Keyboard / J-K Map:** BREAKS. Replaced by Voice/Mic input + Quick-Verb rail.
    * *Why:* Mobile keyboards are slow and visually intrusive.
*   **Density Tiers:** BREAKS. Replaced by standard 1x density (Spine + timestamp always visible).
    * *Why:* Touch targets (44pt min) override desktop data-density cramming.

---

## RIPCORD ITEMS
**None.** 
R-A (Inbox-onto-Oracle) and R-C (Push deep-link) definitively resolve the independent RIPCORDs raised in Run 1 (failure to interrogate) and Run 2 (failure under clock without typing). The synthesis holds.
