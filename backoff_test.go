package fetcher

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func Test_noBackoff_waitDuration(t *testing.T) {
	tests := []struct {
		name    string
		delay   time.Duration
		attempt int
		want    time.Duration
	}{
		{
			name:    "1s on attempt 1",
			delay:   1 * time.Second,
			attempt: 1,
			want:    1 * time.Second,
		},
		{
			name:    "1s on attempt 10000",
			delay:   1 * time.Second,
			attempt: 10000,
			want:    1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := noBackoff{
				delay: tt.delay,
			}
			if got := b.waitDuration(tt.attempt); got != tt.want {
				t.Errorf("noBackoff.waitDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_exponentialBackoff_waitDuration(t *testing.T) {
	type fields struct {
		min       time.Duration
		max       time.Duration
		useJitter bool
	}
	type args struct {
		attempt int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
	}{
		{
			name: "1s on attempt 1",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 1},
			want: 1 * time.Second,
		},
		{
			name: "2s on attempt 2",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 2},
			want: 2 * time.Second,
		},
		{
			name: "10s on attempt 5 (hit max)",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 5},
			want: 10 * time.Second,
		},
		{
			name: "9.34s on attempt 4 (with jitter)",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				useJitter: true,
			},
			args: args{attempt: 4},
			want: 9342031108 * time.Nanosecond,
		},
		{
			name: "10s on attempt 5 (hit max with jitter)",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				useJitter: true,
			},
			args: args{attempt: 5},
			want: 10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := exponentialBackoff{
				min:       tt.fields.min,
				max:       tt.fields.max,
				useJitter: tt.fields.useJitter,
			}
			rand.Seed(1)
			if got := b.waitDuration(tt.args.attempt); got != tt.want {
				t.Errorf("exponentialBackoff.waitDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_linearBackoff_waitDuration(t *testing.T) {
	type fields struct {
		min       time.Duration
		max       time.Duration
		interval  time.Duration
		useJitter bool
	}
	type args struct {
		attempt int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
	}{
		{
			name: "1s on attempt 1",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				interval:  1 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 1},
			want: 1 * time.Second,
		},
		{
			name: "10s on attempt 10",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				interval:  1 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 10},
			want: 10 * time.Second,
		},
		{
			name: "10s on attempt 10000 (hit max)",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				interval:  1 * time.Second,
				useJitter: false,
			},
			args: args{attempt: 1000},
			want: 10 * time.Second,
		},
		{
			name: "5s on attempt 4 (with jitter)",
			fields: fields{
				min:       1 * time.Second,
				max:       10 * time.Second,
				interval:  1 * time.Second,
				useJitter: true,
			},
			args: args{attempt: 5},
			want: 4178582128 * time.Nanosecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := linearBackoff{
				min:       tt.fields.min,
				max:       tt.fields.max,
				interval:  tt.fields.interval,
				useJitter: tt.fields.useJitter,
			}
			rand.Seed(1)
			if got := b.waitDuration(tt.args.attempt); got != tt.want {
				fmt.Println(got.Nanoseconds())
				t.Errorf("linearBackoff.waitDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
