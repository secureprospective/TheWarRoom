package serverhttp

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// ErrorCodeHeader carries a server-generated TWR-XXXX-XXXX failure code.
	ErrorCodeHeader = "X-TWR-Error-Code"

	errorCodeAlphabet       = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	serverCodeFirstAlphabet = "23456789ABDEFGHJKLMNPQRSTUVWXYZ"
)

// NewServerErrorCode creates one server failure code formatted TWR-XXXX-XXXX.
// The first code character is never C, reserving TWR-C for client-only codes.
func NewServerErrorCode() string {
	var out [13]byte
	copy(out[:4], "TWR-")
	out[8] = '-'
	out[4] = randomAlphabetByte(serverCodeFirstAlphabet)
	var raw [7]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	for i := range 3 {
		out[5+i] = errorCodeAlphabet[raw[i]&31]
	}
	for i := range 4 {
		out[9+i] = errorCodeAlphabet[raw[3+i]&31]
	}
	return string(out[:])
}

func randomAlphabetByte(alphabet string) byte {
	limit := 256 - (256 % len(alphabet))
	var b [1]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			panic("crypto/rand unavailable: " + err.Error())
		}
		if int(b[0]) < limit {
			return alphabet[int(b[0])%len(alphabet)]
		}
	}
}

// ValidErrorCode reports whether code matches TWR-XXXX-XXXX over the unambiguous alphabet.
func ValidErrorCode(code string) bool {
	if len(code) != 13 || code[:4] != "TWR-" || code[8] != '-' {
		return false
	}
	for _, idx := range [8]int{4, 5, 6, 7, 9, 10, 11, 12} {
		if strings.IndexByte(errorCodeAlphabet, code[idx]) < 0 {
			return false
		}
	}
	return true
}

type failureMetaKey struct{}

type failureMeta struct {
	mu      sync.Mutex
	command string
	league  string
	account string
}

func (m *failureMeta) snapshot() (command, league, account string) {
	if m == nil {
		return "", "", ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.command, m.league, m.account
}

func setFailureAccount(ctx context.Context, accountID string) {
	meta, ok := ctx.Value(failureMetaKey{}).(*failureMeta)
	if !ok || meta == nil || accountID == "" || len(accountID) > 128 {
		return
	}
	meta.mu.Lock()
	meta.account = accountID
	meta.mu.Unlock()
}

func validFailureLeague(league string) bool {
	if !strings.HasPrefix(league, "native.") ||
		len(league) <= len("native.") || len(league) > 128 {
		return false
	}
	for i := len("native."); i < len(league); i++ {
		c := league[i]
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		if !isLower && !isUpper && !isDigit && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func setFailureLeague(ctx context.Context, league string) {
	meta, ok := ctx.Value(failureMetaKey{}).(*failureMeta)
	if !ok || meta == nil || !validFailureLeague(league) {
		return
	}
	meta.mu.Lock()
	meta.league = league
	meta.mu.Unlock()
}

func setFailureCommand(ctx context.Context, command string) {
	meta, ok := ctx.Value(failureMetaKey{}).(*failureMeta)
	if !ok || meta == nil || command == "" || len(command) > 128 {
		return
	}
	meta.mu.Lock()
	meta.command = command
	meta.mu.Unlock()
}

func recordNativeFailureMeta(ctx context.Context, raw RawNativeRequest) {
	setFailureLeague(ctx, raw.leagueName())
	if raw.Workflow != nil {
		setFailureCommand(ctx, raw.Workflow.Command)
	} else if raw.Command != nil {
		setFailureCommand(ctx, raw.Command.Command)
	}
}

func (h Hosted) withErrorCodes(next http.Handler) http.Handler {
	logger := h.ErrorLog
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := &failureMeta{}
		ctx := context.WithValue(r.Context(), failureMetaKey{}, meta)
		now := h.now
		if now == nil {
			now = time.Now
		}
		wrapped := &errorCodeWriter{
			ResponseWriter: w,
			req:            r.WithContext(ctx),
			meta:           meta,
			logger:         logger,
			start:          now(),
			now:            now,
		}
		next.ServeHTTP(wrapped, wrapped.req)
	})
}

type errorCodeWriter struct {
	http.ResponseWriter
	req    *http.Request
	meta   *failureMeta
	logger *slog.Logger
	start  time.Time
	now    func() time.Time
	wrote  bool
}

func (w *errorCodeWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *errorCodeWriter) WriteHeader(status int) {
	if !w.wrote {
		w.wrote = true
		elapsed := w.now().Sub(w.start)
		w.Header().Set("Server-Timing", serverTimingValue(elapsed))
		if elapsed >= 250*time.Millisecond {
			command, league, _ := w.meta.snapshot()
			w.logger.WarnContext(
				w.req.Context(), "slow request",
				slog.Int("status", status),
				slog.String("route", safeErrorRoute(w.req.URL.Path)),
				slog.String("command", command),
				slog.String("league", league),
				slog.Float64("ms", float64(elapsed)/float64(time.Millisecond)),
			)
		}
		if status >= http.StatusBadRequest {
			code := w.Header().Get(ErrorCodeHeader)
			if !ValidErrorCode(code) {
				code = NewServerErrorCode()
				w.Header().Set(ErrorCodeHeader, code)
			}
			command, league, account := w.meta.snapshot()
			w.logger.ErrorContext(
				w.req.Context(), "request failed",
				slog.String("code", code),
				slog.Int("status", status),
				slog.String("route", safeErrorRoute(w.req.URL.Path)),
				slog.String("command", command),
				slog.String("league", league),
				slog.String("account", account),
			)
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *errorCodeWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	if err != nil {
		return n, err //nolint:wrapcheck // Preserve http.ResponseWriter write error.
	}
	return n, nil
}

func serverTimingValue(d time.Duration) string {
	return "app;dur=" + strconv.FormatFloat(float64(d)/float64(time.Millisecond), 'f', 1, 64)
}

func safeErrorRoute(path string) string {
	switch path {
	case "/", AppSessionPath, RecoveryPath, RecoveryCodesPath, EnrollmentPath,
		PasswordPagePath, PasswordChangePath, JoinPagePath, JoinFranchisesPath,
		JoinClaimPath, TOTPBeginPath, TOTPConfirmPath, TOTPStepUpPath,
		AgentTokenPath, AgentTokenRevokePath, RevokePath, LoginPath, DispatchPath,
		LedgerPath, PeoplePath, PreviewPath, ExecutePath, LineupPath, PickPath,
		ConfirmPath, WorkflowPath, AccountPreferencePath, InvalidationPath:
		return path
	}
	if path == "/app" || strings.HasPrefix(path, AppPath) {
		return AppPath
	}
	return "/other"
}
