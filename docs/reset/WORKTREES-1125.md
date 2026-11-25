# TheWarRoom worktrees — Stage 1 steps 1, 3, and 7

Inventory captured: 2026-11-25T12:59:34.384036+00:00. Origin main: `2ae2f9fd7a7701c5fab32647969db473b4e567e5`.
Current and baseline inventories enumerate 114 registered worktrees. An earlier independent preflight reported 115; the registration changed before this saved snapshot, and this inventory does not identify which entry disappeared.
Snapshot counts: 114 registered; 37 currently dirty; 56 meet clean AND (merged OR pushed). Baseline dirty count: 36; current status captured: 2026-11-25T13:00:50+00:00.
Removal outcomes: 41 removed; 73 retained. 74 worktrees remain, including the checkout created to file this report.
Cleanup used only ordinary `git worktree remove`. All 36 pre-existing dirty worktrees were left alone. The requested RESUME update and this report account for the two other dirty entries. Merged means HEAD is an ancestor of origin/main or exactly matches a verified merged PR head. Pushed means exact remote HEAD or verified ancestry. Local main is 59 commits behind the fetched origin/main. No branches were deleted.

## Worktree inventory (115 rows, including post-snapshot additions)

| Path | Branch | HEAD | Merged | Pushed | Dirty | Last commit | Disposition / why |
| --- | --- | --- | --- | --- | --- | --- | --- |
| /home/chris/work/TheWarRoom | main | c7b6f2337fa3 | yes | yes | yes | 2026-11-22 | retained: Uncommitted changes or unreadable status |
| /home/chris/fleet/runs/pulse-lab5/build | (detached) | b597b61228bc | yes | no | yes | 2026-11-18 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/alpha-2 | session/lineup-standings | 1bbb5edfdb9a | no | yes | no | 2026-11-21 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/astra-brief-1125 | session/astra-brief-1125 | 5838732fd64a | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/auto-lineup | session/auto-lineup | 1a4b28343f3f | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/auto-pick | session/auto-pick | c51403cc9794 | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/build-plan-wt-list | session/build-plan-wt-list | bb4d8379678d | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/codex-shape-release-1125 | session/codex-shape-release-1125 | 2ae2f9fd7a77 | yes | no | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/dv3-shell | session/desktop-v3-shell | e81224424b79 | no | no | yes | 2026-11-21 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/error-codes | session/error-codes | 9528d8cf0682 | no | yes | no | 2026-11-20 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/fix-today | session/fix-today | 578ac110b1c2 | no | no | no | 2026-11-19 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/glossary-adr | session/glossary-adr | 12ccd292f19f | no | no | no | 2026-11-20 | retained: Explicitly protected by BUILD-PLAN stage 1; registered worktree. |
| /home/chris/work/TheWarRoom/.worktrees/handoff-1125 | session/handoff-1125 | cdde2c4acff5 | no | yes | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/hk-newsread | session/nfl-news-read | ed4eac2191a5 | no | yes | no | 2026-11-20 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/lab-dev | session/lab-dev | 5c10eb16eb1e | no | no | yes | 2026-11-22 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/lab-v52 | session/lab-v52-1122 | ce74171900a3 | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/lab-v53 | session/lab-v53-1122 | a211f8f9e195 | no | yes | no | 2026-11-23 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/labrel-11242038 | (detached) | 6f9c477239c5 | yes | no | no | 2026-11-24 | retained: Registered detached staged-release worktree; ignored GoReleaser/dist artifacts present. |
| /home/chris/work/TheWarRoom/.worktrees/line-budget | session/line-budget-1122 | 847154bce9fe | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/lineup-auth | session/lineup-auth | 578ac110b1c2 | no | no | yes | 2026-11-19 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/live-panel-ball | session/live-panel-ball | 087c96b957f6 | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/live-wall | session/live-wall | dd7daa07ef04 | no | yes | no | 2026-11-20 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/mfl-sync | session/mfl-sync | d85693da177e | no | yes | no | 2026-11-20 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/mstub-plan | docs/mobile-v1-stub-plan | d92a9311d898 | no | yes | no | 2026-11-19 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/nativegen-off-desktop | session/nativegen-off-desktop | 3cbd4256be6f | no | yes | no | 2026-11-23 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/oline-handout | session/oline-handout | 642878130cc0 | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/phone-live | session/phone-live | c788da5ad876 | no | yes | no | 2026-11-20 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010 | session/a010-docs | 96601135d534 | no | yes | no | 2026-11-20 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010c2 | session/pulse-a010c2 | 702a8c0a99d1 | yes | no | yes | 2026-11-16 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010c3 | session/pulse-a010c3 | 702a8c0a99d1 | yes | no | yes | 2026-11-16 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010o | session/pulse-a010o | b68ccbd75751 | no | yes | no | 2026-11-16 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010s | session/pulse-a010s | 6ec32123e833 | yes | no | yes | 2026-11-16 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010t | session/pulse-a010t | 89b91d17e1c3 | yes | no | yes | 2026-11-16 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-adp | session/pulse-adp | 0517a2c537a8 | no | yes | no | 2026-11-17 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1a | session/pulse-astra1a | 5068968af24c | yes | no | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1c | session/pulse-astra1c | 5068968af24c | yes | no | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1c2 | session/pulse-astra1c2 | 5068968af24c | yes | no | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra2a | session/pulse-astra2a | 5068968af24c | yes | no | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra8b | session/pulse-astra8b | e85334123f19 | yes | no | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astraw0 | session/pulse-astraw0 | bac81da05bd7 | no | yes | no | 2026-11-25 | retained: open rehearsal PR worktree |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astraw3 | session/pulse-astraw3 | 5068968af24c | yes | no | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astraw3b | session/pulse-astraw3b | 5068968af24c | yes | no | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b11lab3 | (detached) | 578ac110b1c2 | no | no | no | 2026-11-19 | retained: Both Lab release scripts reference its ignored local Python environment; executable exists. |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b1hqdec | session/pulse-b1hqdec | 7685de0e8480 | no | no | no | 2026-11-17 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b7lab | session/pulse-b7lab | 89e5cfdbcb06 | no | yes | no | 2026-11-18 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b7lab2 | session/pulse-b7lab2 | c0d60f35b62a | no | no | no | 2026-11-18 | retained: Both Lab release scripts reference its ignored local client config; required file exists. Do not inspect contents. |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b8plan | session/pulse-b8plan | 9f5fa6b4d0fb | no | yes | no | 2026-11-18 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b9lab | session/pulse-b9lab | a10900742325 | no | no | no | 2026-11-18 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b9lab2 | (detached) | 527d6ba9b654 | no | no | no | 2026-11-19 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-bills | session/pulse-bills | 5a582c47ac4b | yes | no | yes | 2026-11-19 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-burstcpu | session/pulse-burstcpu | 8f1fa86ea13b | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-burstcpu2 | session/pulse-burstcpu2 | d0c75c37625a | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-canary | session/pulse-canary | 292aefe10817 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-catalogue | session/pulse-catalogue | 46243124f925 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-chatlink | session/pulse-chatlink | 3bc834721c36 | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopb1 | session/pulse-desktopb1 | 1bae5045de15 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopb2 | session/pulse-desktopb2 | 0e9c0f3d1616 | yes | no | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopc | session/pulse-desktopc | 4bbd0dae8dd4 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-draftexpire | session/pulse-draftexpire | 7410ed0ae8fd | no | no | yes | 2026-11-25 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-draftexpirea | session/pulse-draftexpirea | 3e4210a0973e | no | yes | no | 2026-11-25 | retained: open rehearsal PR worktree |
| /home/chris/work/TheWarRoom/.worktrees/pulse-draftpause | session/pulse-draftpause | 91ca86e1ce93 | no | yes | no | 2026-11-25 | retained: open rehearsal PR worktree |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f28 | session/pulse-f28 | 11a459e1d61c | yes | no | yes | 2026-11-15 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51a | session/pulse-f51a | 0cc632b4a247 | no | yes | no | 2026-11-16 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51b | session/pulse-f51b | 6dd2f22d470d | no | yes | no | 2026-11-16 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51mem | f51mem | 4ebcf4687184 | no | no | no | 2026-11-17 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51t | session/pulse-f51t | fd1f7ede8e49 | no | no | no | 2026-11-16 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-freshlab | session/pulse-freshlab | 5d284b490ed6 | no | yes | no | 2026-11-19 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-gate2shapes | session/pulse-gate2shapes | d325dd87ddae | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-hqnotes | session/pulse-hqnotes | 2ea17a588e67 | no | yes | no | 2026-11-16 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-instancelock | session/pulse-instancelock | 5f553d3c8bf8 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lab4 | session/pulse-lab4 | b6bb3a9aa329 | no | yes | yes | 2026-11-17 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lab5 | session/pulse-lab5 | 8b3b5c9165c2 | no | yes | no | 2026-11-18 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-loadreview | session/pulse-loadreview | f1e76f11488d | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lostpicks | session/pulse-lostpicks | 5068968af24c | yes | no | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-mflcents | session/pulse-mflcents | 2c1397ffdc06 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-migcover | session/pulse-migcover | d9dcafadc35e | yes | no | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-perf | session/pulse-perf | d670ee3cbdd6 | no | yes | no | 2026-11-18 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-phonetool | session/pulse-phonetool | e83fb7f1d623 | no | yes | no | 2026-11-19 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-phonewalk | session/pulse-phonewalk | 5a582c47ac4b | yes | no | yes | 2026-11-19 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-pincache | session/pulse-pincache | 6990e4930bcb | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-rel9 | session/pulse-rel9 | 4d8d42e755f5 | no | yes | no | 2026-11-16 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-repin | session/pulse-repin | 70cd86653619 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-s502 | session/pulse-s502 | 2f4fe99fc8af | no | yes | no | 2026-11-17 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sbplayoff | session/pulse-sbplayoff | c86c8e3dbc0f | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-schedtimer | session/pulse-schedtimer | 4d810a81d969 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-scorecarry | session/pulse-scorecarry | a2d17e1c487a | yes | no | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-scores11 | session/pulse-scores11 | 9d37d464aead | no | yes | no | 2026-11-19 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-seasonrestore | session/pulse-seasonrestore | 36143994c669 | yes | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-seasonsetup | session/pulse-seasonsetup | 33c5a001a66d | no | no | no | 2026-11-24 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sleeper | session/pulse-sleeper | c12e0d295cbe | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sleeperacc | session/pulse-sleeperacc | 9b5d5371ab55 | yes | yes | yes | 2026-11-24 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wr4 | session/pulse-wr4 | 6a646ebe6eec | no | yes | no | 2026-11-17 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wr5 | session/pulse-wr5 | 6e9e6b308ee3 | no | yes | no | 2026-11-17 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wrplan | session/pulse-wrplan | 88ebbcacf795 | yes | no | yes | 2026-11-17 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pwa-sw-shell | session/pwa-sw-shell | b94fe7d1cf9a | no | yes | yes | 2026-11-22 | retained: Uncommitted changes or unreadable status |
| /home/chris/work/TheWarRoom/.worktrees/pwa-update | session/pwa-update | fc44d4e7dbeb | no | yes | no | 2026-11-20 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/rehearsal-a | session/rehearsal-a | 98a94bcffc58 | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/rehearsal-auto | session/rehearsal-auto | 75597c1631a9 | no | yes | no | 2026-11-23 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/resume-1124 | session/resume-1124 | eec34bf84b0a | no | yes | no | 2026-11-25 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/resume-dv3 | session/resume-desktop-v3 | f7c766d336c9 | no | yes | no | 2026-11-20 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/ro-main-1124 | (detached) | 13e10830ff5e | yes | no | no | 2026-11-24 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/season-box | session/autopick-1122 | 95d2b13e3f8a | no | no | no | 2026-11-23 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/should10 | session/should10-1122 | a31bd2db0a11 | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/should9 | session/should9-1122 | 4708d5bfae9f | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/t677-t679 | session/t677-t679 | 6ebae8b0d17d | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t685 | session/t685-1122 | ba9d770b5f3b | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t688 | session/t688-replay-1122 | 2461ed59607e | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t705 | session/t705-1122 | bf893112ac20 | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t705s | session/t705-screen-1122 | ce74171900a3 | no | no | no | 2026-11-22 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/twr-read | session/twr-read-1122 | cded5ccef0c0 | no | yes | no | 2026-11-22 | removed: Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/uiux-fold1 | session/uiux-fold1-record | 71811c29ee12 | no | no | no | 2026-11-15 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/v3-error-codes | session/v3-error-codes | d2034e9041a0 | no | no | no | 2026-11-21 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/values | session/server-values | 624538c3410c | no | yes | no | 2026-11-16 | retained: Ignored local files need review before removal |
| /home/chris/work/TheWarRoom/.worktrees/wall-v2 | session/wall-v2 | 18eac730a275 | no | no | no | 2026-11-21 | retained: HEAD is neither proven merged nor proven preserved on its remote branch |
| /home/chris/work/TheWarRoom/.worktrees/worktree-audit-1125 | session/worktree-audit-1125 | 2ae2f9fd7a77 | yes | no | yes | 2026-11-25 | retained: New checkout for this Stage 1 report |

