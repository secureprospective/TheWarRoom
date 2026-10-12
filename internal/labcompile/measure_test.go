package labcompile

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type rssResult struct {
	Peak int64
	Err  error
}

func residentKiB() (int64, error) {
	body, err := readWithin("/proc/self", "status")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			n, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse RSS: %w", err)
			}
			return n, nil
		}
	}
	return 0, fmt.Errorf("process status has no RSS")
}
func sampleRSS() (int64, func() rssResult, error) {
	before, err := residentKiB()
	if err != nil {
		return 0, nil, err
	}
	stop, done := make(chan struct{}), make(chan rssResult, 1)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		peak := before
		for {
			select {
			case <-stop:
				done <- rssResult{Peak: peak}
				return
			case <-ticker.C:
				n, err := residentKiB()
				if err != nil {
					done <- rssResult{Err: err}
					return
				}
				peak = max(peak, n)
			}
		}
	}()
	return before, func() rssResult { close(stop); return <-done }, nil
}
