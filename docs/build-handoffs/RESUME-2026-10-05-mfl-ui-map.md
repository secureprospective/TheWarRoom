# RESUME — TheWarRoom: MFL UI map (Bee) + league archive · 2026-10-05 07:10

Supersedes `RESUME-2026-10-04-league-history.md` for next actions. That document's §2 (archive
facts) and §4 (decisions) still hold.

## 0. Next actions, in order
1. **Gate whatever the driver has finished** (§1). For each sector that logs `END`:
   - `python3 ~/fleet/bin/gate-mfl-ui-map.py S<n>`: the mechanical gate.
   - `python3 ~/fleet/bin/coverage-mfl-ui-map.py S<n>`: page kinds linked from the sector's opened
     pages that are not captured anywhere. Level-3 links from Level-2 pages are allowed.
   - Then read nodes against their screenshots.
   - **Accept:** append a line to `logs/gate.log`.
   - **Send back:** write `reports/FEEDBACK-S<n>.md`, delete `S<n>.done`, and relaunch the driver
     if it has finished.
   - Pending: **S9** (home face, running), **S10/S11** (the 17 NEW home destinations; feedback
     queued), **S4** round 2 (three Draft Results views, feedback queued).
   - Also read **S5** and **S7** by hand. They passed only the mechanical gate, overnight.
2. **Re-run S8 last**, once S1–S7 and S9–S11 are accepted. Its first run happened before S6 was
   finished, and reported only 6 nodes.
   - Write `FEEDBACK-S8.md`: rebuild from all accepted sectors plus `REVIEW-NOTES.md`.
   - Delete `S8.done`, then relaunch the driver.
3. **Assemble the deliverable for Christopher:** the S8 menu tree, the S9 home face, the S10/S11
   trees, and the screenshots. Decide the format with him; a PDF suits his habit. This is research
   only: no redesign, no build.
4. **Run `cd ~/work/TheWarRoom/league-archive && python3 archive.py players` on 10-05, after about
   18:00.** MFL asks for its player database once a day, and the basic list was pulled 10-04
   evening. The cookie file is already shredded; `players` needs no login.
5. **Write the phase 1–3 plan (his 4A)** from the previous RESUME §0 item 3, now that the archive is
   complete.
6. Queued on his go only: remove the NFL contract data.

## 1. In flight
- **The driver:** `~/fleet/bin/run-bee-mfl-ui-map.sh`, PID 577782, started 06:58, detached
  (setsid).
  - It runs sectors S1–S11 in order and skips any sector with an `S<n>.done`.
  - Each sector is a fresh `pi -p … gpt-6-luna --thinking high` session, 3 h timeout.
  - A session that ends without its `.done` is resumed, up to 2 more times.
  - Then a second pass over every unfinished sector, then `ALLDONE`.
  - Feedback is moved to `reports/.history/` once its sector completes.
  - It snapshots a sector's reports into `reports/.history/<S>-<stamp>/` before every run.
- **At 07:08 it was on S9, resume attempt 2** (session 04a01ecf…). Its queue: S9 → S10 → S11, then
  the second pass, which reaches S4.
- **Is it alive?**
  - `ps -eo pid,args | grep -E "bash /home/chris/fleet/bin/run-bee-mfl|bee-mfl-ui-map-S" | grep -v grep`
  - `grep -E "^(START|END|ALLDONE)" ~/fleet/runs/bee-mfl-ui-map-2026-10-04/logs/driver.log | tail`
- **Is a session alive?** Check `ss -tnpi` for the pi PID's `bytes_received` growing. Transcript
  mtime only moves when a turn completes, and Luna can stream for 30+ minutes in one write.
- **Recovery:** each sector's session transcript is named in the driver log, under
  `~/.pi/agent/sessions/`.
- **Watch for:** `n=$(grep -c '^END\|^ALLDONE' …/driver.log); until [ "$(grep -c '^END\|^ALLDONE' …)" -gt "$n" ]; …`.
  Count the same pattern on both sides: a stale `ALLDONE` fooled the watcher once.

## 2. What this is
- **Christopher's head-brain task (10-04):** Bee (Luna) maps MFL's menus (Scores, Transactions,
  Player Research, Draft/Auction, League, My Franchise) and the league home page, including every
  click and the tree beneath. It is a research pass for a later UI/UX redesign: cards, graphics and
  colour instead of lists. **No builds.**
- His bar: "maintain excellence, don't let Bee (luna-6) slack off or cut corners."
- **Run dir:** `~/fleet/runs/bee-mfl-ui-map-2026-10-04/`.
  - `capture/pages/*.json`: page structure. `capture/shots/`: full-page JPEGs and viewed PNGs.
  - `capture/menus/0002-menu.json`: the authoritative menu.
  - `capture/nav.log`, `capture/denied.log`, `capture/site-chrome.json`.
  - `reports/S<n>-*.md|.nodes.tsv|.done`, `REVIEW-NOTES.md`, `logs/`.
- **Brief:** `~/fleet/briefs/bee-mfl-ui-map-2026-10-04.md`. It now includes "Rules learned from S1"
  and "How a node is written".

