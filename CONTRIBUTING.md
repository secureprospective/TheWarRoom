# Contributing to TheWarRoom

Thanks for your interest — contributions are welcome, and a growing homelab
community is exactly how a project like this spreads. A few things to know
before you open a pull request.

## The short version

- **License:** TheWarRoom is source-available under the [PolyForm Noncommercial
  License 1.0.0](LICENSE). It is **free to use non-commercially, forever** —
  but it is **not** OSI "open source," because commercial use and sale are
  reserved to the owner, SecureProspective LLC.
- **Ownership:** copyright is held solely by **SecureProspective LLC**. See
  [`docs/licensing/README-license-summary.md`](docs/licensing/README-license-summary.md)
  for the full, plain-English picture.
- **Contributing = agreeing to the CLA:** by submitting any contribution, you
  agree to the [Contributor License Agreement](CLA.md). This is what keeps
  ownership consolidated so the project can be maintained, licensed, and one
  day potentially sold as a single, clean asset — while every contributor's
  own right to use their work is fully preserved.

## Why a CLA (in plain terms)

When you write code, **you** own the copyright on it by default. If your code
lands in TheWarRoom without an agreement, the project would be co-owned by
everyone who ever contributed — which would make it impossible to license
cleanly or maintain long-term. The CLA solves this: you grant SecureProspective
LLC a broad, irrevocable license to your contribution (you keep your own
copyright and can reuse your work freely), and in return your code becomes part
of the project on stable footing. See [`CLA.md`](CLA.md) for the exact terms.

## Ground rules for pull requests

TheWarRoom holds itself to a strict engineering bar. Before you open a PR:

- **Build green.** `GOMEMLIMIT=3000MiB GOGC=40 make lint` reports 0, and
  `go test -race ./...` passes. No `--no-verify`, ever.
- **Respect the architecture.** The three-layer separation, the single-writer
  law, and the pure engine are compiler-enforced — a violation is a build
  failure, not a style note. Read [`SYSTEM_MAP.md`](SYSTEM_MAP.md) first.
- **Match the surrounding code.** Naming, comment density, and idiom should be
  indistinguishable from the code around your change.
- **One focused change per PR**, with a clear description of what and why.

## Questions

Open an issue, or reach [SecureProspective](https://secureprospective.com) for
anything licensing- or commercial-related.
