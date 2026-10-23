# Frontend toolchain upgrade — 2026-10-23

## Baseline (origin/main worktree, before dependency changes)

Node 24.22.1; Corepack pnpm 10.34.3 (packageManager pin). Installed with
`corepack pnpm install --frozen-lockfile`; the shell's bare pnpm was 9.15.9.

- Full audit: 32 advisories (3 critical, 10 high, 16 moderate, 3 low).
- Vitest 0.25.8: 125 files / 1,395 tests passed.
- Vite 3.2.11, plugin-react 2.2.0, TypeScript 4.9.5, React/React DOM 18.3.1.
- Other direct tooling: Tailwind 3.4.19, PostCSS 8.5.15, Autoprefixer 10.5.0;
  React types 18.3.31 / React DOM types 18.3.7. Node types overridden to 20.11.30.
- Build succeeded: entry 306,371 bytes (299.19 KiB, gzip 78.04 KiB),
  entry CSS 44,915 bytes; models 242,512 bytes; snapshot 491,595 bytes.
  Entire dist: 1,734,158 bytes.
- Bundle budget passed at 306,371 / 321,818 bytes; lazy MyMoves, rail,
  commissioner proxy and commissioner forms verified.
- CSP check passed: default-src 'self'; script-src 'self'; style-src 'self'
  'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src
  'self'; object-src 'none'; base-uri 'none'; form-action 'none'.
- Existing act warnings occurred. Initial attempts before dependency installation
  failed (missing Vitest / shell-global TypeScript); these are not the baseline.

## Migration references

- https://v4.vite.dev/guide/migration
- https://v5.vite.dev/guide/migration
- https://v6.vite.dev/guide/migration
- https://v7.vite.dev/guide/migration
- https://vite.dev/guide/migration
- https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react/CHANGELOG.md
- https://vitest.dev/guide/migration.html
- https://tailwindcss.com/docs/upgrade-guide

## Result

Full audit: **0 advisories**. Tests: **125 files / 1,396 tests**, up one test:
Vitest 5 rejects a nested `it`; promoting it to file scope preserves every
assertion and makes the photos-off case independently collected. Two pending
Promise mocks now name their resolved types. No tests deleted, skipped or weakened.
No custom test environment, globals, setup-file or mock-reset override was needed.
React 18.3.1, React DOM 18.3.1 and TypeScript 4.9.5 are unchanged.

| Direct tooling | Before | After |
|---|---|---|
| vite | 3.2.11 | 8.3.5 |
| @vitejs/plugin-react | 2.2.0 | 6.1.2 |
| vitest | 0.25.8 | 5.0.5 |
| @types/node | 20.11.30 (transitive override) | 22.20.5 (explicit devDependency) |
| postcss | 8.5.15 | 8.5.31 |
| tailwindcss | 3.4.19 | 4.3.3 |
| @tailwindcss/postcss | absent | 4.3.3 |
| autoprefixer | 10.5.0 | removed |

### Config and security decisions

- Vite 8 uses Oxc/Rolldown and Lightning CSS. Removed a stray CSS semicolon
  rejected by Lightning CSS; retained the existing CSP policy verbatim.
- Preserve dotted JavaScript asset names for lazy-chunk checks. Emit a manifest;
  the budget now sums the entry's complete static JavaScript import graph,
  not just its now-smaller entry file. Lazy imports remain excluded and tested.
- Vite now HTML-escapes injected apostrophes. The CSP checker decodes `&#39;`
  before validating the same policy. It still rejects eval, remote sources and
  inline scripts. In-memory negative probes rejected unsafe-eval and inline script;
  bundle probes rejected excessive size and forbidden commissioner text in a
  shared eager chunk, not merely the entry. Existing budget self-tests pass.
- Updated Node types to meet Vite/Vitest peers; removed the obsolete override.
- PostCSS upgrade fixes GHSA-fxqj-rqcc-2cmp and GHSA-r28c-9q8g-f849 and updates
  nanoid/source-map-js. Tailwind 3 still pulled unpatched braces 3.0.3 through
  chokidar (GHSA-vfj7-8cjw-p6xm) and vulnerable selector-parser
  (GHSA-rj75-hqrm-r3gf). No fixed braces release exists. Upgraded Tailwind rather
  than override chokidar across its incompatible glob-watching API.
