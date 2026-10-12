# Ring 3 RESUME — 2026-10-12 16:35 CT (third compaction of the day)

## Goal / deadline
/goal: complete ring 3 to excellence (Christopher 16:00: "I would really like to get ring 3 done"). He demos
TheWarRoom to league mates at MNF BUF @ LAR 7:15 PM CT tonight. Freeze ~17:30, build ready ~18:45 (tell him
numbered steps), then watch live polling on BUF@LAR (strip, followed bar, field, spine, MFL live scores).
Endpoint index cards: LEAVE AS IS (Christopher 15:15).

## Branch state (session/ring-3, worktree ~/work/TheWarRoom/.worktrees/ring-3, local, unpushed)
Committed since 14:45: def06ff (one Pools report, trophy names, survivor words, unset CFBD source = "not set up"
not the red alarm), d2a87b8 (league_event reviewed), 445a80a (fact exam vs MFL standings pages 160/160; pages in
RUN/mfl-pages; opt-in TWR_MFL_PAGES), e8e2645 (archive warmed after launch), b748979 (HQ "Reading lineup from
MFL…", seasonChanged right after lineups, "Reading the league…" before snapshot, Comms drawer buttons), then a
RESUME-only commit. Claude-OS runs ring3-c01a (first asset-board build; tiles overflowed; superseded).

### UNCOMMITTED in ring-3 (C-01 asset board + checks grouping), 21 paths — NOT YET GATED
- asset_board.go (+asset_board_test.go): TargetAssetBoard → AssetBoard{Values[{id, now, dynasty, market:
  AssetCell{text, sub, detail, held}, nowValue, dynastyValue}], nowSource, dynastySource, how}. On-field-now =
  PPG over held weekly playerScores (calls hydrateResearch first), Sub "QB 14 of 54 · top 26%" (topPercent).
  Dynasty/market via app interface `assetModel{player(pid, at) dynastyValue}`; app.go field
  `assetModel atomic.Pointer[assetModel]` nil → "Not valued yet · Lab model not compiled".
- frontend/src/app/assets/{contract.ts,state.ts,AssetNumber.tsx,AssetBoardView.tsx,assets.test.ts}; card
  tiles + matrix columns (matrix.ts assetColumn) + PlayerInspector (On-field-now/Dynasty value/Market; Dead cap
  and Form stay "not wired"); AssetBoardView lazy above Pool in War Room › Valuations and pool (data-endpoint
  C-01; franchise filter default mine, position filter, sort by nowValue). look.css .asset-value/.asset-sub.
- Registry: C-01 status built + ring_note; endpoints.gen.ts regenerated; registry.test built list has 'C-01'.
- data_sources_reading.go checkItemRows: check differences sharing reason+class listed once ("N differences\n…");
  TestCheckDifferencesSharingAnExplanationAreListedOnce.
- Bindings regenerated (make bindings) at 16:30 after the last Go struct change.
- 16:33 a background gate run was KILLED (a pre-commit hook stashed/restored the tree mid-run); nothing deployed.
NEXT 1: run `make bindings`, verify, build (foreground or background, but DO NOT commit or edit the repo while a
gate runs — the pre-commit hook stashes unstaged files), deploy as ring3-c01c, live look (HQ tiles "22.00 PPG"
over "QB 14 of 54 · top 26%"; Inspector rows; Valuations and pool board; Control Room › Data and sources checks
grouping), then commit.

## Sol p3g2-s2b: DONE 16:23, awaiting Claude review in ring-3-d
Report RUN/p3g2-s2b3/REPORT.md: 508,219 values, 0 differences on 18 keys (P4 with Go analysis, no recorded
analysis); goldens (2.98 MB) pass; P3 reduced smoke; valuer API Build/Model.Player/Model.Pick checked on 258 real
legs; Sol's mutant (in-season prior 6→7 weeks) caught; verify+build 0. Lab compile +8.9 s, +69 MB RSS.
Worktree ~/work/TheWarRoom/.worktrees/ring-3-d (session/ring-3-d at 4f65ea7; Sol's work UNCOMMITTED, 19 paths).
ring-3 does NOT contain internal/labcompile at all (4f65ea7 is not in ring-3).
NEXT 2 (s2b review): my mutant = internal/labcompile/forecast.go ageBand edge 24.5 → 25.5; run
`go test ./internal/labcompile/ -run Goldens` and P4 `TWR_ARCHIVE_PARITY=1
TWR_FACTS_ARCHIVE=/home/chris/work/TheWarRoom/league-archive/raw go test ./internal/labcompile/ -run
TestFullAnalysisParity`; both should fail; revert; rerun P4 clean. Skim model.go asset(), valuer.go,
value_trades.go; commit in ring-3-d (`git commit -F`). Then delete RUN/p3g2-s2b3/cache and tmp.
NEXT 3 (s3 wiring): merge session/ring-3-d into ring-3 (expect SYSTEM_MAP.md/.golangci.yml conflicts); export
labcompile ClassLabel; add asset price range (Discounted*classLo/Hi + perAsset) to AssetValue; app adapter
→ dynastyValue (Seasons=len(Forecast)); after compileFacts + warmArchive, labcompile.Build with app Evidence,
store a.assetModel, emit target:season. Then the Lab tab reads the runtime Lab + Control Room source lines.
Demo inclusion decided by ~18:00: ship only if gates + live look are clean; otherwise the demo runs on c01c.

## Remaining ring 3 after that
Spec revision 3 (BACKLOG.md "Spec revision 3 notes": Control Room League tab rename with shell.test, M-141 as
articles, S-05/S-10, §13 deferred-row wording, shell verb drift); registry gate (C-01 was the only kept ring-3
row still spec; the other 19 are merged/retired); IR alert: alerts_app.go irAlerts marks IR "unavailable"
unless injuries are FreshLive — look at Home › Alert tray (415,86) live before changing; Christopher's
click-through; PR squash-merge after his OK.

## Environment notes
- Gate env: PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
  GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1 (set BOTH vitest vars or it
  refuses to start). Frontend lines ≤110 incl. tests. Bloat ratchet: no new Go file under 40 code lines.
- Deploy: gzip build → scp claudeos:warroom-ring1/thewarroom.new.gz → gunzip → deploy.sh <label> (keeps 2
  backups now).
- Claude-OS SSH outage 15:49 was the host passt crashing; if `pgrep -fa passt` shows no Claude-OS passt, run
  `passt --socket /run/user/1000/libvirt/qemu/run/passt/1-Claude-OS-net0.socket --tcp-ports 127.0.0.1/2222:22`.
- Hygiene: delete <phase>/cache and <phase>/tmp when a phase merges. Claude-OS 22 GB
  coding-standards-verification is ClaudeBox's; Christopher will have it cleaned.
- Never pkill -f with a pattern that appears in your own command line (it killed the checking shell once).
- Clicks: nav Home (63,204) War Room (75,240) HQ (87,276) Trade (80,312) Pulse (87,348) Control (87,384);
  War Room tabs Valuations (430,86) Research (540,86); Pulse tabs Now (338,86) Race (393,86) Archive (479,86);
  Home Alert tray (415,86); Comms C (1256,90).
