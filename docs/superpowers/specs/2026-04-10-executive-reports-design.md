# Executive Reports — Design Spec

**Date:** 2026-04-10
**Status:** Draft — awaiting user review
**Scope:** v1 of an "Executive Report" feature that correlates cross-source data (Jira, commits, WhatsApp) from Weaviate over a user-selected date range, generates an LLM narrative with actionable suggestions, and renders it in the existing `ReportsPage` as a new tab.

---

## 1. Motivation

The current `ReportsPage` produces templated daily/monthly summaries (commits + Jira), rendered from fixed aggregations. They answer "what did I touch today?" but not:

- **Where are the gaps?** — topics discussed on WhatsApp that never became a Jira ticket, or commits shipped without a linked card.
- **What's stuck?** — cards marked `In Progress` that have had no commit activity for a meaningful window.
- **What should I do next?** — grounded recommendations based on the full cross-source picture.

The new feature uses Weaviate's semantic search to *link* items across the three existing collections (`WaMessageEmbedding`, `JiraCardEmbedding`, `CommitEmbedding`) without requiring explicit IDs, and hands the linked dataset to an LLM agent that produces a narrative plus structured suggestions.

## 2. Scope

**In scope (v1):**
- New "Executive" tab on `ReportsPage`.
- Backend pipeline: hybrid correlator → `ExecutiveReportAgent` → SSE stream → Postgres persistence → history.
- Three suggestion categories: **Gap detection (A)**, **Stale work (B)**, **Next-step recommendations (E)**.
- Three visuals: **activity timeline (stacked area)**, **linkage gauges**, **stale-work table**.
- Max range 90 days, default 14 days, stale threshold 7 days.
- Per-user scoping; optional `workspace_id` scoping for Jira.

**Out of scope (v1 — phase 2 candidates):**
- Focus-drift analysis (suggestion C).
- Collaboration-signal mining (suggestion D).
- Risk flags (suggestion F).
- Topic cluster bubble chart, focus heatmap.
- Distance-threshold auto-tuning.
- Shared/team-level reports (no cross-user aggregation).

## 3. Existing Assets Reused

| Asset | Path | Usage |
|---|---|---|
| Weaviate Go client v4 | `backend/internal/services/weaviate/` | `Search`, `SearchJira`, `SearchCommits` with date + user/workspace filters |
| `ReportAgent` pattern | `backend/internal/ai/agent/report.go:16` | Template for new `ExecutiveReportAgent` (tool-use, system prompt) |
| MiniMax/Anthropic client | `backend/internal/ai/minimax/client.go:14` | LLM backend, `ChatStream` already supports SSE |
| Recharts | `frontend/package.json` (v3.7.0) | `CommitActivityChart`, `LinkageGaugeChart` reused |
| shadcn/ui + Tailwind | `frontend/src/presentation/components/ui/` | Cards, tabs, dialogs, date picker |
| RTK Query | `frontend/src/infrastructure/services/` | Pattern for list/get/delete endpoints |
| SSE pattern | Existing chat stream | Template for `EventSource` consumer |

## 4. Architecture

### 4.1 Module Layout

**Backend (Go):**
```
backend/internal/
├── services/executive/
│   ├── correlator.go        # (userID, workspaceID?, A, B) → CorrelatedDataset
│   ├── correlator_test.go
│   ├── metrics.go           # linkage %, staleness, daily buckets
│   └── types.go             # DTOs
├── ai/agent/
│   └── executive_report.go  # ExecutiveReportAgent (follows report.go pattern)
├── handlers/
│   └── executive_report.go  # Generate (SSE), List, Get, Delete
└── models/
    └── executive_report.go  # GORM model
```

**Frontend (React + TypeScript + Vite):**
```
frontend/src/
├── infrastructure/services/
│   └── executiveReport.service.ts      # RTK list/get/delete
├── presentation/hooks/
│   └── useGenerateExecutiveReport.ts   # SSE consumer hook
└── presentation/components/executive/
    ├── ExecutiveReportTab.tsx
    ├── ExecutiveReportView.tsx
    ├── SuggestionList.tsx
    ├── StaleWorkTable.tsx
    └── index.ts
```

