# Live week 9 replay

`live-week9.tar.gz` losslessly packages the lane L4 fixture from the pulse1114/week9 run,
including its original manifest and all raw-final/synthetic JSON bodies. Raw finals are unmodified;
synthetic frames are proportional approximations, not historical observations. The manifest records
sources, acquisition/simulated instants and mutations. These public packets contain no league credentials.

Collector tests replay all raw finals. Native-overlay tests exercise the first game's progressive,
correction and missing-column frames and compare the recorded quarterback's native score with the
same raw stat line under the same rules. A separate results test scores two two-player lineups
(QB/WR) from an unmodified raw final and matches their totals to native scoring of those same
lines/positions/rules. This is not MFL lineup parity or a certification of official finals.
