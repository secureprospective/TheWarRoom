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

*(Append entries below as T1/T2/T3 proceed.)*
