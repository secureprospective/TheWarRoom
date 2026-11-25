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
