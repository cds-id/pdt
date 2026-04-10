package agent

import (
	"context"
	"testing"
	"time"

	"github.com/cds-id/pdt/backend/internal/services/executive"
)

type stubLLM struct {
	events []ExecutiveEvent
}

func (s *stubLLM) Stream(_ context.Context, _, _ string, out chan<- ExecutiveEvent) {
	for _, e := range s.events {
		out <- e
	}
	close(out)
}

func TestExecutiveReportAgent_ForwardsEvents(t *testing.T) {
	suggestion := executive.Suggestion{Kind: "gap", Title: "x", Detail: "d", Refs: []string{"jira:A-1"}}
	stub := &stubLLM{events: []ExecutiveEvent{
		{Kind: "delta", Delta: "## Summary\nhello"},
		{Kind: "suggestion", Suggestion: &suggestion},
		{Kind: "done"},
	}}
	agentObj := &ExecutiveReportAgent{LLM: stub}
	ds := &executive.CorrelatedDataset{
		Range: executive.DateRange{Start: time.Now().Add(-7 * 24 * time.Hour), End: time.Now()},
	}
	ch := make(chan ExecutiveEvent, 10)
	agentObj.Run(context.Background(), ds, ch)

	var kinds []string
	for e := range ch {
		kinds = append(kinds, e.Kind)
	}
	if len(kinds) != 3 || kinds[0] != "delta" || kinds[1] != "suggestion" || kinds[2] != "done" {
		t.Fatalf("unexpected event order: %v", kinds)
	}
}

func TestTrimForLLM_CapsLongContent(t *testing.T) {
	long := make([]byte, 1000)
	for i := range long {
		long[i] = 'a'
	}
	ds := &executive.CorrelatedDataset{
		Topics: []executive.Topic{{
			Messages: []executive.WAMessage{{Content: string(long)}},
		}},
		OrphanWA: []executive.WAGroup{{
			Messages: []executive.WAMessage{{Content: string(long)}},
		}},
	}
	trimmed := trimForLLM(ds)
	if len(trimmed.Topics[0].Messages[0].Content) != 500 {
		t.Fatalf("topic message content not capped: %d", len(trimmed.Topics[0].Messages[0].Content))
	}
	if len(trimmed.OrphanWA[0].Messages[0].Content) != 500 {
		t.Fatalf("orphan wa message content not capped: %d", len(trimmed.OrphanWA[0].Messages[0].Content))
	}
	// original should be untouched
	if len(ds.Topics[0].Messages[0].Content) != 1000 {
		t.Fatalf("original was mutated")
	}
}
