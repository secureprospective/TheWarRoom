# RESUME — Legacy NFL Behavior Lab (2026-10-05)

## 1. What we are doing
Built an interactive app (the "Behavior Lab") so Christopher can study long-term GM
habits in Legacy NFL (32 teams, IDP, salary cap, MFL league 14432), as input for
building a better TheWarRoom. It is **not** part of TheWarRoom.
Repo `~/work/TheWarRoom` on the Beelink, branch `session/league-history-pwa`, folder
`tools/league-history/`. It is live at http://localhost:8765/ (Beelink only).

**Christopher's instruction at compact: "We are in a really good place … when we get
back from compact let's talk before changing anything."** → On resume, make NO code
changes. Talk first.

## 2. Agents + harnesses
Claude built everything this session, with no subagents. Bee's earlier explorer (same
folder, commit 94bcf88) is replaced; Christopher was unimpressed by it. No dispatches are
running.

## 3. Gates / status
| Check | Result |
|---|---|
| `python3 -m unittest compile/test_build_lab.py` (12 data checks vs raw archive) | pass |
| `node --test tests/engine.test.mjs` (11 filter-engine tests) | pass |
| `node tests/screens.mjs http://127.0.0.1:8765/ <dir>` (11 screens, no errors) | pass |
| `node tests/interact.mjs http://127.0.0.1:8765/ <dir>` (10 click-through checks incl. offline) | pass |
| Repo `make verify` | not run (no Go/frontend changes) |
| Bee review (R4) | not done |
| Christopher acceptance | **pending; he wants to talk first** |

## 4. Artifacts
- App: `tools/league-history/app/` (index.html, style.css, sw.js, manifest, icons, `vendor/d3.min.js` v7.9.0,
  `js/{main,model,lens,ui,charts,stats}.js`, `js/views/{overview,calendar,market,standings,cycles,contracts,draft,franchises,explore,tradelog,about}.js`).
- Compiler: `tools/league-history/compile/build_lab.py` + `test_build_lab.py`. Server: `tools/league-history/serve.py`.
- Tests: `tools/league-history/tests/{engine.test.mjs,screens.mjs,interact.mjs}`.
- Data (private, gitignored): `data/lab.json` 4,868,891 bytes, sha256 prefix `1cf745a2f6688044` (= revision);
  `data/lab.json.gz` 629,027 bytes; `data/db_playerids.csv` (DynastyProcess, sha256 `de241875e4ae91d4…`).
- Docs: `tools/league-history/README.md` (run/check/where to change things);
  `docs/league-history/Dynasty_Market_Psychology_Research_2026-10.md` (research pass);
  `docs/league-history/League_Behavior_Lab_Build_Plan_2026-10.md` (plan, status updated).
- Screenshots: `~/fleet/runs/league-lab-2026-10-05/shots/`.
- Session transcript: `~/.claude/projects/-home-chris/35b807e1-9d84-4024-b8c1-c4ae4b4ca27e.jsonl`.

## 5. Current bug
None open.

## 6. Settled facts — do not retest
- MFL `DP_r_p` current-year pick codes are **zero-based**: round r+1, pick p+1. Zero-based last holder = drafter in
  837/837; as-written ~4%. Bee read them one round low. Bee's "Arizona March 2024: 2 R1 received" is wrong; true = 0 received, 1 sent (test locks it).
- March 28–29 2017 spike = 133 one-sided commissioner pick-ownership transfers. 361 one-sided records in total, excluded by default.
- Washington's volume (~49 trades/season, 634 total) is genuine, not an artifact; Rams–Commanders = 116 trades.
- Picks only recorded in trades from 2017; contracts fully on MFL from 2019 (~45% 2016–18, ~0 before). Pick-weighing views start 2017; cap room only where ≥85% salaries present.
- Draft order is NOT a clean reverse-standings rule (comp picks push slots past 32; standings estimate matched 51/119 proven slots). So a future pick is tied to a drafted player only when its slot is proven; else early/middle/late only.
- Archive players.json has no birthdates → ages from DynastyProcess crosswalk (98.8% coverage).
- Manifest `at` times are Beelink local, no zone.
- Selling-later analysis must be rank-adjusted (regression to the mean); the scatter now compares with teams that started within 3 places.
- The CSS `svg text {fill}` rule overrides SVG fill attributes; use `.style('fill')` for text colour.

## 7. Decisions (Christopher, this session)
- Owners/GMs are not tracked; franchise slots only ("many owners for different franchises").
- Beelink desktop only (no LAN/HTTPS).
- Bee's explorer is replaced (its files are `git rm`'d, recoverable from 94bcf88).
- Eras weighted in the data → era weight lever (presets + per-era sliders). Eras: founding 2013–14, cap 2015–16, draft-on-MFL 2017–20, taxi/13 weeks 2021–23, current 2024–26.
- No jargon or technical codes in the UI.
- Claude is the engineering authority (stack: vanilla JS modules + bundled D3, Python stdlib compiler, offline PWA).

## 8. Ledger state
- Branch HEAD `94bcf88` (Bee). **The whole Lab build is uncommitted**: new files untracked, Bee's files staged as deletions, `serve.py` modified.
  It is uncommitted because Christopher has not accepted it and asked to talk first. Only this RESUME file is committed.
- Do not merge to main; merge only on his go-ahead.

## 9. Next actions, in order
1. When Christopher says "we are back": read this file and confirm the server is up (`ss -ltnp | grep 8765`; restart with `cd tools/league-history && setsid -f python3 serve.py`).
2. **Talk with Christopher before any change.** Help him decide what to change: offer a short pick-from matrix of options (e.g. which screens to deepen, the hit definition, the contender lines, the questions most useful for WarRoom).
3. After he agrees on changes: make them, rerun the four checks, re-screenshot.
4. On his go-ahead: commit the Lab to the branch (`git add tools/league-history docs/league-history`, then commit; never `--no-verify`).
5. Possible later work (only if he asks): owner map, autostart service, feeding Lab findings into WarRoom design docs.

## 10. Environment notes
- Server: PID 1254811 at compact time, `python3 serve.py --port 8765`, cwd `tools/league-history`, started with setsid. It does not autostart after a reboot.
- Headless checks use `/usr/bin/brave-browser` and pnpm's cached playwright-core (the scripts find it themselves).
- Rebuild the data after an archive refresh: `python3 compile/build_lab.py` (~4 s).

## 11. Honest status
Built and verified headless, screen by screen and by interaction. **Not yet seen by Christopher in his own browser.** If Bee's old app shows instead, press Ctrl+Shift+R once (the new service worker replaces the old cache). Findings are descriptive patterns, not causal claims.
