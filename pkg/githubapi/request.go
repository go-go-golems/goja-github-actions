package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

func BuildRequest(
	ctx context.Context,
	baseURL string,
	route string,
	params map[string]interface{},
) (*http.Request, error) {
	method, pathTemplate, err := splitRoute(route)
	if err != nil {
		return nil, err
	}

	path, remainingParams, err := interpolatePath(pathTemplate, params)
	if err != nil {
		return nil, err
	}

	requestURL := strings.TrimRight(baseURL, "/") + path
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		requestURL = path
	}
	var body io.Reader

	if method == http.MethodGet || method == http.MethodDelete {
		query := url.Values{}
		for key, value := range remainingParams {
			query.Set(key, fmt.Sprint(value))
		}
		if encoded := query.Encode(); encoded != "" {
			requestURL += "?" + encoded
		}
	} else if len(remainingParams) > 0 {
		payload, err := json.Marshal(remainingParams)
		if err != nil {
			return nil, errors.Wrap(err, "marshal request body")
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func DecodeResponse(response *http.Response) (*RequestResult, error) {
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}

	result := &RequestResult{
		Status:  response.StatusCode,
		Headers: response.Header,
	}
	if len(bodyBytes) == 0 {
		return result, nil
	}

	var decoded interface{}
	if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
		result.Data = decoded
		return result, nil
	}

	result.Data = string(bodyBytes)
	return result, nil
}

func splitRoute(route string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(route), " ", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid route %q", route)
	}
	return strings.ToUpper(parts[0]), parts[1], nil
}

func interpolatePath(pathTemplate string, params map[string]interface{}) (string, map[string]interface{}, error) {
	remaining := map[string]interface{}{}
	for key, value := range params {
		remaining[key] = value
	}

	path := pathTemplate
	for {
		start := strings.Index(path, "{")
		if start < 0 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end < 0 {
			return "", nil, errors.Errorf("unclosed route placeholder in %q", pathTemplate)
		}
		end += start
		key := path[start+1 : end]
		value, ok := remaining[key]
		if !ok {
			return "", nil, errors.Errorf("missing route param %q", key)
		}
		path = path[:start] + url.PathEscape(fmt.Sprint(value)) + path[end+1:]
		delete(remaining, key)
	}

	if !strings.HasPrefix(path, "/") &&
		!strings.HasPrefix(path, "http://") &&
		!strings.HasPrefix(path, "https://") {
		path = "/" + path
	}
	return path, remaining, nil
}
