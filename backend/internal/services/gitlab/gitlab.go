package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
