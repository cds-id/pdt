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

func TestCorrelator_ExplicitAndSemanticDedupe(t *testing.T) {
	r := mkRange()
	fake := &fakeWeaviate{
		jira: []JiraCard{{CardKey: "PROJ-1", Title: "X", Content: "x content", Status: "Done", UpdatedAt: r.Start}},
		commits: []Commit{{SHA: "aaa", Message: "PROJ-1: impl", CommittedAt: r.Start.Add(time.Hour)}},
		semanticCommits: map[string][]CommitHit{
			"x content": {{Commit: Commit{SHA: "aaa", Message: "PROJ-1: impl"}, Distance: 0.1}},
		},
	}
	ds, err := NewCorrelator(fake).Build(context.Background(), 1, nil, r, 7)
	if err != nil { t.Fatal(err) }
	if len(ds.Topics[0].Commits) != 1 {
		t.Fatalf("explicit + semantic should dedupe: %+v", ds.Topics[0].Commits)
	}
}

func TestCorrelator_StalenessBoundary(t *testing.T) {
	r := mkRange()
	base := r.End
	card := JiraCard{CardKey: "S-1", Title: "T", Status: "In Progress", UpdatedAt: base.Add(-7 * 24 * time.Hour)}
	fake := &fakeWeaviate{jira: []JiraCard{card}}
	c := NewCorrelator(fake)
	c.Now = func() time.Time { return base }
	ds, _ := c.Build(context.Background(), 1, nil, r, 7)
	if !ds.Topics[0].Stale {
		t.Fatalf("expected stale at exactly threshold days")
	}

	card.UpdatedAt = base.Add(-6*24*time.Hour - time.Hour)
	fake.jira = []JiraCard{card}
	ds, _ = c.Build(context.Background(), 1, nil, r, 7)
	if ds.Topics[0].Stale {
		t.Fatalf("expected not stale at threshold-1 days")
	}
}

func TestCorrelator_Truncation(t *testing.T) {
	r := mkRange()
	var jira []JiraCard
	for i := 0; i < MaxAnchors+5; i++ {
		jira = append(jira, JiraCard{CardKey: "T-1", Title: "x", UpdatedAt: r.Start})
	}
	ds, err := NewCorrelator(&fakeWeaviate{jira: jira}).Build(context.Background(), 1, nil, r, 7)
	if err != nil { t.Fatal(err) }
	if len(ds.Topics) != MaxAnchors {
		t.Fatalf("expected %d topics, got %d", MaxAnchors, len(ds.Topics))
	}
	if !ds.Metrics.Truncated {
		t.Fatalf("expected Truncated=true")
	}
}

func TestCorrelator_EmptyRange(t *testing.T) {
	r := mkRange()
	ds, err := NewCorrelator(&fakeWeaviate{}).Build(context.Background(), 1, nil, r, 7)
	if err != nil { t.Fatal(err) }
	if ds.Metrics.LinkagePctCommits != 0 || ds.Metrics.LinkagePctCards != 0 {
		t.Fatalf("expected zero pcts on empty, got %+v", ds.Metrics)
	}
	if len(ds.DailyBuckets) != 30 {
		t.Fatalf("expected 30 daily buckets (Apr 1..30), got %d", len(ds.DailyBuckets))
	}
}

func TestCorrelator_OrphanWANoiseFloor(t *testing.T) {
	r := mkRange()
	base := r.Start
	fake := &fakeWeaviate{
		wa: []WAMessage{
			{MessageID: "w1", SenderName: "alice", Content: "hi", Timestamp: base},
			{MessageID: "w2", SenderName: "alice", Content: "hi again", Timestamp: base.Add(time.Minute)},
		},
	}
	ds, _ := NewCorrelator(fake).Build(context.Background(), 1, nil, r, 7)
	if len(ds.OrphanWA) != 0 {
		t.Fatalf("2-message group should be filtered as noise: %+v", ds.OrphanWA)
	}
}

func TestCorrelator_WorkspaceScoping(t *testing.T) {
	r := mkRange()
	ws3 := uint(3)
	ws5 := uint(5)
	fake := &fakeWeaviate{
		jira: []JiraCard{
			{CardKey: "A-1", Title: "in 3", WorkspaceID: &ws3, UpdatedAt: r.Start},
			{CardKey: "B-1", Title: "in 5", WorkspaceID: &ws5, UpdatedAt: r.Start},
		},
	}
	ds, err := NewCorrelator(fake).Build(context.Background(), 1, &ws3, r, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Topics) != 1 || ds.Topics[0].Anchor.CardKey != "A-1" {
		t.Fatalf("expected only A-1 (ws=3), got %+v", ds.Topics)
	}
}
