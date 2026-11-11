# RESUME: TheWarRoom landing page (warroom.secureprospective.com), 2026-11-10 (third compaction, ~11:35 CST)

Written by Tom (Claude Code on the Beelink). On "we are back": read this, check the in-flight job (section 2), start at NEXT ACTIONS item 1. Do not recap or re-plan.

## 1. What we are doing
The public landing page is finished and Christopher approved it ("absolutely fantastic"). His order: "push/merge - spin up a subagent,
Opus 5.5 low, to build the subdomain landing page to go live so I can start sharing it with real people."
Now: (a) get PR 224 merged, (b) make the TheWarRoom hosted server serve the page at `https://warroom.secureprospective.com/`, (c) deploy to the live lab server.

- Repo `~/work/TheWarRoom`, worktree `~/work/TheWarRoom/.worktrees/website`, branch `session/website`, pushed. PR https://github.com/secureprospective/TheWarRoom/pull/224 (base main).
- Never touch the main checkout (`session/league-history-pwa`) or branch `session/ui-to-mocks`. Squash merges only. Never `--no-verify`.
- Viewing copy for Christopher: `file:///home/chris/Downloads/twr-landing/index.html` (matches the committed `website/`).

## 2. In flight
- Background shell job `bxs2154za` (started ~11:13 CST, 10 min timeout): `gh pr checks 224 --watch` then `gh pr merge 224 --squash`. It has probably timed out.
  Check: `gh pr view 224 --json state -q .state`. If OPEN, merge by hand once checks are green (next action 1).
- CI run 42862521418 on the first push: job "checks" FAILED. A later push (the commit carrying this file) starts a fresh CI run.
- No subagent has been spawned yet. No deploy has been started.

## 3. Status
| Item | State |
|---|---|
| Landing page (hero reel, trade versions A/B, league switches, 3D angled panels, phone pass) | Built; rendered at 1920x1080, 1366x768, 360x740, 390x844, 412x915; no console errors; commit 1013e2863 |
| Reel pauses off-screen, restarts on return | Proven by script (`CHECK=1`): "reel frozen away: true, moving after return: true" |
| Reduced motion | Rendered: static deck in finished state |
| Flights between scenes after this pass (shatter, weave) | NOT re-filmed |
| Sound | Never heard by anyone |
| PR 224 | Open; CI "checks" job failed on an unrelated frontend test (section 5) |
| Server serves the page | Not built |
| Buttons email Christopher | Not built |
| Live deploy | Not started |

## 4. Artifacts
- `website/`: `index.html` (328 lines), `site.css` (338), `site.js` (570), `steel.js`, `fonts/` (5 woff2 + OFL.txt). Committed in 1013e2863.
- Share-preview image, 1200x630, hero with deck: `/tmp/twr-site/out3/og-reel-04300.png` (171,752 B). Not in the repo yet. Re-make: `VP=1200x630 T=4300 /tmp/twr-site/go.sh og`.
- Render harness (Beelink `/tmp/twr-site/`, may vanish on reboot): `go.sh <tag>` with env `VP=WxH T=ms,ms STOPS=0,1.. CLICKS=1 CHECK=1 STILL=1` copies the site by scp to
  `claudeos:work/twr-site/`, runs `claudeos:work/site-study/reel.mjs` (Playwright, chromium), pulls shots to `/tmp/twr-site/out3/`. `sheet.sh <tag> <cols> <resize%> <per-sheet>` makes contact sheets.
  A hook forbids `rsync` of the work tree: use `scp`.
- CI failure log: `/tmp/twr-site/ci-checks-224.log`.

How the page works (new this pass): hero scene 0 holds three `.ly.rc` cards (classes p0/p1/p2 set position and depth; `place()`, `reel()`, `drag()`, `moves`, `rewind`, `reelSet()` in `site.js`).
`.ly` supports `--ry` (rotateY, desktop only), `--pz` (phone depth), `--ps` (phone shout size). Trade desk: `offers[]`, `setOffer()`, `#ver`. League switches toggle `no-yrs/no-cap/no-def` on `#league`.
Phone-only removals: franchise names (numbered tiles instead), `.more` paragraph, "Meanwhile" plate, end wordmark.

## 5. Current bug
PR 224 CI job "checks", step "Lint (with placement and root ratchet), bindings, frontend checks":
`FAIL src/app/shell/leaguePicker.interaction.test.tsx > opens the Control Room on the locked mock for a picked app league` / `AssertionError: expected null not to be null` (1 failed, 754 passed).
Hypothesis: flaky or already failing on main; PR 224 touches only `website/` and `docs/`, no frontend code. Caveat: not checked against main's latest run; confirm before blaming flakiness.

## 6. Already tried / settled, do not repeat
- Hero junk windows: removed from the hero on purpose (exhibits keep them).
- Soft dark box behind centred copy looked smudgy on the lit field: centred copy uses text-shadow instead; left-aligned copy keeps a padded backing.
- Scene on/off state must be decided before the "not near" early exit in `frame()`; otherwise a fast jump leaves a far scene on (fixed).
- Playwright clicks move the mouse and shift the camera by parallax in later shots: an artefact, not a phone bug.
- Design hook "cramped padding" (2 flags, attribution unknown): padded the copy backing and the roster chips; no ignores added.
- Earlier list still stands: thin line-only field, far fade to 4200, DOM yard signs, hazard underline, kickers above headings, outline type with transparent fill, Sleeper under "ads".

