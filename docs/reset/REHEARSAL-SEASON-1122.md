# TWR rehearsal season — handoff for Tom (Claudebox, 11-22 ~2 PM CT)
Christopher's rulings from today's grill. This now goes AHEAD of Stages 3-7 in TWR-BUILD-PLAN-1122.md
(Stage 2 v50/v51 tonight still happens first). Questions: TOM-ASKS-CLAUDEBOX-<topic>.md as before.

## Why
About 6 weeks of real NFL regular season left (then playoff weekends as extras). Use them as a dress rehearsal so
week 1 next season runs with no hitches. You only get one first impression: the live screens must be right.

## Rulings (do not reopen)
1. Two NATIVE 32-team leagues, run entirely by TWR (TWR is the record): the default 32-team league and the
   twin-pool league (AFC 16 / NFC 16, each side has its own copy of every NFL player; same lineups, rules, scoring).
2. Christopher owns one team in each. The other 62 teams: automatic best-projected healthy lineup every week
   (a plain rule, not AI; also the fallback for an owner who forgets). No waiver claims for them yet.
3. Stand-ins (AI owners): not a focus before Dec 6; no date. They take over from the auto-lineup rule later.
4. Draft: in the EXISTING draft room in the desktop PWA. No clock on Christopher. Every other pick is automatic and
   near-instant: next best available on the list. Picks drop off the board, then his turn. Build nothing new.
   One full TEST draft in each league first; real drafts only after the tests are clean. If clean, he live-streams
   the real drafts from that draft room. Backup if the draft is not ready: auto-fill rosters so Sunday does not slip.
5. First rehearsal Sunday: Nov 29, on the Proxmox lab.
6. Laptop = season server (being set up by Christopher this week). Leagues move there once it is ready.
   It updates Saturday night only, with Christopher's OK each time (hand him the exact command). Frozen on game days.
7. Proxmox lab = test bench. Open for live fixes during build-session Sundays (each fix on a branch with its test,
   merged properly afterwards). This replaces the old "no lab release noon-10 PM" rule for the lab only.
8. Monday report card (automatic, on Hermes): recompute every team's score from official stats + our scoring rules
   and flag any difference; lineups locked at kickoff; every game final; standings match results; jobs ran on time.
   Posted on the tracker as green/red lines.
9. The wall shows the REAL live screens of that week's build, rendering hitches fixed (overlaps, uncentered text,
   e.g. T683 panel split, T686 football over score). Must be in the Sat Dec 5 update.
10. Dec 6, two separate tracks:
    a. Report card: record all Dec 6 games. Overnight, Hermes replays every play and captures the live screens at
       every PHONE and TABLET size in Chrome DevTools' device list (one of each) plus the S25 size (size only;
       the real S25 is not touched). Automatic checks: text overflow, overlap, off-centre. Output = exactly what the
       auto-resizing code must handle. One week only.
    b. Live build session: the real Samsung S8 and project iPhone on a stand, connected over the INTERNET (not home
       wifi) to the Proxmox lab, screen-recording the whole game window. Christopher and Tom fix issues live and
       hot-patch the lab. Lag over the internet is a known issue to chase.
11. Reuse what exists before building anything new (standing rule from Christopher).

## Your order of work (after tonight's v50/v51)
A. By Thu Nov 26: both native leagues created (default + twin-pool), Christopher's 2 teams, auto best-projected
   lineup rule, draft room auto-pick working. Test draft in each league. Card to needs_you: "ready for real drafts".
   First check what exists (draft room, twin-pool support, lineup lock, settlement for native leagues) and post the
   gaps on the tracker before building.
B. Thu/Fri Nov 26-27: Christopher runs the real drafts (maybe streamed). Fallback: auto-fill.
C. Sun Nov 29: first rehearsal Sunday on the lab. Christopher sends screenshots; Claudebox turns them into tasks Mon.
D. Mon Nov 30: first Monday report card on Hermes (build the checker this week, after A).
E. Week of Nov 30: wall rebuilt on the real live screens + fixes; Hermes overnight replay sweep built alongside;
   confirm the lab is reachable from the internet for the real phones. All in the Sat Dec 5 update.
F. Sun Dec 6: both tracks (10a, 10b).
G. Each Saturday night: season-server update with Christopher's OK. Each Monday: report card. Repeat to end of season.
Stages 3-6 of the build plan wait until after Dec 6 unless they directly help this.

## Done lines
- Both leagues drafted and showing real matchups on the lab before kickoff Nov 29 (tracker card + screenshot).
- First Monday report card on the tracker Nov 30, every line green or explained.
- Dec 6: recordings + overnight sweep report (pass/fail per device size) + real-phone recordings, all on the tracker.
- End of regular season: several clean report cards in a row, and every flagged issue closed.
