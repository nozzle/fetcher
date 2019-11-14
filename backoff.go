package fetcher

import (
	"math/rand"
	"time"
)

var defaultBackoffStrategy = &exponentialBackoff{
	min:       1 * time.Second,
	max:       30 * time.Second,
	useJitter: true,
}

// backoffStrategy is used to determine how long a retry request should wait until attempted
type backoffStrategy interface {
	waitDuration(attempt int) time.Duration
}

type noBackoff struct {
	delay time.Duration
}

func (b noBackoff) waitDuration(_ int) time.Duration {
	return b.delay
}

type exponentialBackoff struct {
	min       time.Duration
	max       time.Duration
	useJitter bool
}

func (b exponentialBackoff) waitDuration(attempt int) time.Duration {
	// use 0 based attempts since waiting only applies to retries
	attempt--
	delay := b.min * 1 << uint(attempt)

	if b.useJitter {
		delay = jitter(delay)
	}

	return normalizeDelay(delay, b.min, b.max)
}

type linearBackoff struct {
	min       time.Duration
	max       time.Duration
	interval  time.Duration
	useJitter bool
}

func (b linearBackoff) waitDuration(attempt int) time.Duration {
	// use 0 based attempts since waiting only applies to retries
	attempt--
	delay := b.min + b.interval*time.Duration(attempt)

	if b.useJitter {
		delay = jitter(delay)
	}

	return normalizeDelay(delay, b.min, b.max)
}

// jitter adjusts the baseDelay +/- 33%
func jitter(baseDelay time.Duration) time.Duration {
	delayNs := baseDelay.Nanoseconds()
	maxJitter := delayNs / 3

	delayNs += rand.Int63n(2*maxJitter) - maxJitter

	if delayNs <= 0 {
		delayNs = 1
	}

	return time.Duration(delayNs) * time.Nanosecond
}

func normalizeDelay(baseDelay, min, max time.Duration) time.Duration {
	if baseDelay > max {
		return max
	}

	if baseDelay < min {
		return min
	}

	return baseDelay
}
