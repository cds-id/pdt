# Executive Reports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a new "Executive" tab in `ReportsPage` that correlates Jira, commit, and WhatsApp data from Weaviate over a user-selected date range, streams an LLM narrative with structured suggestions, and persists the result for history.

**Architecture:** A pure `correlator` service builds a `CorrelatedDataset` from three parallel Weaviate queries (Jira anchors + raw commits/WA) then runs a Jira-anchored linking pass (explicit key match + semantic match) plus an orphan pass for gap detection. An `ExecutiveReportAgent` consumes the dataset and streams narrative + `emit_suggestion` tool calls. A handler orchestrates correlator → agent → SSE → Postgres persistence. Frontend consumes SSE via a custom hook, renders charts immediately on the `dataset` event, and streams the narrative into markdown.

**Tech Stack:** Go (Gin, GORM, Weaviate Go client v4), Postgres, Anthropic SDK via MiniMax endpoint, React 18 + TypeScript + Vite, RTK Query, Recharts, shadcn/ui, Tailwind.

**Spec:** `docs/superpowers/specs/2026-04-10-executive-reports-design.md`

---

## File Structure

### Backend (Go)

| File | Responsibility |
|---|---|
| `backend/internal/services/executive/types.go` | DTOs: `DateRange`, `CorrelatedDataset`, `Topic`, `WAGroup`, `Metrics`, `DailyBucket`, `Suggestion` |
| `backend/internal/services/executive/tunables.go` | Package-level knobs (`MaxAnchors`, distance thresholds, worker count, etc.) |
| `backend/internal/services/executive/weaviate_client.go` | `WeaviateClient` interface the correlator depends on — lets tests use a fake |
| `backend/internal/services/executive/correlator.go` | `Correlator.Build(ctx, userID, workspaceID, DateRange, staleThresholdDays) (*CorrelatedDataset, error)` |
| `backend/internal/services/executive/metrics.go` | Pure aggregation helpers: `computeMetrics`, `buildDailyBuckets`, `groupMessages` |
| `backend/internal/services/executive/testdata/*.json` | Fixture Jira cards, commits, WA messages + expected outputs |
| `backend/internal/services/executive/metrics_test.go` | Pure unit tests for metrics/grouping |
| `backend/internal/services/executive/correlator_test.go` | Table-driven tests using a fake `WeaviateClient` |
| `backend/internal/ai/agent/executive_report.go` | `ExecutiveReportAgent` — builds prompt, streams deltas and `emit_suggestion` tool calls |
| `backend/internal/ai/agent/executive_report_test.go` | Structural test with a stub LLM client |
| `backend/internal/models/executive_report.go` | GORM model `ExecutiveReport` |
| `backend/internal/handlers/executive_report.go` | `Generate` (SSE), `List`, `Get`, `Delete` |
| `backend/internal/handlers/executive_report_test.go` | Handler tests with fake correlator + stub agent |
| `backend/cmd/server/main.go` (modify) | Route registration under `/api/protected/reports/executive` |
| `backend/internal/database/migrations/NNNN_executive_reports.up.sql` | Schema migration |

### Frontend (React/TS)

| File | Responsibility |
|---|---|
| `frontend/src/infrastructure/services/executiveReport.service.ts` | RTK Query slice: list/get/delete |
| `frontend/src/presentation/hooks/useGenerateExecutiveReport.ts` | SSE consumer: phase/dataset/narrative/suggestions state |
| `frontend/src/presentation/components/executive/ExecutiveReportTab.tsx` | Tab root: history sidebar + controls + view |
| `frontend/src/presentation/components/executive/ExecutiveReportView.tsx` | Renders metrics row, timeline, stale table, narrative, suggestions |
| `frontend/src/presentation/components/executive/SuggestionList.tsx` | Grouped suggestion cards |
| `frontend/src/presentation/components/executive/StaleWorkTable.tsx` | Stale cards table |
| `frontend/src/presentation/components/executive/ExecutiveActivityChart.tsx` | Thin wrapper over `CommitActivityChart` with stacked WA/Jira series |
| `frontend/src/presentation/components/executive/__fixtures__/*.ts` | Completed report, failed report, captured SSE script |
| `frontend/src/presentation/components/executive/*.test.tsx` | Vitest + Testing Library tests |
| `frontend/src/presentation/pages/ReportsPage.tsx` (modify) | Add the Executive tab |
| `frontend/src/infrastructure/store/index.ts` (modify) | Register the new RTK slice |

---

## Task 1: Backend — Types

**Files:**
- Create: `backend/internal/services/executive/types.go`
- Create: `backend/internal/services/executive/tunables.go`

- [ ] **Step 1: Write types.go**

```go
package executive

import "time"

type DateRange struct {
    Start time.Time
    End   time.Time
}

type JiraCard struct {
    CardKey   string    `json:"card_key"`
    Title     string    `json:"title"`
    Status    string    `json:"status"`
    Assignee  string    `json:"assignee"`
    Content   string    `json:"content"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Commit struct {
    SHA         string    `json:"sha"`
    Message     string    `json:"message"`
    RepoName    string    `json:"repo_name"`
    Author      string    `json:"author"`
    CommittedAt time.Time `json:"committed_at"`
}

type WAMessage struct {
    MessageID  string    `json:"message_id"`
    SenderName string    `json:"sender_name"`
    Content    string    `json:"content"`
    Timestamp  time.Time `json:"timestamp"`
}

type Topic struct {
    Anchor   JiraCard    `json:"anchor"`
    Messages []WAMessage `json:"messages"`
    Commits  []Commit    `json:"commits"`
    Stale    bool        `json:"stale"`
    DaysIdle int         `json:"days_idle"`
}

type WAGroup struct {
    Summary   string      `json:"summary"`
    Messages  []WAMessage `json:"messages"`
    StartedAt time.Time   `json:"started_at"`
}

type Metrics struct {
    CommitsTotal      int     `json:"commits_total"`
    CommitsLinked     int     `json:"commits_linked"`
    CardsActive       int     `json:"cards_active"`
    CardsWithCommits  int     `json:"cards_with_commits"`
    WATopicsTicketed  int     `json:"wa_topics_ticketed"`
    WATopicsOrphan    int     `json:"wa_topics_orphan"`
    StaleCardCount    int     `json:"stale_card_count"`
    LinkagePctCommits float64 `json:"linkage_pct_commits"`
    LinkagePctCards   float64 `json:"linkage_pct_cards"`
    Truncated         bool    `json:"truncated"`
}

type DailyBucket struct {
    Day         time.Time `json:"day"`
    Commits     int       `json:"commits"`
    JiraChanges int       `json:"jira_changes"`
    WAMessages  int       `json:"wa_messages"`
}

type CorrelatedDataset struct {
    UserID        uint          `json:"user_id"`
    WorkspaceID   *uint         `json:"workspace_id,omitempty"`
    Range         DateRange     `json:"range"`
    Topics        []Topic       `json:"topics"`
    OrphanWA      []WAGroup     `json:"orphan_wa"`
    OrphanCommits []Commit      `json:"orphan_commits"`
    Metrics       Metrics       `json:"metrics"`
    DailyBuckets  []DailyBucket `json:"daily_buckets"`
}

type Suggestion struct {
    Kind   string   `json:"kind"`   // "gap" | "stale" | "next_step"
    Title  string   `json:"title"`
    Detail string   `json:"detail"`
    Refs   []string `json:"refs"`
}
```

- [ ] **Step 2: Write tunables.go**

```go
package executive

import "time"

var (
    MaxAnchors          = 200
    PerAnchorWorkers    = 8
    CommitDistanceMax   = 0.30
    WADistanceMax       = 0.32
    SemanticCommitLimit = 10
    SemanticWALimit     = 15
    WAGroupWindow       = 10 * time.Minute
    OrphanWAGroupWindow = 30 * time.Minute
    OrphanWANoiseFloor  = 3
    StaleThresholdDefault = 7
    MaxRangeDays          = 90
    DefaultRangeDays      = 14
)
```

- [ ] **Step 3: Build**

Run: `cd backend && go build ./internal/services/executive/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/executive/
git commit -m "feat(executive): add DTOs and tunables for executive reports"
```

---

## Task 2: Backend — Weaviate client interface

**Files:**
- Create: `backend/internal/services/executive/weaviate_client.go`

Purpose: decouple the correlator from the concrete Weaviate client so tests can substitute a fake without mocking a whole SDK.

- [ ] **Step 1: Write the interface**

```go
package executive

import "context"

// WeaviateClient is the narrow interface the correlator depends on. The
// production implementation wraps backend/internal/services/weaviate.
type WeaviateClient interface {
    // ListJiraCards returns Jira cards for a user/workspace whose primary
    // timestamp falls in [start, end]. Empty query, ordered by most recent.
    ListJiraCards(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time, limit int) ([]JiraCard, error)

    // ListCommits returns commits for a user in [start, end]. Empty query.
    ListCommits(ctx context.Context, userID uint, start, end time.Time) ([]Commit, error)

    // ListWAMessages returns WA messages for a user in [start, end]. Empty query.
    ListWAMessages(ctx context.Context, userID uint, start, end time.Time) ([]WAMessage, error)

    // SemanticCommits runs a near-text query on CommitEmbedding filtered by user + date.
    // Returns results sorted by distance ascending. The float is the distance.
    SemanticCommits(ctx context.Context, userID uint, query string, start, end time.Time, limit int) ([]CommitHit, error)

    // SemanticWA runs a near-text query on WaMessageEmbedding filtered by user + date.
    SemanticWA(ctx context.Context, userID uint, query string, start, end time.Time, limit int) ([]WAHit, error)
}

type CommitHit struct {
    Commit   Commit
    Distance float64
}

