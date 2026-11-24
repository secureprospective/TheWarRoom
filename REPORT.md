Done lines: 1 NOT MET — both tests are GAP skips; no server guard to mutation-test.
Done line 2 MET — `go test ./cmd/twr-server -run 'MflSeparation|SelfProtect' -count=1` passed (tests skipped).
Done line 3 NOT MET — lint exited 2: `check-line-budget.sh` reports existing docs/go-test budget overruns and requires a branch reason.
GAP: MFL separation — composition `app.contracts().TargetWhatIf` has no server twin.
GAP: Self-protection — composition `runNative` has no server twin.
No production changes. Existing composition tests remain untouched.
