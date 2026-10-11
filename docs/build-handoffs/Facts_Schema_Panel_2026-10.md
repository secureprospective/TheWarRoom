# Facts-schema panel, 2026-10 (triage record)

Copied verbatim from the ring 2 run folder (`~/fleet/runs/warroom-ring2-2026-10-10/panel-facts-run4-triage.md`), where it was the only copy. It is a recommendation to Christopher, not a decision. Nothing in it is built: it is the plan for the stand-alone app's data pipeline (s0–s5). The UI spec summarises it in `docs/ui/Target_UI_Spec_2026-10.md` §12.

## Facts-schema panel, run 4 (2026-10-11 02:56–03:02 CDT) — Claude's triage

Run: `~/fleet/runs/panel-panel-ring2-facts-schema-20261011T075559Z-2950223/`. Seats: visionary Kimi K3
(6/10 quotes ok), mechanic GLM 5.3 Flash standing in for rate-limited GLM 5.3 (13/13 ok), devil
Nemotron Ultra (0/11 by the checker: every failure is formatting, `** ` prefix and code fences, not
fabrication; spot-checked). Sol's engineer seat still to run after p2h2 (fourth lab).

- **D1 split.** Kimi (a) typed facts.db; GLM-flash (c) compile raw MFL bodies into the `internal/lab`
  column structs, cached by input hash; Nemotron against (c): the Lab's checks.py proofs "need indexed
  joins". **Triage FS1 stands: (c).** Nemotron's own evidence refutes its premise: checks.py:139 runs
  the roster-ledger proof over in-memory dicts (`ros`, `snap`), not SQL, so the proofs port as compile
  assertions. Decisive asymmetry: compiled facts are a pure function of archived bodies, so a facts.db
  can be materialized LATER at zero migration cost (rebuild from raw) if a surface ever needs ad-hoc
  relational queries; a stored facts.db now is a second copy that can drift.
- **New prerequisite (GLM-flash F2, verified):** the 2013–2026 archive lives in
  `TheWarRoom/league-archive/raw` (1,720 files, 110 MB); history.db's `raw_archive` holds only bodies
  the app fetched. **s0 = ingest the archive through `Record` (archive.go:30) as a registered source**,
  so raw_archive + fetch_log are the single sha256-provenanced input. If blob size is a problem, s1
  reads the frozen dir directly; D1 survives either input.
- **D2: all three say both proofs.** Adopt FS2: row-level diff of compiled columns against the frozen
  lab.json (integer-exact money, UTC-second times, floats 1e-9) AND the existing goldens on the
  compiled input; the frozen pair stays a permanent regression fixture when the archive moves on.
- **D3:** Kimi `origin` column + `supersedes`; GLM-flash provenance on imported rows, app-authored
  events in their own append-only stores (envelopes, bid_facts, ledger); Nemotron a `source` column on
  ledger tables. **Adopt:** imported compiled rows carry provenance (source, as-of, body sha); app-
  authored events stay in their own append-only stores (PW3 holds); when the app becomes the authority
  and overrides an imported fact, the authored row names what it supersedes; the compiler merges.
  FS3 stands (no bids in observations).

### Engineer seat (Sol, gpt-6.1-sol max), 2026-10-11 04:08–04:2x CDT — 14/14 quotes OK
Run: `~/fleet/runs/panel-brief-20261011T090833Z-3776547/engineer.out` (10.4 KB). Verdict (c), so all
four seats now converge on FS1 except Kimi's (a). Adopted additions:
- **FS1 refined:** no persistent fact tables; cache key = selected-input manifest + compiler/schema
  version + rules/config hashes (not body hashes alone); validate before publishing; a failed compile
  keeps the previous verified snapshot, marked stale. Disproof: measured compile latency/RSS over
  budget (unmeasured; s1 must measure).
- **New FS4, input manifest (verified: archive.go:87 `ArchivedBodies` has no cutoff, newest wins):**
  s1 first pins a manifest (endpoint/file, historical league id, season/week, owner scope, capture
  time, body sha, row locator) and resolves bodies through it, never the newest-body helper. Missing
  frozen bytes block a same-cutoff parity claim; today's refetch does not substitute.
- **FS2 refined (gate table):** exact for ids/times/counts/enums/nulls/cents; imported decimal scores
  exact normalized; derived floats 1e-9 rel+abs; the <0.01 price anchor stays separate; the 0.011
  MFL score reconciliation keeps pinned exception identities, not counts. Engine goldens must run on
  the newly compiled Data injected (suites calling Load() prove nothing about s1–s5). Never use Go
  output as its own oracle; new goldens come from pinned reference JS on independently checked input.
- **FS3/D3 refined:** one append-only `league_event` table (id, league, stream, revision, effective/
  recorded time, actor, scope, kind, versioned payload in cents, input hash, correlation, supersedes;
  UPDATE/DELETE forbidden by trigger; expected-revision append). Envelopes stay plans; bid_facts and
  plan audit are not league events. Authority policy per league/domain/date: MFL wins until cutover,
  then local events; never newest-timestamp-wins. Not built in ring 2: this is the stand-alone app's
  spine, recorded for spec rev 2 (p2l) and the commissioner's office mapping.
