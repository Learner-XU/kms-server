package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Issue represents a Gitea issue
type Issue struct {
	ID        int       `json:"id"`
	Index     int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"` // open, closed
	Labels    []Label   `json:"labels"`
	Assignees []UserRef `json:"assignees"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	ClosedAt  *string   `json:"closed_at"`
	Comments  int       `json:"comments"`
}

// Label represents a Gitea label
type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UserRef is a minimal user reference
type UserRef struct {
	Login string `json:"login"`
}

// CreateIssueReq is the request body for creating an issue
type CreateIssueReq struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// UpdateIssueReq is the request body for updating an issue
type UpdateIssueReq struct {
	Title  *string  `json:"title,omitempty"`
	Body   *string  `json:"body,omitempty"`
	State  *string  `json:"state,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

// Comment represents a Gitea issue comment
type Comment struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      UserRef `json:"user"`
}

// GetIssue gets a single issue by index
func (c *Client) GetIssue(ctx context.Context, index int) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues/%d", encodeRepo(c.repo), index)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// ListIssuesDetailed lists issues with optional state and label filters
func (c *Client) ListIssuesDetailed(ctx context.Context, state string, labels string, page, limit int) ([]Issue, error) {
	params := url.Values{}
	if state != "" {
		params.Set("state", state)
	}
	if labels != "" {
		params.Set("labels", labels)
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues?%s", encodeRepo(c.repo), params.Encode())
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(body, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// CreateIssueDetailed creates an issue with full request body
func (c *Client) CreateIssueDetailed(ctx context.Context, req CreateIssueReq) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues", encodeRepo(c.repo))
	body, err := c.doPost(ctx, endpoint, req)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// UpdateIssue updates an issue
func (c *Client) UpdateIssue(ctx context.Context, index int, req UpdateIssueReq) (*Issue, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues/%d", encodeRepo(c.repo), index)
	body, err := c.doMethod(ctx, "PATCH", endpoint, req)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// ListLabels lists all labels
func (c *Client) ListLabels(ctx context.Context) ([]Label, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/labels", encodeRepo(c.repo))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var labels []Label
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, err
	}
	return labels, nil
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(ctx context.Context, name, color string) (*Label, error) {
	payload := map[string]interface{}{
		"name":  name,
		"color": color,
	}
	endpoint := fmt.Sprintf("/api/v1/repos/%s/labels", encodeRepo(c.repo))
	body, err := c.doPost(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var label Label
	if err := json.Unmarshal(body, &label); err != nil {
		return nil, err
	}
	return &label, nil
}

// ListComments lists comments on an issue
func (c *Client) ListComments(ctx context.Context, issueIndex int) ([]Comment, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues/%d/comments", encodeRepo(c.repo), issueIndex)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var comments []Comment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// AddComment adds a comment to an issue
func (c *Client) AddComment(ctx context.Context, issueIndex int, bodyText string) (*Comment, error) {
	payload := map[string]interface{}{
		"body": bodyText,
	}
	endpoint := fmt.Sprintf("/api/v1/repos/%s/issues/%d/comments", encodeRepo(c.repo), issueIndex)
	body, err := c.doPost(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var comment Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}
