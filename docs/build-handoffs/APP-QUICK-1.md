# APP-QUICK-1 — partial

- Odds skips simulated all-play tallies only when all-play is absent from the tie-break criteria. Fixed-seed 32-team golden counts were recorded before the change and match after it. The 2,000 runs and random streams are unchanged.
- Synthetic 32-team, 14-open-week fixture, Ryzen 9 6900HX, Go 1.26.10, `-benchtime=3x`: 524.7 ms before, 203.9 ms after. This is not the requested 32-team application fixture or an end-to-end `odds.read` measurement; the <150 ms target is not met.
- Commissioner targeted `go test -list` then `go test -race -p 2`: 613 PASS. Frontend build (including tsc) and `make build` PASS. No frontend changes, vitest or full suite.
- Home clipping remains unresolved. Home's work and inspector are grid siblings, and the Home shell already uses `minmax(0,1fr)` for its work track. Outer shell rules also match nested `.insp` in expanded/closed states; determine whether those rules or intrinsic child sizing explain the reported busy-state clipping with a real browser layout reproduction before changing CSS. No desktop launched.
- Next: reproduce clipping in a headless browser with the real busy Home content and inspector states; verify visually with Christopher. Benchmark the actual fixture and optimize further, including all-play criteria. No remote operations.
