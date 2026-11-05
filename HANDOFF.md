# Handoff

## Baton
Bee → ClaudeBox/operator, 2026-11-05, `session/market-rework`. Slice 2 implemented; review needed.

## Where it stands
Native Market age/STRING/value reuse Offers' directory/value join. Minimum shares native bid admission's settings table and experience policy. Native PTS scores only pool players represented in held stats, retains one bounded league/season/finished-week/revision table, and uses the existing weekly scorer. Unknown facts stay absent. Header preparing resolved normally (249.8 Best-21, 3/32); relevant producer/read paths unchanged from 23eb29e7.

## Evidence
Plain make build, tsc, golangci-lint (0 issues), 93 targeted Go race cases and 65 Vitest tests / four files PASS. Native weekly-sum, absent evidence, cache invalidation and offseason tests added. Native/MFL 1920×1080 screenshots inspected. Copied profile deleted after stopping only this run's process.
Warm full pool reads 107–159 ms; 869 rows, 616 PTS totals. Cold 19–25 s. Windowless reader RSS 68,516 → 683,956 KiB, peak 782,608, ending 681,760; includes held-stat decoding and other reader work, not only totals.

## Next / blocked
Review size and cold acquisition: +204 production lines this slice, +377 across slices against 19358a9f; requested +173 ceiling NOT met. No unrelated deletion to hide growth. make lint is blocked by inherited bloat metrics, worsened by recursively counted existing worktrees; standalone golangci and ifaceguard pass. Missing bundle remote prevents pull/push. Hosted UI, live settings edit invalidation, baseline-binary click comparison not verified.

## Debt / rejected
Native READY plans vs placed bids and private targets remain separate owners (unchanged). No inferred status or invented age/points; no MFL minimum applied to native rows. The first cache key reused franchise-score trial-ledger checks and took 197–303 ms warm; replaced with the durable assets reader's database/rules/facts key, without another scorer. Full evidence: docs/audit/MARKET-TRACE-2026-11-05.md.
