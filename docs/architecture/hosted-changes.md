# Hosted change notifications

The hosted desktop uses no local App or Wails event bus. Local mode is unchanged.

| Owner | Flow and failure behavior |
|---|---|
| `serverhttp.Publisher` | One instance per server process, injected as `Hosted.Publisher`. `NewPublisher` creates its restart epoch. Handler supplies one when omitted. Call `PublishScores(league, season, week)`, `PublishValues(league, season, run)` or `PublishClock(league, season, week, nextBoundary)` **after** the new snapshot becomes readable. |
| Authorized recipient view | The existing head watcher and ledger projection remain the visibility gate. `ObserveView(accountPrincipal, league, revision)` bumps the recipient's broad `league` version only when that view changes; no raw head escapes. The ledger projection does not identify a changed franchise, so no guessed `team` scope is emitted. Clock timers deduplicate the same public boundary across streams. |
| Version reads | After admission and an authorized recipient-ledger read, call `ObserveView`, then `Versions(accountPrincipal, league)`. The returned copy includes only that recipient's counters plus public scopes. This API alone does not authorize a caller or certify a workflow snapshot; the tags owner must preserve snapshot/read ordering. |
| SSE | `X-TWR-Invalidation: 2` opts into epoch/league/scoped decimal versions. No header retains the exact protocol-1 frame. Public publication wakes share the publisher, but recheck session and recipient admission before delivery. Budgets, proof lapse, heartbeats and write deadlines remain. A large frame becomes an empty-topic full-resync hint to stay inside the existing 4096-byte scanner bound. |
| Desktop session | Successful login, enter or proof replacement starts/restarts the watcher; forget, revoke and Close cancel it. Every reconnect reuses immutable captured credentials. `Desktop.generation` fences old callbacks, including old 401/403 responses. Explicit auth refusal clears credentials and publishes signed-out/refused; transport loss backs off and publishes reconnecting. |
| Webview seam | Same-origin/no-store POST `/desktop/changes` accepts `{after: seq}` and waits up to 25 seconds. It returns seq, topics and state, plus epoch/league when available. A bounded 64-notification hint history coalesces crossing topics; overflow, v1 or connection/session transitions require a full authorized-scope reconciliation (empty topics). Timeout returns the current seq and an empty array. |
| Frontend transport | Only `nativeApi.subscribeChanged` calls the long-poll endpoint. It advances the cursor after a validated reply, backs off on failure, and aborts/stops on unsubscribe. The shared read owner must consume hints before notifying screens and perform the separately owned version reconciliation on connect/reconnect. |

## Thin identity fence

`DesktopView.authGeneration` and `DesktopChanges.authGeneration` expose only the decimal session
counter, never credential material. Desktop does not retain the account ID from the login reply;
`hostedReads.ts` uses that generation for the account key too. Status rechecks the admitted seat
through People without incrementing the generation. Other account/helper views retain that admission.

Validated startup/action/helper replies → `observeView` → `hostedReads.setIdentity` → query reset
and late-reply rejection → new identity snapshot → thin screen render. `NativeAccount` keys its
hosted subtree by that snapshot, not by its locally queued presentation state. Identity-changing
actions hide private state before transport; failures leave it hidden.

The module lazily starts exactly one `subscribeChanged` after an admitted view. It consumes topics
before `onHostedChange` listeners. League/team hints immediately suspend private publication and
rendering, then coalesce status rechecks into one in flight and one trailing. A changed epoch or
empty resync hint takes the same conservative path. Status failures remain closed; a later hint or
explicit admission can retry. Signed-out/refused purges and installs an empty view; old generations
are ignored. Version reconciliation (`versions.read`) remains the separately owned C5 seam.

Public entries survive same-server/league account or role switches, including their persisted old
keys. Session reads reuse only public values whose server, epoch, league, season, command/schema
and normalized arguments match. A server/league/epoch switch removes held public entries and disk
copies; pending persistence is removed if its lifetime changes while writing. Local mode never
observes a native view, so it starts no subscription.

Consumer exports: `hostedReads`, `observeView`, `onHostedChange`. Evidence:
`hostedReads.test.ts`, `nativeAccount.test.tsx`, `desktop_identity_test.go`, and the existing workflow
401 generation regression.