## Dirty worktrees (38)

Summaries list status and sanitized path labels only; no file contents are read. Keep dirty work until its owner reviews and backs it up.

| Path | Branch | Sanitized summary | Recommendation |
| --- | --- | --- | --- |
| /home/chris/work/TheWarRoom | main | ?? REPORT.md | KEEP — primary checkout and existing report |
| /home/chris/fleet/runs/pulse-lab5/build | (detached) | ?? deploy/lab/__pycache__/stress.cpython-313.pyc; ?? deploy/lab/__pycache__/stress_storm.cpython-313.pyc | POSSIBLE DROP — only after owner confirms scratch/generated; no removal now |
| /home/chris/work/TheWarRoom/.worktrees/codex-shape-release-1125 | session/codex-shape-release-1125 |  M scripts/lab/README.md;  M scripts/lab/shapes.sh; ?? scripts/lab/test_shapes_release.sh | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/dv3-shell | session/desktop-v3-shell | ?? docs/alpha-checklist-draft.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/handoff-1125 | session/handoff-1125 |  M docs/resume/RESUME-astra-plan.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/lab-dev | session/lab-dev | ?? frontend/tsconfig.node.tsbuildinfo | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/lineup-auth | session/lineup-auth | ?? internal/commissioner/lineup_owner_test.go | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010c2 | session/pulse-a010c2 |  M local path withheld; ?? internal/serverhttp/build/pulse-a010c2/allocs.pprof; ?? internal/serverhttp/build/pulse-a010c2/block.pprof; ?? internal/serverhttp/build/pulse-a010c2/cpu.pprof; ?? internal/serverhttp/build/pulse-a010c2/heap.pprof; ?? internal/serverhttp/build/pulse-a010c2/mutex.pprof; ?? internal/serverhttp/pulse_repro_test.go | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010c3 | session/pulse-a010c3 |  M internal/commissioner/rules.go;  M internal/leaguerules/catalogue.go;  M internal/leaguerules/catalogue_test.go; ?? local path withheld; ?? local path withheld; ?? local path withheld; ?? local path withheld; ?? local path withheld; … +2733 more — includes local scratch or build material | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010s | session/pulse-a010s | ?? deploy/lab/__pycache__/stress.cpython-313.pyc; ?? deploy/lab/__pycache__/test_stress.cpython-313.pyc | POSSIBLE DROP — only after owner confirms scratch/generated; no removal now |
| /home/chris/work/TheWarRoom/.worktrees/pulse-a010t | session/pulse-a010t | ?? deploy/lab/__pycache__/security-sweep.cpython-313.pyc; ?? deploy/lab/__pycache__/stress.cpython-313.pyc; ?? deploy/lab/__pycache__/stress_storm.cpython-313.pyc; ?? deploy/lab/__pycache__/test_security_sweep.cpython-313.pyc; ?? deploy/lab/__pycache__/test_storm.cpython-313.pyc; ?? deploy/lab/__pycache__/test_stress.cpython-313.pyc | POSSIBLE DROP — only after owner confirms scratch/generated; no removal now |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1a | session/pulse-astra1a |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra8b | session/pulse-astra8b |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astraw3b | session/pulse-astraw3b |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-bills | session/pulse-bills |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-canary | session/pulse-canary | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-catalogue | session/pulse-catalogue | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopb1 | session/pulse-desktopb1 | ?? local path withheld — includes REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopb2 | session/pulse-desktopb2 | ?? local path withheld — includes RESULT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-desktopc | session/pulse-desktopc | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-draftexpire | session/pulse-draftexpire |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f28 | session/pulse-f28 | ?? local path withheld; ?? local path withheld; ?? local path withheld; ?? local path withheld — includes log or profiling output (4) | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-instancelock | session/pulse-instancelock | ?? local path withheld; ?? cmd/twr-server/.lock — includes REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lab4 | session/pulse-lab4 | ?? deploy/lab/__pycache__/stress.cpython-313.pyc; ?? deploy/lab/__pycache__/stress_storm.cpython-313.pyc | POSSIBLE DROP — only after owner confirms scratch/generated; no removal now |
| /home/chris/work/TheWarRoom/.worktrees/pulse-loadreview | session/pulse-loadreview | ?? cmd/twr-server/.lock | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-mflcents | session/pulse-mflcents | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-migcover | session/pulse-migcover | ?? local path withheld; ?? local path withheld — includes REPORT.md, RESULT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-phonewalk | session/pulse-phonewalk |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-pincache | session/pulse-pincache | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-repin | session/pulse-repin | ?? local path withheld; ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md, RESULT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sbplayoff | session/pulse-sbplayoff | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-schedtimer | session/pulse-schedtimer | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-scorecarry | session/pulse-scorecarry |  M internal/store/history/score_carry_test.go;  M internal/store/history/season_score_cache.go | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sleeper | session/pulse-sleeper | ?? local path withheld; ?? local path withheld — includes HANDOFF.md, REPORT.md | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-sleeperacc | session/pulse-sleeperacc | ?? cmd/twr-server/.lock | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wrplan | session/pulse-wrplan |  M local path withheld — includes HANDOFF.md | KEEP — tracked or in-flight changes; back up and ask owner |
| /home/chris/work/TheWarRoom/.worktrees/pwa-sw-shell | session/pwa-sw-shell | ?? frontend/tsconfig.node.tsbuildinfo | KEEP — untracked files may be work; ask owner before any removal |
| /home/chris/work/TheWarRoom/.worktrees/worktree-audit-1125 | session/worktree-audit-1125 | ?? docs/reset/WORKTREES-1125.md | KEEP — active report checkout |

