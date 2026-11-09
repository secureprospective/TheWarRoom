//go:build labpprof

package main

// Lab-only profiling: built only with -tags labpprof, never in a release. Loopback only.

import (
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // lab-only build tag; bound to loopback
	"time"
)

func init() {
	go func() {
		srv := &http.Server{Addr: "127.0.0.1:6060", ReadHeaderTimeout: 5 * time.Second}
		log.Println(srv.ListenAndServe())
	}()
}
