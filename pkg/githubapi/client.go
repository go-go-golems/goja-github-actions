package githubapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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
	started := time.Now()
	request, err := BuildRequest(ctx, c.BaseURL, route, params)
	if err != nil {
		log.Debug().
			Str("component", "githubapi").
			Str("route", route).
			Err(err).
			Msg("Failed to build GitHub API request")
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if strings.TrimSpace(c.Token) != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}

	log.Debug().
		Str("component", "githubapi").
		Str("method", request.Method).
		Str("route", route).
		Str("base_url", c.BaseURL).
		Str("request_url", request.URL.String()).
		Bool("auth_present", strings.TrimSpace(c.Token) != "").
		Msg("Sending GitHub API request")

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		log.Debug().
			Str("component", "githubapi").
			Str("method", request.Method).
			Str("route", route).
			Str("request_url", request.URL.String()).
			Dur("duration", time.Since(started)).
			Err(err).
			Msg("GitHub API request failed")
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result, err := DecodeResponse(response)
	if err != nil {
		log.Debug().
			Str("component", "githubapi").
			Str("method", request.Method).
			Str("route", route).
			Str("request_url", request.URL.String()).
			Int("status", response.StatusCode).
			Dur("duration", time.Since(started)).
			Err(err).
			Msg("Failed to decode GitHub API response")
		return nil, err
	}

	log.Debug().
		Str("component", "githubapi").
		Str("method", request.Method).
		Str("route", route).
		Str("request_url", request.URL.String()).
		Int("status", response.StatusCode).
		Dur("duration", time.Since(started)).
		Msg("Received GitHub API response")

	if response.StatusCode >= http.StatusBadRequest {
		return nil, &APIError{
			Status:     response.StatusCode,
			Message:    extractErrorMessage(result),
			Route:      route,
			RequestURL: request.URL.String(),
			Result:     result,
		}
	}
	return result, nil
}
