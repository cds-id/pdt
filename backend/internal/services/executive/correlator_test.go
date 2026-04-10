package executive

import (
	"context"
	"testing"
	"time"
)

func mkRange() DateRange {
	return DateRange{
		Start: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	}
}

func TestCorrelator_HappyPath(t *testing.T) {
	r := mkRange()
	fake := &fakeWeaviate{
		jira: []JiraCard{
			{CardKey: "PROJ-1", Title: "Login flow", Content: "login flow", Status: "Done", UpdatedAt: r.Start.Add(2 * 24 * time.Hour)},
			{CardKey: "PROJ-2", Title: "Billing", Content: "billing rewrite", Status: "In Progress", UpdatedAt: r.Start},
		},
		commits: []Commit{
			{SHA: "aaa", Message: "PROJ-1: add login form", CommittedAt: r.Start.Add(3 * 24 * time.Hour)},
			{SHA: "bbb", Message: "random cleanup", CommittedAt: r.Start.Add(4 * 24 * time.Hour)},
		},
		wa: []WAMessage{
			{MessageID: "w1", SenderName: "alice", Content: "hey about the auth", Timestamp: r.Start.Add(1 * time.Hour)},
			{MessageID: "w2", SenderName: "alice", Content: "also still about auth", Timestamp: r.Start.Add(1*time.Hour + 10*time.Minute)},
			{MessageID: "w3", SenderName: "alice", Content: "third same convo", Timestamp: r.Start.Add(1*time.Hour + 20*time.Minute)},
		},
		semanticCommits: map[string][]CommitHit{},
		semanticWA:      map[string][]WAHit{},
	}

	c := NewCorrelator(fake)
	c.Now = func() time.Time { return r.End }

	ds, err := c.Build(context.Background(), 42, nil, r, 7)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(ds.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(ds.Topics))
	}

	var proj1 *Topic
	for i := range ds.Topics {
		if ds.Topics[i].Anchor.CardKey == "PROJ-1" {
			proj1 = &ds.Topics[i]
		}
	}
	if proj1 == nil || len(proj1.Commits) != 1 || proj1.Commits[0].SHA != "aaa" {
		t.Fatalf("PROJ-1 should link commit aaa via explicit match: %+v", proj1)
	}

	if ds.Metrics.CommitsTotal != 2 || ds.Metrics.CommitsLinked != 1 {
		t.Fatalf("commit metrics wrong: %+v", ds.Metrics)
	}
	if len(ds.OrphanCommits) != 1 || ds.OrphanCommits[0].SHA != "bbb" {
		t.Fatalf("expected bbb as orphan commit: %+v", ds.OrphanCommits)
	}
	if len(ds.OrphanWA) != 1 || len(ds.OrphanWA[0].Messages) != 3 {
		t.Fatalf("expected single orphan WA group with 3 messages: %+v", ds.OrphanWA)
	}
}
