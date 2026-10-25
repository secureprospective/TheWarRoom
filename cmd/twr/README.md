# twr

`twr` lets a person or an AI helper work a War Room league from a terminal. It
never signs in, never makes tokens, and never confirms a change on its own.

1. In The War Room, a signed-in commissioner makes a connection token
   (`POST /v1/auth/agent-token`, which needs a recent second-factor check). Only
   that person's session can make or revoke one.
2. On the helper's computer:

   ```
   twr connect --address https://league.example --league native.example
   ```

   Paste the token when asked. `twr` checks it against the league, then saves it
   in the computer's keychain (service `TheWarRoom.Agent`). Only the address and
   league go in `twr.json` in the user settings folder.

| Command | What it does |
|---|---|
| `twr ledger` | prints the league's recorded changes |
| `twr preview CHANGE.json` | shows what a change would do; records nothing |
| `twr execute CHANGE.json [--confirmation CODE]` | records a change; G2/G3 need the commissioner's code |
| `twr disconnect` | forgets the saved connection on this computer |
| `twr mcp` | serves `league_ledger`, `preview_change` and `execute_change` to an AI helper over MCP (stdio) |

`CHANGE.json` is one league change (`command`, `args`, `reason`, …); `-` reads it
from stdin. When `expected` is missing, `twr` reads the current revision first.
Every call goes through `/v1/native/dispatch`, the same dispatch the desktop
uses, so token scope, expiry and revocation are checked by the server each time.
A revoked token stops both the CLI and a running `twr mcp` on the next call.

Tests: `go test -race ./cmd/twr` runs both against a throwaway HTTPS league.