## 3. Tooling (in `~/fleet/bin`, local git, commit b804e3a)
- **`mflmap.py`** (also at Claude-OS `~/bin/mflmap.py`) and its Beelink wrapper **`mfl`**
  (`MFL_RUN=<run> mfl go|here|menu|hover|click|shot|text|status`). Behaviour:
  - Drives its own Brave tab on Claude-OS over CDP port 9222; the id is in Claude-OS
    `~/.cache/mflmap/tab`.
  - Opens only `www<N>.myfantasyleague.com`, over https.
  - Deny list: logout, csetup, commish, delete, submit, cancel, `action=`, `cmd=`, save, clear,
    cache, export, and others.
  - ALLOW exceptions: `csetup?C=FRANCHISE`, and `O=17 … CMD=GRID|SB|RECAP`.
  - At least 10 s between navigations; on 429, a 15-minute cooldown.
  - HTTP ≥ 400 is recorded as an error and the run continues. "NOT LOGGED IN" fires only on a real
    MFL page with no `franchise_id`.
  - Splits site chrome (from `~/.cache/mflmap/chrome.json`) from page-body clickables.
  - Redacts email addresses before writing JSON.
  - Records select options.
  - After any `mflmap.py` edit, `scp` it to `claudeos:bin/`.
- **`mfl-facts.py <capture.json>`:** the generated Facts block that every node must append
  verbatim. The gate checks it against a fresh run.
- **`mfl-destinations.py`:** the home-page destination table, NEW / MAPPED / external /
  out-of-scope / action. It is installed as `reports/S9-home-face.destinations.tsv`: 17 NEW
  kinds, numbered `new#` 1–17.
- **`gate-mfl-ui-map.py S<n>`** and **`coverage-mfl-ui-map.py S<n>`:** see §0.

## 4. Gate status (07:08)
| Sector | Status |
|---|---|
| S1 Scores | ACCEPTED (4 rounds). Its 403 and guard denials are carried in REVIEW-NOTES for S8. |
| S2 Transactions | ACCEPTED. The trade builder by URL is "Commissioner Access Required". |
| S3 Player Research | ACCEPTED (26 nodes). "Likely" hits are the player Isaiah Likely. |
| S4 Draft/Auction | Sent back: open the CMD=GRID/SB/RECAP views, now allowed. |
| S5 League / S7 My Franchise | Mechanical pass only, overnight. Read by hand. |
| S6 All Reports | ACCEPTED: 33 of 34 hub report kinds captured; Commissioner News guard-denied. |
| S8 Menu tree | Stale (ran before S6). Re-run last. |
| S9 Home face | Running (feedback: finish the inner tabs and the face regions). |
| S10/S11 Home trees | Redo against the generated list: NEW 1–9 and 10–17. Old versions in `.history`. |

## 5. Facts established (do not re-derive)
- **Login:** Christopher is logged in as franchise **0025 (Arizona Cardinals)**, an owner, not
  commissioner. Re-verified 06:56.
- **`detailed?L&W&P[&F]`** (weekly scoring detail) returns **HTTP 403** to owners.
- **Trades:** `trade_response?…&ACTION=revoke` is a plain GET link on a pending offer, and the guard
  refuses it. "Submit Lineup" (`O=02`) redirects to `lineup?L&FRANCHISE`.
- **Home page:** 8 custom tabs (HOME, MY TEAM, REPORTS, PLAYERS, LATEST NEWS, PLAYOFFS, POOLS,
  WAR ROOM), plus inner tab rows. The skin is from mflscripts.com. Site chrome is about 254
  clickables on every page.
- **The six in-scope menus have 50 entries:** 11, 8, 13, 4, 9 and 5.
- **Claude-OS never suspends** (`suspend.target` is masked). X blanking was turned off with
  `xset s off` for this session only.

## 6. Refuted — do not retest
- **Handing Bee the mechanical counts:** they came out wrong (negative counts, "0 ×"). Facts are now
  generated by `mfl-facts.py`.
- **"Rebuild from scratch" in feedback:** Bee deleted good work. Say "keep X, append Y", and the
  driver now snapshots first.
- **`pkill -f`:** it killed my own shell. Kill by PID only.
- **Editing a running bash driver in place:** never. Write to `/tmp`, then `mv` over it (the running
  copy keeps the old inode).
- **sed-redacting emails inside JSON text:** it ate `\n` escapes. Redact the parsed strings
  (`redact()`).
- **Background-task notifications can arrive hours late** (01:41 → 06:55). Do not rely on them alone
  overnight. Ungated sectors ran on autopilot.

## 7. MFL archive: COMPLETE (10-04 21:32 → 10-05 ~00:20, exit 0)
- **Totals:** `~/work/TheWarRoom/league-archive/raw/`, 1,705 files, 96 MB, all valid JSON, 0
  FAILED. `manifest.jsonl` has the latest status per file: ok 1,559, mfl-error 97, empty 49.
- **Every error or empty result is expected:**
  - Auctions, survivor pool and accounting were never used.
  - 2026 weekly rosters W05–W17 are future weeks.
  - The MFL-wide top* lists have no data.
  - **2013–2015** messageBoard, polls, calendar and assets need membership of league ids
    51719/47710/21225, which Christopher's account lacks. That includes 2015; the earlier note
    said only 2013–2014.
- **Credentials:** `.secrets/mfl_cookies.txt` is shredded. `archive.py` falls back to no cookie.
- **Tooling:** `health.sh` gives one-line health; the 15-minute checks are finished.

## 8. Ledger
- **Committed:**
  - `~/fleet/bin` b804e3a (the tooling).
  - This RESUME, in the TheWarRoom repo on `session/after-power-rankings`.
- **Not in any repo:**
  - The brief (`~/fleet/briefs` is not git).
  - The run dir, which holds league data and must stay out of the public repo.
  - `league-archive/` (git-excluded).
- **Task list (Hermes):** T378, plus the MFL-map item under PARKED.
