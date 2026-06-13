# Session Brief: Raising TheWarRoom's Pre-B0 Confidence to 80%+ on All Four Metrics

**Written:** 2026-06-13, at the close of the Pre-Build Friction Testing session
(T1-T3 + synthesis, full detail in `Fable_Friction_Log.md`).
**Audience:** the Claude session that picks this up next (recommended:
Opus, given the engineering + judgment density of what's left).
**Recommended branch:** `session/confidence-to-80` (new — both prior session
branches, `session/prebuild-friction-testing` in this repo and
`session/go-overlay-g0` in `christopher-coding-standards`, stay open and
committed; this session's work either lands on a new branch or continues one
of those — your call once you've read the state).

---

## 0. How to use this document

This is a full briefing, written so you don't need to re-derive anything from
`Fable_Friction_Log.md` to get started — but that log (13 numbered frictions
+ final synthesis, ~460 lines) is the primary source of truth for *why* each
item below exists, and you should read it before making any of the
Christopher-facing decision proposals concrete. `Fable_TheWarRoom_code_plan.md`
(Section 9 especially) is the primary source for the collaboration-workflow
rebuild. The companion `CLAUDE.md` for this project has the current build
state, open items table, and hard constraints — read it first per the normal
Session Start Protocol.

**Framing from Christopher for this session:** session-credit budget has
reset — there is real room to do actual engineering here (a struct-wrap
refactor, a custom analyzer if that's the chosen path, diagnosing CT106
networking, a full T1-T3-style re-run), not just documentation. Don't
artificially scope this down to "just write more docs." At the same time:
Christopher is not a Go/security expert. Every decision gate below must be
presented with a clear plain-language explanation of **what the risk
actually is if left unaddressed**, so he can make an informed call — he is
capable of making the right call when the tradeoff is laid out concretely.
**Every fix in this session must preserve Claude's own integrity (no
shortcuts that compromise correctness to "make a number go up") and must
keep the whole ecosystem (CT105, CT104/agy, the GitHub repos, credentials)
secure** — SHA-pin everything pinnable, no hardcoded secrets, no loosened
permissions beyond what's explicitly decided below.

---

## 1. The four metrics, current values, and what moves each one

These four numbers were measured (not estimated) at the end of the friction
session. Target: all four ≥ 80% by the end of this session, OR a concrete,
Christopher-approved reason why a given metric's remaining gap requires work
beyond this session (e.g., the weak-model test depends on CT106 being
reachable, which may be a Proxmox-level fix outside Claude's reach).

### 1a. Plan-fidelity: 78% → 80%+

Remaining gaps, in priority order:
1. **Friction #13 finding #5** — `Fable_TheWarRoom_code_plan.md`'s WF1A
   skeleton says "Do is the ONLY exported surface," but the same plan's B1
   "do better" guidance explicitly asks for an exported `DiscoverHost`. This
   is a real contradiction inside the planning artifacts, surfaced by
   actually building B1. **Fix:** amend the WF1A skeleton text (in the
   companion plan) to read something like *"`Do` and `DiscoverHost` are the
   two sanctioned exported surfaces for B1 — `DiscoverHost` is
   host-discovery-specific and MUST itself route through `Do` (it does, as
   implemented) so it inherits rate-limiting/retry; no other exported
   methods are permitted."* This is a low-stakes wording fix — draft it,
   show Christopher the diff, get a one-line confirmation.
2. **AD-06 and `interface{}` decisions** (see Section 2 below) need to be
   written into the plan as LOCKED decisions, not "documented gaps." A gap
   that's "documented" but not "decided" is still a plan-fidelity hole —
   close it by recording Christopher's actual choice (with a one-paragraph
   rationale) in `Fable_TheWarRoom_code_plan.md` and/or a new short ADR.
3. **Optional, time-permitting:** a final read-through pass cross-checking
   AD-01 through AD-25 (session-3 audit) against what the G0 overlay
   (`.golangci.yml` + README) can actually enforce, looking for any OTHER
   "the plan assumes tool X enforces Y" claims that — like the depguard glob
   bug (#8) and AD-06/forbidigo (#6) — don't hold up. You already found the
   two most likely candidates; this is a sweep for a third, not expected to
   be large.

### 1b. Realized-enforcement: 72% → 80%+

This metric is almost entirely gated on the AD-06 and `interface{}`
decisions (Section 2). Once a mechanism is chosen and implemented for each:

1. Implement the chosen mechanism(s) in `christopher-coding-standards`
   templates/go overlay (on top of `session/go-overlay-g0` @ `86bcde6`, or a
   new branch off it).
2. **Re-run the T1 deliberate-violation test** (`/tmp/g0-verify/` pattern, or
   recreate it — it's a throwaway scratch module, not committed) with TWO new
   `bad.go` cases added: one direct `playerid.PlayerID("99")` bypass
   conversion, one bare `interface{}`/`any` parameter+return. Confirm BOTH
   now produce a non-zero exit from `make lint` / fail `git commit` via
   pre-commit — i.e., go from 5/7 to 7/7 (or however many total invariants
   now exist) gates firing.
3. Document the final gate-fire table in `Fable_Friction_Log.md` as a new
   "T1 re-run" entry (don't overwrite the original — append, this log is a
   running record).

### 1c. Collab-plumbing: 58% → 80%+

This is the big structural item — see Section 3 ("Collaboration workflow
rebuild") for the full scope. Summary of what closes the gap:

1. **Friction #12 fix** (Section 2, decision gate) — once Christopher
   provides a working PAT, push BOTH pending branches:
   - `session/prebuild-friction-testing` (this repo, @ `d573420`) — contains
     the companion plan, the full friction log, and the fixed B1
     `internal/mfl` files.
   - `session/go-overlay-g0` (`christopher-coding-standards`, @ `86bcde6`) —
     the G0 overlay itself.
2. **Run the ACTUAL T3-as-designed validation** that never happened: from
   CT104 (`ssh antigravitybox`), have agy `git pull` (or fetch) the pushed
   `session/prebuild-friction-testing` branch in its standing TheWarRoom
   clone, and confirm it can read the committed `.golangci.yml` + companion
   plan + friction log directly from git — NOT via the `/tmp/t3-review` SSH-relay
   workaround. This is the missing data point: does Section 9.2's "shared
   committed config, both pull" mechanism actually work once the credential
   gap is closed?
3. **Formalize the triage step.** Friction #13 found that a confident,
   line-specific, "blocking"-severity finding from agy can simply be wrong.
   Section 9.4 already says "Claude triages" but doesn't say HOW. Add a
   concrete protocol to Section 9 (e.g., a new "9.8 Triage Protocol"): every
   agy finding that cites a specific file:line MUST be checked against the
   actual source (read the cited range, confirm the claim) before being
   reported to Christopher as a defect or acted on. Findings that don't
   survive this check are logged (for the friction record / agy calibration
   tracking) but not escalated.
4. **Decide SSH-relay's role.** It worked cleanly (~6 min round trip,
   T3's actual review). Recommend formalizing it as a SECONDARY channel for
   fast, ad-hoc, read-only review requests (scp files to `/tmp/<topic>/` on
   the target machine, prompt, scp results back) — while the PRIMARY channel
   for "shared enforcement config both agents build against" (9.2's actual
   purpose) stays git, now that #12 is fixed. Document this split explicitly
   in Section 9.2/9.3 so future sessions don't have to rediscover it.
5. **Update 9.6's "memory symmetry" framing.** Friction #11 found agy has a
   STANDING, self-updating TheWarRoom clone with live research — it does NOT
   actually behave like a stateless, self-contained-prompt-only reviewer, no
   matter how the prompt is framed. T3 showed this is a FEATURE (agy's
   repo-aware review caught real bugs a blind review might have missed
   structural context for). Rewrite 9.6 to say so explicitly: "agy will use
   its standing repo context regardless of prompt framing; this is expected
   and beneficial for review tasks, but means review prompts should NOT
   assume agy is blind to repo state."
6. **Optional, time-permitting — a "T4":** once 1-5 are done, re-run a small
   end-to-end cycle (pick any small, real, OPEN piece of work — even
   re-reviewing the now-fixed `internal/mfl/client.go` for a second-pass
   First-Instance Template Review using the new triage protocol) to confirm
   the rebuilt workflow holds up. This is the highest-value optional item if
   time allows, because it's the only way to know the rebuild actually works
   rather than just reads well.

### 1d. End-goal: ~60% → 80%+

The single largest lever here is the ORIGINAL T2 question that never got
answered: **can a genuinely weak local model (qwen2.5-coder:14b via
AiderBox/CT106 + Ollama on the Beelink) produce standards-conforming code
from a self-contained brief?** Predicted ~35%, never measured (CT106 was
"NO ROUTE TO HOST" this session — Friction #2).

1. **Diagnose CT106 reachability.** Start with `ping 192.168.1.106` and
   `ssh aiderbox` from CT105. If still unreachable, this may be a
   Proxmox-host-level issue (container stopped, network bridge issue) that
   Claude cannot fix from CT105 — in that case, this becomes a
   Christopher-action item (check via Proxmox console / `pct status 106` /
   `pct start 106`, possibly relayed via `/root/paste.md` since CT105 has no
   route to `.200` either per recent connectivity checks). **Do not spend
   excessive time on this if it's clearly infra-level** — flag it cleanly and
   move on to the items that don't depend on it.
2. **If CT106 comes up:** run the ORIGINAL T2 design — qwen2.5-coder:14b via
   Aider, given the SAME self-contained B1 brief used for agy (`/tmp/b1_brief.md`
   content, reusable), scored on the SAME Conformance Rubric (Structural /
   Standards / Correctness / No-hallucination / Slop-tax, 0-2 each, max 10).
   **Remember the Beelink hard-reboot constraint: do NOT exceed
   `num_ctx: 16384`** (AiderBox is already configured at this cap per prior
   session memory — verify it's still set, don't change it upward).
3. Whatever the result — even a low score — this REPLACES the unmeasured
   "~35% predicted" with a real number, which is itself the point. A low
   score isn't a session failure; it's the data point this whole exercise
   was missing.
4. The other end-goal contributors (AD-06/`interface{}` decisions, #12 fix,
   B1 fixes already done) are shared with 1a-1c — no separate work needed
   beyond what's listed there.

---

## 2. Decision gates requiring Christopher's confirmation

Present each of these to Christopher with the explanation given (adjust
tone/length as needed, but keep the risk framing — he is not a Go/security
expert and needs to understand *what breaks if we don't fix this* in plain
terms, not just *what the fix is*).

### Gate 1 — AD-06: `playerid.PlayerID` bypass conversion

**What it is:** `internal/playerid` is supposed to be the ONLY way to create
a `PlayerID` value, because `playerid.New()` validates and normalizes the ID
(MFL IDs under 1000 need leading zeros — a hard constraint in this project's
`CLAUDE.md`). But as currently typed (`type PlayerID string`), any code
anywhere can write `playerid.PlayerID("99")` directly — skipping validation
entirely — and the type system won't stop it.

**What breaks if this isn't fixed:** a player ID created via the bypass
(e.g., `"99"` instead of the correct `"0099"`) would silently fail to match
records elsewhere in the system that DO use the validated form. The failure
mode is "this player's stats/roster/contract don't show up somewhere" — a
data-integrity bug that's hard to trace back to its source, and could
surface anywhere across 38 build sessions, in code written by different
models.

**Options:**
- **A — Struct-wrap (RECOMMENDED).** Change `PlayerID` from
  `type PlayerID string` to `type PlayerID struct { id string }` with the
  field unexported. Outside the `playerid` package, Go's compiler makes it
  IMPOSSIBLE to construct a `PlayerID` except via `playerid.New()` — this is
  a permanent, zero-maintenance guarantee enforced by the compiler, not a
  linter. **Cost:** a one-time refactor of every place that currently treats
  `PlayerID` as a raw string (string formatting, JSON, SQLite binding) to go
  through accessor methods (`.String()`, etc.). Right now this cost is
  near-zero — almost nothing is built on top of `PlayerID` yet. It only grows
  with every session built before this is fixed.
- **B — Custom static-analysis checker.** Write a small Go tool (using the
  standard `go/analysis` framework) that specifically flags
  `playerid.PlayerID(...)` conversions outside the `playerid` package, wired
  into the lint pipeline. **Cost:** new tool to build, test, and maintain;
  must be SHA-pinned/vendored like any other CI dependency (per the
  trivy-action lesson — no loosely-versioned third-party tool in CI). Carries
  the SAME risk class as the depguard bug found this session: a bug in a
  custom linter can silently not fire, and nobody would know until something
  breaks downstream.
- **C — Code-review checklist only.** Zero engineering cost. Relies on every
  future session (Claude, agy, or a local model) remembering to check this —
  for 38 sessions. This is exactly the "Invisible Risk" category from Section
  9.5's own taxonomy: the risk that's invisible BECAUSE nothing automated
  catches it.

**Recommendation:** A. It's cheap now, free forever after, and doesn't add a
new tool to maintain.

### Gate 2 — `interface{}`/`any` escapes

**What it is:** A function signature like `func Anything(v interface{})
interface{}` accepts/returns "any type at all" — it turns off Go's type
checker for that boundary. T1 confirmed NO currently-enabled linter catches
this (the obvious-sounding `interfacebloat` only checks interface
*declarations* with >5 methods, not bare `any` usage).

**What breaks if this isn't fixed:** this project has hard layer-boundary
rules (e.g., "Layer 2/Layer 4 zero scoring leaks" — no sub-signal in Film/
RAS/Breakout may reference fantasy-points or scoring config). The TYPE SYSTEM
is what would normally make a violation of that rule a compile error. An
`interface{}` escape at a layer boundary defeats that — a scoring-leak
violation could pass through an `interface{}` parameter completely
undetected by the compiler OR any current linter.

**Options:**
- **A — Code-review checklist.** Zero cost, weakest guarantee — same concern
  as Gate 1 Option C.
- **B — Custom `go/analysis` checker** that flags `interface{}`/`any` in
  exported function signatures, with an explicit `//nolint` escape hatch for
  legitimate cases (e.g., generic JSON helpers). Moderate cost; same
  SHA-pin/maintenance considerations as Gate 1 Option B.
- **C — Combine with Gate 1 if B is chosen there.** One custom analyzer
  covering both AD-06 and `interface{}` rules amortizes the "build and
  maintain a custom tool" cost across two real gaps instead of one.

**Recommendation:** if Gate 1 → A (struct-wrap, no new tool), then for this
gate, B/C (one small custom analyzer covering just `interface{}`/`any`) is
proportionate — OR, if Christopher would rather avoid ANY custom tooling
right now, fall back to A (checklist) with an explicit note to revisit if
this class of bug ever causes a real incident. This is more genuinely
Christopher's call than Gate 1 — lay out both paths evenly.

### Gate 3 — GitHub PAT scope/rotation (Friction #12)

**What it is:** CT105's stored credential (`~/.git-credentials`, user
`secureprospective`) can read `github.com/secureprospective/TheWarRoom` but
gets a 403 on push. Two branches with real work (the friction log, companion
plan, fixed B1 code, and the G0 overlay) are stuck local-only.

**What's needed:** Christopher generates a new (or re-scoped) **fine-grained**
GitHub PAT scoped to ONLY `secureprospective/TheWarRoom`, with "Contents:
Read and write" (and "Pull requests: Read and write" if Claude will open PRs
from this repo). Then update `~/.git-credentials` on CT105.

**Why fine-grained / single-repo, not a classic PAT with full `repo` scope:**
a classic PAT with `repo` scope grants read/write to EVERY repo Christopher
owns. If CT105 (or its stored credentials file) were ever compromised, the
blast radius is "every repo." A fine-grained PAT scoped to just `TheWarRoom`
limits the blast radius to this one repo — same capability for the actual
work, much smaller exposure.

**How to do this safely:** per global `CLAUDE.md`, Claude should NOT
generate, see, or insert the actual token. Claude prepares the exact commands
(with a placeholder for the token) in `/root/paste.md`, Christopher runs them
in his own terminal session, pasting the real token only there.

### Gate 4 — CT106/AiderBox reachability (for the weak-model test)

**What it is:** CT106 was "NO ROUTE TO HOST" this session (Friction #2) —
different from the prior "paused at Phase 7" status. Needed for 1d (the
single biggest end-goal lever).

**What's needed:** if `ping`/`ssh aiderbox` from CT105 still fail, this is
likely a Proxmox-level issue (container state, network bridge) that CT105
cannot diagnose or fix on its own — Christopher may need to check via the
Proxmox console directly (CT105 also currently has no route to `.200`, the
Proxmox host itself).

**Framing:** this is an availability issue, not a security decision — flag it
plainly, don't over-explain. If it's a quick fix, great; if not, 1d's weak-
model number stays unmeasured for one more session, which is an acceptable
(already-documented) gap, not a crisis.

### Gate 5 — WF1A skeleton wording (Friction #13, finding #5)

Low-stakes. Draft the specific wording change (Section 1a, item 1 above),
show Christopher the before/after, get a one-line "yes."

### Gate 6 — Collaboration workflow rebuild scope

Present as a scope choice, not a yes/no:
- **Minimal:** Gate 3 fix + the actual T3-as-designed git pull/fetch
  validation (1c.2) + the triage-protocol writeup (1c.3). Closes the loop on
  what was directly tested this session.
- **Expanded (recommended, per Christopher's "not a blind spot" instruction):**
  minimal, PLUS formalize SSH-relay's role (1c.4), update the 9.6 memory-
  symmetry framing (1c.5), and — if time allows — the optional T4 re-run
  (1c.6).

---

## 3. Hard constraints carried forward (do not relax any of these)

- **Branch discipline:** never work on main, branch `session/<short-desc>`,
  never `git --no-verify`.
- **SHA-pin every CI action/hook** — applies doubly if Gate 1/2 produces a
  new custom analyzer: it must be vendored or fetched via a pinned commit
  SHA, never `@latest`/`@main`.
- **No hardcoded secrets anywhere** — including the PAT fix (Gate 3): use the
  `/root/paste.md` relay with a placeholder, never a real token in any file
  Claude writes.
- **agy never commits, never pushes, never edits living docs** — findings
  only. This does NOT change as part of the collab-workflow rebuild (Section
  3 of this brief / Gate 6) — if anything, Friction #13 (agy's one
  hallucinated "blocking" finding) is a reason to keep this boundary firm,
  not loosen it.
- **Christopher confirms before any merge to main**, and before any push
  that makes work visible to others (GitHub pushes count).
- Full bash output, no truncation/collapsing (Christopher's standing
  preference).

---

## 4. Success criteria / end-of-session deliverable

By the end of this session, append a new section to `Fable_Friction_Log.md`
("Confidence-80 session results," 2026-06-13 or later) containing:

1. Re-measured values for all four metrics, using the same evidence-based
   approach as the original synthesis (cite specific gate-fire results,
   specific findings resolved, specific commits).
2. A record of each decision gate's outcome (Christopher's actual choice +
   one-line rationale) for Gates 1, 2, and 6.
3. Status of Gates 3, 4, 5 (fixed / blocked-on-infra / confirmed).
4. If any metric remains below 80%, an explicit, named reason and what
   session would close it (don't paper over a gap with a higher number than
   the evidence supports — that would be exactly the kind of "slop" this
   whole exercise exists to catch).
5. Updated `CLAUDE.md` (Current Build State, Open Items table, Next branch)
   and a new backbone `context.md` session-close entry, per the normal
   Session Close protocol.
