# Legacy NFL — Scouting Schema (B2b-Schema)
Version: 1.0 — June 2026
Status: **LOCKED** (AD-16 human-review gate passed 2026-06-19). Resolves the unified scouting field set for all 10 positions. Implemented at `internal/scouting`.

## What this is

The single unified data shape every scouting fetcher (B2b-Fetch-*) populates and the engine's Layer 4 (Film / RAS / Breakout) consumes. One `Profile` type covers all ten positions; position-conditional inputs are present only where a position uses them.

This is a **Layer-1 data shape** — a leaf, like `internal/domain`. It does not fetch (fetchers populate it) and it does not score (the engine consumes it). It imports only `internal/playerid`.

## The zero-leak invariant (hard constraint)

**No field may hold or reference fantasy points, projected volume, MFL scoring config, or format-dependent volume stats.** Every field is a film, athletic, or college signal. This is the entire reason scouting is its own schema and not roster fields — Layer 2 (fantasy scoring) and Layer 4 (scouting) must never see each other's signals. Verified field-by-field at the AD-16 walk.

## Structure (`internal/scouting/Profile`)

Keyed by `MFLID`; the engine joins it to a `domain.PlayerRecord` (which already carries position), so the profile deliberately does **not** duplicate position — that would break the leaf boundary.

- **Universal core** (flat, present at every scored position): `PFFGrade`, `DraftNetwork`, `MaddenFilm`, `NFLProduction`, `RAS`, `BreakoutAge`, `SchoolTier`, `CollegeProductionShare`, `AgeTrajectory`.
- **Position-conditional groups** (pointers; nil when the position does not use them): `OffenseFilm` (QB/RB/WR/TE), `IDPFilm` (DT/DE/LB/CB/S), `Coverage` (**CB/S only**), `TouchShare` (RB only).
- **Reserved**: `SafetyRole` (S only; unset in v1.0).

The pointer groups encode the position boundaries **structurally** — most importantly the NGS coverage boundary: `Coverage` is non-nil at CB and S exclusively.

**Scale convention:** every numeric input is the source's normalized sub-signal in `[0,1]`. For a position that uses a core field, `0` is a legitimate floor grade (the fetcher always populates it). `CollegeProductionShare` stays one slot — its position-specific *definition* (target % for WR, sack+TFL % for DE, etc.) is computed upstream in the fetcher, not encoded in the schema.

## Per-position coverage (the AD-16 walk)

| Position | Universal core | OffenseFilm | IDPFilm | Coverage (NGS) | TouchShare | SafetyRole | SL-019 |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| QB | ✓ | ✓ | — | — | — | — | excluded |
| RB | ✓ | ✓ | — | — | ✓ | — | excluded |
| WR | ✓ | ✓ | — | — | — | — | excluded v1.0 (SL-022) |
| TE | ✓ | ✓ | — | — | — | — | applied |
| DE | ✓ | — | ✓ | — | — | — | applied |
| DT | ✓ | — | ✓ | — | — | — | excluded (Cushion Guard) |
| LB | ✓ | — | ✓ | — | — | — | excluded |
| CB | ✓ | — | ✓ | **✓** | — | — | applied |
| S | ✓ | — | ✓ | **✓** | — | reserved | applied |
| K | `MaddenFilm` (majority) + `NFLProduction`¹ | — | — | — | — | — | excluded |

¹ **K is sparse but Madden-driven** (DECISION-011, Christopher 2026-06-19): K's Layer-4 valuation is driven by `MaddenFilm` as the **majority weight** (Madden kick power/accuracy — the one genuinely scoutable kicker signal), with `NFLProduction` secondary. RAS still excluded (SL-020); no film/breakout sub-signals scouted. Keeping K in the unified type is cleaner than a carve-out. **This reverses the K rubric's prior structural-exclusion model (flat 1.000 Layer 4) and AD-10 — see the DECISION-011 note below; the K rubric, AD-10, and SL-020's K application must be updated to match (out of scope for this schema session).**

**SL-019 is engine logic, not a schema field** — the schema holds the inputs SL-019 reads (RAS + age), which the universal core already carries. DT's Cushion-Guard exclusion is likewise engine logic.

## SL-OQ-035 / SL-OQ-036 reservation decision (AD-16)

**Decision:** reserve a single `SafetyRole` enum (`box`/`deep`/`hybrid`/unset), left **unset for v1.0**. `CollegeProductionShare` stays monolithic.

**Rationale:** SL-OQ-035 (split S into S_BOX/S_DEEP) and SL-OQ-036 (role-conditional weighting in a monolithic S rubric) resolve together, post-live-data; the standing lean is monolithic-for-v1.0. The `SafetyRole` slot is forward-compatible with *either* outcome without inventing production-share structure that a discarded option would orphan. This is "reserve deliberately, do not invent fields" (handoff constraint). S is scored monolithically until the empirical decision lands.

## Verification

- `make lint` → 0 issues (ifaceguard, filelen, golangci-lint, revive).
- `go test -race ./...` → green (no test file; pure type leaf, mirrors `domain`).
- Leaf boundary: imports only `internal/playerid`.

## DECISION-011 — K Layer-4 Madden-majority valuation (2026-06-19)

**Directive (Christopher):** Madden ratings are the **majority weight** in K's Layer-4 valuation.

**What it reverses (locked items that must be updated to match):**
- **K rubric "structural exclusion" model** — currently Layer 4 hardcoded to flat 1.000, Madden explicitly "archival only — not Layer 4 valuation" (Christopher's prior Option 1 call). Madden now becomes K's primary Layer-4 driver.
- **AD-10** — "K `combine` yields 1.000, not special-cased." No longer holds once K's Film/Madden component is active.
- **SL-020 at K** — currently forces all three K Layer-4 components to 1.000. The Madden-driven component is no longer forced.

**Schema impact (this session): none structural.** `MaddenFilm` is already a universal core field; K simply populates it as its majority signal. The schema supported this without change — only K's *characterization* was corrected.

**Sub-signal split PINNED (Christopher 2026-06-19): Madden 0.60 / NFLProduction 0.40** within K's now-active Film component.

**Still deferred to B5b-K (calibration, not data-shape):** the Film component's **cap and curve** (`film_cap` / inflection / steepness — how strongly Layer 4 may move K's valuation), and the formal reconciliation of the K rubric mechanics + AD-10 + SL-020's K application. The pinned split sets the sub-signal weighting; the component strength is a live-data calibration. The K rubric on disk carries a supersession pointer to this decision; the full mechanics rewrite executes at B5b-K.

## Carried forward

- Fetchers (B2b-Fetch-Offense / -Defense / -Kicker) populate this shape in later sessions. Each must respect the zero-leak invariant and the `[0,1]` scale convention, and reuse `ingestion.CheckAPIError` on any MFL-sourced fetch (B3 learning: MFL returns HTTP 200 with an error body).
- `SafetyRole` stays unset until SL-OQ-035/036 resolves post-live-data.

---

*Built by: Christopher Campbell + Claude (Anthropic)*
