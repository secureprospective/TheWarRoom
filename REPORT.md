Done lines:
1. Met: real hosted handler accepts amountCents=125000000 and returns salaryCents=125000000; legacy amount key is refused with HTTP 400.
2. Met: panel converts typed dollars to rounded cents; test covers $1,250,000 -> 125000000.
3. Met: Go race command, frontend tsc/Vitest, and make lint all pass. Lint required updating the line-budget baseline for regression coverage.
Generator checks passed for native and browser surfaces.
