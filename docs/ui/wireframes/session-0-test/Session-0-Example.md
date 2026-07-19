# Session 0 — The Example Session (test run, 2026-07-19)
**Purpose:** prove the GLM design-session pipeline end-to-end at minimal scope, produce the plan's canonical worked example, and harvest frictions into plan tweaks before Session A. **This is the template every real session follows.**

---

## What ran

| Step | Artifact | Result |
|---|---|---|
| 1. Brief written (thin Session-A slice: base unit, grid thesis, panel anatomy, M1 treatment, 1 seed) | `brief.md` (~40 lines) | Lean brief, one provocation seed ("instrument cluster"), explicit ≤120-line output cap |
| 2. Runner fired via background task | `scripts/glm-design-session.sh brief.md glm-direction.md glm-5.2` | Brief scp'd to the Beelink, z.ai coding-endpoint call ran detached there (key never left the Beelink), `EXIT=` sentinel collected. **Zero hand-polling — one completion notification.** |
| 3. GLM-5.2 responded | `glm-direction.md` (42 lines) | **40 seconds.** Fully committed specifics: 4px scale table, three-zone thesis, earned-elevation rule, 28px-row M1 spec, 1 ripcord item. No filler, no mood language. |
| 4. Claude triaged | draft's meta strip | ADOPTED: 4px scale (no half-units) · three-zone instrument thesis ("the table is the instrument, everything else is bezel") · earned-elevation rule ("there when you reload → border; there because you did something → elevation") · full M1 treatment. ADAPTED: hover-only "delta edge" → folded into the inspector. RIPCORD: 1 (below). |
| 5. Wireframe written | `session-0-draft.html` | Self-contained greyscale HTML: 4-column shell + M1 board + inspector + 48px quick-dash strip, 5 numbered callouts, triage strip, ripcord box. Presented rendered to Christopher (the visual gate). |

## The ripcord item (pending Christopher)

GLM challenged the comms panel as a persistent fourth column (~18% of a 1440px viewport for an interrupt surface) → proposed a summoned overlay with the 48px strip as the idle state, density-aware default. **Claude's read: this substantially agrees with the locked quick-dash rule; recommend "collapsed by default, summon-over" as the Session A baseline.** Christopher pulls or blesses at the Session 0 gate.

## Frictions found → fixes applied

1. **SSH auth:** the bare `ssh chris@beelink` failed (`Permission denied`) — the Beelink requires the dedicated key. Fix: `-i /root/.ssh/beelink` is hardcoded in the runner; never rely on default key resolution.
2. **Key residency:** `ZAI_API_KEY` lives only in the Beelink's `~/.config/opencode/zai.env` (and is NOT exported for non-interactive SSH). Fix: the runner sources it explicitly inside the remote job; the key never crosses to CT105 or the repo.
3. **Lean briefs beat fat briefs:** a ~40-line brief with one seed and an explicit output cap produced sharper commitment than a digest-stuffed prompt would. Fix (plan §6): attach digest *excerpts* relevant to the session, not whole digest files, and give every brief an explicit output-length target.
4. **Runner quoting is the fragile part:** the remote detached job nests bash→setsid→python heredoc quoting. It is proven as written — treat edits to that block as risky and re-test with a Session-0-sized run after any change.
5. **Time budget:** 40s actual vs 900s allowed. Keep the generous timeout (the Beelink is shared; contention is real) — the sentinel pattern makes over-budgeting free.

## What a "session" costs

One divergent run ≈ 40s + ~3k tokens of cheap GLM. A full session (2 divergent + 1 synthesis) is minutes, not hours — the human gate is the long pole, exactly as designed.
