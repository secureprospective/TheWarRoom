# SESSION E — Mobile Learning Harvest: The Reactive Triage Console

**Design Doctrine:** The mobile instance is not a desktop shrunken. A phone is where things REACH you. It is a reactive triage station. The command/chat channel is the shell itself.

---

## 1. THE SURVIVAL TABLE

Desktop surfaces triaged for mobile state-of-matter.

| Desktop Surface | Mobile Fate | The "Why" (One-line rationale) |
| :--- | :--- | :--- |
| **M1 Asset Rankings board** | **CHAT-ANSWERABLE** | A dense 32x7 matrix is unreadable on a phone; operators query it via `> /query players pos=RB` and get a summarized card. |
| **M2 Weekly Power Rankings** | **DEAD** | Desktop-only context; mobile operators only need to know where they stand if it changes, which triggers a Push Notification. |
| **M4 Transaction workspace** | **DEAD** (Proactive) / **PUSH NOTIFICATION** (Reactive) | Nobody builds complex multi-leg trades on a phone. The phone only receives trade pushes and resolves them via Control Cards. |
| **Inspector (Player/Team)** | **CHAT-ANSWERABLE** | Deep data-dive blocks are too dense for a thumb-scroll; requested via `> /inspect player` and returned as a stacked answer-card. |
| **Home (2x2 landing)** | **DEAD** | Replaced entirely by the Triage Queue; the phone opens directly to what needs action, not a navigational board. |
| **The Feed / Pulse** | **LIVE CARD** | The ambient event substrate compresses into a dismissible "Pulse" card pinned to the top of the shell for glanceable context. |
| **The Comms thread** | **LIVE CARD** (The Shell) | Promoted from a right-edge overlay to the primary spatial canvas; the thread IS the operating system. |
| **The Calendar** | **CHAT-ANSWERABLE** / **PUSH** | Viewable on-demand (`> /calendar view`), but deadlines project as Push Notifications when they escalate to `<1hr`. |
| **Cross-league reality** | **PUSH NOTIFICATION** | The 25-league portfolio operator relies on the phone to aggregate escalations across all leagues into a single triage queue. |

---

## 2. THE CHAT-FIRST MOBILE SHELL + THE FLAWLESS FLOW

### Concept: The Triage Stack
On mobile, the Session D "one-thread event grammar" becomes the spatial canvas. 
- **Anatomy:** The screen is a single, thumb-scrollable column. Ambient context (Pulse, Cross-League seams) lives at the top as compact **Live Cards**. Below that, the space is owned by chronological events and **System Answer Cards**. 
- **The `/`-Pivot without a Keyboard:** The prompt is a persistent bottom-edge input bar. Tapping it summons a hardware-density Quick Verb Rail (`/query`, `/calendar`, `/offer`, `/summon`). Selecting a verb formats the prompt to Mono and drops a parameter card. Typing is optional; the verb rail acts as the keyboard's mechanical stand-in.
- **VAV without Hover (The Swipe-to-Act Model):** Desktop hides affordances until `:hover`. Mobile replaces hover with **horizontal swipe**. Resting rows display only the 2px spine, timestamp, subject, and predicate. Swiping left reveals mechanical actions (Acknowledge, Dismiss). Escalated events (`--signal-red-loud` spine) bypass VAV and surface their Control Cards directly inline, requiring immediate resolution.

### The Flawless Flow: Escalated Trade Resolution Under Clock
*The operator is at dinner. A trade was accepted, but a buyout deadline is in 45 minutes.*

