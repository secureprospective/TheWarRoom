# Handoff 49 — Alpha Versioning: Tier 1 (version stamping)

**From:** the 2026-07-21 Versioning & Releases planning session (planning-only, no code).
**For:** the next build session.
**Type:** SMALL BUILD (~half a session). The planning is DONE — do not re-plan it.

## READ FIRST (in this order)
1. **`docs/roadmap/Alpha_Versioning_and_Releases_DECIDED.md`** — the decided plan.
   §0 (why the scope collapsed), §1 (locked decisions), §2 (the tiers), §4 (verified facts
   — DO NOT re-derive), §5 (why Wails), §6 (Beta docket). **Start here.**
2. `docs/roadmap/Versioning_and_Releases_Planning.md` — the original D-V1…D-V7 framing
   scaffold. Historical context only; the answers now live in the DECIDED doc.
3. Handoff `40` — superseded by this one. Its "hard gate before Alpha" framing is
   **downgraded** (see below).

## ⚠ THE STATE CHANGE THAT RESHAPED THIS PHASE
**The league was polled 2026-07-21 — nobody else wants to run the binary. Alpha's audience
is Christopher alone, on Linux.** Signing, cross-platform, hosting, recovery UI, first-run
polish, distro matrix, changelog → **all deferred to Beta.**

**The UI build track is effectively UNBLOCKED.** `CLAUDE.md` + handoff 40 said B-2…B-5 →
ALPHA hard-depends on this phase. That was justified by "before any binary leaves the
machine." No binary is leaving. **Tier 1 is worth doing first anyway** — but because it
pays for itself in B-2's gate loop, not because it blocks it.

## THE TASK — Tier 1 only
Branch `session/alpha-version-stamping` off main.

1. Build script: `git describe --tags --always --dirty`.
2. `-ldflags -X main.version=… -X main.commit=… -X main.buildDate=…`.
   Source defaults **`"dev"` / `""` / `""`** — a dev build must be visibly distinct.
3. Wails-bound `AppInfo()` struct → Zustand → a UI surface. **No `version.json`, no
   duplicated TS constant** — the binary is the authority, one hop.
4. Sync `wails.json productVersion` from the same `git describe` (packaging metadata only;
   inert on Linux, matters for Windows/macOS at Beta).
5. `make release` target that **refuses to tag from a dirty tree**.
6. **Tag current main `v0.5.0`** — NOT the scaffold's stale `v0.4.0` placeholder.

**Explicitly NOT in this handoff:** Tier 2 (schema_migrations / backup / downgrade check)
and Tier 3 (instance lock / logging / dev-DB guard). Tier 2 goes before the next
**schema-touching** work — B-2 is a frontend restyle and does not qualify.

## VERIFY / CLOSE GATE
- `make lint` 0 · `go test -race ./...` green · frontend tsc+vite clean · pre-commit green.
- **Live Beelink gate:** `wails build -tags webkit2_41`; the About surface reports the tag
  + short SHA; a deliberately dirty build visibly reports `-dirty`. Reset the Beelink clone
  to clean main before AND after (`reference_beelink_functional_gate`).
- GLM 5.2 blind review if up (leads-not-findings, triage vs source); waive with a note if down.

## OPEN / CARRIED
- OQ-013 (created→official id ramp), OQ-014 (Money type), OQ-016 (RookieConsensus CSV
  loader), harness 3G/3H pending, AgeTrajectory field vestigial.
- K2 NFLProduction seat reserved neutral in both `applyScouting` branches (small swap-in).
- ⚠️ ROTATE the free CFBD key at beta. ⚠️ ROTATE the z.ai `GLM_API_KEY`.
- **Panel this at Phase 2:** exposure path — Cloudflare Tunnel + Zero Trust (the ORIGINAL
  documented plan, `Backend_Architecture.md`) vs hosted web vs local-server+browser.

## ALTERNATIVES IF CHRISTOPHER PICKS SOMETHING ELSE
B-2 module migration (UI build track, now unblocked) · M2/M4 refactors (handoffs 43/44) ·
league-calendar backend · M2 slice-2 · K2 seat.
