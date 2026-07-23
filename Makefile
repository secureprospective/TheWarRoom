# TheWarRoom — Makefile. Targets merged from the christopher-coding-standards
# Go overlay (templates/go/Makefile.snippet). `make lint` runs ifaceguard +
# filelen + golangci-lint; all must pass to clear. Never bypass with --no-verify.

.PHONY: lint fmt vet test test-coverage build dev mutation-test ifaceguard filelen release sync-product-version

# ── Build stamp (D-V2) ────────────────────────────────────────────────────────
# The git tag is the single source of truth. `git describe` yields the tag
# (or a bare short SHA before the first tag), `--dirty` marks an uncommitted
# tree so an unreleased build can never masquerade as a clean tag. These flow
# into the `main.version/commit/buildDate` vars via `-ldflags -X` (version.go).
# A plain `go build` / `wails dev` leaves the vars at their "dev" defaults —
# deliberately visibly distinct from a stamped release.
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

# ifaceguard — custom go/analysis vettool flagging empty-interface
# (interface{}/any) escapes in EXPORTED signatures — the one gap no enabled
# golangci-lint linter covers (Fable Friction #10). Built from its own pinned
# module under tools/ifaceguard.
IFACEGUARD_BIN := tools/ifaceguard/bin/ifaceguard

$(IFACEGUARD_BIN): tools/ifaceguard/ifaceguard.go tools/ifaceguard/cmd/ifaceguard/main.go tools/ifaceguard/go.mod
	cd tools/ifaceguard && go build -o bin/ifaceguard ./cmd/ifaceguard

ifaceguard: $(IFACEGUARD_BIN)
	go vet -vettool=$(abspath $(IFACEGUARD_BIN)) ./...

# filelen — enforces the 400-line file cap (a design constraint, not cleanup;
# AD-14/AD-17 pre-splits). golangci-lint's funlen caps FUNCTION size, not file
# size, and no enabled linter measures file length. Test files exempt.
FILE_CAP := 400
filelen:
	@offenders=$$(find . -name '*.go' -not -name '*_test.go' -not -path './vendor/*' -not -path './tools/*' \
	  -exec awk 'END { if (NR > $(FILE_CAP)) printf "  %s (%d lines)\n", FILENAME, NR }' {} \;); \
	if [ -n "$$offenders" ]; then \
	  echo "FILE CAP ($(FILE_CAP) lines) exceeded — pre-split per the wireframe (AD-14/AD-17) BEFORE commit:"; \
	  echo "$$offenders"; exit 1; \
	else echo "filelen: all source files within the $(FILE_CAP)-line cap"; fi

# lint runs ifaceguard, filelen, AND golangci-lint — all must pass to clear.
lint: ifaceguard filelen
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

# -race is non-negotiable (failure mode #2: concurrent map access from Wails IPC
# goroutines hitting B3c). Never make this optional.
test:
	go test -race ./...

COVERAGE_THRESHOLD := 80

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@total=$$(go tool cover -func=coverage.out | grep '^total:' | grep -oE '[0-9]+\.[0-9]+'); \
	awk -v t="$$total" -v min="$(COVERAGE_THRESHOLD)" 'BEGIN { if (t+0 < min+0) { printf "coverage %.1f%% < threshold %d%%\n", t, min; exit 1 } else { printf "coverage %.1f%% >= threshold %d%%\n", t, min } }'

# Desktop shell. Wails embeds the built frontend; before the frontend is wired
# substitute `go build ./...`. The build stamp is injected here (see LDFLAGS).
build:
	wails build -ldflags "$(LDFLAGS)"

# dev is intentionally UNstamped — the "dev" var defaults make a `wails dev`
# binary read as a dev build in the UI, which is the point.
dev:
	wails dev

# ── Release (D-V4) ────────────────────────────────────────────────────────────
# Tags a version off already-gated main. There is NO separate release gate: the
# merge gate (lint 0 / go test -race / tsc+vite / live Beelink gate) IS it.
# Refuses a dirty tree, enforces a SemVer vX.Y.Z tag, syncs the packaging
# metadata, then creates an annotated tag. Usage: make release TAG=v0.5.0
release:
	@test -n "$(TAG)" || { echo "release: usage — make release TAG=vX.Y.Z"; exit 1; }
	@echo "$(TAG)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || \
	  { echo "release: TAG must be SemVer vX.Y.Z (got '$(TAG)')"; exit 1; }
	@test -z "$$(git status --porcelain)" || \
	  { echo "release: refusing to tag a dirty tree — commit or stash first"; exit 1; }
	@$(MAKE) --no-print-directory sync-product-version TAG=$(TAG)
	@if [ -n "$$(git status --porcelain)" ]; then \
	  git commit -am "release: $(TAG) — sync wails.json productVersion"; \
	else echo "release: wails.json productVersion already $(TAG:v%=%)"; fi
	git tag -a "$(TAG)" -m "The War Room $(TAG)"
	@echo "release: tagged $(TAG). Push with: git push origin $$(git rev-parse --abbrev-ref HEAD) --tags"

# sync-product-version writes the SemVer core (TAG minus leading 'v') into
# wails.json info.productVersion — Wails installer metadata, inert on the Linux
# alpha, consumed by the Windows/macOS packagers at Beta. Idempotent: the binary
# stamp (LDFLAGS) is the runtime authority; this is packaging only. Usage:
# make sync-product-version TAG=v0.5.0
sync-product-version:
	@test -n "$(TAG)" || { echo "sync-product-version: TAG required (vX.Y.Z)"; exit 1; }
	@tmp=$$(mktemp) && jq --arg v "$(TAG:v%=%)" '.info.productVersion = $$v' wails.json > "$$tmp" \
	  && mv "$$tmp" wails.json && echo "sync-product-version: wails.json productVersion = $(TAG:v%=%)"

# gremlins v0.6.0, pinned. Scoped to internal/engine/... (pure-logic packages
# are where a mutation score has signal). NOT run in CI by default — running a
# pinned third-party module@version is an external-code-exec action requiring
# explicit per-action authorization (Go overlay README).
mutation-test:
	go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash ./internal/engine/...
