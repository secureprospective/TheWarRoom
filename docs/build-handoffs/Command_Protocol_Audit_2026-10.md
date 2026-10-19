# Command Protocol — Direction Notes and Audit, 2026-10-19

**Status:** evidence for `Hardening_Plan_2026-10.md` (phases 8–12). Nothing built.
**Read by Claude from source:** `internal/commissioner/command.go`, `ledger.go`; root
`commissioner_app.go`; `frontend/src/app/commands/registry.ts`, `search.ts`, `CommandBar.tsx`;
`internal/governance/census.go`.

## 1. Direction (Christopher, 2026-10-19)

- **The goal is a first-class command surface:** a typed CLI and a league-aware assistant used by
  people and by AI, "as if we are building a terminal for an operating system." The 10-13 ruling
  stands: one surface; every control is a registry command with permission and gravity; nothing is
  screen-only.
- **Real people are interested in running native leagues.** This week is the cleanup before any new
  direction.
- **The command work must cover the whole range of users, seamlessly:**

  | Mode | Who | Shape |
  |---|---|---|
  | Solo owner | One league, working free-agent pools | Desktop app, MFL harness local, engine embedded |
  | Owner in a hosted league | Client only | Desktop or web client against a server, MFL harness local with the owner's own key |
  | Self-hosting commissioner | Many leagues, many owners | Runs the TheWarRoom server: league clock, ledger, bids, per-league storage, sign-in |

- **Client app plus server install is the likely native future,** but not to be dug into yet. The
  architecture decision goes to the expert panel when that door opens.

### What that range requires of the protocol
1. **One catalog, every surface.** The GUI bar, the `twr` CLI/REPL, AI tools and a future server API
   all run the same commands, generated from one definition.
2. **Transport-neutral.** One command, three transports: in-process (solo), a local socket (CLI
   beside the app), the network (server).
3. **Identity on every call.** Principals come from a session, never from arguments. Per-league
   roles. AI uses scoped tokens: read and preview by default, execute only by grant.
4. **League on every call.** The CLI gets contexts (`use league X`), like kubectl.
5. **Server-owned clock.** Bids and waivers must process at the deadline with nobody's laptop open.
6. **Versioned.** Self-hosters run mixed versions: catalog/protocol version negotiation and
   upcasting for stored commands.
7. **Introspectable.** Each command describes itself (arguments, types, who may run it, gravity,
   undo, preview), so help text, tab completion and AI tool schemas are generated, never hand-written.
8. **Reads are commands too.** A CLI or AI needs `show standings`, not only writes.
9. **MFL stays client-side.** No owner's MFL key on a shared server; never write to MFL.

## 2. Audit — what exists today

```
Frontend (TypeScript)                         Go
command bar = fuzzy picker                    TargetCommissionerExecute(body)  ← one generic write door
registry: ~50 verbs, roles gm/commish/admin,    refuses 6 kinds of owner command → they use their own IPC doors
gravity, undo; run() = UI code                commissioner catalog: ~47 commish.* commands
'commissioner.execute' = tunnel (G0, all roles)  RawCommand envelope · census permissions · gravity
                                                 · preview/dry-run · receipts · retry keys
MFL side: ~78 Target* IPC methods, no commands  Reads: Target* methods, not commands
```

## 3. Verdict

### Sound: keep and build on
| # | What | Why it is right for the future |
|---|---|---|
| S1 | The `RawCommand` envelope: league, command, args, reason, expected revision, request key, effective time, visibility | Already league-scoped, idempotent and concurrency-safe. This is a network-grade envelope. |
| S2 | Principal from a `Session`, never from arguments (`commissioner_app.go:25`) | Exactly what a server with real sign-in needs. Swap the session; nothing else changes. |
| S3 | `Authorizer.Allow(league, principal, command)` + the governance census (CM-001…) | Per-league, per-role permission, decided inside the write transaction. |
| S4 | Preview shares every check with execute (`checkCommissionerCommand`) | Preview and execute can never disagree, which makes "AI previews by default" safe. |
| S5 | Gravity G0–G3, required reasons, receipts in plain words, refusals separated from failures (`Refusal.Reason`) | The confirmation model a CLI and an AI both need. |
| S6 | Commands are the ledger's records | The native league is command-sourced already, which is what a server replays. |

