package services

import (
	"regexp"
	"strings"
	"time"
)

type CommitInfo struct {
	SHA         string
	Message     string
	Author      string
	AuthorEmail string
	Branch      string
	Date        time.Time
}

type CommitDiff struct {
	SHA     string       `json:"sha"`
	Message string       `json:"message"`
	Files   []FileChange `json:"files"`
	Stats   DiffStats    `json:"stats"`
}

type FileChange struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // "added", "modified", "removed", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"` // actual diff (truncated)
}

type DiffStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

type CommitProvider interface {
	FetchCommits(owner, repo, token string, since time.Time) ([]CommitInfo, error)
	FetchBranches(owner, repo, token string) ([]string, error)
	FetchBranchCommits(owner, repo, branch, token string, since time.Time) ([]CommitInfo, error)
	ValidateAccess(owner, repo, token string) error
}

type DiffProvider interface {
	FetchCommitDiff(owner, repo, sha, token string) (*CommitDiff, error)
}

var jiraKeyRegex = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

func ExtractJiraKey(message string) string {
	// Take first line only for matching
	firstLine := strings.SplitN(message, "\n", 2)[0]
	match := jiraKeyRegex.FindString(firstLine)
	return match
}