type WAHit struct {
    Message  WAMessage
    Distance float64
}
```

- [ ] **Step 2: Add `import "time"` and build**

Run: `cd backend && go build ./internal/services/executive/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/executive/weaviate_client.go
git commit -m "feat(executive): define WeaviateClient interface for correlator"
```

---

## Task 3: Backend — Metrics helpers + unit tests

**Files:**
- Create: `backend/internal/services/executive/metrics.go`
- Create: `backend/internal/services/executive/metrics_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
        {SenderName: "alice", Timestamp: base.Add(45 * time.Minute)}, // outside 30min
        {SenderName: "bob",   Timestamp: base.Add(6 * time.Minute)},  // different sender
    }
    groups := groupMessages(msgs, 30*time.Minute)
    if len(groups) != 3 {
        t.Fatalf("expected 3 groups, got %d: %+v", len(groups), groups)
    }
}
```

- [ ] **Step 2: Run and watch them fail**

Run: `cd backend && go test ./internal/services/executive/ -run TestComputeMetrics -v`
Expected: compile error (`computeMetrics` undefined).

- [ ] **Step 3: Implement metrics.go**

```go
package executive

import (
    "fmt"
    "sort"
    "time"
)

func computeMetrics(topics []Topic, orphanCommits []Commit, orphanWA []WAGroup) Metrics {
    var m Metrics
    m.CardsActive = len(topics)

    linkedCommitShas := make(map[string]struct{})
    for _, t := range topics {
        if len(t.Commits) > 0 {
            m.CardsWithCommits++
        }
        if t.Stale {
            m.StaleCardCount++
        }
        for _, c := range t.Commits {
            linkedCommitShas[c.SHA] = struct{}{}
        }
        if len(t.Messages) > 0 {
            m.WATopicsTicketed++
        }
    }
    m.CommitsLinked = len(linkedCommitShas)
    m.CommitsTotal = m.CommitsLinked + len(orphanCommits)
    m.WATopicsOrphan = len(orphanWA)

    if m.CommitsTotal > 0 {
        m.LinkagePctCommits = float64(m.CommitsLinked) / float64(m.CommitsTotal)
    }
    if m.CardsActive > 0 {
        m.LinkagePctCards = float64(m.CardsWithCommits) / float64(m.CardsActive)
    }
    return m
}

func buildDailyBuckets(r DateRange, commits []Commit, jiraChanges []JiraCard, wa []WAMessage) []DailyBucket {
    startDay := truncateDay(r.Start)
    endDay := truncateDay(r.End)

    index := map[time.Time]*DailyBucket{}
    var buckets []DailyBucket
    for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
        buckets = append(buckets, DailyBucket{Day: d})
    }
    for i := range buckets {
        index[buckets[i].Day] = &buckets[i]
    }
    for _, c := range commits {
        if b, ok := index[truncateDay(c.CommittedAt)]; ok {
            b.Commits++
        }
    }
    for _, j := range jiraChanges {
        if b, ok := index[truncateDay(j.UpdatedAt)]; ok {
            b.JiraChanges++
        }
    }
    for _, m := range wa {
        if b, ok := index[truncateDay(m.Timestamp)]; ok {
            b.WAMessages++
        }
    }
    return buckets
}

