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
