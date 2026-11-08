# Future seams: where the next builds plug in

**Read this before building any new server feature.** That especially means the social builds: moments and brag cards, share links, invites and the "start your own league" link, Tuesday and Thursday shows, chat, and new AI-helper powers.

TheWarRoom is AI first and CLI first, with a GUI overlay for people who aren't technical. The server is built so these features plug into existing paths. They don't get new side doors.

**Ruling (Christopher, 2026-11-08):** keep the code open for these builds, but don't build them now. Leave labelled places to connect them. Security stays top notch. The background is in `docs/RESUME-2026-11-08.md` §12 (the community and growth-loop ruling of 10-28).

## Rules

1. **These are seams, not stubs.** There is no empty placeholder code. Each seam is an existing, working path that a new feature extends.
2. **Every seam is marked in the code** with a comment that starts `FUTURE-SEAM(<name>):`. Find them with `grep -rn "FUTURE-SEAM(" internal`.
3. **Every marker has a row below, and every row has a marker.** `internal/seams` enforces this. Add, move or remove a marker and this table in the same commit.
4. **A new feature uses the seam.** It doesn't build beside it. If a feature needs a door that isn't listed here, that needs a ruling from Christopher first.
5. **Security doesn't bend at a seam.** Every seam is already behind login, the role and confirmation gates, the revision check and the "who did it" ledger. Plugging in must keep all of them.

## The seams

| Name | File | What plugs in | How |
|---|---|---|---|
| `commands` | `internal/commandcatalog/workflow_commands.go` | New actions (share a moment, post in chat, send an invite) | Add the command definition in `internal/commissioner`. The catalog picks it up from `nativeMutations()` ∩ `accounts.DelegableCommand`. CLI, screens and AI all get it through the same route. Don't hand-list it here. |
| `scopes` | `internal/accounts/tokens.go` | New token powers (e.g. a share-only or chat helper) | Powers are command-name prefixes. A new family goes into the allow or deny prefix rules with a test. Anything that touches roles, seats or credentials stays non-delegable. |
| `events` | `internal/store/history/events.go` | Moments, weekly shows, chat history, activity feeds | Every confirmed league change is appended here as a revisioned event. Moment detectors **read** this ledger on the server. They never write a second log. |
| `stream` | `internal/serverhttp/invalidations.go` | Live push to phones and desktops (a new moment, a chat message) | The stream only says "something changed for you". By design it never carries event contents, history or a bearer token. Clients reread through authorized reads. Keep it that way. |
| `storage` | `internal/db/migrations.go` | A feature's own tables (moments, cards, chat, invites) | New tables come in as new migration steps. A feature never alters the league, roster or contract tables to hold its data. Backups and the snapshot pipeline cover the new tables automatically. |

Research behind the social builds: `docs/research/social-moments-2026-11-08.md`.
Tiny specialised models (prospecting): `docs/research/tiny-models-2026-11-08.md`.
Session summary of this direction: `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md`.
