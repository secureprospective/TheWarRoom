# Rulings — 2026-11-25 (Christopher, chat and the Ask button)

Context: owner chat was never built (the 11-04 "retire" removed a planned row; `docs/ui/endpoint-registry.csv` M-120/M-121).
The 11-22 grill (Q2/Q9) made chat a must-match before the campaign. Research: `~/fleet/runs/pulse-chatscan/RESULT.md`.

1. **Rehearsal season: link out.** League talk stays in the league's own group chat (GroupMe, Discord, text).
   A league setting holds the link; a "League chat" button opens it (phone League tab, browser League Pulse).
   Built 11-25 as `league.chatLink` (lane chatlink).
2. **Native chat, first version: one chat store with rooms.** The league room first; draft-room chat, a private
   thread on each trade offer and replies to commissioner notices are rooms in the same store, not separate features.
   Designed with the phone study after Dec 6. Plugs into the existing seams (`docs/architecture/future-seams.md`:
   commands, scopes, events, stream).
3. **Alerts: mentions only.** A phone buzzes when someone names you or your team, or replies to you; everything else
   is an unread badge. Chat and phone alerts (T1826) are designed together.
4. **Phone Ask button: real AI later.** It stays canned for now. Later it connects to the owner's own AI helper,
   which sees and does only what that owner may and confirms before any change.

## Draft: a human's clock runs out (Christopher, 11-25)

Found in the GATE 2 computer drafts: when a human-held seat's clock runs out, today's setting (`draft.onExpire = SkipPick`)
skips the turn for good. The commissioner-held seat ended with zero players.

**Ruling:** "Lets build a commissioner setting that does the above, default is that pick is skipped but can be picked at
any time."
- The commissioner chooses what happens at expiry: the computer picks the best-ranked available player; the pick is
  skipped; or the draft waits with no clock.
- **Default:** the turn is skipped so the draft moves on, but the pick is **kept**. The owner can make that missed
  pick at any time later in the draft, from the players still available.
- It goes through the one rule-change flow like every setting. No preset decides it.

## Draft: computer picks lost while writes are paused (Christopher, 11-25)

Bug: after a restart the server refuses writes until the off-site backup is current, but the draft clock keeps running.
17 computer picks were lost in the 10/12/16-team test drafts. Fix: a seat with no owner always gets its automatic pick,
even if late. **Ruling:** merge it before Sunday 11-29 once the evidence is shown, so the practice drafts test it.
Until it is on the Lab: no Lab restarts while any draft is open.
