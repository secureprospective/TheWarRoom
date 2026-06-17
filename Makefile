# TheWarRoom — Makefile. Targets merged from the christopher-coding-standards
# Go overlay (templates/go/Makefile.snippet). `make lint` runs ifaceguard +
# filelen + golangci-lint; all must pass to clear. Never bypass with --no-verify.

.PHONY: lint fmt vet test test-coverage build dev mutation-test ifaceguard filelen

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
# substitute `go build ./...`.
build:
	wails build

dev:
	wails dev

# gremlins v0.6.0, pinned. Scoped to internal/engine/... (pure-logic packages
# are where a mutation score has signal). NOT run in CI by default — running a
# pinned third-party module@version is an external-code-exec action requiring
# explicit per-action authorization (Go overlay README).
mutation-test:
	go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash ./internal/engine/...
