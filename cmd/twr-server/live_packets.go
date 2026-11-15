package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const livePacketBytes = 64 << 20

type rawLivePacket struct {
	URL    string
	Body   []byte
	Digest string
	At     time.Time
}

type rawLiveCheckpoint struct {
	Reduced []string
	Version int
	Season  int
	Week    int
	Packets map[string]rawLivePacket
	IDs     map[string]map[string]string
}

func (r rawLiveCheckpoint) Validate() error {
	if r.Version != 1 || r.Season < 1920 || r.Season > 9999 || r.Week < 1 || r.Week > 22 || len(r.Packets) > 20 || len(r.IDs) > 3 || len(r.Reduced) > 20 {
		return fmt.Errorf("invalid live checkpoint scope or count")
	}
	total := 0
	for address, p := range r.Packets {
		if address != p.URL {
			return fmt.Errorf("live packet key differs from URL")
		}
		if err := p.Validate(); err != nil {
			return err
		}
		total += len(p.Body)
	}

	if total > livePacketBytes {
		return fmt.Errorf("live packets exceed byte budget")
	}
	for kind, ids := range r.IDs {
		if err := validateLiveIDs(kind, ids); err != nil {
			return err
		}
	}

	return nil
}

func (p rawLivePacket) Validate() error {
	u, err := url.Parse(p.URL)
	if err != nil {
		return fmt.Errorf("live packet URL: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil || len(p.URL) > 2048 || p.At.IsZero() || len(p.Body) > 4<<20 || p.Digest != liveDigest(p.Body) {
		return fmt.Errorf("invalid live packet or digest")
	}
	return nil
}

func validateLiveIDs(kind string, ids map[string]string) error {
	if (kind != "espn" && kind != "sleeper" && kind != "gsis") || len(ids) > 30000 {
		return fmt.Errorf("invalid live identity set")
	}
	for player, id := range ids {
		if len(player) == 0 || len(player) > 40 || len(id) == 0 || len(id) > 200 {
			return fmt.Errorf("invalid live identity")
		}
	}
	return nil
}

func liveDigest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

// livePackets serializes packet admission; summary requests may fetch concurrently.
type livePackets struct {
	mu             sync.Mutex
	boardURL       string
	boardInterval  time.Duration
	nextSummary    map[string]time.Time
	summaryReduced map[string]bool
	network        http.RoundTripper
	held           rawLiveCheckpoint
	replay         bool
	prefetched     *rawLivePacket
	nextSleeper    time.Time
	sleeperReduced bool
	failed         error
}

func (p *livePackets) RoundTrip(request *http.Request) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	address := request.URL.String()
	if p.prefetched != nil && p.prefetched.URL == address {
		packet := *p.prefetched
		p.prefetched = nil
		return livePacketResponse(packet), nil
	}
	if p.replay {
		packet, ok := p.held.Packets[address]
		if !ok {
			p.failed = fmt.Errorf("live replay has no requested packet")
			return nil, p.failed
		}
		return livePacketResponse(packet), nil
	}
	if deadline, ok := p.nextSummary[address]; ok && time.Now().Before(deadline) {
		if packet, exists := p.held.Packets[address]; exists {
			if p.summaryReduced[address] {
				p.held.Reduced = append(p.held.Reduced, address)
			}
			return livePacketResponse(packet), nil
		}
	}
	// Keep Sleeper's weekly gap feed on its own slower clock, including failed attempts.
	if strings.HasPrefix(request.URL.Path, "/v1/stats/nfl/regular/") {
		if time.Now().Before(p.nextSleeper) {
			if packet, ok := p.held.Packets[address]; ok {
				if p.sleeperReduced {
					p.held.Reduced = append(p.held.Reduced, address)
				}
				return livePacketResponse(packet), nil
			}
			p.failed = fmt.Errorf("sleeper live retry not due")
			return nil, p.failed
		}
		p.sleeperReduced = false
		p.nextSleeper = time.Now().Add(time.Minute)
	}
	return p.fetch(request)
}

