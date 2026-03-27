package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Base   string
	Token  string
	Client *http.Client
}

func New(base, token string) *Client {
	return &Client{
		Base:   strings.TrimRight(base, "/"),
		Token:  token,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) doJSON(method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Base+path, rdr)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil {
		if len(raw) == 0 {
			return resp.StatusCode, nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("%s: %w", string(raw), err)
		}
	}
	return resp.StatusCode, nil
}

type CLIStartResp struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	Interval        int    `json:"interval"`
	VerificationURL string `json:"verification_url"`
}

type CLIPollReq struct {
	DeviceCode string `json:"device_code"`
}

type CLIPollResp struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
}

func (c *Client) CLIStart() (*CLIStartResp, error) {
	var out CLIStartResp
	code, err := c.doJSON(http.MethodPost, "/v1/auth/cli/start", nil, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("start failed: %d", code)
	}
	return &out, nil
}

func (c *Client) CLIPoll(deviceCode string) (*CLIPollResp, error) {
	var out CLIPollResp
	code, err := c.doJSON(http.MethodPost, "/v1/auth/cli/poll", CLIPollReq{DeviceCode: deviceCode}, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("poll failed: %d", code)
	}
	return &out, nil
}

type MeResp struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	OrgID  string `json:"org_id"`
	Plan   string `json:"plan"`
}

func (c *Client) Me() (*MeResp, error) {
	var out MeResp
	code, err := c.doJSON(http.MethodGet, "/v1/me", nil, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("me failed: %d", code)
	}
	return &out, nil
}

type ProjectCreate struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

func (c *Client) CreateProject(req ProjectCreate) (string, error) {
	var out map[string]string
	code, err := c.doJSON(http.MethodPost, "/v1/projects", req, &out)
	if err != nil {
		return "", err
	}
	if code >= 300 {
		return "", fmt.Errorf("create project: %d", code)
	}
	return out["id"], nil
}

type ProjectsResp struct {
	Projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"projects"`
}

func (c *Client) ListProjects() (*ProjectsResp, error) {
	var out ProjectsResp
	code, err := c.doJSON(http.MethodGet, "/v1/projects", nil, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("list projects: %d", code)
	}
	return &out, nil
}

func (c *Client) PostConnection(projectID, provider, source string) error {
	payload := map[string]string{"provider": provider}
	if strings.TrimSpace(source) != "" {
		payload["source"] = source
	}
	code, err := c.doJSON(http.MethodPost, "/v1/projects/"+projectID+"/connections", payload, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("connect: %d", code)
	}
	return nil
}

type Connection struct {
	ID         string  `json:"id"`
	Provider   string  `json:"provider"`
	Status     string  `json:"status"`
	Source     string  `json:"source"`
	LastError  *string `json:"last_error,omitempty"`
	LastSyncAt *string `json:"last_sync_at,omitempty"`
}

func (c *Client) ListConnections(projectID string) ([]Connection, error) {
	var out struct {
		Connections []Connection `json:"connections"`
	}
	code, err := c.doJSON(http.MethodGet, "/v1/projects/"+projectID+"/connections", nil, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("connections: %d", code)
	}
	if out.Connections == nil {
		return []Connection{}, nil
	}
	return out.Connections, nil
}

type SyncSnapshot struct {
	Date      string         `json:"date"`
	Provider  string         `json:"provider"`
	AmountUSD float64        `json:"amount_usd"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type PostSyncReq struct {
	Snapshots []SyncSnapshot `json:"snapshots"`
}

func (c *Client) PostSync(projectID string, req PostSyncReq) error {
	code, err := c.doJSON(http.MethodPost, "/v1/projects/"+projectID+"/sync", req, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("sync: %d", code)
	}
	return nil
}

func (c *Client) GetReport(projectID, from, to string) ([]byte, error) {
	q := ""
	if from != "" || to != "" {
		q = "?"
		if from != "" {
			q += "from=" + from + "&"
		}
		if to != "" {
			q += "to=" + to
		}
	}
	req, _ := http.NewRequest(http.MethodGet, c.Base+"/v1/projects/"+projectID+"/report"+q, nil)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) GetAnomalies(projectID string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, c.Base+"/v1/projects/"+projectID+"/anomalies", nil)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

type AskResp struct {
	Answer string `json:"answer"`
}

func (c *Client) Ask(projectID, question string) (*AskResp, error) {
	var out AskResp
	code, err := c.doJSON(http.MethodPost, "/v1/projects/"+projectID+"/ask", map[string]string{"question": question}, &out)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("ask: %d", code)
	}
	return &out, nil
}
