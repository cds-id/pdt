package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cds-id/pdt/backend/internal/services/executive"
)

// ExecutiveEvent is a streaming event emitted by the agent during a report generation.
type ExecutiveEvent struct {
	Kind       string                // "delta" | "suggestion" | "done" | "error"
	Delta      string                // when Kind == "delta"
	Suggestion *executive.Suggestion // when Kind == "suggestion"
	Err        error                 // when Kind == "error"
}

// ExecutiveLLM is the narrow interface the agent depends on. Test code uses a stub;
// production wiring supplies a MiniMax/Anthropic-backed implementation (Task 13).
// Stream MUST close out when done (either via a final "done" event or by returning).
type ExecutiveLLM interface {
	Stream(ctx context.Context, system, user string, out chan<- ExecutiveEvent)
}

type ExecutiveReportAgent struct {
	LLM ExecutiveLLM
}

func (a *ExecutiveReportAgent) Run(ctx context.Context, ds *executive.CorrelatedDataset, out chan<- ExecutiveEvent) {
	system := buildExecutiveSystemPrompt(ds.Range)
	user, err := renderExecutiveUserPayload(ds)
	if err != nil {
		out <- ExecutiveEvent{Kind: "error", Err: err}
		close(out)
		return
	}
	a.LLM.Stream(ctx, system, user, out)
}

func buildExecutiveSystemPrompt(r executive.DateRange) string {
	return "You are an engineering executive assistant. You will receive a structured dataset describing a developer's work between " +
		r.Start.Format(time.DateOnly) + " and " + r.End.Format(time.DateOnly) +
		". Produce sections in exactly this order: ## Summary, ## Topics, ## Gaps, ## Stale Work, ## Next Steps. " +
		"Cite evidence inline as [jira:KEY], [commit:sha], [wa:sender@time]. " +
		"Whenever you identify a gap, stale item, or next-step recommendation, call the emit_suggestion tool with kind=gap|stale|next_step."
}

func renderExecutiveUserPayload(ds *executive.CorrelatedDataset) (string, error) {
	trimmed := trimForLLM(ds)
	b, err := json.Marshal(trimmed)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// trimForLLM returns a deep-enough copy of the dataset with WA message content capped at 500 chars.
func trimForLLM(ds *executive.CorrelatedDataset) *executive.CorrelatedDataset {
	cp := *ds
	cp.Topics = make([]executive.Topic, len(ds.Topics))
	for i, t := range ds.Topics {
		t2 := t
		t2.Messages = capMessages(t.Messages)
		cp.Topics[i] = t2
	}
	cp.OrphanWA = make([]executive.WAGroup, len(ds.OrphanWA))
	for i, g := range ds.OrphanWA {
		g2 := g
		g2.Messages = capMessages(g.Messages)
		cp.OrphanWA[i] = g2
	}
	return &cp
}

func capMessages(msgs []executive.WAMessage) []executive.WAMessage {
	out := make([]executive.WAMessage, len(msgs))
	for i, m := range msgs {
		if len(m.Content) > 500 {
			m.Content = m.Content[:500]
		}
		out[i] = m
	}
	return out
}
