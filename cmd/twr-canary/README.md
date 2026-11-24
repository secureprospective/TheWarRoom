# twr-canary

`twr-canary` checks a repeatable slice of weekly journeys against a disposable **Lab** league. It reads a connection token but never signs in. The `-target lab` flag is mandatory; the league must be `native.*` and listed in `-allow-file`. Non-Lab targets and unlisted leagues exit 2 before contacting the server.

Example:

```sh
twr-canary -target lab -server https://lab.example -token-file canary.token \
  -league native.canary -season 2026 -week 14 -allow-file canary-leagues.txt \
  -seed 202614-ab12 -out report.md
```

The seed format is `YEARWEEK-hexnonce`; reusing it selects the same subset and order. Reports begin with the seed. Exit 0 means no failed checks, 1 means a selected read failed, and 2 means the run could not start or report. Journeys without a safe scripted action or available read are explicitly skipped. This initial runner performs reads only; it does not execute league changes.
