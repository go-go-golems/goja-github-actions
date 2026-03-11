package githubapi

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type RequestResult struct {
	Status  int                 `json:"status"`
	Data    interface{}         `json:"data"`
	Headers map[string][]string `json:"headers"`
}

func NewClient(token string, baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.github.com"
	}

	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) DoRoute(
	ctx context.Context,
	route string,
	params map[string]interface{},
) (*RequestResult, error) {
	request, err := BuildRequest(ctx, c.BaseURL, route, params)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if strings.TrimSpace(c.Token) != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result, err := DecodeResponse(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return nil, &APIError{
			Status:  response.StatusCode,
			Message: extractErrorMessage(result),
			Result:  result,
		}
	}
	return result, nil
}
