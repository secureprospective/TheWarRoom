# Throwaway hosted league for desktop click tests

Development only. Neither this command nor its fixture package is linked into desktop/server
release binaries or included in installer bundles. Linux only.

## Build on the machine you are testing

Run from the repository root. Requires the normal desktop build dependencies (Go, WebKitGTK 4.1), OpenSSL, and built frontend
assets. On a fresh worktree: `cd frontend && pnpm install && pnpm build`, then return to the root.

```sh
go run ./cmd/twr-practice up .scratch/practice
# .scratch must already exist; any existing practice directory must be empty.
go run ./cmd/twr-practice code .scratch/practice
go run ./cmd/twr-practice down .scratch/practice
```

## Build once, copy to a test machine without Go

On a Linux build machine with Go, pnpm and the normal WebKitGTK 4.1 development
libraries, run this one-liner from the repository root:

```sh
(cd frontend && pnpm install && pnpm build) && mkdir -p .scratch/practice-bundle && go build -o .scratch/practice-bundle/twr-practice ./cmd/twr-practice && go build -tags practice -o .scratch/practice-bundle/twr-server ./cmd/twr-server && go build -tags desktop,production,webkit2_41 -o .scratch/practice-bundle/thewarroom . && cp docs/ui/endpoint-registry.csv .scratch/practice-bundle/registry.csv
```

The desktop embeds the freshly built frontend assets. Use the same Linux architecture
and compatible system libraries on both machines; this is not a cross-build or a static
bundle. Copy the **whole directory**, including `registry.csv`, preserving executable bits
(replace `tester@test-machine` with your SSH destination):

```sh
scp -r .scratch/practice-bundle tester@test-machine:~/practice-bundle
```

On the test machine, with OpenSSL and the desktop runtime libraries already installed:

```sh
mkdir -p "$HOME/practice-sandboxes"
"$HOME/practice-bundle/twr-practice" up -bin "$HOME/practice-bundle" "$HOME/practice-sandboxes/click-test"
# Launch the exact Desktop command printed by up, in the graphical session.
"$HOME/practice-bundle/twr-practice" code "$HOME/practice-sandboxes/click-test"
# Close the desktop window first.
"$HOME/practice-bundle/twr-practice" down "$HOME/practice-sandboxes/click-test"
```

No checkout, Go, pkg-config, development headers, pnpm or frontend files are needed on the
test machine. `-bin` copies `twr-server` and `thewarroom` into the sandbox, refusing missing,
non-regular or non-executable files, and copies the bundle's preview registry. OpenSSL still
creates fresh disposable TLS keys. Without `-bin`, `up` keeps its repository-based build
behavior. `code` and `down` never build anything. Keep the launcher in place until cleanup;
the printed code/stop commands use its absolute path when `-bin` is supplied.

`up` prints the random loopback HTTPS address, commissioner username, random password,
authenticator secret, current-code command, and an exact desktop launch command. Copy that
command as printed. It inherits `DISPLAY`, including an Xvfb display, and uses its own HOME,
XDG configuration/data/cache/runtime folders and CA. Its deliberately unavailable session bus
prevents reading/writing the real OS keychain; sign-in stays in memory for this window only.
No real app profile or league archive is opened. Do not remove the printed environment settings.
Close the desktop window before `down`.

Sign in as the printed commissioner, use **Authenticate** with the code from `code`, then
click **Open my franchise**. The commissioner owns Practice team 1. Logins **owner** and **partner** own Practice
teams 2 and 3; the remaining franchises are vacant. All three use the printed disposable
password. Ordinary owners need no authenticator step-up. The league has disposable football
evidence, four roster players, a free agent, a published one-week schedule, and a 2027
rookie draft with one available rookie. No MFL data is imported. The confirmation uses the preceding time
step, so the current code is immediately usable. If a code has already been used, wait for the
next 30-second authenticator step before using `code` again.

TLS and authenticator keys are disposable. TLS expires after one day: stop and recreate the
sandbox then. Keep printed credentials and the sandbox directory private; they are not real
accounts. `down` checks the recorded PID's executable before signaling, waits for shutdown, and
then deletes the directory. Missing/invalid ownership records or a PID belonging to a different
executable fail closed and leave the directory untouched. An already stopped server can still
be cleaned up. A startup failure stops its child and deletes only the claimed empty directory.
Never point these commands at a directory you want to keep.

## Implementation and proof

Flow: claim an empty owner-only directory → copy preview registry/build binaries/generate
loopback TLS (or copy bundle binaries/registry and generate TLS) → initialize one database through trusted account/history APIs → create canonical
league/owner seat/enroll TOTP → launch the real `twr-server hosted` → record PID and credentials.
No network bootstrap or authorization override is added. The shared league creation helper in
`internal/hostedfixture` is also used by the hosted transport tests. The hosted server retains
its normal admission and clock-health checks; this tool does not disable them.

`go test -race ./cmd/twr-practice ./internal/serverhttp` exercises the real up → HTTPS password
sign-in → TOTP step-up → enter league → down lifecycle, verifies nonempty-directory refusal
preserves the running league, and rejects unowned directories, symlinks and unrelated PIDs.
An RFC 6238 vector independently checks the disposable code generator. The lifecycle test builds
the desktop but does not launch a GUI; visual acceptance still requires the printed command.

