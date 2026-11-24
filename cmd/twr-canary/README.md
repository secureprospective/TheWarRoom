# twr-canary

`twr-canary` walks a deterministic slice of weekly owner journeys against a disposable **Lab** league using a pre-minted connection token. It never signs in.

## Safety gates

- `-target lab` is mandatory; any other target exits `2` before reading the token or contacting the server.
- `-league` must start with `native.` and appear in `-allow-file`. MFL league IDs (such as `mfl.14432`) and unlisted leagues exit `2` with zero server requests.
- Writes (`commish.lineup.set`) execute only when `-target lab`, `native.*`, and allowlist checks all pass, and only after a clean preview in the same run. Because `commish.lineup.set` is G2, executing with an agent token also requires `-confirmation CODE` from `/v1/native/confirm`; without a human confirmation code, journey 3 verifies the clean preview and stops before execute.
- Commissioner actions never execute.

## Usage

```sh
twr-canary -target lab -server https://lab.example -token-file canary.token \
  -league native.canary -franchise franchise.01 -season 2026 -week 1 \
  -allow-file canary-leagues.txt -seed 202601-ab12 -out report.md
```

## Seed, subset and journeys

The seed format is `YEARWEEK-hexnonce`. When `-seed` is omitted, the runner pairs `season*100+week` with a random 16-bit nonce and prints `seed=<n>-<nonce>` as the first line of output and of the report. Passing the same `-seed` replays the exact same subset and execution order.

- **Always-run set (7):** 1 connect, 2 week and lineup, 3 set lineup (reads `lineup.read`, previews `commish.lineup.set` with the same starters, then executes if all gates pass), 4 live score, 8 waiver claim (preview only), 13 standings (checked against `results.read`), 20 rule read-back.
- **Random slice (5):** drawn without replacement from the remaining eligible journeys. Journeys without a command or requiring a second token are listed as `SKIPPED` with their reason, never silently omitted.
- **Waiver claim preview rule (journey 8):** preview only, never executed. A plain-word rule refusal (such as no open waiver window or a closed window) counts as `PASS`; transport errors, 5xx responses, unauthorized/forbidden errors, and malformed answers count as `FAIL`.

## Exit codes

- `0`: all selected checks passed (skips do not fail a run).
- `1`: one or more selected checks failed.
- `2`: invalid flags, safety refusal, unreadable files, or unable to write the report.
