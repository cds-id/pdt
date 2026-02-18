package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cds-id/pdt/backend/internal/services"
)

const baseURL = "https://api.github.com"

type Client struct{}

func New() *Client {
	return &Client{}
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type githubBranch struct {
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
	var allBranches []string
	page := 1

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=100&page=%d",
			baseURL, owner, repo, page)

		body, err := c.doRequest(url, token)
		if err != nil {
			return nil, err
		}

		var branches []githubBranch
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
	var allCommits []services.CommitInfo
	page := 1

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/commits?sha=%s&since=%s&per_page=100&page=%d",
			baseURL, owner, repo, branch, since.Format(time.RFC3339), page)

		body, err := c.doRequest(url, token)
		if err != nil {
			return nil, err
		}

		var commits []githubCommit
		if err := json.Unmarshal(body, &commits); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(commits) == 0 {
			break
		}

		for _, gc := range commits {
			allCommits = append(allCommits, services.CommitInfo{
				SHA:         gc.SHA,
				Message:     gc.Commit.Message,
				Author:      gc.Commit.Author.Name,
				AuthorEmail: gc.Commit.Author.Email,
				Branch:      branch,
				Date:        gc.Commit.Author.Date,
			})
		}

		if len(commits) < 100 {
			break
		}
		page++
	}

	return allCommits, nil
}

func (c *Client) ValidateAccess(owner, repo, token string) error {
	url := fmt.Sprintf("%s/repos/%s/%s", baseURL, owner, repo)
	_, err := c.doRequest(url, token)
	return err
}

func (c *Client) doRequest(url, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid token")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repository not found: %s", url)
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
