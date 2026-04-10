package executive

import (
	"testing"
	"time"
)

func TestComputeMetrics_EmptyDataset(t *testing.T) {
	m := computeMetrics(nil, nil, nil)
	if m.LinkagePctCommits != 0 || m.LinkagePctCards != 0 {
		t.Fatalf("expected zeroed pct on empty input, got %+v", m)
	}
}

func TestComputeMetrics_HappyPath(t *testing.T) {
	topics := []Topic{
		{Anchor: JiraCard{CardKey: "A-1"}, Commits: []Commit{{SHA: "aaa"}}},
		{Anchor: JiraCard{CardKey: "A-2"}, Commits: nil, Stale: true},
	}
	orphanCommits := []Commit{{SHA: "bbb"}}
	orphanWA := []WAGroup{{Summary: "x"}}
	m := computeMetrics(topics, orphanCommits, orphanWA)

	if m.CommitsTotal != 2 || m.CommitsLinked != 1 {
		t.Fatalf("commits counts wrong: %+v", m)
	}
	if m.CardsActive != 2 || m.CardsWithCommits != 1 {
		t.Fatalf("card counts wrong: %+v", m)
	}
	if m.StaleCardCount != 1 {
		t.Fatalf("stale count wrong: %+v", m)
	}
	if m.LinkagePctCommits != 0.5 || m.LinkagePctCards != 0.5 {
		t.Fatalf("pct wrong: %+v", m)
	}
}

func TestBuildDailyBuckets_FillsZeroDays(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	commits := []Commit{{CommittedAt: start}}
	buckets := buildDailyBuckets(DateRange{Start: start, End: end}, commits, nil, nil)
	if len(buckets) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(buckets))
	}
	if buckets[0].Commits != 1 || buckets[1].Commits != 0 {
		t.Fatalf("bucket 0/1 commit counts wrong: %+v", buckets)
	}
}

func TestGroupMessages_SameSenderWithinWindow(t *testing.T) {
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	msgs := []WAMessage{
		{SenderName: "alice", Timestamp: base, Content: "hello world long enough to clip"},
		{SenderName: "alice", Timestamp: base.Add(5 * time.Minute)},
		{SenderName: "alice", Timestamp: base.Add(45 * time.Minute)},
		{SenderName: "bob", Timestamp: base.Add(6 * time.Minute)},
	}
	groups := groupMessages(msgs, 30*time.Minute)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %+v", len(groups), groups)
	}
}