## 7. Christopher's decisions
- All earlier rulings stand (eight rulings; punchy copy; names rolled slightly; never ads or betting odds; competitor claims confirmed true).
- 2026-11-10 ~11:10 CST, by pick-list:
  - **Go-live:** "Same address, server serves it". The TheWarRoom server shows the landing page at warroom.secureprospective.com; join links and the app keep working. Full test gate first.
  - **Buttons:** "Email you". "Open your league" and "Bring your league over" open the visitor's email app with a pre-filled message. Address not named by him;
    memory says his business address is christopher@secureprospective.com. Tell him which address is on the page before it goes live.
- He asked for the builder to be an Opus 5.5 low-effort subagent, built to my spec, corrected by me.

## 8. Ledger state
- `session/website`: 93e6de9fa, 7f8860ea2, 1013e2863, then the commit carrying this file. Pushed. Nothing uncommitted.
- Hermes task T600 rewritten to this state.

## 9. Next actions, in order
1. Get PR 224 green and squash-merged: `gh pr checks 224`; if the same frontend test fails again, check whether it fails on main (`gh run list --branch main`), then `gh run rerun <id> --failed`
   or report it. Merge with `~/.local/bin/gh pr merge 224 --squash`.
2. Create a new worktree from merged main, never touching the main checkout: `git -C ~/work/TheWarRoom fetch origin && git -C ~/work/TheWarRoom worktree add .worktrees/landing-serve -b session/landing-serve origin/main`.
3. In it, do the page edits myself: both CTA buttons in scene 10 become `mailto:` links with subject and body; add Open Graph and Twitter meta tags; add the share image as `og.png`.
4. Spawn the builder (Agent tool, model opus, effort low) in that worktree with this spec:
   - Move `website/` into a Go package that embeds it (go:embed cannot reach outside its package), e.g. `internal/serverhttp/landing/` or files beside `join_page.go`; one copy of the site, no duplicate.
   - In `internal/serverhttp/hosted.go`: serve `GET`/`HEAD` `/` (index), `/site.css`, `/site.js`, `/steel.js`, `/og.png`, `/fonts/*.woff2` with correct content types.
     These landing paths must be answered BEFORE the `r.URL.RawQuery != ""` refusal in `serveHTTP` (shared links get `?fbclid=` and `?utm_` added; today that returns 400), and must be in `browserRoute` (browser.go) so a same-origin Origin header is admitted.
   - CSP for the landing page: `default-src 'none'; script-src 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; base-uri 'none'; frame-ancestors 'none'; form-action 'none'`
     (the page uses external script files and inline style attributes, so the join page's nonce CSP does not fit). Keep `X-Content-Type-Options: nosniff`.
   - Caching: index `no-store`; css/js short max-age; fonts and og.png long max-age. Everything else on the server keeps `no-store`.
   - Must pass through `versionGate`, `resourceLimits`, `backupAdmission` the way `/join` does (see `join_page_test.go` for the fixture `newHostedFixture`, `f.get`).
   - Tests (each must fail without the change): `/` returns 200 HTML with the CSP; `/?fbclid=x` returns 200; each asset returns its content type; POST `/` is refused; `/join` and its strict CSP unchanged; an unknown path still 404s.
   - No branching on league or source; no new infrastructure; keep it small. Sol/builder never runs git.
5. Review every line, plant one mutant, run `timeout 600 nice -n 10 make verify VERSION=dev COMMIT=<x> BUILD_DATE=<x>` once at the end (toolchain env in section 10), `~/fleet/bin/heavy-ok.sh` first.
6. Load the built server on Claude-OS or a throwaway lab CT and render the page from it (desktop and phone) before any live step; check `/join` still works.
7. PR, squash-merge on green.
8. Deploy to the live lab server (CT 256 on the lab VM, address warroom.secureprospective.com, holds the phone-test league with claimed franchises): signed bundle via `deploy/lab/lab-bundle.sh`,
   then the installed `/usr/local/libexec/twr-upgrade` per `deploy/server/UPGRADE.md` and `deploy/lab/LAB.md`. Read `docs/RESUME-debloat-1109.md` lines 245-262 first (CT 256 history; tunnel init script quirk).
   Christopher's go-live order covers this deploy; anything beyond it (DNS, tunnel changes) needs his word. Cloudflare token on stdin only.
9. Verify live from outside: `/` shows the page on desktop and phone, `/join` works, a link with `?fbclid=test` works. Then tell Christopher, including the email address on the buttons.

## 10. Environment notes
- Address him as Christopher; short plain replies, outcome first, Central time.
- Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1`.
- ssh aliases: `claudeos`, `claudebox` (CT105, root), `hermes`, `lab`. Lab scripts refuse to run anywhere but host `twr-lab`.
- Server facts found: `/` is a 404 today; `/join` is served by `internal/serverhttp/join_page.go` (embedded HTML, nonce CSP); router is `Hosted.route` in `hosted.go`; all replies get `Cache-Control: no-store`;
  requests with any query string or without TLS are refused in `serveHTTP`.
- Never real league data, NFL logos or real-player photos on the page.

## 11. Honest status
The page is done and pushed; it is not merged and not live. ETA to merge: one green CI run (about 5 minutes) if the failing test is a flake. ETA to live: unknown; it needs the server change,
the full gate, a staged check and a signed upgrade of the live lab server, none of which is started. Unproven: the sound, phone frame rate, flights after the layout pass.