- Tailwind 4 uses its own PostCSS plugin and CSS import; explicitly loads the
  existing JS theme with `@config`. Removed obsolete `@tailwind` directives.
  Autoprefixer is redundant because Tailwind 4 handles prefixing; removing it also
  removes the vulnerable browserslist/baseline-browser-mapping chain.
- **No overrides added. No audit suppressions.**

### Bundle bytes

| Output | Before | After | Change |
|---|---:|---:|---:|
| Eager JavaScript graph | 306,371 | 300,304 | -2.0% |
| Physical entry JS (not directly comparable after splitting) | 306,371 | 159,411 | -48.0% |
| Entry CSS | 44,915 | 46,281 | +3.0% |
| models lazy JS | 242,512 | 215,336 | -11.2% |
| snapshot lazy JS | 491,595 | 566,781 | **+15.3%** |
| Entire dist, including new manifest | 1,734,158 | 1,821,697 | **+5.05%** |

New asset total excluding HTML and manifest: 1,771,564 bytes. Snapshot remains
lazy; Vite warns it exceeds 500 kB. Different bundling/minification changes shared
chunk ownership, so individual lazy file comparisons are not identical load graphs.
Other named lazy chunks growing over 5% (before → after bytes):

- ReportDrawer 1397 → 1644; ReceiptStatus 471 → 504; LeagueLedger 4093 → 4439;
  HomePulseMount 1472 → 2026; FranchisePicker 344 → 380; WhatsNew 1198 → 1323.
- Pool 16361 → 17213; WhatIfFigures 586 → 697; ReadingView 3897 → 4319;
  draftEditor 385 → 407; sessionMoves 960 → 1013; TeamPanel 4132 → 4390.
- AppSettings 1212 → 1832; LeagueRoom 2122 → 2696; ReportView 4531 → 4942;
  LeagueOps 3042 → 3288; RuleTray 6480 → 6812; DataSources 5812 → 6250.
- WatchButton 583 → 639; Counterparties 11729 → 12884; Comms 2898 → 3113;
  AlertTray 2903 → 3212; CalendarPanel 2619 → 2868; Now 2955 → 3228;
  SeasonalCard 1755 → 1999.

### Gate evidence

All exit 0 on the final implementation:

- Corepack pnpm 10.34.3 frozen install: lockfile up to date.
- Full pnpm audit: No known vulnerabilities found.
- TypeScript: no diagnostics.
- Frontend tests: 125 files / 1,396 passed.
- Frontend build: succeeds; snapshot-size warning only.
- Bundle budget: 300,304 eager bytes / 321,818; all lazy-boundary checks pass.
- CSP: unchanged policy, check passes.
- make lint: 0 issues. A later shared-cache contamination named another worktree;
  `golangci-lint cache clean` cleared it and the full gate was rerun successfully.
- make test: Go race suite passes (first full run uncached; subsequent runs cached).
- make verify: exit 0; govulncheck reports no vulnerabilities; bindings fresh.
- make build: exit 0; executable `build/bin/thewarroom`, x86-64 ELF.
- make bindings-fresh: exit 0; **133** exported bound methods, unchanged.

Step 6 skipped as instructed: the named dependency-audit step does not exist in
`.github/workflows/verify.yml` on this base. Workflow left unchanged.

UNVERIFIED: interactive desktop launch and visual parity after Tailwind 4.
Operator test: open the built app and inspect shell, inputs/buttons, channel chips,
Market, Draft room and commissioner trays; fonts, spacing, focus and colors should
match the previous build. No desktop process was launched against operator data.
No React/React DOM/TypeScript major was needed; no task stop. No push.

## Exhaustive resolved-package changes

Platform-specific optional packages are included; unchanged packages omitted.

