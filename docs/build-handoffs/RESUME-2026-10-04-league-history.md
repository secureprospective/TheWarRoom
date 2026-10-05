# RESUME — TheWarRoom, league-history archive (2026-10-04, late evening)

Supersedes `RESUME-2026-10-04-after-power-rankings.md` for next actions; its §1–§3 still hold.

## 0. Next actions, in order
1. **Christopher has head-brain (CT105) work for you after the compact.** Take it while the MFL
   archive keeps running. The standing CT105 rules still apply unless he directs otherwise:
   observe, record, report; do not race it.
2. **Watch the MFL archive** (§1, in flight) until `DONE`. Then:
   - Summarise `manifest.jsonl` by year × type: ok / empty / mfl-error / failed.
   - Re-run `python3 archive.py` to fill any failures. It skips valid files.
   - **On 2026-10-05 or later**, run `python3 archive.py players`. That pulls the per-season
     `players&DETAILS=1` (birthdates, college, draft). MFL asks that its player database be
     fetched once a day, and the basic lists were already pulled on 10-04.
   - Then **shred `league-archive/.secrets/mfl_cookies.txt`** (Christopher's MFL login).
3. **Write the full phase 1–3 plan (his 4A)** for approval. Plan only; build nothing until he
   approves:
   - Phase 1: bring the archive into our own database.
   - Phase 2: refit the matchup model on 13 seasons, judging each game only on what was known
     before it. Bee's caution: test pooled eras against recent-only.
   - Phase 3: the Tuesday roundup and Thursday preview as text cards, plus a printable PDF and a
     computed fact sheet built for him to paste into Sol 6.1 on Pi. No in-app LLM.
   - Use the archive inventory from step 2, and Bee's report (§4).
4. Queued from before, on his go only: remove NFL contract data (footprint in the previous
   RESUME §0 item 2).

## 1. In flight: the MFL archive pull
- **What:** `~/work/TheWarRoom/league-archive/archive.py`, run as `python3 archive.py`. It is
  PID 192753, started 21:37 from a Claude Code background shell. It requests every MFL export
  for 2013–2026 one at a time, 6 s apart, including per-week rosters, playerScores,
  projectedScores, injuries and nflSchedule (weeks 1–17).
- **Rate and ETA:** about 9 requests a minute, about 1,700 requests in all. It was in 2014 week 8
  at 21:53. **ETA about 00:30–01:00** on 10-05.
- **On HTTP 429:** it waits 900 s, then 1,800 s, then exits with status 2.
- **Is it alive?**
  - `ps -o pid,etime,args -p 192753`
  - `tail -3 ~/work/TheWarRoom/league-archive/archive.log`
  - The log ends with `DONE` and then `exit=0`.
- **If it died:** re-run `cd ~/work/TheWarRoom/league-archive && python3 archive.py >> archive.log 2>&1`
  in the background. It resumes. Never run two at once: MFL throttles.
- **Output:**
  - `raw/<year>/<type>[_param].json` and `raw/<year>/weekly/<type>_Wnn.json`.
  - `raw/<year>/playoffBracket_<id>.json`.
  - `manifest.jsonl`, one line per request.
  - `archive.log`.
- `league-archive/` is in `.git/info/exclude`: league data and members' posts never enter the
  public repo. **The code is not in git.** `archive.py` lives only there, and so does the
  record-book `build.py` (§3).

## 2. Archive facts learned (do not re-derive)
- **Hosts and league ids:**
  - 2013: `www48`, L=51719. 2014: `www46`, 47710. 2015: `www45`, 21225. 2016–2021: `www45`,
    14432. 2022 on: `www47`, 14432.
  - Franchise ids 0001–0032 are the same team in every season.
- **Shared exports go to `api.myfantasyleague.com` with no `L=`:** injuries, nflByeWeeks,
  nflSchedule, adp, aav and the top* lists. A request that includes `L` is redirected to the
  league's host, which rejects it: "must go to api.myfantasyleague.com".
- **Christopher's MFL login** (Brave on Claude-OS) is in `league-archive/.secrets/mfl_cookies.txt`
  (mode 600). It holds 4 MFL cookies; `archive.py` sends them as a Cookie header.
  - **His account is not a member of the 2013–2014 leagues** (51719, 47710): "API requires
    logged in user in league ID 51719". So messageBoard, polls, calendar and assets are refused
    for those years.
  - The MFL messageBoard for 14432 is empty (`{}`): the league talks on ProBoards.
  - The Claude-OS cookie file was shredded. `~/twr-history/secrets/export_cookies.py` on
    Claude-OS decrypts Brave's v11 cookies through libsecret (no values printed). Re-run it only
    if the MFL login expires (MFL_PW_SEQ expires 2026-11-04).
- **WeeklyResults has "W=YTD":** one request returns every week of a season (`allWeeklyResults`).
- **The home flag is a schedule artifact.** Some teams are home 12 times a season and others once
  or never, so do not treat it as home-field advantage.
- **Defining a season:**
  - Regular season = the weeks before the first playoff bracket: 13 in 2013, 12 in 2014–2020,
    13 from 2021.
  - 2013 has no MFL bracket. Its playoffs are schedule games in weeks 14–17, with BYE
    placeholders.
  - MFL's standings PF counts weeks 1–17.
  - MFL's standings records include placeholder games in 2013, 2014 and 2021.
  - Our records match MFL's exactly for 2015–2026.

## 3. Artifacts
- **The record book:**
  - `~/fleet/runs/warroom-league-history-2026-10-04/Legacy_NFL_Complete_Record_2013-2025.pdf`
  - 6,267,354 bytes, sha256 `7c9a36a7…a41f9`, 63 landscape pages.
  - Built by `build.py` from `raw/`, with every number in `facts.json`, rendered by headless
    Chromium on Claude-OS.
  - Totals: 2,592 regular-season games, 155 playoff games, 104,961 starter performances; 2026
    through week 3.
  - Champions: ARI 3 (2013, 2016, 2017); TEN 2 (2020–21); CLE 2 (2022, 2024); then NYG, LAR, LV,
    GB, ATL and BUF 1 each.
- **Bee's research** (Sol 6.1):
  - Report: `~/fleet/runs/sol61-bee-warroom-league-history-2026-10-04/stdout.log` (54,459 B).
  - Brief: `~/fleet/briefs/bee-warroom-league-history-2026-10-04.md`.
  - Its top 5:
    1. Historical calibration plus the Thursday preview.
    2. The Tuesday roundup from computed facts.
    3. Era-aware records and milestones.
    4. Offense/IDP stories.
    5. Playoff paths by simulation.
  - Its cautions:
    - Forecasts must use only what was known before kickoff.
    - Test pooled eras against recent-only.
    - Rivalry and manager history is story, not predictor.
- **Memory:** `warroom-archive-everything-from-mfl`, with his answers and the ProBoards ruling.

## 4. Decisions (Christopher, 2026-10-04)
- **His idea:** use all-play plus W/L history across seasons to find league norms for a better
  matchup model, a Tuesday roundup reel and a Thursday matchup-preview reel. Planning only until
  approved.
- **Answers:**
  - 1A: text cards in the app.
  - 2A: just him, plus a printable PDF.
  - 3B: templates only; he pastes the facts into GPT Sol 6.1 on Pi himself.
  - 4A: write the full plan for approval.
- **"I want everything, literally everything":** archive all MFL history to data-mine human
  behaviour and season norms.
- **"We don't need ProBoards; the behaviour is better data-mined on MFL."** No board archive.
  (Its robots.txt bans AI and data-mining crawlers by name, and ProBoards has no export.)
- Earlier rulings stand: no NFL contract data; free tools only; the repo is public.

## 5. Refuted — do not retest
- Probing export types instead of pulling everything: he rejected it. Pull everything.
- **Throttling:** 2.5 s between requests, plus error-returning calls, drew HTTP 429 after about
  60 calls. 6 s spacing has held with no 429 so far.
- Sending an `L=` with shared exports fails (see §2).
- `players` was not re-pulled on 10-04 (once-a-day courtesy). Use the `players` subcommand
  tomorrow.

## 6. Environment
- Claude-OS: `ssh claudeos`; Brave runs there with `--remote-debugging-port=9222`. Leave it
  running.
- PDF rendering:
  `chromium --headless=new --disable-gpu --no-pdf-header-footer --print-to-pdf=out.pdf file://…`
  on Claude-OS.
- Kill processes by PID, never with `pkill -f` (it matches the calling shell).
