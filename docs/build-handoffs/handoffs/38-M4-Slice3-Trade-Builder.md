HANDOFF — Session 38: M4 slice-3 TRADE multi-leg builder — BUILT, gate-deferred (RC session)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-12 · Branch: session/m4-slice3-trade (off main, NOT merged) · HEAD 2bf7b3e
UPDATED 2026-07-12 (second RC session): two more slice-3 builds landed on this branch —
franchise-name league-fetcher (a1551f7) + D6 commissioner surfaces (2bf7b3e). See == FOLLOW-ON == below.

== WHERE WE ARE ==
- M4 slice-1 (ROSTER_STATUS+WAIVER+SIGN) and slice-2 (TAG+EXTENSION+BUYOUT+RESTRUCTURE) are
  MERGED to main (squashes 1f0abfe, 64bf9fc). This session opened slice-3 — the TRADE builder.
- Session ran on RC (CT105), NOT the Beelink — so the live functional gate + GLM SSH review are
  DEFERRED to a Beelink session. Everything runnable on RC is done and green.

== WHAT LANDED (commit d759f7b on session/m4-slice3-trade) ==
The multi-franchise/multi-player TRADE surface as its OWN "Trade" tab (design doc §"TRADE gets its
own builder surface" — it is the only multi-franchise/multi-player op, so it is deliberately NOT
folded into the subject-centric single-player workspace).

KEY FACT: the TRADE backend was ALREADY COMPLETE before this session — transactions.Trade (request +
validate: dedup, empty-leg, maxTradeLegs=256) + acquisitions.Trade (atomic per-leg MovePlayer) +
buildRequest KindTrade + the generic Coordinator.Preview dry-run path all existed and are tested.
So slice-3 TRADE is a PURE FRONTEND build — zero Go change. (Verified: go test -race green, untouched.)

