# SESSION E — Mobile Learning Harvest: The Oracle Terminal

**Thesis:** The phone is not a desktop shrunk; it is a proactive query terminal. Dense boards die; the app becomes a command-line conversation. The operator asks, the system answers with actuation-bearing cards. 

## 1. THE SURVIVAL TABLE

Dense surfaces do not survive the jump to 360px; they change state of matter. 

| Desktop Surface | Mobile Fate | The "Why" (One-Line Rationale) |
| :--- | :--- | :--- |
| **M1 Asset Rankings** | **CHAT-ANSWERABLE** | A 32×7 matrix is unreadable on a phone. "Show me top available RBs" returns a compact, sorted Live Card. |
| **M2 Power Rankings** | **CHAT-ANSWERABLE** | Replaced by an on-demand system summary: "Where do I rank?" returns a top-5 list card. |
| **M4 Transaction WS** | **DEAD** | The complex multi-leg workspace dies. It is replaced by push-alert Control Cards and chat-driven `/offer` commands. |
| **Inspector** | **CHAT-ANSWERABLE** | Deep-dive entity stats are queried ("Inspect Bijan") and returned as a single, scroll-frozen answer card. |
| **Home Dashboard** | **DEAD** | No room for a 2×2 grid. The chat thread itself is the home screen. |
| **Feed / Pulse** | **LIVE CARD** | The live event substrate persists as a thumb-scrollable timeline within the active chat thread. |
| **Comms Thread** | **LIVE CARD** | Promoted to the entire shell. The terminal-log thread *is* the mobile application. |
| **Calendar** | **CHAT-ANSWERABLE** | Month/Week grids die. "What's due this week?" returns an append-only list card of chips. |
| **Cross-League Reality**| **PUSH NOTIFICATION** | The 25-league operator doesn't browse; they are interrupted. Fires only appear as escalated push alerts. |

---

## 2. THE CHAT-FIRST MOBILE SHELL & FLAWLESS FLOW

The Session D terminal-log grammar becomes the entire mobile shell. It is a single, thumb-scrollable column. 

**Anatomy & VAV Interrogation:**
*   **The `/`-Pivot:** Typing `/` activates the command channel. Because mobile is keyboard-optional, a persistent **`/>` hardware-actuation button** sits left of the send key, instantly snapping the keyboard to Mono and injecting `/`. 
*   **VAV without Hover:** Hover-dependent affordance visibility is replaced by **Tap-to-Expand**. Resting rows show zero chrome (spine + text). Tapping a row expands a slide-down action drawer (verb affordances). 
*   **Escalation Override:** If an event hits amber/red escalation state, the affordance drawer is *force-persisted* and the card glows with a semantic spine, surviving until `feed.ack` or `ui.event.resolve` is fired.

### THE FLAWLESS FLOW: The 11PM Snipe Counter
*Context: The operator has 25 leagues. At 23:58, a bid expires. They are in bed, phone in hand, one thumb available.*

**1. THE PUSH (Reactive):** The phone wakes. A push notification slides in: `⚠️ [L: ALPHA] SNIPE ALERT: Bijan bid expiring 00:01.`
**2. THE THREAD (The Shell):** Operator taps the alert. The app opens directly to the terminal-log thread. The escalated event is pinned at the top.
**3. THE QUERY (Proactive Oracle):** Operator taps the `/>` command button. Keyboard snaps to Mono. Operator types: `/show bijan` (verb: `cmd.execute`). 
**4. THE ANSWER CARD:** The system prints `↳ SYS Rendering active bid...`. A `/offer Control Card` renders in the conversation. It shows: `Score: 9.2 (Elite)`, `Current Bid: $42 (You)`, `Top Bid: $45 (Rival)`.
**5. THE ACTUATION:** The Control Card's embedded action button is active: `[ HOLD TO COUNTER $46 ]`.
**6. THE COMMIT:** The operator presses and holds the button. The desktop's 480px modal is dead; the hold-to-fire *gesture* lives on the button itself. It fills with `--signal-amber` (≤600ms fill).
**7. THE RESOLUTION:** The engine validates. The button locks, fades to 50%, and prints `[COMMITTED — 23:59]` (verb: `txn.commit action=counter`). The spine recedes to achromatic gray.

### ASCII Phone-Frame Sketch (~360px logical width)

