# MFL UI map delivered as a PDF
**Date:** 2026-10-05
**Branch:** session/after-power-rankings
**Status:** done; delivered to Christopher. Research only, nothing to merge but this note.

## What changed
- **Bee (gpt-6-luna) mapped all 11 sectors of the MyFantasyLeague site, and Claude accepted every one against the screenshots:**
  - the six menus (Scores, Transactions, Player Research, Draft/Auction, League, My Franchise);
  - the All Reports hub;
  - the league home page's face and tabs;
  - the 17 destinations only the home page reaches;
  - the consolidated menu tree.
- **The deliverable PDF** (175 pages) is on the Beelink at `~/fleet/runs/bee-mfl-ui-map-2026-10-04/deliverable/MFL-UI-map-2026-10-05.pdf`. It holds league data, so it stays out of this public repo.
- **Tooling** is in `~/fleet/bin` (local git, commits b804e3a..d7bc1ba):
  - `mflmap.py` / `mfl`: the guarded CDP mapper.
  - `mfl-facts.py`, `mfl-destinations.py`, `mfl-menu-counts.py`, `mfl-menu-tree.py`: generators whose output is correct by construction.
  - `gate-mfl-ui-map.py`, `coverage-mfl-ui-map.py`: the quality checks.
  - `run-bee-mfl-ui-map.sh`: the sector driver.
  - `mfl-map-pdf.py`: the PDF build. It prints through Claude-OS's Brave over CDP, because headless Brave never prints on the Beelink.

## Why
Christopher asked for a map of how MFL's old, deep menus and home page are built. It is input to a later TheWarRoom UI/UX redesign built on cards, graphics and colour. No build.

## Verification
- Every sector passes `gate-mfl-ui-map.py` (mechanical), and each was read by hand against its screenshots. The acceptance log is `logs/gate.log` in the run directory.
- The PDF was rendered and inspected page by page with `pdftoppm`.

## Open items / what Claude or Christopher should check
- **Christopher reads the PDF.** The redesign discussion starts from its summary page.
- **Lessons for future Luna work:**
  - Luna scripted counts and tree dumps wrongly every time. Generate the mechanical parts, and let Luna write only judgement.
  - The gate now requires every template field.
- **Tonight after about 18:00:** `cd ~/work/TheWarRoom/league-archive && python3 archive.py players` (MFL serves the player database once a day).
