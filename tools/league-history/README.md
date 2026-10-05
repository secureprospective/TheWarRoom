# Legacy NFL league history

Independent read-only HTML PWA over the local MFL archive. It does not change the
WarRoom desktop application, its databases, or the archive. No runtime dependencies,
no external fonts, no MFL calls.

## Open

On the Beelink: **http://localhost:8765/**.

To build or restart, from the repository root:

```sh
python3 tools/league-history/build_archive.py
python3 tools/league-history/serve.py
```

The server binds only to localhost. Installation/offline caching requires localhost
or HTTPS. Chromium-family browsers offer an Install button when eligible; their
browser menu can also install the app. Wait for **Offline ready** before going offline.
Other devices need HTTPS hosting of the same allowlisted assets, or a local tunnel.
No HTTPS host or reboot-autostart service has been configured.

## Explore

- **Calendar:** 2013–2026 against January–December. Click a cell for a month, then a
  day for its ledger. Click a month heading to compare that month across every year.
- Filter by franchise, player, UTC date range and activity. Switch between deals
  involving picks and individual picks moved. Export the complete filtered ledger
  as CSV, not just the visible page.
- **League:** all fourteen seasons, recorded Super Bowl winners, activity summaries,
  MFL standings and recorded drafts. Click a season to change the detail.
- **Teams:** results through time, historical names, reported roster observations,
  completed trades and other movements.
- **Players:** search by name or MFL ID; open season scores, draft and movement
  history, and reported rosters. Player links and franchise links join the views.
- **Archive:** search all 1,705 source files by year, type or status and read the
  counting/coverage methodology.

### Known limits

2026 is partial. Calendar dates and source seasons are deliberately different.
Early draft results can be missing even when the endpoint returned usable JSON.
126 trades have comment-only pick references, not trustworthy encoded quantities.
These are separately discoverable and excluded from the 1,725 encoded-pick deals
and 3,780 recorded individual pick transfers. Proposals never count as completed
trades. Standings are ordered by wins then points, not the complete MFL tiebreaker.
Roster observations do not prove continuous ownership or owner identity.

The app indexes trades, movement records, draft selections, standings, championships,
YTD scores and general/weekly rosters. Other endpoints are inventoried, not fully
interpreted. Source evidence points to original local filenames and array positions;
indexed records are normalized, not original JSON.

## Data and privacy

`data/archive.json` and `data/revision.json` are generated, ignored by git and necessary
for a working deployment. Building takes the latest manifest entry for each file.
Owner contacts, credentials, messages and arbitrary transaction comments are omitted;
MFL draft pick notes are retained. The allowlist server denies raw archive access,
source scripts, directory listing and traversal paths. Do not substitute a server
that exposes the entire repository. This is private league data: do not publish
its generated index without Christopher's approval.

Service worker scope is this app only. It caches the full index for offline use,
refreshes data using its generated content revision, and falls back to cached responses
when the server is unreachable. For shell-breaking changes bump the cache version in
`sw.js`. To clear local data, clear site storage for this origin in browser settings.

## Verify

```sh
cd tools/league-history
python3 -m unittest -v
node --test model.test.mjs
```

Optional browser test, with an existing Playwright Core installation and a running
local server (no browser-test dependency is installed by this project):

```sh
PLAYWRIGHT_CORE=/path/to/playwright-core BROWSER_PATH=/path/to/chromium \
  node browser.test.mjs
```

It checks the 168 heatmap cells, future periods, month/day counts, entities, activity
filters, histories, export, source denial, 1440/390px page overflow and offline reload.
Screenshots and test CSV live in ignored `.impeccable/review/`. Repository gates remain
`make lint` and `make test` with the toolchain specified in `CLAUDE.md`.

**Christopher's acceptance test:** open July 2025 on the default calendar. It should
show **73 completed trades involving encoded picks** and reveal their dated ledger.
Switch to individual picks moved; the same deals now count each transferred pick.
Select Arizona, then open Teams or a linked player. After “Offline ready”, stop the
server and reload: all five views should still work.

## Extension points

`build_archive.py` owns MFL parsing/provenance; `model.mjs` owns pure filtering and
aggregation; `app.js` owns view rendering and interaction; `style.css` owns the UI;
`sw.js` owns offline caching; `serve.py` owns the local allowlist boundary. Keep raw
archive reads in the compiler, not the browser, and keep the generated data out of git.

Icons are original authored calendar-grid artwork: `icon.svg`; PNG variants were
rendered locally with Pillow from the same coordinate/color design. No third-party
image or font assets are shipped.
