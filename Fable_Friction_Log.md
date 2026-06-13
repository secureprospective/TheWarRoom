# Fable Friction Log — Pre-Build Friction Testing Phase
**Started:** 2026-06-13
**Branches:** `session/prebuild-friction-testing` (legacy-nfl-fantasy), `session/go-overlay-g0` (christopher-coding-standards)
**Purpose:** Append-only running log of every gate, divergence, relay round-trip, and surprise from T1/T2/T3 and any exploratory work. Pass/fail is binary; this log is the map.

---

## Pre-flight — environment confirmation (2026-06-13)

| Check | Result | Note |
|---|---|---|
| CT105 Go toolchain | **MISSING** — `go: command not found` | apt candidate is `golang-go 2:1.19~1` (2022-era). Will install a modern Go via official tarball, not apt. **Friction #1: G0 assumed Go existed on CT105; it does not.** |
| CT105 golangci-lint | **MISSING** | Must install — version pin TBD, agy/web-research to confirm current stable in Section 8B style. |
| CT105 pre-commit | **MISSING** | pipx/pip install needed. |
| CT105 gitleaks | **MISSING** | Binary release install needed. |
| agy / CT104 (`ssh antigravitybox`) | **REACHABLE** — `agy --version` → 1.0.8 | Good for T2 (redesigned) and T3. |
| AiderBox / CT106 (`ssh aiderbox`, 192.168.1.106) | **NO ROUTE TO HOST** | Not "paused at Phase 7" — fully unreachable from CT105 right now. Moot: Christopher ruled out Aider for this phase regardless. **Friction #2.** |
| Beelink (192.168.1.190) | Ping OK, **SSH refused (port 22)** | Possibly related to the unresolved hard-reboot issue. Not currently needed — T2 redesigned around agy — but flagging since the homelab doc marks this URGENT. **Friction #3.** |
| Internet (CT105) | OK — `https://go.dev` returns 200 | Toolchain installs from official sources are viable. |

## Decision log

- **2026-06-13 — T2 redesign.** Original T2 used qwen2.5-coder:14b via AiderBox/CT106 or Beelink Ollama. Christopher: "Do not use aider, only agy." With CT106 unreachable and Beelink SSH refused anyway, Christopher chose: **agy itself (CT104, Antigravity) is the weak-model build subject for T2** — `ssh antigravitybox 'agy -p "..."'` builds B1 directly, scored on the same rubric. This is a structural change from the original plan (agy was cast as Recon/Audit in T3); for T2 it is the builder. Both roles are tracked as separate, self-contained interactions — agy's statelessness (Section 9.6) makes this safe to do without role confusion bleeding across.

---

## T1 — G0 Overlay Build (2026-06-13)

### Toolchain bring-up (CT105 was bare)

Installed from scratch, all verified-checksum manual installs (no `--no-verify`,
no disabled checks):

| Tool | Version | Note |
|---|---|---|
| Go | 1.26.4 | Official tarball. |
| golangci-lint | 2.12.2 | **Friction #4: golangci-lint's own `install.sh` fails checksum verification** for v2.12.2/linux-amd64 — the script compares the tarball's hash against the `.sbom.json` line in `checksums.txt`, not the tarball's own line. Worked around with manual `sha256sum -c` against the correct line. Documented in overlay README as a confirmed upstream bug. |
| pre-commit | 3.0.4 | apt install OK alone. |
| gitleaks | 8.30.1 | **Friction #5: `apt-get install -y -qq pre-commit gitleaks` fails entirely** ("Unable to locate package gitleaks") — apt aborts the WHOLE transaction on one missing package, including the package that WAS available. Had to split into separate installs; gitleaks via direct binary release. |

### Baseline lint findings — clean "good" code fails lint as specified (Friction #6, #7)

Before writing any deliberate violations, built a scratch verification module
(`/tmp/g0-verify`) mirroring TheWarRoom's canonical `internal/` layout
(`playerid`, `engine`, `ingestion` per Section 3.1) using the companion plan's
OWN skeleton patterns, and ran `golangci-lint run ./...` as a sanity check.
**Result: 2 issues on code that should be "good."**

