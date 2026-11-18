# Handoff

- Baton: Bee, 2026-11-18; PR #351 open against `bundle/b5`, head `74b8034b5` plus final handoff update.
- State: Hosted Server ledger uses `createHostedLedgerQuery` / `createReadQuery` and `query.use()`, with parsed league identity validation. Backup-now remains unavailable; restore/feed/upgrade remain operator-only.
- Slop fixed: removed raw ledger port use; fixed long lines in the touched Control Room test fixture.
- Verified: full frontend `pnpm test` (920/920); touched frontend tests (20 passed); `pnpm exec tsc --noEmit`; `make lint`; `make bloat`; push hook verify passed. PR #351 checks and desktop package/install checks are green.
- Prior full-run failures named in PR: `Readable app source (§M2/§M3) > forbids lines longer than 110 characters` (fixture fixed) and `opens from the HQ rail and Back returns to HQ` in `hostedPulse.test.tsx` (flaky); both passed in isolated rerun (40 tests).
- Tidy: `~/fleet/bin/tidy-check.sh` flags active other lane `b2docket` (pid 3627613) and its required scratch KEEP file. Left it untouched; cannot create outside permitted paths.
- Blocked: hosted `backup.now` command path missing; no new command/read.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
