package fetcher

import (
	"context"
	"testing"
	"time"
)

func Test_rateLimit_limit(t *testing.T) {
	type args struct {
		rate        int
		duration    time.Duration
		ctxDeadline time.Duration
		runCount    int
	}
	tests := []struct {
		name string
		args args
		want *rateLimit
	}{
		{
			"1 per second",
			args{
				rate:        5,
				duration:    5 * time.Millisecond,
				ctxDeadline: 5 * time.Millisecond,
				runCount:    3,
			},
			&rateLimit{
				enforcedRate: time.Millisecond,
			},
		},
		{
			"10 per second",
			args{
				rate:        10,
				duration:    1 * time.Millisecond,
				ctxDeadline: 3 * time.Millisecond,
				runCount:    20,
			},
			&rateLimit{
				enforcedRate: time.Millisecond / 10,
			},
		},
		{
			"10 per second - killed by context",
			args{
				rate:        10,
				duration:    1 * time.Millisecond,
				ctxDeadline: 2 * time.Millisecond,
				runCount:    30,
			},
			&rateLimit{
				enforcedRate: time.Millisecond / 10,
			},
		},
		{
			"no rate limit",
			args{
				rate:        0,
				duration:    0,
				ctxDeadline: 2 * time.Millisecond,
				runCount:    5,
			},
			&rateLimit{
				enforcedRate: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := newRateLimit(tt.args.rate, tt.args.duration)

			timeToProcess := rl.enforcedRate * time.Duration(tt.args.runCount)
			finishAtOrAfter := time.Now().UTC().Add(timeToProcess)
			if timeToProcess >= tt.args.ctxDeadline {
				finishAtOrAfter = time.Now().UTC().Add(tt.args.ctxDeadline)
			}

			c, cancelFunc := context.WithDeadline(context.Background(), time.Now().UTC().Add(tt.args.ctxDeadline))
			defer cancelFunc()

			for i := 0; i < tt.args.runCount; i++ {
				rl.limit(c)
			}

			if tt.want.enforcedRate != rl.enforcedRate {
				t.Errorf("rateLimit = %s, want %s", rl.enforcedRate.String(), tt.want.enforcedRate.String())
			}

			tm := time.Now().UTC()
			if !(tm.After(finishAtOrAfter) || tm.Equal(finishAtOrAfter)) {
				t.Errorf("time = %s, want finishAtOrAfter %s", tm.Format("2006-01-02 15:04:05.9999"), finishAtOrAfter.Format("2006-01-02 15:04:05.9999"))
			}
		})
	}
}