**Friction #6 — forbidigo cannot enforce AD-06 as specified.** The companion
plan (Section 3.3) proposed a forbidigo rule banning the bypass conversion
`playerid.PlayerID("0531")` outside `internal/playerid`. As configured
(`analyze-types: true`, pattern `^playerid\.PlayerID$`), it ALSO flagged the
type reference in a normal function signature
(`func ValidateRawID(raw string) (playerid.PlayerID, error)`) — forbidigo has
no call-vs-type-position distinction. A pattern narrow enough to catch the
bypass conversion bans using the type at all outside its own package, which
contradicts AD-06's own requirement that domain structs use
`playerid.PlayerID` as a field type. **This is not a config typo — it's a
structural limitation of forbidigo for this use case.** Rule removed from
`.golangci.yml`. Real fix recommended for B0: struct-wrap `PlayerID` with an
unexported field (the plan's own "heavier alternative") so the bypass
conversion fails to *compile*, not just to lint. Documented as "Known
limitation: AD-06" in the overlay README with three ranked options.

**Friction #7 — Section 4 WF 1B skeleton fails wrapcheck as written.** The
WF1B skeleton (`return playerid.New(raw)`, unwrapped) is flagged by
`wrapcheck` (default config — no exemption added) because it crosses a
package boundary without `fmt.Errorf("...: %w", err)`. This is NOT a config
bug — Section 3.3's own `normalize.Roster` example wraps correctly; WF1B's
skeleton is the inconsistent one. **Errata documented in overlay README**:
WF1B's skeleton needs the wrap added before B0, so a weaker model copying it
verbatim doesn't inherit a day-one lint failure.

**Resolution applied:** `.golangci.yml` forbidigo rule removed (defaults
retained — still bans `fmt.Print*`/`println`); `/tmp/g0-verify`'s
`good.go` updated to wrap the error. Re-ran `golangci-lint run ./...` →
**0 issues.** Baseline is now clean; proceeding to the deliberate-violation
(`bad.go`) test.

**Why these matter for the friction-test hypothesis:** the test predicted
"1-2 tooling/config gaps." These ARE that — but notably, both gaps were found
not by writing bad code, but by writing the plan's own "good" skeleton code
and running the plan's own proposed linter config against it. The companion
plan's skeletons and its linter spec were authored without cross-checking
each other against a real toolchain. **If a weaker model (agy, T2) is handed
the WF1B skeleton verbatim and the G0 overlay verbatim, it inherits a
pre-existing lint failure it did not cause** — this is exactly the kind of
"silent ungated drift" the friction-test phase exists to surface before B0.

### Deliberate-violation test (`bad.go`) — gate-fire results

Built `/tmp/g0-verify/internal/scratch/bad.go` (7 violations, one per gate per
README spec) plus `/tmp/g0-verify/internal/ingestion/bad_depguard.go` (the
depguard cross-layer case, since it requires a file actually located inside
`internal/ingestion/`). Ran `go build ./...` (OK — lint issues don't block
compilation), `golangci-lint run ./...`, and `git commit` against
pre-commit-installed hooks.

**Friction #8 — SEVERE, FIXED. depguard `files:` globs need a `**/` prefix
or they never match — every layering rule was silently inert, AND the
negated SQL-confinement rule would false-positive on its own intended
exception.** The companion plan's Section 3.1 layout gave `files:` patterns
like `"internal/ingestion/**"`, `"internal/engine/**"`,
`"!internal/db/**"` — written exactly as the directory tree reads. Empirically
verified (isolated single-rule configs, multiple pattern variants):
- `"internal/ingestion/**"` → **0 matches**, even for
  `internal/ingestion/bad_depguard.go`. Only `"**/internal/ingestion/**"`
  (leading `**/`) matched.
- The negated form `"!internal/db/**"` has the SAME bug in reverse: since
  `internal/db/**` never matches anything, the negation never excludes
  anything, so `sql-confined-to-data-layer` would ALSO fire on
  `internal/db/db.go`'s own legitimate `database/sql` import — a **false
  positive on the one package the rule exists to allow**. Confirmed with a
  throwaway `internal/db/db.go` (`sql.Open(...)`): OLD pattern flagged it,
  NEW pattern (`"!**/internal/db/**"`) correctly excluded it.

**Impact if shipped as-written:** all SIX depguard layering rules
(layer1-no-upward-import, engine-is-pure, store-*-no-siblings ×3,
transactions-only-through-coordinator) would NEVER fire — the entire
"three-layer architectural law becomes a build error" mechanism (Section
8A.2, the central justification for AD-19/G0) would be **silently
decorative**. Simultaneously, `sql-confined-to-data-layer` would
false-positive on `internal/db` and `internal/store` themselves, breaking
the build for CORRECT code on day one of B0 — a weaker model (or Christopher)
would see a lint failure on legitimate code and have no reason to suspect the
*rule's file-matching*, not the code, was wrong.

**Fixed:** all 9 `files:` glob entries across the 6 depguard rules in
`.golangci.yml` now prefixed with `**/`. Re-verified:
`internal/ingestion/bad_depguard.go` (imports `internal/engine`) → depguard
fires correctly; `internal/db/db.go` (imports `database/sql`) → no longer
flagged.

**This is the single highest-value finding of T1** — it's exactly the class
of "1-2 tooling/config gaps" the test hypothesized, except structural rather
than cosmetic: a config that *looks* correct, passes `golangci-lint run`
without error, and produces zero issues — making it indistinguishable from
"the rules are working and the code is clean" until tested against a
deliberate violation.

**Friction #9 — gitleaks allowlists the canonical AWS-docs example key.**
`AKIAIOSFODNN7EXAMPLE` (used as the hardcoded-credential violation in
`bad.go`) was caught by `gosec` (G101) but **gitleaks reported "Passed" — did
not flag it**. This string is AWS's own documentation placeholder and is
allowlisted by gitleaks' default ruleset as a known test fixture. Two
takeaways: (a) gosec provides real defense-in-depth here — if it were ever
disabled, this exact credential shape would slip past gitleaks too; (b) don't
reuse `AKIAIOSFODNN7EXAMPLE` as the "must-be-caught" fixture in any future
gate test — it's a false negative by gitleaks design, not a gap. Use a
fake-but-non-allowlisted key shape instead.

**Friction #10 — confirmed silent gate, no clean fix found. `interface{}`/
`any` parameter/return escapes are NOT caught by the configured linter set.**
`func Anything(v interface{}) interface{} { return v }` produces **zero
findings** from `gocritic`, `interfacebloat`, or any other enabled linter.
Root cause: `interfacebloat` only checks INTERFACE TYPE DECLARATIONS with too
many methods (default max 5) — irrelevant to a function parameter/return
type. `revive`'s `use-any` rule (cosmetic — flags literal `interface{}`
syntax, suggests the `any` alias) is not in revive's default rule set as
configured here, and even if enabled would not flag `any` itself, so it
would not address the underlying weak-typing concern either way. **Section 2
item 1's claim ("gocritic + interfacebloat reject interface{}/any escapes")
is FALSE for this configuration.** No lint-level fix identified — this
appears to require either a custom `go/analysis` checker or a code-review
checklist item. Flagged for B0; not fixed in this overlay.

