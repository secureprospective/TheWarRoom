# R2 caller-team handoff

## Baton
ClaudeBox/reviewer — 2026-10-22. Bee completed the R2 caller-team brief on `session/r2-caller-team`. Not pushed.

## Where it stands
- Saved MFL key → api-host `myleagues` → validated league/franchise → `core.State.MyFranchise`. Startup and successful saves resolve it; failed lookup/deletion clear it. The discovered-host keyed path stays restricted.
- `RequireOwnFranchise` checks the actor on all 11 brief rows before effects. Contract/release/trade plan IDs and hand-off/withdrawal use the stored franchise too. Native roles do not bypass this check.
- Trade counterparty projection uses held MFL truth, never the counterparty's private what-if session.
- Picker views any team. Existing key status supplies identity and Go refusal text to team-action buttons. Bound methods remain 133; bindings regenerated.
- Commits: identity `02cf9d26`; enforcement `718cba02`; picker/status is the final commit containing this handoff.
- Requested gates, `make lint`, `make test`, and final `make verify` pass. Frontend: 1,394 tests; entry bundle 306,306 / 321,818 bytes; CSP passes. No vulnerabilities found.
- Regression tests: `TestMyFranchise`, `TestLoginFranchiseLookupFailsClosed`, `TestOwnerActionsRefuseWithoutChangingState`, `TestTradeCounterpartyProjectionIgnoresPrivateWhatIf`, and the picker/action case in `frontend/src/app/shell/mflKey.test.ts`. Removing the shared check failed the refusal regression; restored before final gates.

## Next move
Review the three commits, then live-test a throwaway profile: connect a login, view another team, verify action refusal and unchanged plans; return to the login's team and verify planning works; delete the key and verify refusal. Nothing may be submitted to MFL.

## Blocked on
Independent review and live GUI/MFL verification are not done. No implementation blocker.

## Tried and rejected
- Parallel full test suites exceeded the targeted command's default timeout. Sequential rerun passed; the project's full gate uses 60 minutes.
- An in-tree toolchain TMPDIR let a source-scanning test encounter disappearing linker files. Normal tooling temporary storage passed; no test/gate was weakened.
- System pnpm 9 rejects `pnpm test --run`; the pinned `corepack pnpm` 10.34.3 ran that gate successfully.
