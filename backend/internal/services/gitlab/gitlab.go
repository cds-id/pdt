package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cds-id/pdt/backend/internal/services"
)

const defaultBaseURL = "https://gitlab.com"

type Client struct {
	BaseURL string
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{BaseURL: baseURL}
}

type gitlabCommit struct {
	ID             string    `json:"id"`
	Message        string    `json:"message"`
	AuthorName     string    `json:"author_name"`
	AuthorEmail    string    `json:"author_email"`
	AuthoredDate   time.Time `json:"authored_date"`
}

type gitlabBranch struct {
	Name string `json:"name"`
}

// FetchCommits fetches all commits across all branches with branch info.
func (c *Client) FetchCommits(owner, repo, token string, since time.Time) ([]services.CommitInfo, error) {
	branches, err := c.FetchBranches(owner, repo, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch branches: %w", err)
	}

	seen := map[string]bool{}
	var allCommits []services.CommitInfo

	for _, branch := range branches {
		commits, err := c.FetchBranchCommits(owner, repo, branch, token, since)
		if err != nil {
			continue
		}
		for _, ci := range commits {
			if !seen[ci.SHA] {
				seen[ci.SHA] = true
				allCommits = append(allCommits, ci)
			}
		}
	}

	return allCommits, nil
}

func (c *Client) FetchBranches(owner, repo, token string) ([]string, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	var allBranches []string
	page := 1

	for {
		reqURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/branches?per_page=100&page=%d",
			c.BaseURL, projectPath, page)

		body, err := c.doRequest(reqURL, token)
		if err != nil {
			return nil, err
		}

		var branches []gitlabBranch
		if err := json.Unmarshal(body, &branches); err != nil {
			return nil, fmt.Errorf("failed to parse branches: %w", err)
		}

		if len(branches) == 0 {
			break
		}

		for _, b := range branches {
			allBranches = append(allBranches, b.Name)
		}

		if len(branches) < 100 {
			break
		}
		page++
	}

	return allBranches, nil
}

func (c *Client) FetchBranchCommits(owner, repo, branch, token string, since time.Time) ([]services.CommitInfo, error) {
	projectPath := url.PathEscape(owner + "/" + repo)
	var allCommits []services.CommitInfo
	page := 1

	for {
		reqURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits?ref_name=%s&since=%s&per_page=100&page=%d",
			c.BaseURL, projectPath, url.QueryEscape(branch), since.Format(time.RFC3339), page)

		body, err := c.doRequest(reqURL, token)
		if err != nil {
			return nil, err
		}

		var commits []gitlabCommit
		if err := json.Unmarshal(body, &commits); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(commits) == 0 {
			break
		}

		for _, gc := range commits {
			allCommits = append(allCommits, services.CommitInfo{
				SHA:         gc.ID,
				Message:     gc.Message,
				Author:      gc.AuthorName,
				AuthorEmail: gc.AuthorEmail,
				Branch:      branch,
				Date:        gc.AuthoredDate,
			})
		}

		if len(commits) < 100 {
			break
		}
		page++
	}

	return allCommits, nil
}

func (c *Client) FetchCommitDiff(owner, repo, sha, token string) (*services.CommitDiff, error) {
	projectPath := url.PathEscape(owner + "/" + repo)

	// Fetch commit detail for message
	commitURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s", c.BaseURL, projectPath, sha)
	commitBody, err := c.doRequest(commitURL, token)
	if err != nil {
		return nil, fmt.Errorf("fetch commit detail: %w", err)
	}

	var commitResp struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(commitBody, &commitResp); err != nil {
		return nil, fmt.Errorf("parse commit detail: %w", err)
	}

	// Fetch diff
	diffURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/diff", c.BaseURL, projectPath, sha)
	diffBody, err := c.doRequest(diffURL, token)
	if err != nil {
		return nil, fmt.Errorf("fetch commit diff: %w", err)
	}

	var fileDiffs []struct {
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
		Diff        string `json:"diff"`
	}
	if err := json.Unmarshal(diffBody, &fileDiffs); err != nil {
		return nil, fmt.Errorf("parse commit diff: %w", err)
	}

	diff := &services.CommitDiff{
		SHA:     commitResp.ID,
		Message: commitResp.Message,
	}

	for _, f := range fileDiffs {
		status := "modified"
		if f.NewFile {
			status = "added"
		} else if f.DeletedFile {
			status = "removed"
		} else if f.RenamedFile {
			status = "renamed"
		}

		patch := f.Diff
		if len(patch) > 500 {
			patch = patch[:500] + "\n... (truncated)"
		}

		// Count additions/deletions from diff lines
		additions, deletions := 0, 0
		for _, line := range strings.Split(f.Diff, "\n") {
			if len(line) > 0 && line[0] == '+' && (len(line) < 3 || line[1] != '+' || line[2] != '+') {
				additions++
			} else if len(line) > 0 && line[0] == '-' && (len(line) < 3 || line[1] != '-' || line[2] != '-') {
				deletions++
			}
		}

		filename := f.NewPath
		if filename == "" {
			filename = f.OldPath
		}

		diff.Files = append(diff.Files, services.FileChange{
			Filename:  filename,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Patch:     patch,
		})
		diff.Stats.Additions += additions
		diff.Stats.Deletions += deletions
		diff.Stats.Total += additions + deletions
	}

	return diff, nil
}

func (c *Client) ValidateAccess(owner, repo, token string) error {
	projectPath := url.PathEscape(owner + "/" + repo)
	reqURL := fmt.Sprintf("%s/api/v4/projects/%s", c.BaseURL, projectPath)
	_, err := c.doRequest(reqURL, token)
	return err
}

func (c *Client) doRequest(reqURL, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid token")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("project not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var body []byte
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	return body, nil
}
