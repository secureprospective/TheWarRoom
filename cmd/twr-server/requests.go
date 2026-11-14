package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
)

const requestsPath = "/requests"

type requestMemory struct {
	HeapAllocBytes uint64  `json:"heapAllocBytes"`
	HeapSysBytes   uint64  `json:"heapSysBytes"`
	NumGC          uint32  `json:"numGC"`
	RSSBytes       *uint64 `json:"rssBytes,omitempty"`
}
type requestsReport struct {
	Since    time.Time                           `json:"since"`
	Commands map[string]serverhttp.RequestCounts `json:"commands"`
	Memory   requestMemory                       `json:"memory"`
}

func requestRoutes(t *serverhttp.RequestTally) []serverstatus.Route {
	return []serverstatus.Route{
		{Pattern: "GET " + requestsPath, Handler: func(w http.ResponseWriter, _ *http.Request) {
			since, commands := t.Snapshot()
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			memory := requestMemory{HeapAllocBytes: stats.HeapAlloc, HeapSysBytes: stats.HeapSys, NumGC: stats.NumGC, RSSBytes: residentBytes()}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(requestsReport{Since: since, Commands: commands, Memory: memory})
		}},
		{Pattern: "POST " + requestsPath + "/reset", Handler: func(w http.ResponseWriter, _ *http.Request) { t.Reset(); w.WriteHeader(http.StatusNoContent) }},
	}
}
func residentBytes() *uint64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return nil
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil
	}
	size := os.Getpagesize()
	if size <= 0 {
		return nil
	}
	bytes := pages * uint64(size)
	return &bytes
}
func runRequests(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("twr-server requests", flag.ContinueOnError)
	socket := flags.String("status-socket", "", "running server's private socket")
	reset := flags.Bool("reset", false, "reset the counters first")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse requests flags: %w", err)
	}
	if *socket == "" || flags.NArg() != 0 {
		return fmt.Errorf("requests needs -status-socket")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", *socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: time.Minute, Transport: transport}
	if *reset {
		if err := requestReset(ctx, client); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://twr-server"+requestsPath, nil)
	if err != nil {
		return fmt.Errorf("create requests request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reach status socket: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("requests: server returned %s", resp.Status)
	}
	var report requestsReport
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&report); err != nil {
		return fmt.Errorf("read requests: %w", err)
	}
	return printRequests(output, report)
}
func requestReset(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://twr-server"+requestsPath+"/reset", nil)
	if err != nil {
		return fmt.Errorf("create reset: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reset requests: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("reset requests: server returned %s", resp.Status)
	}
	return nil
}
func printRequests(output io.Writer, report requestsReport) error {
	keys := make([]string, 0, len(report.Commands))
	for key := range report.Commands {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := report.Commands[keys[i]].Calls, report.Commands[keys[j]].Calls
		if a == b {
			return keys[i] < keys[j]
		}
		return a > b
	})
	var b strings.Builder
	fmt.Fprintf(&b, "Since %s\nCOMMAND MODE CALLS FROM-SERVER-CACHE COMPUTED REFUSED UNCHANGED KB-SENT\n", report.Since.Format(time.RFC3339))
	for _, key := range keys {
		c := report.Commands[key]
		fmt.Fprintf(&b, "%s %d %d %d %d %d %.2f\n", key, c.Calls, c.Cached, c.Computed, c.Refused, c.Unchanged, float64(c.ReplyBytes)/1024)
	}
	rss := "unavailable"
	if report.Memory.RSSBytes != nil {
		rss = fmt.Sprintf("%.2f MB", float64(*report.Memory.RSSBytes)/(1024*1024))
	}
	fmt.Fprintf(&b, "Memory: heap allocated %.2f MB, heap system %.2f MB, RSS %s, GC %d\n", float64(report.Memory.HeapAllocBytes)/(1024*1024), float64(report.Memory.HeapSysBytes)/(1024*1024), rss, report.Memory.NumGC)
	if _, err := io.WriteString(output, b.String()); err != nil {
		return fmt.Errorf("print requests: %w", err)
	}
	return nil
}
