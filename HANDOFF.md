Baton: Bee, 2026-11-24
State: Existing seven Go cents changes retained; frontend now sends amountCents and test asserts $1,250,000 => 125000000. Updated hosted handler test for cents and legacy key rejection (expected 400; verify actual route behavior).
Next: Run required Go race tests, frontend tsc + focused vitest, and make lint VERSION=dev COMMIT=b1 BUILD_DATE=b1. Fix failures. Then nativegen -check both surfaces, inspect git diff, commit once, write REPORT.md (<15 lines), tidy-check.
Blocked: none known. No commit yet.
