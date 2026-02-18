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

type CommitProvider interface {
	FetchCommits(owner, repo, token string, since time.Time) ([]CommitInfo, error)
	FetchBranches(owner, repo, token string) ([]string, error)
	FetchBranchCommits(owner, repo, branch, token string, since time.Time) ([]CommitInfo, error)
	ValidateAccess(owner, repo, token string) error
}

var jiraKeyRegex = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

func ExtractJiraKey(message string) string {
	// Take first line only for matching
	firstLine := strings.SplitN(message, "\n", 2)[0]
	match := jiraKeyRegex.FindString(firstLine)
	return match
}
