package githubapi

import (
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	Status     int
	Message    string
	Route      string
	RequestURL string
	Result     *RequestResult
}

func (e *APIError) Error() string {
	if e == nil {
		return "github api error"
	}

	statusText := strings.TrimSpace(http.StatusText(e.Status))
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = statusText
	}
	if message == "" {
		message = fmt.Sprintf("status %d", e.Status)
	}

	parts := []string{fmt.Sprintf("github api error: status %d", e.Status)}
	if statusText != "" {
		parts = append(parts, statusText)
	}
	if message != "" && !strings.EqualFold(message, statusText) {
		parts = append(parts, message)
	}

	details := strings.Join(parts, ": ")
	if strings.TrimSpace(e.Route) != "" {
		details += fmt.Sprintf(" (route %q)", e.Route)
	}

	if hint := strings.TrimSpace(e.Hint()); hint != "" {
		details += ": " + hint
	}

	return details
}

func (e *APIError) Hint() string {
	if e == nil {
		return ""
	}

	route := strings.TrimSpace(e.Route)
	message := strings.TrimSpace(strings.ToLower(e.Message))

	switch e.Status {
	case http.StatusUnauthorized:
		return "check that the token is present, valid, unexpired, and the one the runtime actually used"
	case http.StatusForbidden:
		if strings.Contains(message, "resource not accessible by personal access token") {
			if strings.Contains(route, "/actions/permissions") {
				return "the token is valid but under-scoped for this endpoint; for repository Actions permissions endpoints, a fine-grained PAT usually needs repository permissions Actions: Read and Administration: Read"
			}
			return "the token is valid but lacks permission for this endpoint"
		}
	case http.StatusConflict:
		if strings.Contains(route, "/actions/permissions/selected-actions") {
			return "this endpoint only applies when the repository allowed_actions policy is set to \"selected\"; fetch /actions/permissions first and call selected-actions only when allowed_actions == \"selected\""
		}
	}

	return ""
}

func extractErrorMessage(result *RequestResult) string {
	if result == nil {
		return ""
	}

	payload, ok := result.Data.(map[string]interface{})
	if !ok {
		return strings.TrimSpace(fmt.Sprint(result.Data))
	}

	for _, key := range []string{"message", "error", "title"} {
		if value, ok := payload[key]; ok {
			message := strings.TrimSpace(fmt.Sprint(value))
			if message != "" {
				return message
			}
		}
	}

	return ""
}
