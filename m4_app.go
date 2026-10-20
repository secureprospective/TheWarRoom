package main

import (
	"time"
)

// m4Timeout bounds the M4 reads: the first call after a cold start may fetch the players
// directory.
const m4Timeout = 120 * time.Second
