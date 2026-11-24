# twr-canary

`twr-canary` checks a repeatable slice of weekly journeys against a disposable **Lab** league. It reads a connection token but never signs in. The `-target lab` flag is mandatory; the league must be `native.*` and listed in `-allow-file`. Non-Lab targets and unlisted leagues exit 2 before contacting the server.

Example:

```sh
twr-canary -target lab -server https://lab.example -token-file canary.token \
  -league native.canary -season 2026 -week 14 -allow-file canary-leagues.txt \
  -seed 202614-ab12 -out report.md
```

The seed format is `YEARWEEK-hexnonce`; reusing it selects the same subset and order. With no seed, the runner combines the season/week base with a random 16-bit nonce. Reports begin with the seed and include one row per journey; unselected journeys say so explicitly. Exit 0 means no failed checks, 1 means a selected read failed, and 2 means the run could not start or report.

This version is a read-only first pass. Mutating journeys (including setting a lineup and claiming a waiver) remain SKIPPED: the preview-then-execute path and real-hosted-handler fixture test are not implemented yet. Do not schedule or deploy this as the finished T708 canary. Waiver-claim preview rule for the eventual runner: a plain-language rule refusal (for example, a closed window) is PASS; transport errors, 5xx, and malformed replies are FAIL.
