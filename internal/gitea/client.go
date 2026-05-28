package gitea

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	repo       string // "owner/name"
	httpClient *http.Client
}

func NewClient(baseURL, token, repo string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		repo:    repo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type FileContent struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Name    string `json:"name"`
}

type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

type CommitInfo struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created"`
	Author    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

type BranchInfo struct {
	Commit struct {
		ID string `json:"id"`
	} `json:"commit"`
}

// encodeRepo splits "owner/name" and escapes each segment separately.
func encodeRepo(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		return url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1])
	}
	return url.PathEscape(repo)
}

// encodePath splits a "/" delimited path into segments and applies
// url.PathEscape to each one, then re-joins with "/".
func encodePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func (c *Client) GetFile(ctx context.Context, path string) (*FileContent, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/contents/%s", encodeRepo(c.repo), encodePath(path))
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var file FileContent
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}
	file.Content = string(decoded)
	return &file, nil
}

func (c *Client) PutFile(ctx context.Context, path, content, message, sha string) error {
	payload := map[string]interface{}{
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"message": message,
	}
	if sha != "" {
		payload["sha"] = sha
	}
	endpoint := fmt.Sprintf("/api/v1/repos/%s/contents/%s", encodeRepo(c.repo), encodePath(path))
	_, err := c.doPut(ctx, endpoint, payload)
	return err
}

func (c *Client) DeleteFile(ctx context.Context, path, sha, message string) error {
	payload := map[string]interface{}{
		"sha":     sha,
		"message": message,
	}
	endpoint := fmt.Sprintf("/api/v1/repos/%s/contents/%s", encodeRepo(c.repo), encodePath(path))
	_, err := c.doDelete(ctx, endpoint, payload)
	return err
}

func (c *Client) ListTree(ctx context.Context, path string, recursive bool) ([]TreeEntry, error) {
	branchEndpoint := fmt.Sprintf("/api/v1/repos/%s/branches/main", encodeRepo(c.repo))
	branchBody, err := c.doGet(ctx, branchEndpoint)
	if err != nil {
		return nil, err
	}
	var branch BranchInfo
	if err := json.Unmarshal(branchBody, &branch); err != nil {
		return nil, err
	}

	treeEndpoint := fmt.Sprintf("/api/v1/repos/%s/git/trees/%s", encodeRepo(c.repo), url.PathEscape(branch.Commit.ID))
	if recursive {
		treeEndpoint += "?recursive=1"
	}
	treeBody, err := c.doGet(ctx, treeEndpoint)
	if err != nil {
		return nil, err
	}
	var tree struct {
		Tree []TreeEntry `json:"tree"`
	}
	if err := json.Unmarshal(treeBody, &tree); err != nil {
		return nil, err
	}

	entries := tree.Tree
	if path != "" {
		filtered := make([]TreeEntry, 0)
		for _, e := range entries {
			if strings.HasPrefix(e.Path, path) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	return entries, nil
}

func (c *Client) GetFileHistory(ctx context.Context, path string, page, limit int) ([]CommitInfo, error) {
	endpoint := fmt.Sprintf("/api/v1/repos/%s/commits?path=%s&page=%d&limit=%d",
		encodeRepo(c.repo), url.QueryEscape(path), page, limit)
	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	commits := make([]CommitInfo, 0)
	if err := json.Unmarshal(body, &commits); err != nil {
		return nil, err
	}
	return commits, nil
}

// Issue methods moved to issue.go

func (c *Client) doGet(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitea %s: %d %s", endpoint, resp.StatusCode, string(body[:min(len(body), 200)]))
	}
	return body, nil
}

func (c *Client) doPost(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	return c.doMethod(ctx, "POST", endpoint, payload)
}

func (c *Client) doPut(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	return c.doMethod(ctx, "PUT", endpoint, payload)
}

func (c *Client) doDelete(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	return c.doMethod(ctx, "DELETE", endpoint, payload)
}

func (c *Client) doMethod(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gitea %s %s: %d %s", method, endpoint, resp.StatusCode, string(body[:min(len(body), 200)]))
	}
	return body, nil
}