`TestPrebuiltSandboxHTTPSLifecycleWithoutGo` builds the bundle first, then changes to an
unrelated empty working directory and restricts `PATH` to OpenSSL alone. It exercises real
`up -bin` → HTTPS sign-in → TOTP step-up → enter league → `code` → `down`, including the
nonempty-directory refusal and server shutdown. Go cannot be found during any of these
actions. `TestPrebuiltSandboxRefusesInvalidBinaries` checks both binary names for missing,
non-executable and directory inputs, and verifies cleanup leaves the input bundle untouched.
The desktop is copied but not launched: graphical click acceptance remains a separate test.

## Off-site backup boundary

The launcher builds its server with `-tags practice`, an explicit disposable-test
build, not the deployable binary. This variant refuses R2 configuration, validates
the private sandbox folders and colocated runtime files, and logs that it has no
off-site protection. The release builder never uses this tag. A normal server
binary remains paused without verified R2 coverage; there is no runtime switch
to disable that rule. Prebuilt practice bundles must use the tagged build shown
above. Never use the practice binary for real league state.

### Update-required screen

`up` also prints an **Update-required click test** command. With an unstamped development
binary, this sets `TWR_PRACTICE_UPDATE=app` in the isolated practice profile and opens the real
blocking desktop screen. It offers one button, **Update and reopen**. Without installer trust,
clicking it must explain that update setup is missing, preserve the screen, and permit retry;
no test signing keys or unsigned executable are installed. Release-stamped builds ignore the
practice override. For a valid verified-executable launch use the generated-key staging test
in `internal/releasetrust`; actual signed release/three-OS click drills require the install
handoff in `docs/delivery/Release_Trust.md`.

## Lineup, reserves and draft walkthrough

After opening your franchise, use **Refresh league server** if needed. The location label is
**This league is managed here**; a failed refresh keeps the last reading read-only.

1. **Set lineup**: select Practice quarterback, check the lineup, then hold to confirm.
   Refresh and verify he is a starter. Kickoffs are one day after sandbox creation.
2. **Roster and reserves**: select Injured prospect, choose Injured reserve, check and
   confirm. Refresh shows the new squad. A healthy quarterback must be refused for IR;
   Young receiver is eligible for taxi. Choose Active roster to bring a reserve back.
3. **Draft room**: Practice team 1 is on the clock two seconds after provisioning.
   Select Practice rookie, review, then hold to confirm. Refresh shows the pick and roster.
   Commissioner controls can pause/resume or skip with a recorded reason.
4. **League rules** shows the rules in force and their existing governed explanations.
5. Disconnect the server, refresh: the saved view stays visible with all mutation controls
   disabled. Reconnect and refresh: only reads are sent, never queued changes. A failed
   confirmation warns that it may have reached the server; inspect the refreshed state
   before choosing another action.

Automated workflow coverage: `internal/serverhttp/workflow_test.go` exercises real HTTPS
and desktop sessions, eligibility refusal with unchanged roster, idempotent retry, lineup,
rookie selection and revocation. `frontend/src/app/commissioner/hostedRoom.test.tsx` consumes
real Go response fixtures and checks stale mode, reconnect, authority fencing and no replay.
Regenerate its wire fixture with `TWR_UPDATE_HOSTED_WIRE=1 go test ./internal/serverhttp
-run TestHostedWorkflowFrontendReadings -count=1` (keep the command on one shell line).

## Bids, waivers, trades, results and next-season walkthrough

Use separate disposable windows or sign out and back in for each named login. Launch only
with the printed sandbox command; never reuse a real profile. Refresh each window after
another person records a change. Every action is checked, then held to confirm; a reason
explains commissioner rulings. No action is sent to MFL.

1. **Free-agent bids**, as `owner`: choose Place a private bid, Practice free agent,
   annual salary 5000000, one year, and a reason. Check, inspect and confirm. Refresh:
   only this owner sees the private bid. Try Raise your bid and Withdraw your bid too.
   As `commissioner`, refresh: neither the rival bid nor its existence/closing time is
   visible. Commissioner processing/correction choices cannot peek either. Place another
   bid and leave it standing: practice bids close after **two minutes**, then the real
   server worker awards them. Refresh to see the public award and the winning roster.
2. **Waivers**, as `owner`: release Other team receiver. As `commissioner`, refresh,
   choose Claim a player, select that receiver, check and confirm. The claim is private;
   the roster does not change yet. The practice window closes after **two minutes**.
   Refresh for the worker's award. The releasing owner cannot reclaim his own player.
   Commissioner actions can set priority, process a closed window or correct an award;
   their choices come from the server, never a pasted transaction identifier.