func truncateDay(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// groupMessages clusters messages by (sender, time window). Messages must be
// sorted; we sort defensively.
func groupMessages(msgs []WAMessage, window time.Duration) []WAGroup {
    if len(msgs) == 0 {
        return nil
    }
    sorted := make([]WAMessage, len(msgs))
    copy(sorted, msgs)
    sort.Slice(sorted, func(i, j int) bool { return sorted[i].Timestamp.Before(sorted[j].Timestamp) })

    var groups []WAGroup
    var cur *WAGroup
    for _, m := range sorted {
        if cur != nil && cur.Messages[len(cur.Messages)-1].SenderName == m.SenderName &&
            m.Timestamp.Sub(cur.Messages[len(cur.Messages)-1].Timestamp) <= window {
            cur.Messages = append(cur.Messages, m)
            continue
        }
        groups = append(groups, WAGroup{
            Summary:   summaryFor(m),
            Messages:  []WAMessage{m},
            StartedAt: m.Timestamp,
        })
        cur = &groups[len(groups)-1]
    }
    return groups
}

func summaryFor(m WAMessage) string {
    c := m.Content
    if len(c) > 80 {
        c = c[:80]
    }
    return fmt.Sprintf("%s: %s", m.SenderName, c)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/services/executive/ -run "TestComputeMetrics|TestBuildDailyBuckets|TestGroupMessages" -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/executive/metrics.go backend/internal/services/executive/metrics_test.go
git commit -m "feat(executive): metrics and bucket aggregation helpers"
```

---

## Task 4: Backend — Fake Weaviate client + correlator skeleton

**Files:**
- Create: `backend/internal/services/executive/correlator.go`
- Create: `backend/internal/services/executive/fake_weaviate_test.go`
- Create: `backend/internal/services/executive/correlator_test.go`

- [ ] **Step 1: Write the fake client (test helper)**

```go
package executive

import (
    "context"
    "time"
)

type fakeWeaviate struct {
    jira    []JiraCard
    commits []Commit
    wa      []WAMessage

    // map from anchor card key → list of semantic commit hits to return
    semanticCommits map[string][]CommitHit
    semanticWA      map[string][]WAHit
}

func (f *fakeWeaviate) ListJiraCards(_ context.Context, _ uint, _ *uint, start, end time.Time, limit int) ([]JiraCard, error) {
    var out []JiraCard
    for _, c := range f.jira {
        if (c.UpdatedAt.Equal(start) || c.UpdatedAt.After(start)) && !c.UpdatedAt.After(end) {
            out = append(out, c)
        }
    }
    if limit > 0 && len(out) > limit {
        out = out[:limit]
    }
    return out, nil
}

func (f *fakeWeaviate) ListCommits(_ context.Context, _ uint, start, end time.Time) ([]Commit, error) {
    var out []Commit
    for _, c := range f.commits {
        if (c.CommittedAt.Equal(start) || c.CommittedAt.After(start)) && !c.CommittedAt.After(end) {
            out = append(out, c)
        }
    }
    return out, nil
}

func (f *fakeWeaviate) ListWAMessages(_ context.Context, _ uint, start, end time.Time) ([]WAMessage, error) {
    var out []WAMessage
    for _, m := range f.wa {
        if (m.Timestamp.Equal(start) || m.Timestamp.After(start)) && !m.Timestamp.After(end) {
            out = append(out, m)
        }
    }
    return out, nil
}

func (f *fakeWeaviate) SemanticCommits(_ context.Context, _ uint, query string, _ , _ time.Time, _ int) ([]CommitHit, error) {
    return f.semanticCommits[query], nil
}

func (f *fakeWeaviate) SemanticWA(_ context.Context, _ uint, query string, _, _ time.Time, _ int) ([]WAHit, error) {
    return f.semanticWA[query], nil
}
```

- [ ] **Step 2: Write the correlator skeleton**

```go
package executive

import (
    "context"
    "regexp"
    "strings"
    "sync"
    "time"

    "golang.org/x/sync/errgroup"
)

type Correlator struct {
    Client WeaviateClient
    Now    func() time.Time // injectable for tests
}

func NewCorrelator(client WeaviateClient) *Correlator {
    return &Correlator{Client: client, Now: time.Now}
}

func (c *Correlator) Build(ctx context.Context, userID uint, workspaceID *uint, r DateRange, staleDays int) (*CorrelatedDataset, error) {
    if staleDays <= 0 {
        staleDays = StaleThresholdDefault
    }

    var (
        anchors    []JiraCard
        rawCommits []Commit
        rawWA      []WAMessage
        truncated  bool
    )

    g, gctx := errgroup.WithContext(ctx)
    g.Go(func() error {
        list, err := c.Client.ListJiraCards(gctx, userID, workspaceID, r.Start, r.End, MaxAnchors+1)
        if err != nil {
            return err
        }
        if len(list) > MaxAnchors {
            truncated = true
            list = list[:MaxAnchors]
        }
        anchors = list
        return nil
    })
    g.Go(func() error {
        list, err := c.Client.ListCommits(gctx, userID, r.Start, r.End)
        rawCommits = list
        return err
    })
    g.Go(func() error {
        list, err := c.Client.ListWAMessages(gctx, userID, r.Start, r.End)
        rawWA = list
        return err
    })
    if err := g.Wait(); err != nil {
        return nil, err
    }

    topics := c.buildTopics(ctx, userID, anchors, rawCommits, r, staleDays)

    orphanCommits := subtractCommits(rawCommits, topicCommitSet(topics))
    orphanWAMsgs := subtractWA(rawWA, topicWASet(topics))
    orphanWA := filterNoise(groupMessages(orphanWAMsgs, OrphanWAGroupWindow), OrphanWANoiseFloor)

    ds := &CorrelatedDataset{
        UserID:        userID,
        WorkspaceID:   workspaceID,
        Range:         r,
        Topics:        topics,
        OrphanWA:      orphanWA,
        OrphanCommits: orphanCommits,
        DailyBuckets:  buildDailyBuckets(r, rawCommits, anchors, rawWA),
    }
    ds.Metrics = computeMetrics(topics, orphanCommits, orphanWA)
    ds.Metrics.Truncated = truncated
    return ds, nil
}

func (c *Correlator) buildTopics(ctx context.Context, userID uint, anchors []JiraCard, rawCommits []Commit, r DateRange, staleDays int) []Topic {
    sem := make(chan struct{}, PerAnchorWorkers)
    var mu sync.Mutex
    topics := make([]Topic, len(anchors))

    var wg sync.WaitGroup
    for i, card := range anchors {
        i, card := i, card
        wg.Add(1)
        sem <- struct{}{}
        go func() {
            defer wg.Done()
            defer func() { <-sem }()

            commits := matchCommitsForCard(ctx, c.Client, userID, card, rawCommits, r)
            wa := matchWAForCard(ctx, c.Client, userID, card, r)

            daysIdle := int(c.Now().Sub(card.UpdatedAt).Hours() / 24)
            stale := strings.EqualFold(card.Status, "In Progress") && len(commits) == 0 && daysIdle >= staleDays

            mu.Lock()
            topics[i] = Topic{
                Anchor:   card,
                Messages: flattenGroupMessages(groupMessages(wa, WAGroupWindow)),
                Commits:  commits,
                Stale:    stale,
                DaysIdle: daysIdle,
            }
            mu.Unlock()
        }()
    }
    wg.Wait()
    return topics
}

var cardKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

func matchCommitsForCard(ctx context.Context, client WeaviateClient, userID uint, card JiraCard, rawCommits []Commit, r DateRange) []Commit {
    found := map[string]Commit{}
    for _, c := range rawCommits {
        for _, key := range cardKeyRe.FindAllString(c.Message, -1) {
            if key == card.CardKey {
                found[c.SHA] = c
            }
        }
    }
    hits, err := client.SemanticCommits(ctx, userID, card.Content, r.Start, r.End, SemanticCommitLimit)
    if err == nil {
        for _, h := range hits {
            if h.Distance > CommitDistanceMax {
                continue
            }
            if _, ok := found[h.Commit.SHA]; !ok {
                found[h.Commit.SHA] = h.Commit
            }
        }
    }
    out := make([]Commit, 0, len(found))
    for _, v := range found {
        out = append(out, v)
    }
    return out
}

func matchWAForCard(ctx context.Context, client WeaviateClient, userID uint, card JiraCard, r DateRange) []WAMessage {
    hits, err := client.SemanticWA(ctx, userID, card.Content, r.Start, r.End, SemanticWALimit)
    if err != nil {
        return nil
    }
    var out []WAMessage
    for _, h := range hits {
        if h.Distance > WADistanceMax {
            continue
        }
        out = append(out, h.Message)
    }
    return out
}

func flattenGroupMessages(groups []WAGroup) []WAMessage {
    var out []WAMessage
    for _, g := range groups {
        out = append(out, g.Messages...)
    }
    return out
}

func topicCommitSet(topics []Topic) map[string]struct{} {
    s := map[string]struct{}{}
    for _, t := range topics {
        for _, c := range t.Commits {
            s[c.SHA] = struct{}{}
        }
    }
    return s
}

func topicWASet(topics []Topic) map[string]struct{} {
    s := map[string]struct{}{}
    for _, t := range topics {
        for _, m := range t.Messages {
            s[m.MessageID] = struct{}{}
        }
    }
    return s
}

func subtractCommits(all []Commit, exclude map[string]struct{}) []Commit {
    var out []Commit
    for _, c := range all {
        if _, found := exclude[c.SHA]; !found {
            out = append(out, c)
        }
    }
    return out
}

func subtractWA(all []WAMessage, exclude map[string]struct{}) []WAMessage {
    var out []WAMessage
    for _, m := range all {
        if _, found := exclude[m.MessageID]; !found {
            out = append(out, m)
        }
    }
    return out
}

func filterNoise(groups []WAGroup, floor int) []WAGroup {
    var out []WAGroup
    for _, g := range groups {
        if len(g.Messages) >= floor {
            out = append(out, g)
        }
    }
    return out
}
```

- [ ] **Step 3: Add dependency**

Run: `cd backend && go get golang.org/x/sync/errgroup && go mod tidy`
Expected: clean.

- [ ] **Step 4: Build**

Run: `cd backend && go build ./internal/services/executive/...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/executive/correlator.go backend/internal/services/executive/fake_weaviate_test.go backend/go.mod backend/go.sum
git commit -m "feat(executive): hybrid correlator skeleton + test fake"
```

---

## Task 5: Backend — Correlator happy-path test

**Files:**
- Modify: `backend/internal/services/executive/correlator_test.go`

- [ ] **Step 1: Write the happy-path test**

```go
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
            {MessageID: "w2", SenderName: "alice", Content: "also still about auth", Timestamp: r.Start.Add(2 * time.Hour)},
            {MessageID: "w3", SenderName: "alice", Content: "third same convo", Timestamp: r.Start.Add(3 * time.Hour)},
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
```

- [ ] **Step 2: Run the test**

Run: `cd backend && go test ./internal/services/executive/ -run TestCorrelator_HappyPath -v`
Expected: PASS. If not, fix the correlator — do NOT loosen the test. The most likely issue is the `WAGroupWindow` grouping inside `matchWAForCard` returning no messages (fake returns empty semantic hits for both cards), which is actually correct — the 3 WA messages should all fall into orphan.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/executive/correlator_test.go
git commit -m "test(executive): correlator happy-path"
```

---

## Task 6: Backend — Correlator edge cases

**Files:**
- Modify: `backend/internal/services/executive/correlator_test.go`

- [ ] **Step 1: Add tests for semantic dedupe, staleness, truncation, empty range**

```go
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
    // card updated exactly 7 days before "now"
    base := r.End
    card := JiraCard{CardKey: "S-1", Title: "T", Status: "In Progress", UpdatedAt: base.Add(-7 * 24 * time.Hour)}
    fake := &fakeWeaviate{jira: []JiraCard{card}}
    c := NewCorrelator(fake)
    c.Now = func() time.Time { return base }
    ds, _ := c.Build(context.Background(), 1, nil, r, 7)
    if !ds.Topics[0].Stale {
        t.Fatalf("expected stale at exactly threshold days")
    }

    // one day less → not stale
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
    if ds.Topics == nil { /* ok */ }
    if ds.OrphanWA == nil { /* ok */ }
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
    // only 2 messages in a group → should be filtered
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
```

- [ ] **Step 2: Run all correlator tests**

Run: `cd backend && go test ./internal/services/executive/ -v`
Expected: all PASS. (The truncation test also relies on `ListJiraCards` respecting the `limit` arg — verify the fake already does.)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/services/executive/correlator_test.go
git commit -m "test(executive): correlator edge cases (dedupe, stale, truncation, noise)"
```

---

## Task 7: Backend — Weaviate adapter (production WeaviateClient)

**Files:**
- Create: `backend/internal/services/executive/weaviate_adapter.go`

Purpose: implement the `WeaviateClient` interface by delegating to the existing `backend/internal/services/weaviate` package. Keep all Weaviate-specific details here so the correlator stays SDK-agnostic.

- [ ] **Step 1: Read the existing Weaviate search methods to understand their signatures**

Run: `cd backend && grep -n "func.*Search" internal/services/weaviate/*.go`
Expected: see `Search`, `SearchJira`, `SearchCommits` and any list helpers.

- [ ] **Step 2: Write the adapter**

```go
package executive

import (
    "context"
    "time"

    wv "github.com/nst/pdt/backend/internal/services/weaviate" // adjust import path to actual module
)

type weaviateAdapter struct {
    c *wv.Client
}

func NewWeaviateAdapter(c *wv.Client) WeaviateClient {
    return &weaviateAdapter{c: c}
}

func (a *weaviateAdapter) ListJiraCards(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time, limit int) ([]JiraCard, error) {
    // Delegate to SearchJira with empty query. Map results into executive.JiraCard.
    // TODO during implementation: confirm the exact field name for the card's primary
    // timestamp in JiraCardEmbedding and use it in the filter. See spec §13 open question.
    raw, err := a.c.SearchJira(ctx, "", userID, workspaceID, start, end, limit)
    if err != nil {
        return nil, err
    }
    out := make([]JiraCard, 0, len(raw))
    for _, r := range raw {
        out = append(out, JiraCard{
            CardKey:   r.CardKey,
            Title:     r.Title,       // rename if weaviate result uses "Summary"
            Status:    r.Status,
            Assignee:  r.Assignee,
            Content:   r.Content,
            UpdatedAt: r.UpdatedAt,
        })
    }
    return out, nil
}

func (a *weaviateAdapter) ListCommits(ctx context.Context, userID uint, start, end time.Time) ([]Commit, error) {
    raw, err := a.c.SearchCommits(ctx, "", userID, start, end, 1000)
    if err != nil {
        return nil, err
    }
    out := make([]Commit, 0, len(raw))
    for _, r := range raw {
        out = append(out, Commit{
            SHA:         r.SHA,
            Message:     r.Content,
            RepoName:    r.RepoName,
            Author:      r.Author,
            CommittedAt: r.CommittedAt,
        })
    }
    return out, nil
}

func (a *weaviateAdapter) ListWAMessages(ctx context.Context, userID uint, start, end time.Time) ([]WAMessage, error) {
    raw, err := a.c.Search(ctx, "", userID, 0, start, end, 1000)
    if err != nil {
        return nil, err
    }
    out := make([]WAMessage, 0, len(raw))
    for _, r := range raw {
        out = append(out, WAMessage{
            MessageID:  r.MessageID,
            SenderName: r.SenderName,
            Content:    r.Content,
            Timestamp:  r.Timestamp,
        })
    }
    return out, nil
}

func (a *weaviateAdapter) SemanticCommits(ctx context.Context, userID uint, query string, start, end time.Time, limit int) ([]CommitHit, error) {
    raw, err := a.c.SearchCommits(ctx, query, userID, start, end, limit)
    if err != nil {
        return nil, err
    }
    out := make([]CommitHit, 0, len(raw))
    for _, r := range raw {
        out = append(out, CommitHit{
            Commit: Commit{
                SHA: r.SHA, Message: r.Content, RepoName: r.RepoName,
                Author: r.Author, CommittedAt: r.CommittedAt,
            },
            Distance: r.Distance,
        })
    }
    return out, nil
}

func (a *weaviateAdapter) SemanticWA(ctx context.Context, userID uint, query string, start, end time.Time, limit int) ([]WAHit, error) {
    raw, err := a.c.Search(ctx, query, userID, 0, start, end, limit)
    if err != nil {
        return nil, err
    }
    out := make([]WAHit, 0, len(raw))
    for _, r := range raw {
        out = append(out, WAHit{
            Message: WAMessage{
                MessageID: r.MessageID, SenderName: r.SenderName,
                Content: r.Content, Timestamp: r.Timestamp,
            },
            Distance: r.Distance,
        })
    }
    return out, nil
}
```

⚠️ **Implementation note:** exact types from `services/weaviate` (`r.Content` vs `r.Body`, `r.Distance` vs `r.Extra.Distance`, etc.) may differ. Adapt the adapter to the real return types — the interface shape in `weaviate_client.go` is the source of truth.

- [ ] **Step 3: Build**

Run: `cd backend && go build ./...`
Expected: no errors. If a field name mismatch surfaces, fix the adapter mapping, not the interface.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/executive/weaviate_adapter.go
git commit -m "feat(executive): production WeaviateClient adapter"
```

---

## Task 8: Backend — Postgres model + migration

**Files:**
- Create: `backend/internal/models/executive_report.go`
- Create: `backend/internal/database/migrations/NNNN_executive_reports.up.sql` (use the next sequence number in the repo)
- Create: corresponding `.down.sql`

- [ ] **Step 1: Find the next migration number**

Run: `ls backend/internal/database/migrations/ | tail -5`
Expected: see existing migration filenames; pick the next integer.

- [ ] **Step 2: Write the model**

```go
package models

import (
    "time"

    "gorm.io/datatypes"
)

type ExecutiveReport struct {
    ID                 uint       `gorm:"primaryKey" json:"id"`
    UserID             uint       `gorm:"index;not null" json:"user_id"`
    WorkspaceID        *uint      `gorm:"index" json:"workspace_id,omitempty"`
    RangeStart         time.Time  `json:"range_start"`
    RangeEnd           time.Time  `json:"range_end"`
    StaleThresholdDays int        `json:"stale_threshold_days"`
    Narrative          string     `gorm:"type:text" json:"narrative"`
    Suggestions        datatypes.JSON `json:"suggestions"`
    Dataset            datatypes.JSON `json:"dataset"`
    Status             string     `gorm:"type:varchar(16);not null;default:'generating'" json:"status"`
    ErrorMessage       string     `gorm:"type:text" json:"error_message,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    CompletedAt        *time.Time `json:"completed_at,omitempty"`
}

func (ExecutiveReport) TableName() string { return "executive_reports" }
```

- [ ] **Step 3: Write the migration SQL**

```sql
-- up
CREATE TABLE executive_reports (
    id                   SERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL,
    workspace_id         BIGINT,
    range_start          TIMESTAMPTZ NOT NULL,
    range_end            TIMESTAMPTZ NOT NULL,
    stale_threshold_days INT NOT NULL DEFAULT 7,
    narrative            TEXT NOT NULL DEFAULT '',
    suggestions          JSONB NOT NULL DEFAULT '[]'::jsonb,
    dataset              JSONB NOT NULL DEFAULT '{}'::jsonb,
    status               VARCHAR(16) NOT NULL DEFAULT 'generating',
    error_message        TEXT NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at         TIMESTAMPTZ
);
CREATE INDEX idx_executive_reports_user_created ON executive_reports (user_id, created_at DESC);
CREATE INDEX idx_executive_reports_workspace ON executive_reports (workspace_id);
```

```sql
-- down
DROP TABLE executive_reports;
```

- [ ] **Step 4: Wire auto-migrate (if the repo uses GORM auto-migrate)**

Check: `grep -rn "AutoMigrate" backend/internal/database/ backend/cmd/server/`
If auto-migrate is the pattern, add `&models.ExecutiveReport{}` to the list. Otherwise, ensure the migration files are picked up by the existing migration runner.

- [ ] **Step 5: Build & run migrations**

Run: `cd backend && go build ./... && ./scripts/migrate.sh up` (or equivalent — match the existing repo's migration workflow).
Expected: migration applies cleanly.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/models/executive_report.go backend/internal/database/migrations/
git commit -m "feat(executive): ExecutiveReport model and migration"
```

---

## Task 9: Backend — ExecutiveReportAgent (stub LLM)

**Files:**
- Create: `backend/internal/ai/agent/executive_report.go`
- Create: `backend/internal/ai/agent/executive_report_test.go`

Purpose: get the full streaming pipeline working end-to-end with a stub LLM client. Task 13 replaces the stub with the real MiniMax call.

- [ ] **Step 1: Define the agent interface**

```go
package agent

import (
    "context"

    "github.com/nst/pdt/backend/internal/services/executive"
)

type ExecutiveEvent struct {
    Kind       string                 // "delta" | "suggestion" | "done" | "error"
    Delta      string                 // when Kind == "delta"
    Suggestion *executive.Suggestion  // when Kind == "suggestion"
    Err        error                  // when Kind == "error"
}

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
        return
    }
    a.LLM.Stream(ctx, system, user, out)
}

func buildExecutiveSystemPrompt(r executive.DateRange) string {
    return "You are an engineering executive assistant. You will receive a structured dataset between " +
        r.Start.Format("2006-01-02") + " and " + r.End.Format("2006-01-02") +
        ". Produce sections: ## Summary, ## Topics, ## Gaps, ## Stale Work, ## Next Steps. " +
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

// trimForLLM returns a compact copy of the dataset with WA content capped at 500 chars.
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
```

Add `import "encoding/json"` at the top.

- [ ] **Step 2: Write the structural test with a stub LLM**

```go
package agent

import (
    "context"
    "testing"
    "time"

    "github.com/nst/pdt/backend/internal/services/executive"
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
    agent := &ExecutiveReportAgent{LLM: stub}
    ds := &executive.CorrelatedDataset{
        Range: executive.DateRange{Start: time.Now().Add(-7 * 24 * time.Hour), End: time.Now()},
    }
    ch := make(chan ExecutiveEvent, 10)
    agent.Run(context.Background(), ds, ch)

    var kinds []string
    for e := range ch {
        kinds = append(kinds, e.Kind)
    }
    if len(kinds) != 3 || kinds[0] != "delta" || kinds[1] != "suggestion" || kinds[2] != "done" {
        t.Fatalf("unexpected event order: %v", kinds)
    }
}
```

- [ ] **Step 3: Run**

Run: `cd backend && go test ./internal/ai/agent/ -run TestExecutiveReportAgent -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/agent/executive_report.go backend/internal/ai/agent/executive_report_test.go
git commit -m "feat(executive): ExecutiveReportAgent stub with event stream"
```

---

## Task 10: Backend — Handler (SSE generate + list/get/delete)

**Files:**
- Create: `backend/internal/handlers/executive_report.go`

- [ ] **Step 1: Write the handler skeleton**

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/datatypes"
    "gorm.io/gorm"

    "github.com/nst/pdt/backend/internal/ai/agent"
    "github.com/nst/pdt/backend/internal/models"
    "github.com/nst/pdt/backend/internal/services/executive"
)

type ExecutiveReportHandler struct {
    DB         *gorm.DB
    Correlator interface {
        Build(ctx context.Context, userID uint, workspaceID *uint, r executive.DateRange, staleDays int) (*executive.CorrelatedDataset, error)
    }
    Agent *agent.ExecutiveReportAgent
}

type generateRequest struct {
    RangeStart         time.Time `json:"range_start" binding:"required"`
    RangeEnd           time.Time `json:"range_end" binding:"required"`
    StaleThresholdDays int       `json:"stale_threshold_days"`
    WorkspaceID        *uint     `json:"workspace_id"`
}

func (h *ExecutiveReportHandler) Generate(c *gin.Context) {
    var req generateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if req.RangeEnd.Before(req.RangeStart) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "range_end before range_start"})
        return
    }
    if req.RangeEnd.Sub(req.RangeStart) > time.Duration(executive.MaxRangeDays)*24*time.Hour {
        c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("range exceeds max %d days", executive.MaxRangeDays)})
        return
    }
    if req.StaleThresholdDays == 0 {
        req.StaleThresholdDays = executive.StaleThresholdDefault
    }

    userID := c.GetUint("user_id")
    if userID == 0 {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
        return
    }

    // SSE headers
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.Flush()

    write := func(event, data string) bool {
        _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
        if err != nil {
            return false
        }
        c.Writer.Flush()
        return true
    }

    writeJSON := func(event string, payload any) bool {
        b, err := json.Marshal(payload)
        if err != nil {
            return false
        }
        return write(event, string(b))
    }

    // Pre-insert a "generating" row so interruptions are visible in history.
    row := models.ExecutiveReport{
        UserID:             userID,
        WorkspaceID:        req.WorkspaceID,
        RangeStart:         req.RangeStart,
        RangeEnd:           req.RangeEnd,
        StaleThresholdDays: req.StaleThresholdDays,
        Status:             "generating",
    }
    if err := h.DB.Create(&row).Error; err != nil {
        writeJSON("error", gin.H{"message": err.Error()})
        return
    }

    writeJSON("status", gin.H{"phase": "correlating"})
    ds, err := h.Correlator.Build(c.Request.Context(), userID, req.WorkspaceID, executive.DateRange{Start: req.RangeStart, End: req.RangeEnd}, req.StaleThresholdDays)
    if err != nil {
        h.markFailed(row.ID, err.Error())
        writeJSON("error", gin.H{"message": err.Error()})
        return
    }

    writeJSON("dataset", ds)
    writeJSON("status", gin.H{"phase": "thinking"})

    events := make(chan agent.ExecutiveEvent, 64)
    go h.Agent.Run(c.Request.Context(), ds, events)

    var narrative string
    var suggestions []executive.Suggestion
    var streamErr error

    for ev := range events {
        switch ev.Kind {
        case "delta":
            narrative += ev.Delta
            writeJSON("delta", gin.H{"text": ev.Delta})
        case "suggestion":
            if ev.Suggestion != nil {
                suggestions = append(suggestions, *ev.Suggestion)
                writeJSON("suggestion", ev.Suggestion)
            }
        case "error":
            streamErr = ev.Err
        case "done":
            // handled after loop
        }
    }

    if streamErr != nil {
        h.markFailed(row.ID, streamErr.Error())
        writeJSON("error", gin.H{"message": streamErr.Error()})
        return
    }

    writeJSON("status", gin.H{"phase": "persisting"})

    dsBytes, _ := json.Marshal(ds)
    sugBytes, _ := json.Marshal(suggestions)
    now := time.Now()
    if err := h.DB.Model(&row).Updates(map[string]any{
        "narrative":    narrative,
        "suggestions":  datatypes.JSON(sugBytes),
        "dataset":      datatypes.JSON(dsBytes),
        "status":       "completed",
        "completed_at": &now,
    }).Error; err != nil {
        writeJSON("error", gin.H{"message": err.Error()})
        return
    }

    writeJSON("done", gin.H{"id": row.ID})
    io.WriteString(c.Writer, "") // flush sentinel
}

func (h *ExecutiveReportHandler) markFailed(id uint, msg string) {
    h.DB.Model(&models.ExecutiveReport{}).Where("id = ?", id).Updates(map[string]any{
        "status":        "failed",
        "error_message": msg,
    })
}

func (h *ExecutiveReportHandler) List(c *gin.Context) {
    userID := c.GetUint("user_id")
    var rows []models.ExecutiveReport
    if err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    // Return lightweight list (no dataset/narrative) to keep the payload small.
    type item struct {
        ID         uint       `json:"id"`
        RangeStart time.Time  `json:"range_start"`
        RangeEnd   time.Time  `json:"range_end"`
        Status     string     `json:"status"`
        CreatedAt  time.Time  `json:"created_at"`
        CompletedAt *time.Time `json:"completed_at,omitempty"`
    }
    out := make([]item, len(rows))
    for i, r := range rows {
        out[i] = item{r.ID, r.RangeStart, r.RangeEnd, r.Status, r.CreatedAt, r.CompletedAt}
    }
    c.JSON(http.StatusOK, out)
}

func (h *ExecutiveReportHandler) Get(c *gin.Context) {
    userID := c.GetUint("user_id")
    id := c.Param("id")
    var row models.ExecutiveReport
    if err := h.DB.Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    c.JSON(http.StatusOK, row)
}

func (h *ExecutiveReportHandler) Delete(c *gin.Context) {
    userID := c.GetUint("user_id")
    id := c.Param("id")
    res := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ExecutiveReport{})
    if res.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
        return
    }
    if res.RowsAffected == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    c.Status(http.StatusNoContent)
}
```

- [ ] **Step 2: Build**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handlers/executive_report.go
git commit -m "feat(executive): SSE generate handler + list/get/delete"
```

---

## Task 11: Backend — Handler tests

**Files:**
- Create: `backend/internal/handlers/executive_report_test.go`

- [ ] **Step 1: Write SSE event order test with fake correlator + stub agent**

```go
package handlers

import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "github.com/nst/pdt/backend/internal/ai/agent"
    "github.com/nst/pdt/backend/internal/models"
    "github.com/nst/pdt/backend/internal/services/executive"
)

type fakeCorrelator struct {
    ds *executive.CorrelatedDataset
}

func (f *fakeCorrelator) Build(_ context.Context, _ uint, _ *uint, _ executive.DateRange, _ int) (*executive.CorrelatedDataset, error) {
    return f.ds, nil
}

type scriptedLLM struct {
    events []agent.ExecutiveEvent
}

func (s *scriptedLLM) Stream(_ context.Context, _, _ string, out chan<- agent.ExecutiveEvent) {
    for _, e := range s.events {
        out <- e
    }
    close(out)
}

func setupDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil { t.Fatal(err) }
    if err := db.AutoMigrate(&models.ExecutiveReport{}); err != nil { t.Fatal(err) }
    return db
}

func parseSSE(body string) []string {
    var events []string
    sc := bufio.NewScanner(strings.NewReader(body))
    for sc.Scan() {
        line := sc.Text()
        if strings.HasPrefix(line, "event: ") {
            events = append(events, strings.TrimPrefix(line, "event: "))
        }
    }
    return events
}

func TestGenerate_EventOrder(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupDB(t)

    ds := &executive.CorrelatedDataset{UserID: 42}
    h := &ExecutiveReportHandler{
        DB:         db,
        Correlator: &fakeCorrelator{ds: ds},
        Agent: &agent.ExecutiveReportAgent{LLM: &scriptedLLM{events: []agent.ExecutiveEvent{
            {Kind: "delta", Delta: "hello"},
            {Kind: "suggestion", Suggestion: &executive.Suggestion{Kind: "gap", Title: "t", Detail: "d"}},
            {Kind: "done"},
        }}},
    }

    r := gin.New()
    r.Use(func(c *gin.Context) { c.Set("user_id", uint(42)); c.Next() })
    r.POST("/generate", h.Generate)

    body := `{"range_start":"2026-04-01T00:00:00Z","range_end":"2026-04-10T00:00:00Z"}`
    req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)

    events := parseSSE(rec.Body.String())
    want := []string{"status", "dataset", "status", "delta", "suggestion", "status", "done"}
    if len(events) != len(want) {
        t.Fatalf("event count: got %v want %v", events, want)
    }
    for i, e := range want {
        if events[i] != e {
            t.Fatalf("event %d: got %s want %s (full: %v)", i, events[i], e, events)
        }
    }

    var row models.ExecutiveReport
    db.First(&row)
    if row.Status != "completed" || row.Narrative != "hello" {
        t.Fatalf("row not persisted correctly: %+v", row)
    }
}

func TestGenerate_AgentFailure(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupDB(t)
    h := &ExecutiveReportHandler{
        DB:         db,
        Correlator: &fakeCorrelator{ds: &executive.CorrelatedDataset{}},
        Agent: &agent.ExecutiveReportAgent{LLM: &scriptedLLM{events: []agent.ExecutiveEvent{
            {Kind: "error", Err: fmtErr("boom")},
        }}},
    }
    r := gin.New()
    r.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
    r.POST("/generate", h.Generate)
    req := httptest.NewRequest("POST", "/generate",
        bytes.NewBufferString(`{"range_start":"2026-04-01T00:00:00Z","range_end":"2026-04-02T00:00:00Z"}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)

    if !strings.Contains(rec.Body.String(), "event: error") {
        t.Fatalf("expected error event, got: %s", rec.Body.String())
    }
    var row models.ExecutiveReport
    db.First(&row)
    if row.Status != "failed" || !strings.Contains(row.ErrorMessage, "boom") {
        t.Fatalf("row status wrong: %+v", row)
    }
}

func TestGet_NotOwner_Returns404(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupDB(t)
    db.Create(&models.ExecutiveReport{UserID: 7, Status: "completed"})

    h := &ExecutiveReportHandler{DB: db}
    r := gin.New()
    r.Use(func(c *gin.Context) { c.Set("user_id", uint(99)); c.Next() })
    r.GET("/:id", h.Get)

    req := httptest.NewRequest("GET", "/1", nil)
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)
    if rec.Code != 404 {
        t.Fatalf("expected 404 for non-owner, got %d", rec.Code)
    }
}

func TestGenerate_RejectsOversizedRange(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupDB(t)
    h := &ExecutiveReportHandler{DB: db, Correlator: &fakeCorrelator{ds: &executive.CorrelatedDataset{}}}
    r := gin.New()
    r.Use(func(c *gin.Context) { c.Set("user_id", uint(1)); c.Next() })
    r.POST("/generate", h.Generate)

    start := time.Now().AddDate(0, 0, -200).Format(time.RFC3339)
    end := time.Now().Format(time.RFC3339)
    body := `{"range_start":"` + start + `","range_end":"` + end + `"}`
    req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)
    if rec.Code != 400 {
        t.Fatalf("expected 400 for oversized range, got %d: %s", rec.Code, rec.Body.String())
    }
}

type stringError string
func (e stringError) Error() string { return string(e) }
func fmtErr(s string) error { return stringError(s) }

var _ = json.Marshal // keep import
```

- [ ] **Step 2: Install sqlite driver if not present**

Run: `cd backend && go get gorm.io/driver/sqlite && go mod tidy`

- [ ] **Step 3: Run**

Run: `cd backend && go test ./internal/handlers/ -run TestGenerate -v`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handlers/executive_report_test.go backend/go.mod backend/go.sum
git commit -m "test(executive): handler SSE order, failure, auth, range validation"
```

---

## Task 12: Backend — Route registration

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Find the existing `/api/protected/reports/` group**

Run: `grep -n "reports" backend/cmd/server/main.go`
Expected: see the protected group where `report.go` handlers are registered.

- [ ] **Step 2: Wire the new handler**

Add near the existing reports routes:

```go
execHandler := &handlers.ExecutiveReportHandler{
    DB:         db,
    Correlator: executive.NewCorrelator(executive.NewWeaviateAdapter(weaviateClient)),
    Agent:      &agent.ExecutiveReportAgent{LLM: /* wired in Task 13 */ nil},
}
reports := protected.Group("/reports/executive")
{
    reports.POST("/generate", execHandler.Generate)
    reports.GET("/", execHandler.List)
    reports.GET("/:id", execHandler.Get)
    reports.DELETE("/:id", execHandler.Delete)
}
```

⚠️ Leaving `LLM: nil` is fine because the stub agent is only ever called through `Run`, which is tested separately. The real LLM gets wired in Task 13, before any end-user traffic.

- [ ] **Step 3: Build**

Run: `cd backend && go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(executive): register executive reports routes"
```

---

## Task 13: Backend — Real LLM wiring

**Files:**
- Create: `backend/internal/ai/agent/executive_llm_minimax.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Inspect the existing MiniMax client to find the streaming + tool-use API shape**

