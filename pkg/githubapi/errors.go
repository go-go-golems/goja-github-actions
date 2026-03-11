package githubapi

import (
	"fmt"
	"strings"
)

type APIError struct {
	Status  int
	Message string
	Result  *RequestResult
}

func (e *APIError) Error() string {
	if e == nil {
		return "github api error"
	}
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("github api error: status %d", e.Status)
	}
	return fmt.Sprintf("github api error: status %d: %s", e.Status, e.Message)
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
