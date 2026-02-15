// Package linkedin provides a client for the LinkedIn Member Data Portability APIs.
// It supports both the Snapshot API (for historical data) and the Changelog API
// (for incremental updates within a 28-day rolling window).
package linkedin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL        = "https://api.linkedin.com/rest"
	linkedInAPIVer = "202312"
)

// Client wraps the LinkedIn REST API for Member Data Portability.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a new LinkedIn API client with the given OAuth token.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest executes an authenticated request against the LinkedIn API.
func (c *Client) doRequest(method, endpoint string, params url.Values) ([]byte, error) {
	u := baseURL + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Linkedin-Version", linkedInAPIVer)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ---------------------------------------------------------------------------
// Snapshot API
// ---------------------------------------------------------------------------

// SnapshotResponse is the top-level response from the Member Snapshot API.
type SnapshotResponse struct {
	Paging   Paging            `json:"paging"`
	Elements []SnapshotElement `json:"elements"`
}

// SnapshotElement is a single domain result inside a snapshot response.
type SnapshotElement struct {
	SnapshotDomain string                   `json:"snapshotDomain"`
	SnapshotData   []map[string]interface{} `json:"snapshotData"`
}

// Paging contains pagination metadata returned by LinkedIn APIs.
type Paging struct {
	Start int          `json:"start"`
	Count int          `json:"count"`
	Total int          `json:"total"`
	Links []PagingLink `json:"links"`
}

// PagingLink is a rel link inside a Paging block.
type PagingLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
	Type string `json:"type"`
}

// FetchSnapshot retrieves a single page of Snapshot data for the given domain.
// domain should be one of the LinkedIn snapshot domain enums, e.g.
// "MEMBER_SHARE_INFO", "ARTICLES", "INSTANT_REPOSTS", "ALL_COMMENTS", etc.
// start is the pagination offset (0-based page index).
func (c *Client) FetchSnapshot(domain string, start int) (*SnapshotResponse, error) {
	params := url.Values{}
	params.Set("q", "criteria")
	if domain != "" {
		params.Set("domain", domain)
	}
	if start > 0 {
		params.Set("start", strconv.Itoa(start))
	}

	body, err := c.doRequest(http.MethodGet, "/memberSnapshotData", params)
	if err != nil {
		return nil, err
	}

	var resp SnapshotResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding snapshot response: %w", err)
	}
	return &resp, nil
}

// FetchAllSnapshotPages iterates through all pages for a given domain and
// returns every SnapshotData entry collected.
func (c *Client) FetchAllSnapshotPages(domain string) ([]map[string]interface{}, error) {
	var all []map[string]interface{}
	start := 0

	for {
		resp, err := c.FetchSnapshot(domain, start)
		if err != nil {
			// LinkedIn signals end-of-data with an error "No data found for this memberId"
			if len(all) > 0 {
				break
			}
			return nil, err
		}

		for _, elem := range resp.Elements {
			all = append(all, elem.SnapshotData...)
		}

		// Check if there's a "next" link
		hasNext := false
		for _, link := range resp.Paging.Links {
			if link.Rel == "next" {
				hasNext = true
				break
			}
		}
		if !hasNext {
			break
		}
		start++
	}

	return all, nil
}

// ---------------------------------------------------------------------------
// Changelog API
// ---------------------------------------------------------------------------

// ChangelogResponse is the top-level response from the Member Changelog API.
type ChangelogResponse struct {
	Elements []ChangelogEvent `json:"elements"`
	Paging   Paging           `json:"paging"`
}

// ChangelogEvent represents a single changelog activity record.
type ChangelogEvent struct {
	ID                string          `json:"id"`
	ActivityID        string          `json:"activityId"`
	CapturedAt        int64           `json:"capturedAt"`
	ProcessedAt       int64           `json:"processedAt"`
	ConfigVersion     int             `json:"configVersion"`
	Owner             string          `json:"owner"`
	Actor             string          `json:"actor"`
	ResourceName      string          `json:"resourceName"`
	ResourceID        string          `json:"resourceId"`
	ResourceURI       string          `json:"resourceUri"`
	Method            string          `json:"method"`         // CREATE, UPDATE, PARTIAL_UPDATE, DELETE
	MethodName        string          `json:"methodName"`     // optional, only for ACTION
	ActivityStatus    string          `json:"activityStatus"` // SUCCESS, FAILURE, SUCCESSFUL_REPLAY
	Activity          json.RawMessage `json:"activity"`
	ProcessedActivity json.RawMessage `json:"processedActivity"`
}

// FetchChangelog retrieves changelog events, optionally starting from startTimeMs
// (epoch milliseconds). count controls how many events per request (recommended: 10, max: 50).
func (c *Client) FetchChangelog(startTimeMs int64, count int) (*ChangelogResponse, error) {
	params := url.Values{}
	params.Set("q", "memberAndApplication")

	if startTimeMs > 0 {
		params.Set("startTime", strconv.FormatInt(startTimeMs, 10))
	}
	if count > 0 {
		params.Set("count", strconv.Itoa(count))
	}

	body, err := c.doRequest(http.MethodGet, "/memberChangeLogs", params)
	if err != nil {
		return nil, err
	}

	var resp ChangelogResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding changelog response: %w", err)
	}
	return &resp, nil
}

// CheckAuthorization verifies that changelog event generation is active for
// the authenticated member.
func (c *Client) CheckAuthorization() ([]byte, error) {
	params := url.Values{}
	params.Set("q", "memberAndApplication")
	return c.doRequest(http.MethodGet, "/memberAuthorizations", params)
}

// EnableChangelog explicitly enables changelog event generation by POSTing to
// the memberAuthorizations endpoint (usually auto-enabled on consent).
func (c *Client) EnableChangelog() error {
	req, err := http.NewRequest(http.MethodPost, baseURL+"/memberAuthorizations", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Linkedin-Version", linkedInAPIVer)

	// LinkedIn requires an empty JSON body
	req.Body = io.NopCloser(nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("enable changelog failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