// fetch runs with mu held, releasing it only for the network request.
func (p *livePackets) fetch(request *http.Request) (*http.Response, error) {
	address := request.URL.String()
	p.mu.Unlock()
	response, err := p.network.RoundTrip(request)
	p.mu.Lock()
	if err != nil {
		return p.fallback(address, fmt.Errorf("fetch live packet: %w", err))
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return p.fallback(address, fmt.Errorf("live source HTTP %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil {
		return p.fallback(address, fmt.Errorf("read live packet: %w", err))
	}
	if len(body) > 4<<20 {
		p.failed = fmt.Errorf("live packet exceeds byte budget")
		return nil, p.failed
	}
	if address == p.boardURL {
		p.boardInterval = liveBoardMaxAge(response.Header.Get("Cache-Control"))
	}
	packet := rawLivePacket{URL: address, Body: body, Digest: liveDigest(body), At: time.Now().UTC()}
	if err := p.admit(packet); err != nil {
		p.failed = err
		return nil, err
	}
	p.held.Packets[address] = packet
	if strings.Contains(request.URL.Path, "/summary") {
		if p.nextSummary == nil {
			p.nextSummary = map[string]time.Time{}
			p.summaryReduced = map[string]bool{}
		}
		p.nextSummary[address] = time.Now().Add(8 * time.Second)
		p.summaryReduced[address] = false
	}
	return livePacketResponse(packet), nil
}

func (p *livePackets) admit(packet rawLivePacket) error {
	if _, exists := p.held.Packets[packet.URL]; !exists && len(p.held.Packets) >= 20 {
		return fmt.Errorf("live packet count exceeds budget")
	}
	total := len(packet.Body)
	for address, held := range p.held.Packets {
		if address != packet.URL {
			total += len(held.Body)
		}
	}
	if total > livePacketBytes {
		return fmt.Errorf("live packets exceed byte budget")
	}
	return nil
}

func (p *livePackets) fallback(address string, err error) (*http.Response, error) {
	if strings.Contains(address, "/summary") && p.summaryReduced != nil {
		p.summaryReduced[address] = true
	}
	if strings.Contains(address, "/v1/stats/nfl/regular/") {
		p.sleeperReduced = true
	}
	if packet, ok := p.held.Packets[address]; ok {
		p.held.Reduced = append(p.held.Reduced, address)
		return livePacketResponse(packet), nil
	}
	p.failed = err
	return nil, err
}

func livePacketResponse(packet rawLivePacket) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(packet.Body))}
}

func writeLiveCheckpoint(path string, checkpoint rawLiveCheckpoint) (string, error) {
	if err := checkpoint.Validate(); err != nil {
		return "", err
	}
	body, err := json.Marshal(checkpoint)
	if err != nil {
		return "", fmt.Errorf("encode live checkpoint: %w", err)
	}
	if len(body) > 96<<20 {
		return "", fmt.Errorf("encoded checkpoint exceeds budget")
	}
	file, err := os.OpenFile(path+".pending", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("open live checkpoint: %w", err)
	}
	_, writeErr := file.Write(body)
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return "", fmt.Errorf("write live checkpoint: %w", err)
	}
	if err := os.Rename(path+".pending", path); err != nil {
		return "", fmt.Errorf("replace live checkpoint: %w", err)
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("open checkpoint directory: %w", err)
	}
	if err := errors.Join(directory.Sync(), directory.Close()); err != nil {
		return "", fmt.Errorf("sync live checkpoint directory: %w", err)
	}
	return liveDigest(body), nil
}

func readLiveCheckpoint(path string) (rawLiveCheckpoint, error) {
	file, err := os.Open(path)
	if err != nil {
		return rawLiveCheckpoint{}, fmt.Errorf("open live checkpoint: %w", err)
	}
	defer func() { _ = file.Close() }()
	body, err := io.ReadAll(io.LimitReader(file, (96<<20)+1))
	if err != nil {
		return rawLiveCheckpoint{}, fmt.Errorf("read live checkpoint: %w", err)
	}
	if len(body) > 96<<20 {
		return rawLiveCheckpoint{}, fmt.Errorf("live checkpoint exceeds budget")
	}
	var raw rawLiveCheckpoint
	if err := json.Unmarshal(body, &raw); err != nil {
		return rawLiveCheckpoint{}, fmt.Errorf("decode live checkpoint: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return rawLiveCheckpoint{}, err
	}
	return raw, nil
}

// ESPN's CDN cadence bounds useful scoreboard requests, within the hosted live budget.
func liveBoardMaxAge(header string) time.Duration {
	delay := 5 * time.Second
	for directive := range strings.SplitSeq(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(directive), "=")
		if !ok || !strings.EqualFold(key, "max-age") {
			continue
		}
		seconds, err := strconv.Atoi(strings.Trim(value, "\" "))
		if err == nil && seconds > 5 {
			delay = time.Duration(min(seconds, 10)) * time.Second
		}
	}
	return delay
}
