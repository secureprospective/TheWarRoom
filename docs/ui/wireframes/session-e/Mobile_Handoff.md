# TheWarRoom — Mobile Handoff (Session E Learning Harvest)

**Status:** CONFIRMED 2026-07-19/20. This is the artifact a future mobile design session opens **cold**. It is NOT a mobile build — it is the direction harvested from the completed desktop system (Sessions A–D). Read `docs/ui/Wireframe_Session_Plan.md` §6 (Session E) and the desktop wireframes A–D first; the phone-frame proof is `session-e-wireframe.html`.

**Thesis (R-A): the phone is an _Inbox that opens onto an Oracle._** Reactive at the front door (push-first — escalated events and deadlines reach you); proactive once inside (the `/`-pivot command channel returns answer-cards on demand). Mobile is a **different instrument**, not a responsive shrink of the desktop; its primary surface is the Session-D terminal-log thread, promoted to the entire shell.

---

## 1. The survival table

Every desktop surface gets exactly one mobile **fate** — dense surfaces don't shrink, they change state of matter. Four fates: **CHAT-ANSWERABLE** (returns as an on-demand card through the command channel), **LIVE CARD** (persists in the shell), **PUSH** (no resting surface — only ever an alert demanding a decision), **DEAD** (open the desktop for it).

| Desktop surface | Mobile fate | Why · escalation/push behavior |
|---|---|---|
| **M1 Asset Rankings** | CHAT-ANSWERABLE | A 32×7 matrix is illegible on a thumb. "top available RBs" → a sorted answer-card. |
| **M2 Power Rankings** | CHAT-ANSWERABLE | "where do I rank?" → a top-5 summary card. A query, not death. |
| **Inspector** | CHAT-ANSWERABLE | "breakdown Bijan" → a single scroll-frozen entity card. |
| **Home (2×2)** | DEAD | No landing grid — the thread IS the home screen; it opens to what needs action. |
| **Comms thread** | LIVE CARD | Promoted from a right-edge overlay to the entire shell — the terminal-log thread is the app. |
| **Feed / Pulse** | LIVE CARD | The live event substrate persists inline; ambient events stay calm until they escalate. |
| **Calendar** | CHAT-ANSWERABLE **+ PUSH** | "what's due this week?" → an append-only chip list. **PUSH when a deadline escalates (<1hr / roster-lock).** |
| **M4 Transaction WS** | DEAD **+ PUSH** | The multi-leg builder dies (nobody builds trades on a phone). **Escalated trade/bid → PUSH → `/offer` Control Card.** |
| **Cross-league (25-league)** | PUSH | The operator doesn't browse 25 leagues on a phone — they're interrupted. Fires aggregate into one push queue. |

**The push-spine is the graft that makes the table honest.** Pull-only interrogation (the Oracle alone) is dead when the app is closed at 2am — so **every time-critical axis (deadlines, trades, cross-league fires) carries a PUSH**. Everything else is calm and answers-on-demand. This closes the pull-only gap the Oracle direction admitted in its own RIPCORD.

---

## 2. The chat-first shell + the one flawless flow

The Session-D one-thread event grammar (card + control + answer in one thread) becomes the **spatial canvas**. One thumb-scrollable column. No hover, no keyboard-first assumption.

**Anatomy on a phone:**
- **The thread** is the app: event rows (spine + mono timestamp + subject-600/predicate-400), system log lines (`↳ SYS`, no bubbles/no persona), and inline Control Cards, all in one column.
- **VAV without hover (R-D):** resting rows show spine + text, zero action chrome. **Tap-to-expand** a slide-down action drawer is the primary reveal. **Swipe** is reserved for the single highest-frequency one-shot (acknowledge/dismiss) — not a general action surface. **Escalation force-persists** affordances inline (an escalated Control Card is never hidden).
- **The prompt (R-F):** the input bar is **free natural-language / voice-first** — a first-class **mic** affordance + the `/`-pivot (flips to Mono + blue focus axis). A **quick-verb rail** (`/rank /m1 /inspect /calendar /offer`) is an **accelerator, not the only path**. The verbs are user-facing aliases the LLM resolves to canonical Command-Ledger verbs.
- **Input is noise-tolerant:** input arrives fuzzy — **misspellings** (thumb-typos) and **voice-transcription errors**. The LLM router **normalizes fuzzy input → the canonical verb BEFORE any output** (normalize-then-act; never echoes the garbled token, never runs a mis-parsed verb). This is why the backbone is LLM-backed, not a rigid Guix-simple parser. See `Command_Ledger.md` input-normalization mandate.

