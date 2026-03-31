// backend/internal/scheduler/cron_test.go
package scheduler

import (
	"testing"
	"time"
)

func TestNextCronRun(t *testing.T) {
	ref := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		expr     string
		after    time.Time
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "every weekday at 8am",
			expr:     "0 8 * * 1-5",
			after:    ref,
			expected: time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
		},
		{
			name:     "every hour",
			expr:     "0 * * * *",
			after:    ref,
			expected: time.Date(2026, 3, 30, 11, 0, 0, 0, time.UTC),
		},
		{
			name:    "invalid expression",
			expr:    "not a cron",
			after:   ref,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextCronRun(tt.expr, tt.after)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("NextCronRun(%q, %v)\n  got:  %v\n  want: %v", tt.expr, tt.after, got, tt.expected)
			}
		})
	}
}

func TestNextRunAt(t *testing.T) {
	ref := time.Date(2026, 3, 30, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		triggerType     string
		cronExpr        string
		intervalSeconds int
		after           time.Time
		wantNil         bool
		wantErr         bool
	}{
		{name: "cron trigger", triggerType: "cron", cronExpr: "0 8 * * 1-5", after: ref},
		{name: "interval trigger", triggerType: "interval", intervalSeconds: 900, after: ref},
		{name: "event trigger returns nil", triggerType: "event", after: ref, wantNil: true},
		{name: "unknown trigger", triggerType: "unknown", after: ref, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextRunAt(tt.triggerType, tt.cronExpr, tt.intervalSeconds, tt.after)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil time")
			}
			if tt.triggerType == "interval" {
				expected := ref.Add(900 * time.Second)
				if !got.Equal(expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			}
		})
	}
}
