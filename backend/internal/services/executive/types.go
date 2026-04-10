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
	Kind   string   `json:"kind"`
	Title  string   `json:"title"`
	Detail string   `json:"detail"`
	Refs   []string `json:"refs"`
}