1.  **Arrival:** The push hits the lock screen. Tapping it opens the app directly to the escalated event. 
2.  **Escalation UI:** The event row renders with a 4px `--signal-red-loud` spine. It is pinned to the top of the thread (ALERTS tributary).
3.  **Control Card Expansion:** Below the event, the `/offer` Control Card renders inline. Contract block is Mono `--text-primary`. Hold-to-fire verbs are arrayed horizontally: Decline, Counter, Accept. 
4.  **The Hold-to-Fire (B-Confirm Mobile):** The operator presses and holds the "Accept" actuator. The UI drops the 480px modal (it doesn't fit) and instead **ink-fills the entire Control Card** with `--signal-blue-loud` opacity overlay (≤600ms fill). The card is mechanically locked. 
5.  **Resolution:** The fill completes. The engine commits. The card collapses its state to `[COMMITTED — 21:15]`, the spine fades from red to `--edge-freshness-stale` gray, and the row recedes (fades to 50% opacity). Threat neutralized. One hand, one thumb.

### ASCII Phone-Frame Sketch (Logical ~360px)

```text
┌─────────────────────────────────┐
│ THEWARROOM          [L: ALL]  ⚙ │
├─────────────────────────────────┤
│ ▌ PULSE: 12 events, 1 snipe     │ <- Ambient Live Card (Collapsible)
├─────────────────────────────────┤
│ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ ▌ │
│                                 │
│ 21:15 ↳ Commit logged.          │ <- System Answer (Mono)
│       [COMMITTED — 21:15]       │
│                                 │
│ ║ 20:30 Team X proposed /offer  │ <- Escalated Event (4px Red-Loud Spine)
│ ┃ ┌─────────────────────────┐   │
│ ┃ │ CONTRACT BLOCK          │   │ <- Control Card /offer
│ ┃ │ > /offer accept         │   │
│ ┃ │ ┃ [DECLINE] [ACCEPT ⌛] ┃   │ <- Hold-to-Fire targets (VAV forced)
│ ┃ └─────────────────────────┘   │
│                                 │
│ 19:00 Player X cleared waivers. │ <- Resting State (2px Achromatic Spine)
│                                 │
│                                 │
├─────────────────────────────────┤
│ [ / ]  [ /query ] [ /cal ] [+] │ <- Quick Verb Rail (No keyboard needed)
└─────────────────────────────────┘
```

---

## 3. THE LESSONS LEDGER (Mobile Handoff)

*This is the binding bridge between the desktop system and the future mobile build.*

### 🟩 BINDS MOBILE (Inherited Non-Negotiables)
| Decision | The "Why" |
| :--- | :--- |
| **Color & Restraint Doctrine** | Color is State, not structure. Do not tint backgrounds by value. The notification list must remain a calm, dark, optical-elevation instrument to prevent notification anxiety. |
| **The One-Thread Event Grammar** | The anatomy (Spine + Subject + Predicate + Answer) scales perfectly to a single column. The thread is the shell. |
| **Escalation Model** | 4px spine, amber/red hue, ALERTS tributary. This is the core mechanic of the reactive triage station. |
| **Hold-to-Fire Actuation** | The ≤600ms fill must survive. It prevents catastrophic fat-finger trades. It simply changes shape (card-fill vs. modal). |
| **Token Typography** | Inter for text, JetBrains Mono for data/commands. Hero numerics maintain tabular-nums. Visual hierarchy is preserved. |
| **Command Ledger (A1-D12)** | Every action still requires a verb. Mobile UI is just a generator for `txn.commit`, `feed.ack`, etc. |

### 🟥 MOBILE MAY BREAK (Deliberate Divergences)
| Decision | The "Why" & The Replacement |
| :--- | :--- |
| **4-Column Desktop Instrument Shell** | *Why:* A phone has room for ONE thing. *Replacement:* The Chat-First Triage Stack. Dense boards are stripped entirely or relegated to the Chat-Answerable queue. |
| **Hover-Dependent VAV** | *Why:* Touch surfaces have no hover state. *Replacement:* Swipe-to-reveal interactions for resting states; forced persistent VAV for escalated states. |
| **480px Centered ConfirmModal** | *Why:* Desktop measure; hijacks the small viewport clumsily. *Replacement:* The inline Control Card ink-fill. The commitment happens *in situ*. |
| **Keyboard Navigation (J/K/Enter)** | *Why:* Phones lack physical keyboards by default. *Replacement:* The Quick Verb Rail; mechanical tap targets and swipe gestures. |
| **Density Tiers (Matrix/Tactical)** | *Why:* 22px rows require a mouse pointer. *Replacement:* State-of-matter changes. Dense matrices become summarized cards (`> /query`). |

---

## RIPCORD ITEMS
*Honesty regarding where the "Reactive Triage Station" model fails.*

1.  **The Proactive Interrogation Failure:** An inbox is inherently reactive. If an operator is at the pub and wants to ask, *"How do my running backs look across all 25 leagues?"*, a pure triage push-feed offers nothing. 
    *   *The Fix:* The Quick Verb Rail (`/`) is the ripcord. The LLM router and command ledger must be capable of generating a complex `CHAT-ANSWERABLE` matrix card on demand. The phone must serve proactive interrogation even if its primary state is reactive.
2.  **Notification Anxiety Threshold:** If the system pushes every ambient event from 25 leagues, the phone becomes an anxiety engine.
    *   *The Fix:* The Push Notification fate is strictly reserved for `--signal-amber-loud` and `--signal-red-loud` escalations, plus direct `@mentions`. Standard feed events must die or remain silently in the Pulse Live Card until manually opened.
