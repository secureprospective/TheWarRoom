# Spec: the small MFL helper in the app (T710 step 1)

**Status:** DRAFT for Christopher's OK, 2026-11-23. Written by Tom.
**Ruling it serves (11-23, `docs/reset/ASSUMPTIONS.md` A3):** "rebuild small, use it for reference to keep from making
mistakes". The desktop program goes away (A1, A2); these helpers live only inside it today, so they are rebuilt small in
the app first. Then the desktop can be deleted.

## What the owner sees

An owner whose league still lives on MFL opens TheWarRoom on a phone or a browser and finds **My MFL moves**:

1. **Free agents.** The league's free-agent list from MFL, refreshed when you ask (at most once every few minutes), with
   TheWarRoom's value beside each player.
2. **Plan a waiver bid.** Pick a player, an amount and who you would drop. TheWarRoom checks the bid against your
   league's rules before you go to MFL: cap room after the bid, roster size after the drop, a bid within the league's
   limits. A problem is said in plain words ("this leaves you $2 over the cap").
3. **Plan a lineup.** Choose your starters. TheWarRoom checks the lineup is legal for this league (right number at each
   spot, nobody on bye or on IR without saying so).
4. **Then you make the move on MFL yourself.** Each checked plan has an **Open on MFL** button that takes you to the right
   MFL page, and an **I did it** button once you have. A plan you change your mind on can be withdrawn.

**TheWarRoom never sends anything to MFL.** It reads public league data and checks your plan. You make the move on MFL.

## What is left out (on purpose: small)

- **Incoming trade offers.** Only your own MFL sign-in can see them, so the app would have to hold your MFL password.
  Recommendation: leave them out and read offers on MFL. (The desktop had this; it is the one helper not rebuilt.)
- Bid history, bid strategy tips, a calendar of waiver runs: not rebuilt.
- No new storage kind: plans are kept the way owner-private notes already are (private per owner).

## How we know it works

- A test per check (cap, roster size, lineup legality) that fails without the check.
- The old desktop's checks are used as the reference: the same plan gives the same answer before the desktop is deleted.
- Click-through on the Lab with a real MFL league read-only (Legacy 14432): plan a bid and a lineup, see the checks,
  press Open on MFL (no click on MFL itself).

## Order after this is approved

1. Build the four pieces (Gemini, about a day and a half), reviewed by Tom with a planted break.
2. Delete the desktop program, with a second-model review (Argon) because it deletes a lot of code.
3. Reword CLAUDE.md and `docs/architecture/desktop-modes.md` in the same change.

## Questions for Christopher

- Leave incoming trade offers out (read them on MFL)? Recommended: yes.