**THE ONE FLAWLESS FLOW — reactive, under clock (R-C: zero typing, push deep-linked):**
1. **PUSH.** An escalated bid/trade fires an OS push to the lock screen carrying the decision (`⚠ Bijan bid topped — expires 00:01`).
2. **DEEP-LINK.** Tapping the push opens the app **directly to the thread with the `/offer` Control Card already expanded** — no navigation, no query.
3. **FORCE-PERSIST.** The escalated event renders with a 4px `--red-loud` spine, pinned to the ALERTS tributary; the Control Card's action chrome is visible (escalation overrides tap-to-expand).
4. **HOLD-TO-FIRE.** Press-and-hold the action button (≤600ms fill, `--blue-base` action axis; release cancels; engine-reject holds). The 480px modal is dropped — the gesture lives on the card.
5. **RECEDE.** Engine commits `txn.commit action=counter`. Card flips to `[COMMITTED — 23:59]` in `--green-base`; spine recedes to `--fresh-stale`; row fades to 50%. One thumb, no navigation.

**The everyday counterpart — proactive, calm:** mic/thumb → fuzzy utterance (`"trde cmach for cheetah"`) → `↳ SYS` normalizes → `/trade give=McCaffrey get=Hill` → an actuation-bearing card returns in the thread (tap-to-expand for Accept/Counter/Decline). Dense M1/M2/Inspector return here as **cards, never boards**.

---

## 3. The lessons ledger

The binding contract for the future mobile build. The desktop taught us what is structural (sacred) and what is merely spatial (mobile's to break).

### BINDS mobile — inherited, non-negotiable
| Decision | Why it's sacred |
|---|---|
| **Color & restraint doctrine** | Color is Data + State; structure achromatic. A notification list must never light up like a Christmas tree — hue earned strictly via semantic escalation. |
| **One-thread event grammar** | Spine + timestamp + subject(600)/predicate(400) + verb-affordance is the universal translator; scales perfectly to one column. |
| **Append-only truth** | Events/actions are superseded, never mutated (`EventSuperseded`/`EventUndone`). Temporal honesty maps 1:1 onto a chat thread. |
| **Hold-to-fire actuation** | The 480px modal dies; the gesture survives (≤600ms, release-cancels, engine-reject holds). Prevents fat-finger commits. |
| **Token typography** | Inter = text, JetBrains Mono = data/commands; hero numerics keep tabular-nums. Desktop hierarchy preserved. |
| **Optical elevation (no shadows/blur)** | Hairlines + inset bevels + the surface ramp carry over — cheap-paint on mobile too. |
| **Command Ledger verbs (A1–D12)** | Every action is still a verb; the mobile UI is a generator for `txn.commit`/`feed.ack`/`cmd.execute`. Slash-aliases are skins the LLM resolves. |
| **LLM input-normalization** | Fuzzy voice/typo input resolves to a canonical verb *before* output (normalize-then-act). Router preprocessing, not a mutation verb. |

### MOBILE MAY BREAK — deliberate divergences
| Desktop decision | Why + what replaces it |
|---|---|
| **4-column instrument shell** | A phone has room for ONE thing. → **Single-column chat-first shell**; splits die, dense boards → chat-answerable cards. |
| **Hover-dependent VAV** | Touch has no hover. → **Tap-to-expand drawer (primary) + swipe for ack**; escalation force-persists. |
| **480px centered ConfirmModal** | Desktop measure, heavy under a clock. → **Inline hold-to-fire on the Control Card** — commitment in situ. |
| **Keyboard / J-K-T map** | No physical keyboard; soft one is slow. → **Voice/mic + `/`-pivot & quick-verb rail**; native touch scroll. |
| **Density tiers (Matrix/Tactical)** | 22px rows need a mouse; 44pt touch targets override. → **One comfortable touch density**; "Matrix" need met by summary cards. |
| **Dedicated boards (Home, M1, M4)** | Can't visualize a portfolio at once on a phone. → **Data exists when interrogated**; the board becomes an answer. |

---

## RIPCORD — none new
R-A (Inbox-onto-Oracle) resolves the reactive-Inbox direction's proactive-interrogation gap; R-C (push deep-link) resolves the proactive-Oracle direction's under-clock-typing gap. The two divergent runs' RIPCORDs cancel each other in the composition. Phase-3 mobile inherits this document; nothing in Phase-1 desktop changes.