3. **Trades**, as `owner`: choose Practice team 3, check the 2027 first-round pick
   originally belonging to Practice team 2, then Check trade offer and confirm.
   As `partner`, refresh, choose Accept a trade, select the offer, give a reason and confirm.
   As `commissioner`, refresh, choose Review a trade and Approve (or Veto), give a reason,
   inspect the preview, then confirm. Refresh all windows: the pick changes owner only
   after approval. A veto can be appealed and ruled on; decided awards can be overridden
   with a recorded reason. Also try withdrawing an unaccepted offer and declining a new
   offer. The commissioner cannot review a trade in which they are a party.
4. **Results and standings**: the published games explain missing results rather than
   inventing NFL statistics or missing lineups. As `commissioner`, choose Adjust a franchise
   score (season 2026, week 1, points 1.5), or Set a franchise record (season 2026, Practice
   team 1, one win, zero losses/ties). Check and confirm. Refresh: computed standings remain
   alongside Commissioner standings and the ruling's explanation. Clear standings
   corrections to return to the computed record. This sandbox deliberately has no final NFL
   statistics, so an incomplete game must remain incomplete even after a score adjustment.
5. **Next season**, last: as `commissioner`, choose Move to the next season, enter 2027
   and a reason. Check the full plan, including contracts, rules, picks and cap, before
   holding to confirm. The plan names the awarded free agent and gives its dates in league
   time (US Central). Refresh, as every login, shows roster season 2027. The old NFL
   evidence cannot be used to submit a new-season lineup or bid: the operator must
   explicitly select new season evidence. Other readings remain accessible; lineup states
   its evidence problem and bids say they wait for new-season evidence.
6. Disconnect, refresh and attempt each task: saved readings remain read-only. Reconnect
   and refresh: only reads run, never a replayed bid/claim/trade/correction/rollover. If a
   confirmation is interrupted, inspect the refreshed league before choosing another act.

Automated pass evidence: `internal/serverhttp/market_workflow_test.go` drives real HTTPS
owner/commissioner sessions through bids, releases/claims, offers/acceptance/review,
corrections and rollover. Every execute is repeated with the identical request and must
return the same receipt. The privacy test compares whole commissioner response bodies
and recipient revisions before/after another owner's blind bid. Standings from a fresh
store and assets from ledger replay must agree with hosted readings. Old-season bid
evidence is refused without appending. `rollover_workflow_test.go` lets the worker award a
real bid, rolls the season over, then refreshes as commissioner and owner. Frontend `hostedRoom.test.tsx` and
`hostedWorkflows.test.tsx` check all five tasks, named choices, single confirmation,
read-only stale/reconnect behavior and explicitly unavailable new-season lineups.

These tests do not replace a graphical click acceptance on the final desktop binary.

## Block E acceptance walk: MFL reference versus your league server

Christopher runs this on the final practice build. Automated checks do not mark this live
walk passed. Use the isolated launcher above; no real MFL login, submission or roster move.

1. Start a fresh practice league, launch the printed Desktop command, sign in as
   `commissioner`, authenticate and **Open my franchise**. Expect **Your league server**,
   the explanation that MFL refresh cannot replace recorded changes, and
   **Refresh league server**. No MFL key or local reference screen should open.
2. Open **Roster and reserves**. Move **Injured prospect** to **Injured reserve**,
   check the preview and hold to confirm. Refresh: the server still records that squad.
   Open **Set lineup**, **Draft room** and **League rules**; each is a server reading,
   not a roster imported from MFL.
3. Open **Free-agent bids**, **Waivers**, **Trades**, **Results and standings** and
   **Next season**. Follow the task walks above for disposable changes (next season last).
   Confirmations must say they change your league server, not MFL or a local plan.
   Commissioner rulings remain under commissioner control.
4. To simulate a disconnect without deleting the league, use your process monitor to find
   the exact `twr-server` executable inside this sandbox directory, verify its path, and
   pause that process (not any real league server). Click **Refresh league server** and
   wait for the request to time out. Expect **Stale — read-only saved view**, the last-read
   time, preserved roster and disabled change controls. No MFL/local fallback may appear.
5. Resume that same sandbox process and refresh. Expect the recorded reserve move and
   other confirmed changes to remain. No queued bid, claim, trade, lineup or ruling should
   run. If a confirmation was interrupted, inspect the fresh reading before another action.
6. In a separate fixture-only browser preview (`cd frontend && pnpm dev`, then open its
   printed address), expect **From MFL (reference only)**. In Control Room → League,
   expand the MFL reference details; in Data and sources inspect the refresh explanation;
   on Trade Floor inspect offers history. Expect references, not server offers. The browser
   cannot perform desktop MFL refresh. Return to the practice desktop and refresh: its
   recorded state remains unchanged. Close the preview and desktop, then run the printed
   sandbox stop command.

The fixture preview is a copy/label check, not a live MFL refresh test. The actual
reference-replacement boundary is covered with fixtures by
`TestNativeSeedOnceAndRetryAfterMirrorRefresh` and
`TestNativeCapReadingUsesRecordedAssetsAndSignedDeadMoney`: replacing the MFL mirror changes
its reference values without changing native assets/cap, and seeding again is refused.
No MFL write endpoint is added or exercised. The startup tests also reject failed hosted
startup responses without mounting the local shell or its refresh hooks.
