package market

import (
	"time"
)

type fakeModel map[string]Dynasty

func (f fakeModel) Player(pid string, _ time.Time) Dynasty { return f[pid] }

func (fakeModel) Pick(int, int, string, time.Time) Dynasty { return Dynasty{} }
