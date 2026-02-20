package vestaboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL        = "https://cloud.vestaboard.com"
	writePath      = "/"
	vbmlBaseURL    = "https://vbml.vestaboard.com"
	vbmlFormatPath = "/compose"
	headerName     = "X-vestaboard-token"
)

var errMissingAPIKey = errors.New("VESTABOARD_TOKEN is not set")

type Client struct {
	baseURL    string
	vbmlURL    string
	apiKey     string
	httpClient *http.Client
	verbose    bool
	logWriter  io.Writer
}

type Option func(*Client)

func WithVerboseLogging(enabled bool, writer io.Writer) Option {
	return func(c *Client) {
		c.verbose = enabled
		c.logWriter = writer
	}
}

func NewClient(apiKey string, options ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errMissingAPIKey
	}
	client := &Client{
		baseURL: baseURL,
		vbmlURL: vbmlBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}
	return client, nil
}

func (c *Client) SendText(ctx context.Context, text string) error {
	payload := map[string]string{"text": text}
	return c.postMessage(ctx, payload)
}

func (c *Client) SendCharacters(ctx context.Context, characters [][]int) error {
	payload := map[string][][]int{"characters": characters}
	return c.postMessage(ctx, payload)
}

func (c *Client) FormatMessage(ctx context.Context, message, model, align, justify string) ([][]int, error) {
	payload := map[string]any{
		"components": []map[string]any{
			{
				"template": message,
				"style": map[string]string{
					"align":   align,
					"justify": justify,
				},
			},
		},
	}
	if model == "note" {
		payload["style"] = map[string]int{
			"height": 3,
			"width":  15,
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.vbmlURL+vbmlFormatPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.logHTTP("request", req.URL.String(), body, 0)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read VBML response: %w", err)
	}
	c.logHTTP("response", req.URL.String(), bodyBytes, resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vbml API returned %s: %s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	var response struct {
		Characters [][]int `json:"characters"`
	}
	if err := json.Unmarshal(bodyBytes, &response); err == nil && len(response.Characters) > 0 {
		return response.Characters, nil
	}

	var characters [][]int
	if err := json.Unmarshal(bodyBytes, &characters); err == nil && len(characters) > 0 {
		return characters, nil
	}

	return nil, errors.New("vbml API returned no characters")
}

func (c *Client) postMessage(ctx context.Context, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+writePath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerName, c.apiKey)
	c.logHTTP("request", req.URL.String(), body, 0)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read API response: %w", err)
	}
	c.logHTTP("response", req.URL.String(), respBody, resp.StatusCode)

	if resp.StatusCode == http.StatusConflict {
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vestaboard API returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (c *Client) logHTTP(direction, url string, payload []byte, statusCode int) {
	if !c.verbose || c.logWriter == nil {
		return
	}
	if direction == "request" {
		_, _ = fmt.Fprintf(c.logWriter, "request URL: %s\n", url)
		_, _ = fmt.Fprintf(c.logWriter, "request payload:\n%s\n", prettyJSON(payload))
		return
	}
	_, _ = fmt.Fprintf(c.logWriter, "response URL: %s\n", url)
	_, _ = fmt.Fprintf(c.logWriter, "response status: %d\n", statusCode)
	_, _ = fmt.Fprintf(c.logWriter, "response payload:\n%s\n", prettyJSON(payload))
}

func prettyJSON(payload []byte) string {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return "<empty>"
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, trimmed, "", "  "); err == nil {
		return pretty.String()
	}
	return string(trimmed)
}