| Package | Before | After |
|---|---|---|
| @babel/code-frame | 7.29.7 | removed |
| @babel/compat-data | 7.29.7 | removed |
| @babel/core | 7.29.7 | removed |
| @babel/generator | 7.29.7 | removed |
| @babel/helper-annotate-as-pure | 7.29.7 | removed |
| @babel/helper-compilation-targets | 7.29.7 | removed |
| @babel/helper-globals | 7.29.7 | removed |
| @babel/helper-module-imports | 7.29.7 | removed |
| @babel/helper-module-transforms | 7.29.7 | removed |
| @babel/helper-plugin-utils | 7.29.7 | removed |
| @babel/helper-string-parser | 7.29.7 | removed |
| @babel/helper-validator-identifier | 7.29.7 | removed |
| @babel/helper-validator-option | 7.29.7 | removed |
| @babel/helpers | 7.29.7 | removed |
| @babel/parser | 7.29.7 | removed |
| @babel/plugin-syntax-jsx | 7.29.7 | removed |
| @babel/plugin-transform-react-jsx | 7.29.7 | removed |
| @babel/plugin-transform-react-jsx-development | 7.29.7 | removed |
| @babel/plugin-transform-react-jsx-self | 7.29.7 | removed |
| @babel/plugin-transform-react-jsx-source | 7.29.7 | removed |
| @babel/template | 7.29.7 | removed |
| @babel/traverse | 7.29.7 | removed |
| @babel/types | 7.29.7 | removed |
| @esbuild/android-arm | 0.15.18 | removed |
| @esbuild/linux-loong64 | 0.15.18 | removed |
| @jridgewell/sourcemap-codec | 1.5.5 | 1.5.5, 1.6.0 |
| @nodelib/fs.scandir | 2.1.5 | removed |
| @nodelib/fs.stat | 2.0.5 | removed |
| @nodelib/fs.walk | 1.2.8 | removed |
| @oxc-project/types | absent | 0.155.0 |
| @rolldown/binding-android-arm-eabi | absent | 1.2.15 |
| @rolldown/binding-android-arm64 | absent | 1.2.15 |
| @rolldown/binding-darwin-arm64 | absent | 1.2.15 |
| @rolldown/binding-darwin-x64 | absent | 1.2.15 |
| @rolldown/binding-freebsd-x64 | absent | 1.2.15 |
| @rolldown/binding-linux-arm-gnueabihf | absent | 1.2.15 |
| @rolldown/binding-linux-arm64-gnu | absent | 1.2.15 |
| @rolldown/binding-linux-arm64-musl | absent | 1.2.15 |
| @rolldown/binding-linux-ppc64-gnu | absent | 1.2.15 |
| @rolldown/binding-linux-s390x-gnu | absent | 1.2.15 |
| @rolldown/binding-linux-x64-gnu | absent | 1.2.15 |
| @rolldown/binding-linux-x64-musl | absent | 1.2.15 |
| @rolldown/binding-openharmony-arm64 | absent | 1.2.15 |
| @rolldown/binding-win32-arm64-msvc | absent | 1.2.15 |
| @rolldown/binding-win32-x64-msvc | absent | 1.2.15 |
| @rolldown/pluginutils | absent | 1.0.1 |
| @tailwindcss/node | absent | 4.3.3 |
| @tailwindcss/oxide | absent | 4.3.3 |
| @tailwindcss/oxide-android-arm64 | absent | 4.3.3 |
| @tailwindcss/oxide-darwin-arm64 | absent | 4.3.3 |
| @tailwindcss/oxide-darwin-x64 | absent | 4.3.3 |
| @tailwindcss/oxide-freebsd-x64 | absent | 4.3.3 |
| @tailwindcss/oxide-linux-arm-gnueabihf | absent | 4.3.3 |
| @tailwindcss/oxide-linux-arm64-gnu | absent | 4.3.3 |
| @tailwindcss/oxide-linux-arm64-musl | absent | 4.3.3 |
| @tailwindcss/oxide-linux-x64-gnu | absent | 4.3.3 |
| @tailwindcss/oxide-linux-x64-musl | absent | 4.3.3 |
| @tailwindcss/oxide-wasm32-wasi | absent | 4.3.3 |
| @tailwindcss/oxide-win32-arm64-msvc | absent | 4.3.3 |
| @tailwindcss/oxide-win32-x64-msvc | absent | 4.3.3 |
| @tailwindcss/postcss | absent | 4.3.3 |
| @types/chai | 4.3.20 | 5.2.3 |
| @types/chai-subset | 1.3.6 | removed |
| @types/deep-eql | absent | 4.0.2 |
| @types/estree | absent | 1.0.9 |
| @types/node | 20.11.30 | 22.20.5 |
| @vitejs/plugin-react | 2.2.0 | 6.1.2 |
| @vitest/mocker | absent | 5.0.5 |
| @vitest/spy | absent | 5.0.5 |
| acorn | 8.19.0 | removed |
| acorn-walk | 8.3.5 | removed |
| any-promise | 1.3.0 | removed |
| anymatch | 3.1.3 | removed |
| arg | 5.0.2 | removed |
| assertion-error | 1.1.0 | 2.0.1 |
| autoprefixer | 10.5.0 | removed |
| baseline-browser-mapping | 2.10.37 | removed |
| binary-extensions | 2.3.0 | removed |
| braces | 3.0.3 | removed |
| browserslist | 4.28.2 | removed |
| camelcase-css | 2.0.1 | removed |
| caniuse-lite | 1.0.30001799 | removed |
| chai | 4.5.0 | 6.3.0 |
| check-error | 1.0.3 | removed |
| chokidar | 3.6.0 | removed |
| commander | 4.1.1 | removed |
| convert-source-map | 2.0.0 | removed |
| cssesc | 3.0.0 | removed |
| debug | 4.4.3 | removed |
| deep-eql | 4.1.4 | removed |
| detect-libc | absent | 2.1.2 |
| didyoumean | 1.2.2 | removed |
| dlv | 1.1.3 | removed |
| electron-to-chromium | 1.5.375 | removed |
| enhanced-resolve | absent | 5.26.0 |
| es-errors | 1.3.0 | removed |
| es-module-lexer | absent | 2.3.2 |
| esbuild | 0.15.18 | removed |
| esbuild-android-64 | 0.15.18 | removed |
| esbuild-android-arm64 | 0.15.18 | removed |
| esbuild-darwin-64 | 0.15.18 | removed |
| esbuild-darwin-arm64 | 0.15.18 | removed |
| esbuild-freebsd-64 | 0.15.18 | removed |
| esbuild-freebsd-arm64 | 0.15.18 | removed |
| esbuild-linux-32 | 0.15.18 | removed |
| esbuild-linux-64 | 0.15.18 | removed |
| esbuild-linux-arm | 0.15.18 | removed |
| esbuild-linux-arm64 | 0.15.18 | removed |
| esbuild-linux-mips64le | 0.15.18 | removed |
| esbuild-linux-ppc64le | 0.15.18 | removed |
| esbuild-linux-riscv64 | 0.15.18 | removed |
| esbuild-linux-s390x | 0.15.18 | removed |
| esbuild-netbsd-64 | 0.15.18 | removed |
| esbuild-openbsd-64 | 0.15.18 | removed |
| esbuild-sunos-64 | 0.15.18 | removed |
| esbuild-windows-32 | 0.15.18 | removed |
| esbuild-windows-64 | 0.15.18 | removed |
| esbuild-windows-arm64 | 0.15.18 | removed |
| escalade | 3.2.0 | removed |
| estree-walker | absent | 3.0.3 |
| expect-type | absent | 1.4.0 |
| fast-glob | 3.3.3 | removed |
| fastq | 1.20.1 | removed |
| fill-range | 7.1.1 | removed |
| fraction.js | 5.3.4 | removed |
| function-bind | 1.1.2 | removed |
| gensync | 1.0.0-beta.2 | removed |
| get-func-name | 2.0.2 | removed |
| glob-parent | 5.1.2, 6.0.2 | removed |
| graceful-fs | absent | 4.2.11 |
| hasown | 2.0.4 | removed |
| is-binary-path | 2.1.0 | removed |
| is-core-module | 2.16.2 | removed |
| is-extglob | 2.1.1 | removed |
| is-glob | 4.0.3 | removed |
| is-number | 7.0.0 | removed |
| jiti | 1.21.7 | 2.7.0 |
| jsesc | 3.1.0 | removed |
| json5 | 2.2.3 | removed |
| lightningcss | absent | 1.32.0, 1.33.0 |
| lightningcss-android-arm64 | absent | 1.32.0, 1.33.0 |
| lightningcss-darwin-arm64 | absent | 1.32.0, 1.33.0 |
| lightningcss-darwin-x64 | absent | 1.32.0, 1.33.0 |
| lightningcss-freebsd-x64 | absent | 1.32.0, 1.33.0 |
| lightningcss-linux-arm-gnueabihf | absent | 1.32.0, 1.33.0 |
| lightningcss-linux-arm64-gnu | absent | 1.32.0, 1.33.0 |
| lightningcss-linux-arm64-musl | absent | 1.32.0, 1.33.0 |
| lightningcss-linux-x64-gnu | absent | 1.32.0, 1.33.0 |
| lightningcss-linux-x64-musl | absent | 1.32.0, 1.33.0 |
| lightningcss-win32-arm64-msvc | absent | 1.32.0, 1.33.0 |
| lightningcss-win32-x64-msvc | absent | 1.32.0, 1.33.0 |
| lilconfig | 3.1.3 | removed |
| lines-and-columns | 1.2.4 | removed |
| local-pkg | 0.4.3 | removed |
| loupe | 2.3.7 | removed |
| lru-cache | 5.1.1 | removed |
| magic-string | 0.26.7 | 0.30.21, 1.4.3 |
| merge2 | 1.4.1 | removed |
| micromatch | 4.0.8 | removed |
| ms | 2.1.3 | removed |
| mz | 2.7.0 | removed |
| nanoid | 3.3.12 | 3.3.20 |
| node-releases | 2.0.47 | removed |
| normalize-path | 3.0.0 | removed |
| object-assign | 4.1.1 | removed |
| object-hash | 3.0.0 | removed |
| obug | absent | 3.0.0 |
| path-parse | 1.0.7 | removed |
| pathval | 1.1.1 | removed |
| picomatch | 2.3.2, 4.0.4 | 4.0.4, 4.0.7 |
| pify | 2.3.0 | removed |
| pirates | 4.0.7 | removed |
| postcss | 8.5.15 | 8.5.31 |
| postcss-import | 15.1.0 | removed |
| postcss-js | 4.1.0 | removed |
| postcss-load-config | 6.0.1 | removed |
| postcss-nested | 6.2.0 | removed |
| postcss-selector-parser | 6.1.4 | removed |
| postcss-value-parser | 4.2.0 | removed |
| queue-microtask | 1.2.3 | removed |
| react-refresh | 0.14.2 | removed |
| read-cache | 1.0.0 | removed |
| readdirp | 3.6.0 | removed |
| resolve | 1.22.12 | removed |
| reusify | 1.1.0 | removed |
| rolldown | absent | 1.2.15 |
| rollup | 2.80.0 | removed |
| run-parallel | 1.2.0 | removed |
| semver | 6.3.1 | removed |
| source-map | 0.6.1 | removed |
| source-map-js | 1.2.1 | 1.2.2 |
| sourcemap-codec | 1.4.8 | removed |
| std-env | absent | 4.3.0 |
| strip-literal | 1.3.0 | removed |
| sucrase | 3.35.1 | removed |
| supports-preserve-symlinks-flag | 1.0.0 | removed |
| tailwindcss | 3.4.19 | 4.3.3 |
| tapable | absent | 2.3.3 |
| thenify | 3.3.1 | removed |
| thenify-all | 1.6.0 | removed |
| tinybench | 2.9.0 | 6.2.2 |
| tinyexec | absent | 1.3.1 |
| tinypool | 0.3.1 | removed |
| tinyspy | 1.1.1 | removed |
| to-regex-range | 5.0.1 | removed |
| ts-interface-checker | 0.1.13 | removed |
| type-detect | 4.1.0 | removed |
| undici-types | 5.26.5 | 6.21.0 |
| update-browserslist-db | 1.2.3 | removed |
| util-deprecate | 1.0.2 | removed |
| vite | 3.2.11 | 8.3.5 |
| vitest | 0.25.8 | 5.0.5 |
| why-is-node-running | absent | 3.2.1 |
| yallist | 3.1.1 | removed |
