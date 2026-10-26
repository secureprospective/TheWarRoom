# twr

`twr` lets a person or an AI helper work a War Room league from a terminal. It
never signs in, never makes tokens, and never confirms a change on its own.

1. In The War Room, a signed-in commissioner verifies a fresh authenticator code,
   opens league access, and chooses **AI helpers → Connect a helper**: what it may
   do ("look and suggest" or "also record") and which franchise it acts for. The
   desktop shows a connection line starting `twr1.` once. Only that person's
   session can make or disconnect a helper (`/v1/auth/agent-token`, `/revoke`).
2. On the helper's computer:

   ```
   twr connect --address https://league.example --league native.example
   ```

   Paste the connection line when asked. `twr` checks it against the league, then
   saves the secret in the computer's keychain (service `TheWarRoom.Agent`). Only
   the address, league and the helper's public ID go in `twr.json` in the user
   settings folder.

| Command | What it does |
|---|---|
| `twr ledger` | prints the league's recorded changes |
| `twr preview CHANGE.json` | shows what a change would do; records nothing |
| `twr execute CHANGE.json [--confirmation CODE]` | records a change; G2/G3 need the commissioner's code |
| `twr disconnect` | forgets the saved connection on this computer |
| `twr mcp` | serves `league_ledger`, `preview_change` and `execute_change` to an AI helper over MCP (stdio) |

`CHANGE.json` is one league change (`command`, `args`, `reason`, …); `-` reads it
from stdin. When `expected` is missing, `twr` reads the current revision first.

Bigger changes (G2/G3): `twr preview` adds a `confirmRequest` line starting
`twrc1.`. The commissioner pastes it under **AI helpers → Confirm a helper's
change**, sees the exact change, confirms, and hands back a code that works for
that change only, for five minutes: `twr execute CHANGE.json --confirmation CODE`.
If the league changed in between, the code no longer matches; preview again.
Every call goes through `/v1/native/dispatch`, the same dispatch the desktop
uses, so token scope, expiry and revocation are checked by the server each time.
A revoked token stops both the CLI and a running `twr mcp` on the next call.

Tests: `go test -race ./cmd/twr` runs both against a throwaway HTTPS league.

## Moving a league

`league-export save` creates an encrypted local copy; `league-export check` verifies it
without importing or activating anything. This console operation needs the existing
league database and a private export key, not the saved HTTPS connection token. See
[`internal/leaguetransfer/README.md`](../../internal/leaguetransfer/README.md) for the
commands, key custody, pinned-evidence requirements and authority warnings.

## Authorization regression matrix

`release_matrix_test.go` sends every catalog mutation fixture through the real CLI
and SDK MCP tools to a disposable TLS hosted server. Both helper modes, two
leagues, undelegable commands and revoked retries must leave both histories
unchanged. The fixture inventory is checked against `NativeCommands` in
`internal/app/commissioner`; release evidence is in
`docs/build-handoffs/Release_Blocker_Matrix_2026-10.md`.

## Named league verbs

`twr --help` lists registry-generated mutation names, such as
`twr commish.rules.set CHANGE.json`. These use the same JSON file as `twr preview`
(the command name can be omitted). The default is preview; recording needs
`--mode execute` and, where required, `--confirmation CODE`. A file naming a different
action is refused. The existing `preview`, `execute`, `ledger` and helper/MCP verbs
remain available. These names do not grant access: the server still checks the token,
current permissions and human confirmation. Read-only workflows, held seed and the
special draft-pick door are not published as generic mutation verbs.
