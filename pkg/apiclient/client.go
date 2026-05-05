package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Collection struct {
	BinType string `json:"bin_type"`
	Date    string `json:"date"`
}

type Client struct {
	BaseURL    string
	Token      string
	HTTP       *http.Client
	RetryDelay time.Duration
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		RetryDelay: 500 * time.Millisecond,
	}
}

type pushBody struct {
	ScrapedAt   string       `json:"scraped_at"`
	Collections []Collection `json:"collections"`
}

func (c *Client) PushCollections(label string, scrapedAt time.Time, items []Collection) error {
	if items == nil {
		items = []Collection{}
	}
	body := pushBody{
		ScrapedAt:   scrapedAt.UTC().Format(time.RFC3339),
		Collections: items,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	target := c.BaseURL + "/v1/locations/" + url.PathEscape(label) + "/collections"

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(c.RetryDelay)
		}
		req, err := http.NewRequest(http.MethodPut, target, bytes.NewReader(buf))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("api returned %d: %s", resp.StatusCode, string(respBody))
		if resp.StatusCode < 500 && resp.StatusCode != http.StatusRequestTimeout {
			// Don't retry 4xx other than timeouts.
			return lastErr
		}
	}
	return lastErr
}
