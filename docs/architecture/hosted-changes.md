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

L1's transport generation was not present at this branch's base. The minimal `Desktop.generation`
field fences watcher lifetimes here; integration should reuse the transport/session generation rather
than introduce a second session identity. No changes to `internal/composition/run.go` are needed.
