# Alpha 0.11 — what "done" means
**Status: DRAFT. Not approved. Do not build from this file until Christopher approves it.**
See `RELEASE-0.10.md` for how releases are numbered.

## Headline: a formal Update button in the commissioner console
Rulings so far (Christopher, 2026-11-16):
- The button is required and lives in the commissioner console.
- Only the commissioner may use it. Not co-commissioners, not owners.

## What already exists (do not rebuild)
- `twr-upgrade` on the server: checks the signed bundle, backs up first, migrates, restores on failure,
  refuses tampered, expired or older bundles (`deploy/server/upgrade.sh`, `UPGRADE.md`, `internal/releasetrust/`).
- The desktop's own updater and the "update required" screen (`internal/serverclient/updater.go`,
  `frontend/src/app/control/UpdateRequired.tsx`).
- A Server screen sketch with an Upgrade action in the local app (`frontend/src/app/control/room/ServerView.tsx`).

## What is missing
- A published place the server can fetch releases from, and a step that publishes each release there.
- A way for the server to learn an update exists and say which version it is running.
- The button and version display in the hosted commissioner console.
- A way to carry out the install: the server runs locked down and cannot replace its own program or restart
  itself. **This needs a small privileged helper inside the container. That is a security design decision and
  goes to the expert panel before anything is built.**

## Draft done-benchmarks
1. The commissioner console shows the running version of the server and of the desktop app.
2. With no newer release, it says "up to date". With the release source unreachable, it says so and changes nothing.
3. With a newer release published, the console names it. The commissioner clicks Update, sees a confirmation
   naming the version and the expected outage, and confirms once.
4. The app shows progress, loses the server, reconnects by itself, and shows the new version.
5. League data is intact afterwards: the last ledger entry is still the last, and a fresh backup completes.
6. An owner or co-commissioner signed in to the same league sees no Update button and cannot trigger one.
7. A tampered, expired or older release is refused and nothing stops.
8. A failed install puts the old data back, never reports success, and tells the commissioner what to do next.
9. Releases 0.11.1 and 0.11.2 are each installed with the button alone, no console.

## Open before approval
- Panel: design of the privileged helper.
- Where releases are published.
- What else rides in 0.11 besides the button.