Run: `grep -n "ChatStream\|tool" backend/internal/ai/minimax/*.go`
Expected: see the existing `ChatStream` method and any tool-use helpers.

- [ ] **Step 2: Implement `minimaxExecutiveLLM`**

```go
package agent

import (
    "context"
    "encoding/json"

    "github.com/nst/pdt/backend/internal/ai/minimax"
    "github.com/nst/pdt/backend/internal/services/executive"
)

type minimaxExecutiveLLM struct {
    client *minimax.Client
    model  string
}

func NewMinimaxExecutiveLLM(c *minimax.Client, model string) ExecutiveLLM {
    return &minimaxExecutiveLLM{client: c, model: model}
}

func (m *minimaxExecutiveLLM) Stream(ctx context.Context, system, user string, out chan<- ExecutiveEvent) {
    defer close(out)

    tools := []minimax.Tool{
        {
            Name:        "emit_suggestion",
            Description: "Emit a structured suggestion (gap, stale, or next_step) with title, detail, and refs.",
            InputSchema: map[string]any{
                "type": "object",
                "required": []string{"kind", "title", "detail"},
                "properties": map[string]any{
                    "kind":   map[string]any{"type": "string", "enum": []string{"gap", "stale", "next_step"}},
                    "title":  map[string]any{"type": "string"},
                    "detail": map[string]any{"type": "string"},
                    "refs":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
                },
            },
        },
    }

    events, errCh := m.client.ChatStream(ctx, minimax.ChatRequest{
        Model:    m.model,
        System:   system,
        Messages: []minimax.Message{{Role: "user", Content: user}},
        Tools:    tools,
    })

    for {
        select {
        case <-ctx.Done():
            out <- ExecutiveEvent{Kind: "error", Err: ctx.Err()}
            return
        case err := <-errCh:
            if err != nil {
                out <- ExecutiveEvent{Kind: "error", Err: err}
                return
            }
        case ev, ok := <-events:
            if !ok {
                out <- ExecutiveEvent{Kind: "done"}
                return
            }
            switch ev.Kind {
            case minimax.EventText:
                out <- ExecutiveEvent{Kind: "delta", Delta: ev.Text}
            case minimax.EventToolUse:
                if ev.ToolName == "emit_suggestion" {
                    var s executive.Suggestion
                    if err := json.Unmarshal(ev.ToolInput, &s); err == nil {
                        out <- ExecutiveEvent{Kind: "suggestion", Suggestion: &s}
                    }
                }
            }
        }
    }
}
```

