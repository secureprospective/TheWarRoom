# Handoff

- Baton: Bee, 2026-11-18.
- State: b3home is blocked before implementation; working tree has no feature changes. Thin `NativeAccount` has one configured league's admission; `NativeView.seats` are that league's seats. `commish.leagues.read` is a local/native read returning the server-wide league list, not an account-filtered membership list, and the hosted workflow does not expose it. `Desktop.Workflow` rejects reads for a league other than this window's configured league.
- Next: Christopher/ClaudeBox must resolve the privacy-safe membership enumeration and multi-league read path before building the app-level Home. Do not expose the global league list or infer membership from refused reads.
- Blocked on: authorized server-side account membership listing and cross-league read scope; needed to prove membership before showing league names/cards and fan-out reads.
- Tried/rejected: no local-mode `TargetApp`/`AppLeagueMount` port; no unfiltered `commish.leagues.read` plus per-league probing, because that exposes nonmember league names and thin reads are bound to one league.
