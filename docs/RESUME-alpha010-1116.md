# RESUME — TheWarRoom alpha 0.10 + MNF live stream (Claudebox head brain), Mon 2026-11-16
Updated 3:45 PM CT (fourth compaction today). On "we are back": do his instruction first, else section 10 item 1.
**TONIGHT IS THE PRIORITY: Christopher live-streams Live Pulse on YouTube during MNF (kickoff 7:15 PM CT).**

## 1. WHAT WE ARE DOING
(a) TONIGHT: Live Pulse on the **dev server CT 257** (`https://twr-lab-257.secureprospective.com`, league
`native.mfl-14432-2026`), as close to production as possible, every lineup and box score current, streamed live.
He ruled: "not building a new app, just prepping what we have and fixing bugs"; "focus on the Live Pulse";
"all lineups from all teams and all box scores need to be up to date in live pulse".
(b) Alpha 0.10 proving continues beside it (GPT lanes on the Beelink).
Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (main checkout is HIS; worktrees only). Lab host 192.168.1.182
(from Beelink: `ssh -F ~/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`).
CT 257 = dev (he approved using it tonight, incl. data changes for lineups). CT 258 = 0.10.1 lab. Never touch 110, 256.

## 2. DONE LOOKS LIKE — TONIGHT
| Line | State |
|---|---|
| T1 CT 257 Live Pulse shows week 10, live feed | MET: "WEEK 10 · 0 NFL live · 13 final · 1 to play", sources ESPN/Sleeper live |
| T2 all 32 week-10 lineups on CT 257 | 22 of 32 (Cards + Giants were set earlier; 20 entered by fill). **10 refused**: franchise.01 04 08 10 14 17 18 22 23 32 |
| T3 box scores current | MET for teams with lineups (player lines, points, "N left", MNF players "Mon 7:15 PM", LAC game) |
| T4 app on Beelink for him, walked by me first | NOT DONE. Use CT 257-matching build `claudeos:~/twr-d2/thewarroom-d6b2` (cs45-candidate d6b205491); copy to Beelink; he signs in as `chris` himself |
| T5 go/no-go by 6:30 PM CT with screenshots | NOT DONE |

## 2b. DONE LOOKS LIKE — 0.10 (docs/roadmap/RELEASE-0.10.md)
1 MET · 2 server MET (desktop prints `cs45-candidate`, not 0.10.x: CI does not stamp desktop version — bug) ·
3–7 NOT STARTED · 12 **MET** (CT 258 upgraded to 0.10.1 via twr-upgrade, 27 s, ledger 5006, integrity ok, twr-status clean) ·
13 PARTIAL (tampered refused, services untouched, hashes same; older-bundle not provable on 0.10.x: same release 8 — next patch adds release 9) ·
14 NOT PASSED (stress) · 15 sweep now **11 pass / 0 fail / 3 not proven** (check 9 fixed today, see 4).

## 3. IN FLIGHT
| Job | Check alive | Output | On result |
|---|---|---|---|
| Terra lane `a010o` odds-simulation CPU fix, brief `~/fleet/briefs/alpha010/odds1.md`, started ~3:25 PM | `date -r $(ls -t ~/fleet/runs/pulse-a010o/sessions/*|head -1)` | `~/fleet/runs/pulse-a010o/{stdout.log,completion.txt}` | Review PR diff myself; ask him "merge N" |
| App window on claudeos `:5` signed in to CT 257 (pid 105563, Eagles test acct, `~/twr-d2/run-a.sh`) | `ssh claudeos pgrep -af thewarroom` | screenshots via `DISPLAY=:5 gnome-screenshot -f /tmp/mnf5.png` | my private walk display; keep |

## 4. RESULTS TODAY
- PRs merged with his OK: #297 (lab tools), #299 (Linux-only CI), #300 (docs: 0.10 results, 0.11=PWA draft), #301
  (governance replay copy fix: local p95 2.3→1.4 s, RSS 721→587 MiB, CPU unchanged). Main = a38b32ca3.
- Cloudflare (driven by me in Brave on claudeos :0): redirect rule `twr force https` (http://twr-*.secureprospective.com/* →
  301 https), WAF `twr https and tls13 only` ACTIVE, `twr tls13 only` ACTIVE (redundant). Check 9: http 301, TLS1.2 403, TLS1.3 200.