### 4.2 Boundary Rules

1. `correlator` never imports LLM code. Pure data in → `CorrelatedDataset` out. Fully unit-testable with a fake Weaviate client.
2. `ExecutiveReportAgent` never touches Weaviate. It receives a populated `CorrelatedDataset` and streams narrative + suggestion tool calls.
3. Handler is the only orchestrator: correlator → agent → persistence → SSE framing.

## 5. Data Model

### 5.1 Go Types (`services/executive/types.go`)

```go
type DateRange struct{ Start, End time.Time }

type CorrelatedDataset struct {
    UserID        uint
    WorkspaceID   *uint
    Range         DateRange
    Topics        []Topic
    OrphanWA      []WAGroup
    OrphanCommits []Commit
    Metrics       Metrics
    DailyBuckets  []DailyBucket
}

type Topic struct {
    Anchor   JiraCard
    Messages []WAMessage
    Commits  []Commit
    Stale    bool
    DaysIdle int
}

type WAGroup struct {
    Summary   string    // first sender + first 80 chars of first message
    Messages  []WAMessage
    StartedAt time.Time
}

type Metrics struct {
    CommitsTotal      int
    CommitsLinked     int
    CardsActive       int
    CardsWithCommits  int
    WATopicsTicketed  int
    WATopicsOrphan    int
    StaleCardCount    int
    LinkagePctCommits float64
    LinkagePctCards   float64
    Truncated         bool
}

type DailyBucket struct {
    Day         time.Time
    Commits     int
    JiraChanges int
    WAMessages  int
}
```

### 5.2 Postgres Model (`models/executive_report.go`)

```go
type ExecutiveReport struct {
    ID                 uint      `gorm:"primaryKey"`
    UserID             uint      `gorm:"index"`
    WorkspaceID        *uint     `gorm:"index"`
    RangeStart         time.Time
    RangeEnd           time.Time
    StaleThresholdDays int
    Narrative          string         `gorm:"type:text"`
    Suggestions        datatypes.JSON // []Suggestion
    Dataset            datatypes.JSON // CorrelatedDataset snapshot
    Status             string         // "generating" | "completed" | "failed"
    ErrorMessage       string         // populated on failed
    CreatedAt          time.Time
    CompletedAt        *time.Time
}

type Suggestion struct {
    Kind   string   `json:"kind"`   // "gap" | "stale" | "next_step"
    Title  string   `json:"title"`
    Detail string   `json:"detail"`
    Refs   []string `json:"refs"`   // ["jira:PROJ-42", "commit:abc123", "wa:sender@2026-04-05T10:22"]
}
```

**Persistence rationale:** the `Dataset` snapshot lets historical reports re-render charts/tables without re-querying Weaviate. Executive reports are immutable snapshots of "what the system knew at that moment." Note: the stored snapshot is the full `CorrelatedDataset` (for render fidelity), independent of the compact 500-char-capped rendering sent to the LLM. If row sizes become a concern in practice, trimming is a follow-up.

## 6. Correlator Algorithm (Hybrid Linking)

**Input:** `userID`, optional `workspaceID`, `DateRange{A, B}`, `staleThresholdDays` (default 7).

### 6.1 Steps

1. **Pull anchors (Jira cards).** Empty-query `SearchJira` with date filter on the card's primary timestamp field (exact field name — `updated_at` vs `last_modified` — to be confirmed against the `JiraCardEmbedding` schema during planning; it must be one that's indexed/filterable in Weaviate). Cap at 200 most recently updated; set `Metrics.Truncated = true` if exceeded.

2. **Pull raw slices (in parallel with Step 1, via `errgroup`):**
   - `rawCommits` — empty-query `SearchCommits` filtered to `[A, B]`.
   - `rawWA` — empty-query `Search` filtered to `[A, B]`.

