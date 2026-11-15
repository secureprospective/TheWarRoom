# Move my MFL league

Control Room → League → **Move my MFL league** extends the existing dark, compact
commissioner form language. Operate mode: read a league, inspect its counts and
problems, fill missing rules, then confirm once with a typed reason. No new visual
system, assets or dependencies.

## Flow and ownership

- `control.API.TargetMFLMigration(command, body)` → composition startup gate →
  `catalog.RunMigration` → the existing migration handlers and service. The binding
  is desktop-only; no hosted, CLI or browser command gains MFL credentials.
- `RawMigrationStep.Validate` checks the input. An optional key is write-only,
  redacts itself, and is saved through `mflkey.Store.ForLeague`, retaining the
  existing keyring provider and season while scoping the account to the MFL ID.
- `Service.SourceWithBase` opens a separate read-only MFL client. `Source.Base`
  fills missing values only. The initial suggestion is Mirror for 16 teams,
  Legacy for 32, otherwise 32-Team Default. Changing the suggestion rereads MFL.
- `waitingSettings` supplies both plan notes and structured catalogue inputs.
  `RuleInput` and `ruleDraft` are the same editors used for later rule changes.
- Import rereads the source, rejects answers to non-waiting fields, and batches
  Create + starting snapshot + normal RulesSet answers. RulesSet is batchable only
  with a creation; ordinary rule checks, admission and transaction rollback remain.
- A successful receipt calls `leagueState.choose(importedLeague)`. A reading
  failure is shown without offering a second import. Another league can be moved
  through the reset action. MFL is never written; cutover/invites are not automatic.

## Evidence and limits

Fixture tests cover selected Base threading through the real MFL source adapter,
key isolation, preserving MFL's cap, ordinary rule answers, and rollback after an
answer write fails. Frontend state tests cover prepare → preset → answer → import,
selection, blocked problems, retry and credential clearing. No live MFL calls.

A synthetic mobile Firefox capture was inspected; it shows the expanded form.
The desktop capture remained collapsed, so expanded desktop layout and native
Wails click-through are unverified. Detector returned no findings. No independent
visual review was run; this is not whole-surface visual approval.
