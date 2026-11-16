# Handoff

- Baton: Christopher / ClaudeBox, 2026-11-16 15:43 UTC. Branch `session/pulse-a010s`, PR #296; no merge.
- Round 7: detached run 2 started on CT 258 at 15:32:48 UTC; schedule PID 3581091, sampler PID 3581090. Expected earliest finish 18:38:48 UTC plus recovery/check time. Lab OUTDIR basename `stress-0.10.0-run2`; STATE unchanged from the round-7 brief.
- Same 32 owners / 80 devices; value.read skipped (explicit skip line verified). No server, limits, tunnel, load or script changes. Script hashes match repo. Existing device loop records failed reads and continues; the previous external stop-on-read-error watcher was NOT launched.
- At 15:43:02 UTC (614 seconds): phase steady; sample.csv 62 lines, latency.jsonl 3549, confirmed.jsonl 107, events.jsonl 569. Schedule, load process and sampler alive. Local private evidence: `.scratch/run2-ten-minute-evidence.txt` (all 15 read names, counts 200 / 502 / other). Other statuses were five 409s across corrections, moves, roster check, results and standings; recorded, not a stop.
- At 15:43 UTC: twr-status all OK, zero PROBLEM; league and sign-in backups current at 15:40. CT 258 memory.events oom/oom_kill/max all zero; CT disk 26%, lab disk 26%, lab available memory ~7.4 GB. Reads still include 502s: gate 2 remains FAILED per ClaudeBox ruling, independent of durability evidence.
- First attempt (lab output basename `stress-0.10.0`) untouched. No guard refusal. CT 257/110/256 untouched. Run 2 left running detached.
- Next: let the schedule complete and collect recovery, exactly-once, integrity, backup and memory evidence. Stop only for schedule exit, failure to return within 10 minutes after a scripted stop, or lab host disk/OOM trouble. No further read-error stop watcher. No claim yet on final gates.
- Earlier tool work: stresscheck and skip-read support committed/published; prior handoff in git history. Existing untracked `deploy/lab/__pycache__/` left untouched.