3. **Build Topics.** For each anchor, in a bounded worker pool (concurrency = 8):
   - `Topic.Anchor = card`
   - **Explicit commit match:** scan `rawCommits` for messages containing `card.CardKey` (regex `\b{KEY}\b`). High-confidence.
   - **Semantic commit match:** `SearchCommits(query=card.Content, userID, A, B, limit=10)`, keep distance `< 0.30`. Merge with explicit, dedupe by SHA.
   - **Semantic WA match:** `Search(query=card.Content, userID, A, B, limit=15)`, keep distance `< 0.32`. Group consecutive messages by `(sender, 10-min window)`.
   - **Staleness:** `Topic.Stale = card.Status == "In Progress" && len(Topic.Commits) == 0 && daysSince(card.UpdatedAt) >= staleThresholdDays`. Compute `DaysIdle`.

4. **Compute orphans.**
   - `OrphanCommits = rawCommits \ ∪(Topic.Commits)`.
   - `OrphanWA`: `rawWA` minus any message already attached to a Topic, then clustered by `(sender, 30-min window)`. Drop groups with `< 3` messages (noise floor).

5. **Aggregate.** Fill `Metrics` and build one `DailyBucket` per day in range (including zero-activity days for continuous chart x-axis).

### 6.2 Tunables (package-level vars, marked for revisit after first real data)

```go
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
)
```

### 6.3 Worst-case query budget

90-day range, 200 anchors → `200 × 2 + 2 ≈ 402` Weaviate calls. Bounded, logged with timing. Acceptable for v1.

## 7. ExecutiveReportAgent

**Location:** `backend/internal/ai/agent/executive_report.go`, following `report.go:16` pattern.

### 7.1 System prompt (abbreviated)

> You are an engineering executive assistant. You will be given a structured dataset describing a developer's work between {A} and {B}: Jira-anchored topics (cards with their linked commits and discussions), orphan commits, orphan discussion groups, and aggregate metrics.
>
> Produce a report in exactly this order:
>
> ## Summary — one paragraph.
> ## Topics — one section per non-trivial Topic (`**{CardKey}** — {title}`), with inline evidence refs.
> ## Gaps — bullets. Discussed-but-untracked and untracked-commit orphans only.
> ## Stale Work — each stale card with a concrete next action.
> ## Next Steps — 3–5 prioritized recommendations with evidence refs.
>
> Cite evidence inline as `[jira:KEY]`, `[commit:sha]`, `[wa:sender@time]`.

### 7.2 User payload

Compact JSON rendering of `CorrelatedDataset`. Raw message bodies capped at 500 chars. Only fields the LLM needs (no embeddings, no internal IDs).

### 7.3 Structured suggestion extraction

Tool-calling. Agent is given one tool `emit_suggestion(kind, title, detail, refs)` and the prompt instructs it to call the tool whenever it identifies a suggestion of kind `gap`, `stale`, or `next_step`. Matches the existing `ReportAgent` tool-use pattern and avoids a second LLM pass.

## 8. HTTP Surface