## Removed (41)

| Path | Reason |
| --- | --- |
| /home/chris/work/TheWarRoom/.worktrees/astra-brief-1125 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/build-plan-wt-list | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/lab-v52 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/lab-v53 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/line-budget | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/live-panel-ball | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/live-wall | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/mfl-sync | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/mstub-plan | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/nativegen-off-desktop | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-adp | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1c | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra1c2 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astra2a | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-astraw3 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-b8plan | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-burstcpu | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-burstcpu2 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-chatlink | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51a | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-f51b | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-freshlab | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-gate2shapes | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lab5 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-lostpicks | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-phonetool | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-s502 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-scores11 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-seasonrestore | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wr4 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pulse-wr5 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/pwa-update | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/rehearsal-auto | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/resume-1124 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/resume-dv3 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/ro-main-1124 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t677-t679 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t685 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t688 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/t705 | Clean; HEAD verified merged or pushed; no active use found |
| /home/chris/work/TheWarRoom/.worktrees/twr-read | Clean; HEAD verified merged or pushed; no active use found |

## Disk summary

Before:
```text
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme0n1p1  458G  354G   81G  82% /home
```
Used: 379893760000 bytes; available: 86194982912 bytes.

After:
```text
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme0n1p1  458G  351G   84G  81% /home
```
Used: 376843661312 bytes; available: 89245081600 bytes.
Freed **3.05 GB** (2.84 GiB) based on the change in used blocks.

Audit evidence is in `~/fleet/runs/twr-worktrees-1125/`: `worktrees.json`, `live-paths.tsv`, `removals.json`, `metrics.json`, and `post-snapshot-worktrees.json`.