```text
+-----------------------------------------+
| 23:58               THE WAR ROOM     ⚙ |
+-----------------------------------------+
| [L: ALPHA]                      ⚠️ 00:01|
| ▌SYS  ⚠️ SNIPE ALERT: Bijan bid expiring|  <-- Escalated Red-Loud Spine
|  ▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆▆ |      (Pinned, Tap-to-Expand hidden)
+-----------------------------------------+
| > /show bijan                           |  <-- Human (Inter), /pivot active (Mono)
+-----------------------------------------+
| ↳ SYS  Rendering active bid...          |  <-- System (Mono)
|                                         |
| ▌ ╔═══════════════════════════════════╗ |  <-- Chat-Answerable Live Card
|   ║ BIJAN ROBINSON           [L:ALPHA]║ |      (Optical raise via tile edge)
|   ║ ───────────────────────────────── ║ |
|   ║ Score: 9.2 (Elite)     Age: 22    ║ |
|   ║ Your Bid: $42 / 2yr    Top: $45   ║ |
|   ╠═══════════════════════════════════╣ |
|   ║ [██████████████░░░░] HOLD $46     ║ |  <-- Hold-to-Fire Gesture (<600ms)
|   ╚═══════════════════════════════════╝ |
+-----------------------------------------+
| [/>]  Ask / Command...          [SEND]  |  <-- Hardware-actuation style buttons
+-----------------------------------------+
```

---

## 3. THE LESSONS LEDGER (Handoff Document)

This is the binding contract for the future mobile build session. The desktop taught us what is structural and what is merely spatial.

| BINDS MOBILE (Non-Negotiable) | The "Why" |
| :--- | :--- |
| **Color & Restraint Doctrine** | Color is data, structure is achromatic. A push notification must not light up the screen like a Christmas tree; hue is earned strictly via semantic state. |
| **One-Thread Event Grammar** | The anatomy (spine + timestamp + subject/predicate) is the universal translator. Everything is a message in the terminal log. |
| **Hold-to-Fire Commit** | The 480px modal dies, but the physical actuation of the commit survives. It maps perfectly to a mobile press-and-hold. |
| **Optical Elevation (No Shadows)** | Blurs/dropshadows are expensive on WebKitGTK/mobile. Hairlines and 4% sunken/9% tile elevation rules carry over flawlessly. |
| **Command Ledger (Verbs)** | The `txn.commit` and `cmd.execute` verbs are the API. The mobile UI is just a skin over this command ledger. |
| **Append-Only Truth** | Events and actions are never mutated; they are superseded. This temporal honesty scales perfectly to a chat thread. |

| MOBILE MAY BREAK (Explicit Divergence) | The "Why" & What Replaces It |
| :--- | :--- |
| **4-Column Spatial Shell** | No room for nav/inspector/calendar splits. **Replaced by:** A single-column chat thread; inspectors become answer cards. |
| **Hover/Focus VAV** | A phone has no hover state. **Replaced by:** Tap-to-Expand drawers for resting affordances; escalation forces persistent visibility. |
| **Density Tiers (Matrix/Narrative)** | 32×7 grids are illegible on a thumb surface. **Replaced by:** On-demand queries returning single-entity or filtered summary cards. |
| **Keyboard-First Assumption** | J/K/T/Enter navigation requires a physical keyboard. **Replaced by:** Native touch scrolling, swipe-to-acknowledge, and a dedicated `/>` command pivot button. |
| **Dedicated Boards (Home, M1, M4)** | The operator cannot visualize a portfolio at once. **Replaced by:** The Oracle terminal—data exists only when interrogated. |

---

## RIPCORD ITEMS

The "Proactive Oracle" (pull-only interrogation) is mathematically elegant, but it fails the core use case: **time-critical events the operator didn't think to ask about.** 

1.  **The 2AM Snipe Failure:** If the app is closed, a pull-only terminal is dead. A command-first phone *demands* native Push Notifications (APNs/FCM) for escalated events (`red-loud` snipes, expiring bids). 
2.  **The Thumb-Typing Latency:** Under a ticking clock, requiring the operator to type `/show bijan` is too much friction. 
    *   *Mitigation:* Push notifications must deep-link directly into the thread with the relevant `/offer` Control Card already expanded and rendered, bypassing the query phase entirely. The operator opens the phone and immediately executes the Hold-to-Fire gesture.