All routes mounted under `/api/protected/reports/executive/` (auth required, `user_id` from JWT).

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/generate` | SSE stream. Body: `{range_start, range_end, stale_threshold_days?, workspace_id?}` |
| `GET`  | `/` | List caller's executive reports (id, range, status, created_at) |
| `GET`  | `/:id` | Full report (narrative + suggestions + dataset) |
| `DELETE` | `/:id` | Delete caller's report |

### 8.1 SSE Event Contract (`/generate`)

```
event: status      data: {"phase":"correlating"}
event: status      data: {"phase":"thinking"}
event: dataset     data: {...CorrelatedDataset}            # sent once, before LLM
event: delta       data: {"text":"## Summary\n..."}        # many
event: suggestion  data: {"kind":"gap", "title":..., "detail":..., "refs":[...]}
event: status      data: {"phase":"persisting"}
event: done        data: {"id":42}
event: error       data: {"message":"..."}                 # terminal on failure
```

`dataset` is sent early so the frontend can render charts and stale tables while the LLM is still narrating. On `done`, the handler writes: `Narrative` = accumulated delta text, `Suggestions` = collected suggestion events, `Dataset` = sent dataset, `Status = "completed"`, `CompletedAt = now`. On fatal error: `Status = "failed"`, `ErrorMessage = err.Error()`; the row is still persisted so it shows in history.

### 8.2 Authorization

- `GET/DELETE /:id`: must match `report.UserID == claims.UserID`, otherwise 404 (not 403 — don't leak existence).
- All Weaviate calls inside the correlator pass `userID` (and `workspaceID` when set) into `buildWhereFilter`. No endpoint accepts a `user_id` query param.

## 9. Frontend UX

### 9.1 Tab Placement

`ReportsPage.tsx` gains a third `TabsTrigger` `"Executive"` next to the existing Daily/Monthly tabs. The tab content is `ExecutiveReportTab`.

### 9.2 `ExecutiveReportTab` Layout

```
┌───────────────────┬─────────────────────────────────────────┐
│ History sidebar   │ Controls                                │
│ ─────────────     │  Date: [start] → [end]  [Generate]      │
│ ● May 11 ✓        │  Stale threshold: [7] days              │
│   Apr 27→May 11   │                                         │
│ ○ May 04 ✓        ├─────────────────────────────────────────┤
│   Apr 20→May 04   │ ExecutiveReportView                     │
│ ○ Apr 27 ✗ failed │                                         │
└───────────────────┴─────────────────────────────────────────┘
```

Sidebar uses `useListExecutiveReportsQuery()`. Clicking a historical entry loads it via `useGetExecutiveReportQuery(id)` and renders from persisted data (no re-generation, no SSE).

### 9.3 `ExecutiveReportView` Render Order

Render sections as data arrives to keep the user engaged through LLM latency:

1. **Header strip** — range, generated-at, status badge.
2. **Metrics row** (on `dataset` event): three `LinkageGaugeChart` instances for commit-linkage, card-linkage, and WA-topic-ticketed ratio.
3. **Activity timeline** (on `dataset` event): stacked area chart from `DailyBuckets` (commits / Jira changes / WA messages). Reuses `CommitActivityChart` styling.
4. **Stale work table** (on `dataset` event, actions filled by stream): `StaleWorkTable`, rows are `Topics` with `Stale=true`. Columns: card key, title, days idle, last activity, suggested action (populated as stale suggestions arrive).
5. **Narrative** (streams in via `delta`): markdown rendered with the existing chat markdown renderer.
6. **Suggestion panel** (sticky right rail on desktop, collapsible drawer on mobile): grouped by kind. Click to scroll to the matching narrative section or open the external ref.

### 9.4 State Management

- `executiveReport.service.ts` — RTK Query slice with `listExecutiveReports`, `getExecutiveReport(id)`, `deleteExecutiveReport(id)`.
- `useGenerateExecutiveReport()` — custom hook owning an `EventSource`. State: `{ phase, dataset, narrative, suggestions, error, start(body), reset() }`. On `done` it dispatches `invalidateTags(['ExecutiveReport'])` so the sidebar refreshes.

### 9.5 Error Handling

- Mid-stream `error` event: hook sets `error`, view shows inline error card with **Retry**.
- `EventSource` auto-reconnects on network drop. If the server no longer has generation state, Retry starts a fresh `/generate`.
- Historical failed report: non-interactive card showing `ErrorMessage`. User starts a new generation from scratch.

### 9.6 Reuse vs New

| Reused | New |
|---|---|
| `LinkageGaugeChart` | `ExecutiveReportTab` |
| `CommitActivityChart` (extended series) | `ExecutiveReportView` |
| Markdown renderer | `SuggestionList` |
| `Card`, `Tabs`, `DatePicker`, `Button` | `StaleWorkTable` |
|   | `useGenerateExecutiveReport` hook |

## 10. Defaults & Constraints

| Setting | Default | Cap |
|---|---|---|
| Range | Last 14 days | Max 90 days |
| Stale threshold | 7 days | — |
| Max anchors | 200 | — |
| Concurrency per generation | 8 Weaviate workers | — |

## 11. Testing Strategy

### 11.1 Backend

**`correlator_test.go`** — table-driven tests on a fake Weaviate client:

- Happy path: 3 cards, mixed explicit + semantic + orphan commits, mixed WA.
- Explicit + semantic dedupe: a commit with both matches counted once.
- Staleness boundary: exactly `threshold - 1` vs `threshold` days idle.
- Orphan WA clustering: same sender within 30 min → one group; 31 min → two; `< 3` messages → filtered.
- Truncation: 250 anchors → 200 processed, `Metrics.Truncated = true`.
- Empty range: zeroed metrics, empty arrays (not nil), no division by zero.
- Workspace scoping: Jira card in workspace 5 excluded when dataset built for workspace 3.

**`metrics_test.go`** — pure arithmetic, division-by-zero guards.

**`handlers/executive_report_test.go`** — fake correlator + scripted fake agent:

- SSE event order: `status → dataset → delta* → suggestion* → status → done`.
- Final DB row matches streamed content.
- Agent failure → `Status="failed"` row, `error` event, `ErrorMessage` populated.
- Unauthenticated → 401, correlator never called.
- User A requesting user B's report ID → 404.

**No end-to-end LLM test.** Real narrative quality is validated by manual smoke run.

### 11.2 Frontend

**Vitest + Testing Library:**

- `StaleWorkTable` — renders fixture, formats days-idle, empty state.
- `SuggestionList` — groups by kind, click scroll behavior mocked.
- `useGenerateExecutiveReport` — fed a mocked `EventSource` replaying a captured SSE script; asserts phase transitions and accumulated narrative.
- `ExecutiveReportView` with a completed historical report (dataset + narrative + suggestions provided statically) renders without the hook.

**No browser e2e** (no Playwright infra in repo; not introducing one for this feature).

### 11.3 Fixtures

- `backend/internal/services/executive/testdata/` — JSON fixtures (Jira cards, commits, WA messages) + expected `CorrelatedDataset` outputs, shared across correlator and handler tests.
- `frontend/src/presentation/components/executive/__fixtures__/` — one completed report, one failed report, one captured SSE stream.

### 11.4 Explicitly Not Tested in v1

- Distance-threshold tuning (needs real data; follow-up).
- Recharts pixel output (library trusted; assert props only).
- LLM narrative quality (manual review).

## 12. Rollout & Follow-ups

**v1 delivery order (suggested, will be refined in the implementation plan):**

1. Go types + Postgres model + migration.
2. Correlator + tests (the risky piece — land first).
3. `ExecutiveReportAgent` stub (no LLM yet) + handler + SSE + handler tests.
4. Real agent prompt wired to MiniMax client.
5. Frontend service + hook.
6. UI components + tab integration.
7. Manual smoke test on a real user's data.

**Follow-ups tracked (phase 2):**

- Suggestion categories C (focus drift), D (collaboration signal), F (risk flags).
- Topic cluster bubble chart, focus heatmap.
- Distance-threshold tuning based on observed false positives/negatives.
- Team/shared reports (requires permissions model work).
- Export to PDF/markdown.

## 13. Open Questions (to resolve during implementation planning, not blockers)

- Exact MiniMax/Anthropic streaming API shape for tool-use calls — confirm whether `emit_suggestion` tool calls interleave cleanly with text deltas or require separate handling.
- Whether to send the `dataset` event before or after `status: thinking` — current design sends it before, so chart flashes in immediately while LLM spins up.
- GORM `datatypes.JSON` availability in the current module graph — if not already a dependency, consider `json.RawMessage` + manual marshalling.
- Exact Jira timestamp field name in `JiraCardEmbedding` — confirm during planning so the date filter in correlator Step 1 targets the right property.