⚠️ Adjust `minimax.Tool`, `ChatRequest`, `EventText`, `EventToolUse` to match the actual API in `backend/internal/ai/minimax/`. The test in Task 9 uses a stub so it stays green regardless of the shape here.

- [ ] **Step 3: Wire it into `main.go`**

```go
execLLM := agent.NewMinimaxExecutiveLLM(minimaxClient, "claude-sonnet-4-6")
execHandler.Agent = &agent.ExecutiveReportAgent{LLM: execLLM}
```

- [ ] **Step 4: Build**

Run: `cd backend && go build ./...`
Expected: no errors. If the MiniMax client's stream API shape differs, update `minimaxExecutiveLLM` accordingly.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai/agent/executive_llm_minimax.go backend/cmd/server/main.go
git commit -m "feat(executive): wire MiniMax LLM into ExecutiveReportAgent"
```

---

## Task 14: Frontend — RTK Query service

**Files:**
- Create: `frontend/src/infrastructure/services/executiveReport.service.ts`
- Modify: `frontend/src/infrastructure/store/index.ts`

- [ ] **Step 1: Inspect the existing `report.service.ts` to copy the slice pattern**

Run: `cat frontend/src/infrastructure/services/report.service.ts | head -60`
Expected: see the `createApi` setup, base query, tag types.

- [ ] **Step 2: Write the slice**

```ts
import { createApi } from '@reduxjs/toolkit/query/react';
import { baseQueryWithAuth } from './baseQuery'; // use the project's existing base query

