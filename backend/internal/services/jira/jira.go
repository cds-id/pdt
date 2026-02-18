package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	Workspace string
	Email     string
	Token     string
}

func New(workspace, email, token string) *Client {
	return &Client{
		Workspace: workspace,
		Email:     email,
		Token:     token,
	}
}

type SprintInfo struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	StartDate *time.Time `json:"startDate"`
	EndDate   *time.Time `json:"endDate"`
}

type CardInfo struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Assignee string `json:"assignee"`
}

type IssueDetail struct {
	Key       string          `json:"key"`
	Summary   string          `json:"summary"`
	Status    string          `json:"status"`
	Assignee  string          `json:"assignee"`
	IssueType string          `json:"issue_type"`
	Parent    *IssueRef       `json:"parent,omitempty"`
	Subtasks  []IssueRef      `json:"subtasks,omitempty"`
	Changelog []ChangeHistory `json:"changelog,omitempty"`
}

type IssueRef struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

type ChangeHistory struct {
	Author  string       `json:"author"`
	Created string       `json:"created"`
	Items   []ChangeItem `json:"items"`
}

type ChangeItem struct {
	Field      string `json:"field"`
	FromString string `json:"from_string"`
	ToString   string `json:"to_string"`
}

type boardResponse struct {
	Values []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"values"`
}

type sprintResponse struct {
	Values []SprintInfo `json:"values"`
}

type issueResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
		} `json:"fields"`
	} `json:"issues"`
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://%s/rest", c.Workspace)
}

func (c *Client) FetchBoards() ([]int, error) {
	url := fmt.Sprintf("%s/agile/1.0/board", c.baseURL())
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp boardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse boards: %w", err)
	}

	var ids []int
	for _, b := range resp.Values {
		ids = append(ids, b.ID)
	}
	return ids, nil
}

func (c *Client) FetchSprints(boardID int) ([]SprintInfo, error) {
	url := fmt.Sprintf("%s/agile/1.0/board/%d/sprint", c.baseURL(), boardID)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp sprintResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse sprints: %w", err)
	}

	return resp.Values, nil
}

func (c *Client) FetchSprintIssues(sprintID int) ([]CardInfo, error) {
	url := fmt.Sprintf("%s/agile/1.0/sprint/%d/issue?maxResults=200", c.baseURL(), sprintID)
	body, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var resp issueResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	var cards []CardInfo
	for _, issue := range resp.Issues {
		assignee := ""
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		cards = append(cards, CardInfo{
			Key:      issue.Key,
			Summary:  issue.Fields.Summary,
			Status:   issue.Fields.Status.Name,
			Assignee: assignee,
		})
	}

	return cards, nil
}

func (c *Client) FetchIssue(key string) (*IssueDetail, error) {
	reqURL := fmt.Sprintf("%s/api/2/issue/%s?fields=summary,status,assignee,parent,subtasks,issuetype&expand=changelog", c.baseURL(), key)
	body, err := c.doRequest(reqURL)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Key    string `json:"key"`
		Fields struct {
			Summary   string `json:"summary"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
			Parent *struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string `json:"summary"`
					Status  struct {
						Name string `json:"name"`
					} `json:"status"`
					IssueType struct {
						Name string `json:"name"`
					} `json:"issuetype"`
				} `json:"fields"`
			} `json:"parent"`
			Subtasks []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string `json:"summary"`
					Status  struct {
						Name string `json:"name"`
					} `json:"status"`
					IssueType struct {
						Name string `json:"name"`
					} `json:"issuetype"`
				} `json:"fields"`
			} `json:"subtasks"`
		} `json:"fields"`
		Changelog struct {
			Histories []struct {
				Author struct {
					DisplayName string `json:"displayName"`
				} `json:"author"`
				Created string `json:"created"`
				Items   []struct {
					Field      string `json:"field"`
					FromString string `json:"fromString"`
					ToString   string `json:"toString"`
				} `json:"items"`
			} `json:"histories"`
		} `json:"changelog"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	detail := &IssueDetail{
		Key:       raw.Key,
		Summary:   raw.Fields.Summary,
		Status:    raw.Fields.Status.Name,
		IssueType: raw.Fields.IssueType.Name,
	}

	if raw.Fields.Assignee != nil {
		detail.Assignee = raw.Fields.Assignee.DisplayName
	}

	if raw.Fields.Parent != nil {
		detail.Parent = &IssueRef{
			Key:     raw.Fields.Parent.Key,
			Summary: raw.Fields.Parent.Fields.Summary,
			Status:  raw.Fields.Parent.Fields.Status.Name,
			Type:    raw.Fields.Parent.Fields.IssueType.Name,
		}
	}

	for _, st := range raw.Fields.Subtasks {
		detail.Subtasks = append(detail.Subtasks, IssueRef{
			Key:     st.Key,
			Summary: st.Fields.Summary,
			Status:  st.Fields.Status.Name,
			Type:    st.Fields.IssueType.Name,
		})
	}

	for _, h := range raw.Changelog.Histories {
		history := ChangeHistory{
			Author:  h.Author.DisplayName,
			Created: h.Created,
		}
		for _, item := range h.Items {
			history.Items = append(history.Items, ChangeItem{
				Field:      item.Field,
				FromString: item.FromString,
				ToString:   item.ToString,
			})
		}
		detail.Changelog = append(detail.Changelog, history)
	}

	return detail, nil
}

func (c *Client) Validate() error {
	url := fmt.Sprintf("%s/api/2/myself", c.baseURL())
	_, err := c.doRequest(url)
	return err
}

func (c *Client) doRequest(reqURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.Email + ":" + c.Token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("unauthorized: check jira credentials")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found: %s", reqURL)
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