### T1 gate-fire summary

| # | Violation | Linter | Result |
|---|---|---|---|
| 1 | Unchecked error (`db.Exec`, `db.Query`) | errcheck | **FIRED** ×2 |
| 2 | `fmt.Sprintf`-built SQL | gosec G201 | **FIRED** |
| 3 | Hardcoded AWS-style key | gosec G101 | **FIRED** |
| 3b | (same key) | gitleaks | silent — Friction #9 (by design, documented) |
| 4 | Package-level `var` | gochecknoglobals | **FIRED** |
| 5 | `playerid.PlayerID("99")` bypass conversion | forbidigo | silent — Friction #6 (documented limitation, AD-06 needs struct-wrap) |
| 6 | `interface{}` param/return | gocritic/interfacebloat | silent — Friction #10 (no fix found) |
| 7 | `internal/ingestion` → `internal/engine` cross-layer import | depguard | **FIRED** (after Friction #8 fix) |
| bonus | `database/sql` import outside db/store | depguard | **FIRED** |

`make lint` → exit 2 (non-zero). `git commit` → **blocked** by
`golangci-lint-full` pre-commit hook (gitleaks hook passed — see #9).

### T1 verdict

**Strict pass/fail (companion plan's stated criterion — "fail = any silent
gate"): FAIL.** Two of seven specified gates (#5 AD-06 bypass, #6
`interface{}` escape) stayed silent.

**But measured against the friction test's actual purpose** (find and fix
config gaps before B0, don't just pass/fail a checklist): T1 found FOUR
distinct config issues (#6 forbidigo type/call ambiguity, #7 WF1B wrapcheck
errata, #8 depguard glob prefix, #9 gitleaks allowlist), FIXED two of them in
the overlay (#7, #8), root-caused and documented remediation paths for the
other two (#5/#6 → AD-06 struct-wrap; #6 interface{} → code-review item), and
caught #8 — which on its own would have made G0's central enforcement
mechanism (the three-layer law) completely decorative without ever throwing
an error. **The overlay as committed is materially stronger than the overlay
as first written**, which is the actual deliverable.

Remaining open items for B0 (not fixed here, by design — these are
architectural decisions for Christopher, not config fixes):
- AD-06 enforcement: adopt struct-wrapped `PlayerID` (changes public API
  shape of `internal/playerid`) or accept code-review-only enforcement.
- `interface{}`/`any` escapes: no lint-level enforcement found; needs either
  a custom analyzer or a code-review checklist item.

---

*(Append entries below as T2/T3 proceed.)*