export interface ExecutiveReportListItem {
  id: number;
  range_start: string;
  range_end: string;
  status: 'generating' | 'completed' | 'failed';
  created_at: string;
  completed_at?: string;
}

export interface Suggestion {
  kind: 'gap' | 'stale' | 'next_step';
  title: string;
  detail: string;
  refs: string[];
}

export interface DailyBucket {
  day: string;
  commits: number;
  jira_changes: number;
  wa_messages: number;
}

export interface Metrics {
  commits_total: number;
  commits_linked: number;
  cards_active: number;
  cards_with_commits: number;
  wa_topics_ticketed: number;
  wa_topics_orphan: number;
  stale_card_count: number;
  linkage_pct_commits: number;
  linkage_pct_cards: number;
  truncated: boolean;
}

export interface Topic {
  anchor: { card_key: string; title: string; status: string; assignee: string; content: string; updated_at: string };
  messages: Array<{ message_id: string; sender_name: string; content: string; timestamp: string }>;
  commits: Array<{ sha: string; message: string; repo_name: string; author: string; committed_at: string }>;
  stale: boolean;
  days_idle: number;
}

export interface WAGroup {
  summary: string;
  messages: Topic['messages'];
  started_at: string;
}

export interface CorrelatedDataset {
  user_id: number;
  workspace_id?: number;
  range: { Start: string; End: string };
  topics: Topic[];
  orphan_wa: WAGroup[];
  orphan_commits: Topic['commits'];
  metrics: Metrics;
  daily_buckets: DailyBucket[];
}

export interface ExecutiveReport {
  id: number;
  user_id: number;
  workspace_id?: number;
  range_start: string;
  range_end: string;
  stale_threshold_days: number;
  narrative: string;
  suggestions: Suggestion[];
  dataset: CorrelatedDataset;
  status: 'generating' | 'completed' | 'failed';
  error_message?: string;
  created_at: string;
  completed_at?: string;
}

export const executiveReportApi = createApi({
  reducerPath: 'executiveReportApi',
  baseQuery: baseQueryWithAuth,
  tagTypes: ['ExecutiveReport'],
  endpoints: (b) => ({
    listExecutiveReports: b.query<ExecutiveReportListItem[], void>({
      query: () => '/api/protected/reports/executive/',
      providesTags: ['ExecutiveReport'],
    }),
    getExecutiveReport: b.query<ExecutiveReport, number>({
      query: (id) => `/api/protected/reports/executive/${id}`,
      providesTags: (_r, _e, id) => [{ type: 'ExecutiveReport', id }],
    }),
    deleteExecutiveReport: b.mutation<void, number>({
      query: (id) => ({ url: `/api/protected/reports/executive/${id}`, method: 'DELETE' }),
      invalidatesTags: ['ExecutiveReport'],
    }),
  }),
});

export const {
  useListExecutiveReportsQuery,
  useGetExecutiveReportQuery,
  useDeleteExecutiveReportMutation,
} = executiveReportApi;
```

- [ ] **Step 3: Register the slice in the store**

Open `frontend/src/infrastructure/store/index.ts`, add `executiveReportApi.reducer` to `reducer`, add `executiveReportApi.middleware` to `middleware`.

- [ ] **Step 4: Build**

Run: `cd frontend && pnpm tsc --noEmit` (or `bun run tsc --noEmit`, match the repo's tooling)
Expected: no type errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/infrastructure/services/executiveReport.service.ts frontend/src/infrastructure/store/
git commit -m "feat(executive): RTK Query slice for executive reports"
```

---

## Task 15: Frontend — `useGenerateExecutiveReport` hook

**Files:**
- Create: `frontend/src/presentation/hooks/useGenerateExecutiveReport.ts`

- [ ] **Step 1: Write the hook**

```ts
import { useCallback, useRef, useState } from 'react';
import { useDispatch } from 'react-redux';
import { executiveReportApi, CorrelatedDataset, Suggestion } from '@/infrastructure/services/executiveReport.service';

export type Phase = 'idle' | 'correlating' | 'thinking' | 'streaming' | 'persisting' | 'done' | 'error';

export interface GenerateArgs {
  rangeStart: string; // ISO
  rangeEnd: string;
  staleThresholdDays?: number;
  workspaceId?: number;
}

export function useGenerateExecutiveReport() {
  const dispatch = useDispatch();
  const [phase, setPhase] = useState<Phase>('idle');
  const [dataset, setDataset] = useState<CorrelatedDataset | null>(null);
  const [narrative, setNarrative] = useState('');
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [reportId, setReportId] = useState<number | null>(null);
  const sourceRef = useRef<EventSource | null>(null);

  const reset = useCallback(() => {
    sourceRef.current?.close();
    sourceRef.current = null;
    setPhase('idle');
    setDataset(null);
    setNarrative('');
    setSuggestions([]);
    setError(null);
    setReportId(null);
  }, []);

  const start = useCallback((args: GenerateArgs) => {
    reset();
    // EventSource only supports GET, so we POST the body first and use a 1-shot fetch-based SSE.
    // We use ReadableStream + TextDecoder to parse events.
    setPhase('correlating');
    const ctrl = new AbortController();
    fetch('/api/protected/reports/executive/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      signal: ctrl.signal,
      body: JSON.stringify({
        range_start: args.rangeStart,
        range_end: args.rangeEnd,
        stale_threshold_days: args.staleThresholdDays,
        workspace_id: args.workspaceId,
      }),
    })
      .then(async (res) => {
        if (!res.ok || !res.body) {
          const text = await res.text();
          throw new Error(text || `HTTP ${res.status}`);
        }
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx: number;
          while ((idx = buffer.indexOf('\n\n')) !== -1) {
            const raw = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            handleEvent(raw);
          }
        }
      })
      .catch((e) => {
        setPhase('error');
        setError(e instanceof Error ? e.message : String(e));
      });

    function handleEvent(raw: string) {
      const lines = raw.split('\n');
      let event = 'message';
      let data = '';
      for (const line of lines) {
        if (line.startsWith('event: ')) event = line.slice(7);
        else if (line.startsWith('data: ')) data += line.slice(6);
      }
      if (!data) return;
      const parsed = JSON.parse(data);
      switch (event) {
        case 'status':
          if (parsed.phase === 'correlating') setPhase('correlating');
          else if (parsed.phase === 'thinking') setPhase('thinking');
          else if (parsed.phase === 'persisting') setPhase('persisting');
          break;
        case 'dataset':
          setDataset(parsed as CorrelatedDataset);
          setPhase('streaming');
          break;
        case 'delta':
          setNarrative((n) => n + (parsed.text ?? ''));
          break;
        case 'suggestion':
          setSuggestions((s) => [...s, parsed as Suggestion]);
          break;
        case 'done':
          setPhase('done');
          setReportId(parsed.id);
          dispatch(executiveReportApi.util.invalidateTags(['ExecutiveReport']));
          break;
        case 'error':
          setPhase('error');
          setError(parsed.message ?? 'unknown error');
          break;
      }
    }

    return () => ctrl.abort();
  }, [dispatch, reset]);

  return { phase, dataset, narrative, suggestions, error, reportId, start, reset };
}
```

