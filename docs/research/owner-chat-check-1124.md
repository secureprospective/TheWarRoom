> **DRAFT for Christopher's approval** (11-24 overnight). Written by Luna (gpt-6-luna), citations spot-checked by Tom. Gap A6. Finding verified: chat was retired 11-04 (docs/ui/endpoint-registry.csv M-121). Reviving it is a product call.

# Owner chat check (gap A6)

**What the commissioner/owner sees:** Owners cannot chat with each other inside TheWarRoom. The app has no chat action; league talk is handed off to ProBoards (registry M-120, lines 121–122).

## 1. Verdict

**missing** — owner-to-owner chat is not implemented in TheWarRoom's hosted service; the chat concept is explicitly marked retired and its test journey says no command exists (`docs/ui/endpoint-registry.csv:122`; `cmd/twr-canary/journeys.go:57`).

## 2. Pieces

| piece | file:line | hosted server route? | frontend screen? | test? |
|---|---|---:|---:|---|
| Chat feature entry | `docs/ui/endpoint-registry.csv:122` | n | n | `cmd/twr-canary/journeys.go:57` (excluded: no command) |
| Chat canary journey | `cmd/twr-canary/journeys.go:57` | n | n | same file:57 |
| Assistant UI (not owner chat) | `frontend/src/app/v3/V3Shell.tsx:144-146` | n | y (assistant only) | unknown |
| ProBoards hand-off in pool/trade-related UI | `frontend/src/app/pool/state.ts:189`; `frontend/src/app/pool/Pool.tsx:316` | n (external destination) | y (copy text / hand-off) | `frontend/src/app/pool/` tests: unknown |

## 3. What an owner can do today

- Cannot send a live chat message to other league owners inside the app (`cmd/twr-canary/journeys.go:57`).
- The registry marks the Chat feature retired because its communication target was removed (`docs/ui/endpoint-registry.csv:122`).
- The app provides a ProBoards hand-off for league talk rather than built-in chat (`docs/ui/endpoint-registry.csv:121`).
- Some trade/claim workflows prepare text for the owner to post; the app says it does not post it for them (`frontend/src/app/pool/state.ts:189`).
- The “Assistant” control in the shell is not owner-to-owner chat (`frontend/src/app/v3/V3Shell.tsx:144-146`).

## 4. Smallest build to make it work

1. Specify the minimum chat behaviour and privacy rules (league membership, who can read/post, retention/moderation) before building; current documentation says chat's target was removed (`docs/ui/endpoint-registry.csv:122`).
2. Add persistent chat storage and migrations, following the documented storage seam (`docs/architecture/future-seams.md:27`).
3. Add authenticated hosted-server read/post routes and authorization; exact route files are unknown because no chat route exists (`cmd/twr-canary/journeys.go:57`).
4. Add a league chat screen and message composition/read flow; frontend screen is absent (`cmd/twr-canary/journeys.go:57`; `frontend/src/app/v3/V3Shell.tsx:144-146`).
5. Add server and frontend tests, and replace the excluded chat canary with an end-to-end hosted chat journey (`cmd/twr-canary/journeys.go:57`).
