package fetcher

import (
	"context"
	"time"
)

type rateLimit struct {
	enforcedRate time.Duration
	ticker       *time.Ticker
}

func newRateLimit(rate int, dur time.Duration) rateLimit {
	if rate <= 0 || dur <= 0 {
		return rateLimit{}
	}
	return rateLimit{
		enforcedRate: dur / time.Duration(rate),
		ticker:       time.NewTicker(dur / time.Duration(rate)),
	}
}

func (rl *rateLimit) limit(c context.Context) {
	if rl.enforcedRate == 0 {
		return
	}

	// wait for the ticker or c.Done
	select {
	case <-rl.ticker.C:
		return
	case <-c.Done():
		rl.ticker.Stop()
		return
	}
}