Files:
- frontend/src/components/transactions/TradeBuilder.tsx (NEW): rail = browse ANY franchise roster →
  "Add →" a player to the cart → each cart leg gets a DESTINATION-franchise <select> (excludes the
  player's own franchise) → "Review trade…" → D5 PreviewTransaction → ConfirmModal → D4 re-send-
  intent ExecuteTransaction. One atomic N-leg swap. Dedup (a staged player shows "Staged", can't be
  re-added; the engine also rejects a doubled player). Every list `?? []` (D9). No mflID typed (D2):
  names from the server, id hidden on each leg. stageable = ≥1 leg AND every leg has a destination.
- ConfirmModal.tsx: Pending.kind union extended with 'TRADE'; footer shows "Confirm trade". The trade
  path REUSES the modal wholesale, so it inherits the slice-1/2 GLM hardening: L1 (ExecuteTransaction
  ok:false surfaced in the modal, not a false success), L2 (non-dismissable mid-commit), L3 (stageGen
  ref token discards a preview that resolves after cancel/restage).
- App.tsx: new "Trade" tab between "Transactions" and "League Controls". Single-move workspace's stale
  "trades land in a later slice" note now points to the Trade tab.

== GATE STATUS ==
PASSED on RC:  tsc+vite clean · make lint 0 · go test -race ./... green · pre-commit hooks (golangci-
  lint + gitleaks) green on commit.
GLM 5.2 BLIND REVIEW — DONE (2026-07-12, driven from RC over SSH; header verified `> build · glm-5.2`).
  Verdict: 1 MAJOR + 2 MINOR/NOTE, no path where a wrong trade commits. APPLIED (commit 7658d4f):
  - M1 (MAJOR): pickFranchise awaited GetRoster WITHOUT clearing roster first → mid-fetch the rail
    highlighted the new franchise while the OLD roster showed live Add buttons, so addLeg stamped the
    new franchiseID onto a player on the old one (lying "from" label + self-move rejections; engine
    still blocked a bad commit). Fix: setRoster(null) first + source leg origin from roster.franchiseID.
  - n2 (NOTE): dedup moved inside the setLegs functional updater (same-tick double-click can't double-add).
  TRIAGED OUT (documented): m1 (post-commit GetRoster throw) + n1 (unwrapped mount getters) — both
  conditional on the IPC getters THROWING, but GetRoster/GetFranchises/GetLegalOps return an ok/detail
  struct (don't error the binding), and both patterns are identical to the shipped sibling workspace.
  GLM verified-correct (kept): stageGen L3 token, D4 re-send-intent, L1/L2/L3 via shared ConfirmModal,
  atomic-cart-after-fail, D9 null guards.
DEFERRED to Beelink (NOT done — the ONE remaining gate, blocks merge):
  LIVE FUNCTIONAL GATE — build a real 2-team (then 3-team) swap, confirm, watch BOTH rosters and BOTH
  caps move on the post-commit refresh; verify a same-franchise/no-destination leg can't be staged.
  Reset the Beelink clone to clean main before AND after ([[reference_beelink_functional_gate]]).
  Beelink clone = /home/chris/opencode/TheWarRoom.
Only after the live gate passes: squash-merge to main, then session-close's merge steps.

== BUILD ENV (RC or Beelink) ==
go/golangci-lint/wails are NOT on the default PATH here: export PATH=/usr/local/go/bin:/root/go/bin:$PATH
lint needs GOMEMLIMIT=3000MiB GOGC=40 ; tests GOMEMLIMIT=1500MiB GOGC=20.
NOTE: the pre-commit framework installs golangci-lint + gitleaks envs on FIRST commit (several
minutes) — run the commit with Go on PATH and give it time; do NOT --no-verify.

== REMAINING slice-3 items (each a separate, bigger multi-layer change — NOT started) ==
- Franchise-name league-fetcher: the MFL `league` export carries a franchises block with id+name that
  internal/ingestion/league/types.go does NOT decode (leagueEnvelope has no franchises). Extend that
  fetcher → thread names through ingestion → normalize → state so GetFranchises returns real names;
  the rail + trade destination dropdowns already fall back to id and will pick up names for free.
- D6 commissioner surfaces off the dev panel (ADVANCE_PHASE / ROLLOVER_SEASON / SET_SIGNING_WINDOW to
  League Controls; RETIREMENT / DEATH / CAP_RELIEF segregated under a red Commissioner divider).
- Pre-commit dollar breakdown: needs the sealed apply() (14 reqs) to RETURN a breakdown so WAIVER/
  TAG/BUYOUT/trade cap deltas show in the quote instead of only on the post-commit refresh.

== FOLLOW-ON (second RC session, 2026-07-12) — two more slice-3 builds on this branch ==

(2) FRANCHISE-NAME LEAGUE-FETCHER (commit a1551f7). Backend + a thin frontend polish.
- internal/ingestion/league/types.go + league.go: decode the MFL `league` export's `franchises`
  block (id + name) into a new RawConfig.Franchises (empty-id rows dropped; MFLList collapse-tolerant
  like rosters). New Franchise{ID,Name} type.
- internal/store/rulebook/{helpers,rulebook}.go: the directory RIDES the rulebook's versioned JSON
  payload — ZERO schema change (it's just another RawConfig field). New FranchiseNames() id->name map
  (blank names omitted); cloneConfig deep-copies the slice. NOTE: FranchiseNames landed in helpers.go
  (not rulebook.go) to stay under the 400-line file cap — rulebook.go was at 405 with it inline.
- m4_app.go GetFranchises: stamps M4Franchise.Name from a.rulebook.FranchiseNames(); id fallback.
- frontend TradeBuilder.tsx: RosterPicker now takes a `label` so the browsed-roster header shows the
  name too (rail + destination dropdowns already did name || id).
- New test TestAssemble_DecodesFranchiseDirectory (collapse + empty-id drop).
- EDGE (documented, not a bug): an EXISTING db holds a config version stored BEFORE franchises were
  captured, so names stay empty (id fallback) until a rulebook Reload+Promote re-fetches. Fresh dbs
  get names immediately. Matches the immutable-versioned-snapshot model.
- Gate PASSED on RC: make lint 0 / go build+vet / go test -race ./... EXIT=0 / tsc+vite clean /
  pre-commit hooks green. M4Franchise.Name already existed in the Wails bindings → no regen.

(3) D6 COMMISSIONER SURFACES (commit 2bf7b3e). PURE FRONTEND — backends all already existed.
- frontend LeagueControls.tsx rewritten from a stub into the real surface: a season CALENDAR group
  (ADVANCE_PHASE / ROLLOVER_SEASON §14 / SET_SIGNING_WINDOW §6) and, under a RED divider, the
  destructive COMMISSIONER group (RETIREMENT §13 / DEATH §13 / CAP_RELIEF §13 — the last uses the new
  franchise-name picker from build 2). Every op runs the SAME D5 preview → ConfirmModal → D4
  re-send-intent path as the workspace, reusing the shared modal so it inherits L1 (ok:false surfaced)
  / L2 (non-dismissable mid-commit) / L3 (stageGen token discards stale previews). All 6 kinds route
  through the existing generic buildRequest + Coordinator.Preview/Execute — no new IPC.
- ConfirmModal.tsx: Pending.kind union extended with the 6 commissioner kinds; the footer label chain
  refactored into a confirmLabel(kind) switch (per-kind verbs).
- TransactionWorkspace.tsx: stale "commissioner surfaces land in a later slice" note updated to point
  at the League Controls tab.
- Gate PASSED on RC: tsc --noEmit clean / vite build clean / gitleaks green. No Go change (no
  race-suite delta, no wails regen). No frontend eslint/prettier in the repo — tsc+vite IS the gate.

== NEXT ==
1. Run the deferred Beelink LIVE FUNCTIONAL GATE — it now covers ALL THREE builds in one pass:
   - Trade: build a real 2-team then 3-team swap, confirm, watch BOTH rosters + BOTH caps move.
   - Names: rail + trade destination dropdowns + roster header show real team names (id fallback on a
     stale db that predates the directory).
   - Commissioner: advance a phase → the workspace's legal ops change; grant cap relief → that
     franchise's cap drops; a rejected op (e.g. rollover when not in PLAYOFFS) shows the engine reason.
   Reset the Beelink clone to clean main BEFORE and AFTER ([[reference_beelink_functional_gate]]).
2. GLM 5.2 BLIND review of builds 2 + 3 is DONE (2026-07-12, commit 767f6ea, header verified
   `build · glm-5.2`, combined blind diff over SSH). 4 leads triaged vs source; only L2 [MED] applied
   — LeagueControls.confirm() now refreshes the franchise directory after a successful commit so the
   cap-relief picker's counts can't show stale post-retirement/death state in-tab (frontend-only,
   engine authoritative). Triaged OUT: L1 [HIGH] kind strings + all json field tags match Go exactly
   and buildRequest already routes the 6 kinds (D2/D4 hold — GLM was blind); L3 MFLList handles the
   bare-object collapse and is unit-tested in decode_test.go; L4 MFL franchise ids are unique. tsc+vite
   clean; frontend consistency fix rides the same deferred gate, no re-gate. ALL THREE builds reviewed.
3. Only after the gate → squash-merge to main + session-close merge steps.
4. LAST remaining slice-3 item: pre-commit dollar breakdown — the heaviest, least RC-friendly (14
   apply() signatures + the committed-vs-uncommitted TxWriter read wall). Not pure-frontend.