- CPU diagnosis lanes a010c..c5: reproduction on real CT 258 data copy at lab `/root/twr-lab/a010c-data/` (0600). Top costs:
  governance replay (fixed #301), odds simulation 21.7% CPU (lane a010o), single write conn + pool waits.
- I installed `sqlite` (apk) in CT 258 to take backups.

## 5. ARTIFACTS
- Tags v0.10.0=44a663173, v0.10.1=55be58b03. 0.10.1 Linux deb: Beelink `~/scratch/twr-mnf-1116/pkg/warroom-linux-amd64.deb` (sha256 file beside; CI run 43935385936). Extracted `~/scratch/twr-mnf-1116/root/usr/bin/thewarroom`.
- Lab-only TUF keys matching CT 258 pin (8c91770c…): `.worktrees/pulse-a010s/.scratch/lab-release/` (never print).
- CT 257 tools on claudeos `~/twr-d2/`: `role257.py` (helpers; `chris` login reads password from env `TWR_PW`),
  `accounts-257.json` (test accts claude-commish=f32 Owner only, claude-owner-a=f31, claude-owner-b=f30), `tools/w10.py`
  (fill week-10 lineups from `/tmp/w10/wr10.json` = MFL weeklyResults W=10, all 32 have starters; preview all, execute
  with 5x reread retry; `TWR_SHOW=1` prints previews and saves nothing), `tools/w10diag.py`.
- Cards = franchise.25 (his team). CT 258 test creds for f25 in Beelink `~/scratch/twr-mnf-1116/cred25` (0600) — CT 258 not used tonight.

## 6. CURRENT BUG (tonight's blocker)
10 lineups: preview 200, execute 409 `native change refused; reread before retrying`, 5 retries each, same result.
Server swallows the real error: `internal/serverhttp/hosted.go:~447` writes generic 409 and logs nothing (BUG to fix later).
Hypothesis (unproven): those MFL starters include players not on the CT 257 roster (MFL moves after import) or a
rule the preview does not check. Next: read preview bodies (TWR_SHOW run) to see warnings; owner accounts cannot read other teams.
**His TWR_SHOW command HUNG** (`... > ~/Downloads/w10-preview.txt` with ssh -t; prompt never appeared). Fix: keep the
redirect INSIDE the remote command, e.g.
`ssh -t claudeos 'read -s -p "War Room password (chris): " TWR_PW; echo; export TWR_PW TWR_SHOW=1; python3 ~/twr-d2/tools/w10.py > /tmp/w10/preview.txt 2>&1; tail -3 /tmp/w10/preview.txt'`
then I read `claudeos:/tmp/w10/preview.txt`.

## 7. REFUTED — DO NOT RETEST
- CT 258 for tonight: no NFL week schedule in its reference.db (hosted server has no fetch path; schedule only arrives
  with a desktop league move) → Pulse "The league has not said which week it is playing". Use CT 257.
- The 10 refusals are not a timing race (5 rereads each, still 409).
- Host-side sqlite on CT 258 via /proc/<pid>/root fails (unprivileged CT); use pct exec + apk sqlite.
- cux `click` needs ydotool (missing): drive :0/:5 with `xdotool` directly (DISPLAY set).
- Earlier: league-load order, value.read 409 (F51 absent), guard `>`+`~/.ssh`, Mac CI, sweep tool bugs.

## 8. DECISIONS (today)
0.11 = PWA (+ Update button). Desktop Linux only. Upgrade test: "run now, fix later" (release 9 in next patch).
Tonight: record/stream on CT 257; set all lineups + Cards seat ok; not building new app; focus Live Pulse; YouTube live.

## 9. LEDGER
Main a38b32ca3. This RESUME: branch `session/a010-docs` (worktree `.worktrees/pulse-a010`). Uncommitted: none of mine
except scratch tools on claudeos and briefs in `~/fleet/briefs/alpha010/`.

## 10. NEXT ACTIONS
1. Give him the fixed TWR_SHOW command (section 6); read `/tmp/w10/preview.txt`; find why the 10 refuse; fix data or
   enter corrected lineups (he runs w10.py again; I never hold his password).
2. Copy `thewarroom-d6b2` to Beelink (`~/scratch/twr-mnf-1116/thewarroom-257`), give launch line:
   `~/scratch/twr-mnf-1116/thewarroom-257 -native https://twr-lab-257.secureprospective.com native.mfl-14432-2026`; he signs in as chris.
3. Walk Pulse on claudeos :5 at kickoff-like state; screenshots; check nothing private on screen (test names, tokens).
4. Go/no-go to him by 6:30 PM CT.
5. During game: watch live tick on CT 257 (`grep "live stats" /var/lib/twr/server.log` via pct exec 257, read-only).
6. After: odds lane result; release-9 row + 0.10.2; fix hosted.go swallowed-error 409; CI desktop version stamp; LAB.md root text.

## 11. NEEDS CHRISTOPHER
- Run the section-6 command (his password). - Later: `twr-down` CT 258.

## 12. ENVIRONMENT
Times Central. Never print /etc/twr/* or passwords. Lanes: `ssh -f chris@192.168.1.191 'nohup setsid ~/fleet/bin/launch-pulse-lane.sh terra <lane> <brief> high >/dev/null 2>&1 &'`.
Tidy/filing check false-fails on Bee's `~/scratch/bee-resume-v14-*` symlinks. Pre-push needs `PATH=/usr/local/go/bin:$HOME/go/bin:$PATH`; force-push blocked (merge main instead).

## 13. HONEST STATUS
Pulse on CT 257 works with live sources and 22/32 lineups. Stream readiness ~75%: 10 lineups blank, app not yet on
the Beelink, CT 257 is an older build. Live tick during a game on CT 257 not yet observed today.