### Gaps: off track for the future (ranked)
| # | Gap | Evidence | Why it matters | Cost |
|---|---|---|---|---|
| G1 | **The catalog is not data.** Gravity is a `switch`; arguments are one union struct of about 35 optional fields (`RawArgs`); validation is spread across functions. No command can describe itself. | `command.go:59–128` | Help, completion, AI tool schemas and the GUI can't be generated from it. Every new surface would re-hand-write the catalog. | M |
| G2 | **Two catalogs that disagree.** The frontend registry has its own roles (`gm/commish/admin`) and gravity; `commissioner.execute` is tagged G0 for all roles while the Go side treats many of its commands as G3. | `registry.ts:71–74, 464` | The bar can't show the real gravity or hide commands a role may not run. A CLI would need a third copy. | M |
| G3 | **Policy lives in the IPC door.** The generic execute door refuses six kinds of owner command (trade, waiver and bid actions, lineup set, draft pick, the starting seed), which only work through their own IPC methods because those gather facts (calendar, held data) first. | `commissioner_app.go:45–63` | A CLI or server calling "the command" gets different behaviour by door. Each command should declare the facts it needs, and the engine should gather them. | M |
| G4 | **The MFL side has no commands.** About 78 `Target*` methods, no catalog entries. | root `*_app.go` | The solo-owner mode (free-agent pools) has nothing for a CLI or AI to call. | L |
| G5 | **Reads are not commands.** Standings, results, ledger and pools are separate IPC methods. | `commissioner_app.go:83–225` | `twr show standings` and AI reads need catalog reads with the same permission and privacy filtering. | M |
| G6 | **No versioning.** No protocol or catalog version, and the ledger stores `RawCommand` verbatim, so the wire format *is* the stored format. | `ledger.go:18–27` | Mixed-version self-hosting breaks. Changing an argument shape changes how history decodes. Needs a version per command and upcasters. | M |
| G7 | **The bar is a picker, not a grammar.** Fuzzy search over labels; no parser for `domain.verb key=value`. | `search.ts`, `CommandBar.tsx` | The typed grammar promised in the Target UI Spec §2 doesn't exist yet. Fine for now; the CLI needs it. | S |
| G8 | **Local identity is one machine user.** The principal is a hash of the install dir and UID. | `commissioner_app.go:21` | Correct for solo, but there are no accounts or tokens. Server mode needs real sign-in. S2 makes that a swap, not a rewrite. | L (server) |
| G9 | **The clock runs in the desktop app.** Bid and waiver processing are app workers. | `native_waivers.go`, `native_bid_clock.go` | Hosted leagues need the clock on the server. Fine for solo. | L (server) |

### Bottom line
**The native command engine (S1–S6) is on the right track, and it is the hard part.** What's missing
is one catalog, as data, that every surface reads (G1, G2, G5, G6), and moving policy out of the
IPC doors (G3). The MFL-side commands (G4) and server identity and clock (G8, G9) are bigger and
belong to later tracks.

## 4. Track
Scheduled as phases 8–12 of `Hardening_Plan_2026-10.md`. That plan is the single source for order
and ETA.

## 5. Christopher's rulings (2026-10-19)
1. **CLI local socket:** undecided; it goes to the expert panel in C0.
2. **Order:** C5 before C6 is fine. The MFL-side commands must exist by the time a real owner other
   than Christopher uses the app, which will be near a real Alpha release.
3. **Panel scope:** keep C0 small. It covers the protocol shape and the socket question now, not
   the full client/server design ("no need to launch the space shuttle now").