- [ ] **Step 2: Type-check**

Run: `cd frontend && pnpm tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/hooks/useGenerateExecutiveReport.ts
git commit -m "feat(executive): SSE consumer hook for report generation"
```

---

## Task 16: Frontend — StaleWorkTable + SuggestionList

**Files:**
- Create: `frontend/src/presentation/components/executive/StaleWorkTable.tsx`
- Create: `frontend/src/presentation/components/executive/SuggestionList.tsx`
- Create: `frontend/src/presentation/components/executive/StaleWorkTable.test.tsx`
- Create: `frontend/src/presentation/components/executive/SuggestionList.test.tsx`

- [ ] **Step 1: Write `StaleWorkTable.tsx`**

```tsx
import { Topic, Suggestion } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/presentation/components/ui/card';

interface Props {
  topics: Topic[];
  suggestions: Suggestion[];
}

export function StaleWorkTable({ topics, suggestions }: Props) {
  const stale = topics.filter((t) => t.stale);
  if (stale.length === 0) {
    return <Card className="p-4 text-sm text-muted-foreground">No stale work in this range.</Card>;
  }
  const actionFor = (key: string) =>
    suggestions.find((s) => s.kind === 'stale' && s.refs.includes(`jira:${key}`))?.detail ?? '';

  return (
    <Card className="p-0 overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            <th className="text-left p-2">Card</th>
            <th className="text-left p-2">Title</th>
            <th className="text-right p-2">Days idle</th>
            <th className="text-left p-2">Suggested action</th>
          </tr>
        </thead>
        <tbody>
          {stale.map((t) => (
            <tr key={t.anchor.card_key} className="border-t">
              <td className="p-2 font-mono">{t.anchor.card_key}</td>
              <td className="p-2">{t.anchor.title}</td>
              <td className="p-2 text-right">{t.days_idle}</td>
              <td className="p-2 text-muted-foreground">{actionFor(t.anchor.card_key) || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
```

- [ ] **Step 2: Write `SuggestionList.tsx`**

```tsx
import { Suggestion } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/presentation/components/ui/card';

interface Props {
  suggestions: Suggestion[];
}

const GROUP_LABEL: Record<Suggestion['kind'], string> = {
  gap: 'Gaps',
  stale: 'Stale Work',
  next_step: 'Next Steps',
};

export function SuggestionList({ suggestions }: Props) {
  const grouped: Record<Suggestion['kind'], Suggestion[]> = {
    gap: [], stale: [], next_step: [],
  };
  for (const s of suggestions) grouped[s.kind].push(s);

  return (
    <div className="space-y-4">
      {(['gap', 'stale', 'next_step'] as const).map((k) => (
        grouped[k].length > 0 && (
          <section key={k}>
            <h3 className="text-sm font-semibold text-muted-foreground uppercase mb-2">{GROUP_LABEL[k]}</h3>
            <div className="space-y-2">
              {grouped[k].map((s, i) => (
                <Card key={i} className="p-3">
                  <div className="font-medium">{s.title}</div>
                  <div className="text-sm text-muted-foreground">{s.detail}</div>
                  {s.refs.length > 0 && (
                    <div className="mt-1 text-xs text-muted-foreground/80">
                      {s.refs.join(' · ')}
                    </div>
                  )}
                </Card>
              ))}
            </div>
          </section>
        )
      ))}
    </div>
  );
}
```

- [ ] **Step 3: Write the tests**

```tsx
// StaleWorkTable.test.tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StaleWorkTable } from './StaleWorkTable';

describe('StaleWorkTable', () => {
  it('shows empty state when no stale topics', () => {
    render(<StaleWorkTable topics={[]} suggestions={[]} />);
    expect(screen.getByText(/no stale work/i)).toBeInTheDocument();
  });

  it('renders a row per stale topic and attaches matching action', () => {
    render(
      <StaleWorkTable
        topics={[
          {
            anchor: { card_key: 'A-1', title: 'Login', status: 'In Progress', assignee: '', content: '', updated_at: '' },
            messages: [], commits: [], stale: true, days_idle: 9,
          },
          {
            anchor: { card_key: 'A-2', title: 'Shipped', status: 'Done', assignee: '', content: '', updated_at: '' },
            messages: [], commits: [], stale: false, days_idle: 0,
          },
        ]}
        suggestions={[{ kind: 'stale', title: 't', detail: 'Ping owner', refs: ['jira:A-1'] }]}
      />
    );
    expect(screen.getByText('A-1')).toBeInTheDocument();
    expect(screen.queryByText('A-2')).not.toBeInTheDocument();
    expect(screen.getByText('Ping owner')).toBeInTheDocument();
    expect(screen.getByText('9')).toBeInTheDocument();
  });
});
```

```tsx
// SuggestionList.test.tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SuggestionList } from './SuggestionList';

describe('SuggestionList', () => {
  it('groups suggestions by kind in order gap/stale/next_step', () => {
    render(<SuggestionList suggestions={[
      { kind: 'next_step', title: 'n1', detail: 'd', refs: [] },
      { kind: 'gap', title: 'g1', detail: 'd', refs: [] },
      { kind: 'stale', title: 's1', detail: 'd', refs: [] },
    ]} />);
    const headings = screen.getAllByRole('heading');
    expect(headings.map(h => h.textContent)).toEqual(['GAPS', 'STALE WORK', 'NEXT STEPS']);
  });
});
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && pnpm test -- StaleWorkTable SuggestionList`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/presentation/components/executive/
git commit -m "feat(executive): StaleWorkTable and SuggestionList with tests"
```

---

## Task 17: Frontend — ExecutiveActivityChart wrapper

**Files:**
- Create: `frontend/src/presentation/components/executive/ExecutiveActivityChart.tsx`

- [ ] **Step 1: Inspect `CommitActivityChart` for its prop shape**

Run: `cat frontend/src/presentation/components/charts/CommitActivityChart.tsx`

- [ ] **Step 2: Write the wrapper**

```tsx
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { DailyBucket } from '@/infrastructure/services/executiveReport.service';

interface Props {
  buckets: DailyBucket[];
}

