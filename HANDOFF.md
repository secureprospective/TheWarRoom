# Handoff

- Baton: Bee, 2026-11-18; PR #351 open against `bundle/b5`.
- State: Server ledger now uses `createHostedLedgerQuery` / `createReadQuery` and `query.use()`, with parsed league identity validation. Backup-now remains unavailable; restore/feed/upgrade remain operator-only.
- Slop fixed: removed raw ledger port use; fixed long lines in the touched hosted Control Room test fixture.
- Verified: `HostedControlRoom.test.tsx` + `controls.test.ts` (20 passed); full frontend `pnpm test` (920/920); `pnpm exec tsc --noEmit`; `make lint`; `make bloat`.
- Flake note for PR: `opens from the HQ rail and Back returns to HQ` in `hostedPulse.test.tsx` failed on the prior full run, then passed in the 40-test isolated rerun. The other prior failure, `Readable app source (§M2/§M3) > forbids lines longer than 110 characters`, was due to this lane's fixture and is fixed; it passed on rerun.
- Tidy: `~/fleet/bin/tidy-check.sh` flagged another active lane (`b2docket`, pid 3627613); left it untouched.
- Next: commit and push these updates to PR #351, then verify its CI.
- Blocked: hosted `backup.now` command path missing; no new command/read.
- Open decisions untouched: snooze scope; deputy job visibility; written-request visibility; division naming.
