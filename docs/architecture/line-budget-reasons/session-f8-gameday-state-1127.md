Argon finding #8: keep scripts/load/gameday.py state file when any seat was claimed and the run fails or exits non-cleanly.
Regression tests cover partial onboard CoverageError, mid-onboard exception, load failure, zero-claim cleanup, and full PASS deletion.