export function ExecutiveActivityChart({ buckets }: Props) {
  const data = buckets.map((b) => ({
    day: b.day.slice(5, 10), // MM-DD
    commits: b.commits,
    jira: b.jira_changes,
    wa: b.wa_messages,
  }));
  return (
    <div className="w-full h-64">
      <ResponsiveContainer>
        <AreaChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="day" />
          <YAxis allowDecimals={false} />
          <Tooltip />
          <Area type="monotone" dataKey="commits" stackId="1" stroke="#2563eb" fill="#2563eb" fillOpacity={0.6} />
          <Area type="monotone" dataKey="jira" stackId="1" stroke="#f59e0b" fill="#f59e0b" fillOpacity={0.6} />
          <Area type="monotone" dataKey="wa" stackId="1" stroke="#10b981" fill="#10b981" fillOpacity={0.6} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
```

(Colors match the existing `pdt-primary/accent/neutral` palette roughly; tighten to the real Tailwind tokens during implementation.)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/presentation/components/executive/ExecutiveActivityChart.tsx
git commit -m "feat(executive): activity timeline chart wrapper"
```

---

## Task 18: Frontend — ExecutiveReportView (composes charts, narrative, suggestions)

**Files:**
- Create: `frontend/src/presentation/components/executive/ExecutiveReportView.tsx`

- [ ] **Step 1: Inspect the existing markdown renderer used in chat**

Run: `grep -rn "ReactMarkdown\|MarkdownRenderer" frontend/src/presentation/components/ | head -5`
Expected: find the component or hook already used for chat markdown.

- [ ] **Step 2: Write the view**

```tsx
import { CorrelatedDataset, Suggestion, Metrics } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/presentation/components/ui/card';
import { LinkageGaugeChart } from '@/presentation/components/charts/LinkageGaugeChart';
import { ExecutiveActivityChart } from './ExecutiveActivityChart';
import { StaleWorkTable } from './StaleWorkTable';
import { SuggestionList } from './SuggestionList';
import { Markdown } from '@/presentation/components/ui/markdown'; // adjust to the real path

interface Props {
  dataset: CorrelatedDataset | null;
  narrative: string;
  suggestions: Suggestion[];
  phase: string;
  error: string | null;
}

export function ExecutiveReportView({ dataset, narrative, suggestions, phase, error }: Props) {
  if (error) {
    return <Card className="p-4 border-destructive text-destructive">Error: {error}</Card>;
  }
  if (!dataset) {
    return <Card className="p-4 text-muted-foreground">{phaseLabel(phase)}</Card>;
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
      <div className="space-y-4">
        <MetricsRow metrics={dataset.metrics} />
        <Card className="p-4">
          <h3 className="font-semibold mb-2">Activity Timeline</h3>
          <ExecutiveActivityChart buckets={dataset.daily_buckets} />
        </Card>
        <StaleWorkTable topics={dataset.topics} suggestions={suggestions} />
        {narrative && (
          <Card className="p-4 prose prose-sm max-w-none dark:prose-invert">
            <Markdown>{narrative}</Markdown>
          </Card>
        )}
        {!narrative && phase !== 'done' && (
          <Card className="p-4 text-muted-foreground text-sm">{phaseLabel(phase)}</Card>
        )}
      </div>
      <aside className="space-y-4">
        <SuggestionList suggestions={suggestions} />
      </aside>
    </div>
  );
}

function MetricsRow({ metrics }: { metrics: Metrics }) {
  return (
    <div className="grid grid-cols-3 gap-4">
      <GaugeCard label="Commit linkage" value={metrics.linkage_pct_commits} />
      <GaugeCard label="Card linkage" value={metrics.linkage_pct_cards} />
      <GaugeCard
        label="WA topics ticketed"
        value={
          metrics.wa_topics_ticketed + metrics.wa_topics_orphan === 0
            ? 0
            : metrics.wa_topics_ticketed / (metrics.wa_topics_ticketed + metrics.wa_topics_orphan)
        }
      />
    </div>
  );
}

function GaugeCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <LinkageGaugeChart value={value} />
    </Card>
  );
}

function phaseLabel(phase: string): string {
  switch (phase) {
    case 'correlating': return 'Correlating data from Jira, commits, and WhatsApp…';
    case 'thinking':    return 'Generating narrative…';
    case 'streaming':   return 'Writing report…';
    case 'persisting':  return 'Saving…';
    case 'done':        return 'Ready.';
    default:            return 'Idle.';
  }
}
```

⚠️ `LinkageGaugeChart`'s prop shape may differ from `{ value }` — adjust after reading the component.

- [ ] **Step 3: Type-check**

Run: `cd frontend && pnpm tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/presentation/components/executive/ExecutiveReportView.tsx
git commit -m "feat(executive): ExecutiveReportView composing charts, narrative, suggestions"
```

---

## Task 19: Frontend — ExecutiveReportTab (controls + history sidebar)

**Files:**
- Create: `frontend/src/presentation/components/executive/ExecutiveReportTab.tsx`
- Create: `frontend/src/presentation/components/executive/index.ts`

- [ ] **Step 1: Write the tab**

```tsx
import { useMemo, useState } from 'react';
import {
  useListExecutiveReportsQuery,
  useGetExecutiveReportQuery,
  useDeleteExecutiveReportMutation,
} from '@/infrastructure/services/executiveReport.service';
import { useGenerateExecutiveReport } from '@/presentation/hooks/useGenerateExecutiveReport';
import { Button } from '@/presentation/components/ui/button';
import { Card } from '@/presentation/components/ui/card';
import { DateRangePicker } from '@/presentation/components/ui/date-range-picker'; // or whatever the repo uses
import { ExecutiveReportView } from './ExecutiveReportView';

export function ExecutiveReportTab() {
  const [range, setRange] = useState(() => {
    const end = new Date();
    const start = new Date(); start.setDate(start.getDate() - 14);
    return { start, end };
  });
  const [staleDays, setStaleDays] = useState(7);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const { data: list } = useListExecutiveReportsQuery();
  const { data: selected } = useGetExecutiveReportQuery(selectedId!, { skip: selectedId === null });
  const [deleteReport] = useDeleteExecutiveReportMutation();

  const stream = useGenerateExecutiveReport();

  const activeView = useMemo(() => {
    if (selectedId && selected) {
      return {
        dataset: selected.dataset,
        narrative: selected.narrative,
        suggestions: selected.suggestions,
        phase: selected.status === 'completed' ? 'done' : selected.status,
        error: selected.status === 'failed' ? (selected.error_message ?? 'failed') : null,
      };
    }
    return {
      dataset: stream.dataset,
      narrative: stream.narrative,
      suggestions: stream.suggestions,
      phase: stream.phase,
      error: stream.error,
    };
  }, [selectedId, selected, stream]);

  function generate() {
    setSelectedId(null);
    stream.start({
      rangeStart: range.start.toISOString(),
      rangeEnd: range.end.toISOString(),
      staleThresholdDays: staleDays,
    });
  }

  return (
    <div className="grid grid-cols-[240px_1fr] gap-4">
      <aside className="space-y-2">
        <h3 className="text-sm font-semibold text-muted-foreground">History</h3>
        {list?.map((item) => (
          <Card
            key={item.id}
            role="button"
            onClick={() => setSelectedId(item.id)}
            className={`p-2 cursor-pointer ${selectedId === item.id ? 'border-primary' : ''}`}
          >
            <div className="text-xs font-mono">#{item.id}</div>
            <div className="text-xs">{item.range_start.slice(0, 10)} → {item.range_end.slice(0, 10)}</div>
            <div className="text-xs text-muted-foreground">{item.status}</div>
            <button
              className="text-xs text-destructive mt-1"
              onClick={(e) => { e.stopPropagation(); deleteReport(item.id); if (selectedId === item.id) setSelectedId(null); }}
            >
              delete
            </button>
          </Card>
        ))}
      </aside>

      <div className="space-y-4">
        <Card className="p-4 flex flex-wrap items-end gap-4">
          <DateRangePicker value={range} onChange={setRange} />
          <label className="text-sm">
            Stale threshold (days)
            <input
              type="number"
              min={1}
              value={staleDays}
              onChange={(e) => setStaleDays(parseInt(e.target.value, 10) || 7)}
              className="ml-2 w-16 border rounded px-2 py-1"
            />
          </label>
          <Button onClick={generate} disabled={stream.phase !== 'idle' && stream.phase !== 'done' && stream.phase !== 'error'}>
            Generate
          </Button>
        </Card>

        <ExecutiveReportView {...activeView} />
      </div>
    </div>
  );
}
```

```ts
// index.ts
export { ExecutiveReportTab } from './ExecutiveReportTab';
```

⚠️ `DateRangePicker` import path is a placeholder. Use the project's actual date range component (likely in `components/ui/` or `shared/`). Replace with the native one if none exists.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/presentation/components/executive/ExecutiveReportTab.tsx frontend/src/presentation/components/executive/index.ts
git commit -m "feat(executive): ExecutiveReportTab with controls and history sidebar"
```

---

## Task 20: Frontend — Wire the tab into `ReportsPage`

**Files:**
- Modify: `frontend/src/presentation/pages/ReportsPage.tsx`

- [ ] **Step 1: Find the existing tabs block**

Run: `grep -n "TabsTrigger\|TabsContent" frontend/src/presentation/pages/ReportsPage.tsx`
Expected: see the Daily/Monthly tab setup.

- [ ] **Step 2: Add the Executive tab**

```tsx
import { ExecutiveReportTab } from '@/presentation/components/executive';

// inside <Tabs>:
<TabsList>
  <TabsTrigger value="daily">Daily</TabsTrigger>
  <TabsTrigger value="monthly">Monthly</TabsTrigger>
  <TabsTrigger value="executive">Executive</TabsTrigger>
</TabsList>
// ... existing TabsContent ...
<TabsContent value="executive">
  <ExecutiveReportTab />
</TabsContent>
```

- [ ] **Step 3: Type-check + manual smoke**

Run: `cd frontend && pnpm tsc --noEmit && pnpm dev`
Expected: no errors; load `/reports`, click the Executive tab, see empty history + controls.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/presentation/pages/ReportsPage.tsx
git commit -m "feat(executive): add Executive tab to ReportsPage"
```

---

## Task 21: End-to-end smoke test

**No files; this is a manual verification checklist.**

- [ ] **Step 1: Start the backend**

Run: `cd backend && go run ./cmd/server`
Expected: server starts, no migration errors, `executive_reports` table exists.

- [ ] **Step 2: Start the frontend**

Run: `cd frontend && pnpm dev`

- [ ] **Step 3: Log in and navigate to `/reports`**

Click the Executive tab. Confirm:
- Empty history sidebar.
- Date range defaults to last 14 days.
- Generate button enabled.

- [ ] **Step 4: Generate a report**

Click Generate. Confirm:
- Phase progresses: correlating → thinking → streaming → persisting → done.
- Charts render immediately when `dataset` arrives.
- Narrative streams in markdown.
- Suggestions appear in the right rail.
- History sidebar shows the new report after `done`.

- [ ] **Step 5: Reload and click the historical report**

Confirm it re-renders charts and narrative from the persisted snapshot without triggering a new generation.

- [ ] **Step 6: Delete it**

Confirm it disappears from the sidebar.

- [ ] **Step 7: Generate an invalid range (e.g., 200 days)**

Confirm a 400 error is shown in the view.

- [ ] **Step 8: Final commit**

If any adjustments were needed:

```bash
git add -u
git commit -m "fix(executive): smoke-test adjustments"
```

---

## Self-Review Notes

**Spec coverage check:**

| Spec requirement | Task(s) |
|---|---|
| Hybrid correlator (anchor + orphan pass) | Tasks 3, 4, 5, 6 |
| Explicit + semantic dedupe | Task 6 |
| Staleness logic (threshold boundary) | Tasks 4, 6 |
| Orphan WA noise floor (3 msgs min) | Tasks 3, 6 |
| Truncation at `MaxAnchors` | Tasks 4, 6 |
| `ExecutiveReportAgent` with `emit_suggestion` tool | Tasks 9, 13 |
| SSE event contract (`status → dataset → delta* → suggestion* → status → done`) | Tasks 10, 11 |
| Failure persistence (`Status="failed"` row) | Tasks 10, 11 |
| Per-user 404 (no leak) | Tasks 10, 11 |
| Range cap 90 days | Tasks 10, 11 |
| `List`, `Get`, `Delete` endpoints | Tasks 10, 11 |
| RTK Query slice + store registration | Task 14 |
| SSE consumer hook with phase state | Task 15 |
| Reuse `LinkageGaugeChart`, `CommitActivityChart` | Tasks 17, 18 |
| Stale work table with suggestion join | Task 16 |
| Grouped suggestion panel | Task 16 |
| History sidebar + re-render from persisted snapshot | Task 19 |
| Third tab on `ReportsPage` | Task 20 |

**Types consistency check:** Go `JiraCard`, `Commit`, `WAMessage`, `Topic`, `Suggestion` used identically across correlator, agent, handler. TypeScript mirror types in `executiveReport.service.ts` match the Go JSON tags.

**Placeholder scan:** Task 7 (adapter) and Task 13 (LLM wiring) contain real code but note that exact field names and the MiniMax API shape must be matched against the real packages during implementation — this is honest scope of integration work, not a placeholder for missing design.

**Open items from spec §13** (tracked for planning-time resolution, not blockers):
- Exact Jira timestamp field name in `JiraCardEmbedding` → resolve in Task 7.
- MiniMax tool-use API shape → resolve in Task 13.
- `gorm.io/datatypes` availability → resolve in Task 8 (fall back to `json.RawMessage` if absent).
