# War Room read slice

Locked visual authority: `docs/ui/mocks/warroom-mock-v6.html`. `warroom.css` scopes its CSS to
`wr-v6`, without changing the global shell or HQ styling. `TargetApp` lazy-loads this surface.

`WarRoom` owns the selected tab, partner, private in-memory proposal and inspector. Existing
feature adapters own transport and checked parsing. Settings and contract/cap reads supply rule
values; no presets or mock player data. Native reads use the signed-in league/franchise, never
local MFL private lists. Scope-keyed reads fence late results; missing/failed reads render
“Not available yet”, with failure details in titles. Preview is not execution. The local embedded
Draft room mounts the existing native `draftroom/DraftRoom`, including its commissioner tray and
owner pick; it does not introduce a parallel write path. The selected league goes to
`leagueState.choose`; existing permissions and backend validation govern all changes. MFL and
hosted draft readings remain unchanged. No MFL submission is introduced.

Evidence and remaining gaps: `docs/build-handoffs/WR-WIRE-report.md`.
