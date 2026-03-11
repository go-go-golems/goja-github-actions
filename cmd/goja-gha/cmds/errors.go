package cmds

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/goja-github-actions/pkg/githubapi"
	"github.com/rs/zerolog/log"
)

func FormatCLIError(err error) string {
	return formatCLIError(err)
}

func formatCLIError(err error) string {
	if err == nil {
		return ""
	}

	log.Debug().
		Str("component", "cli").
		Err(err).
		Msg("Formatting CLI error")

	var apiErr *githubapi.APIError
	if stderrors.As(err, &apiErr) {
		return formatGitHubAPIError(apiErr)
	}

	var exception *goja.Exception
	if stderrors.As(err, &exception) {
		return formatGojaException(exception)
	}

	if formatted, ok := formatWrappedJavaScriptError(err); ok {
		return formatted
	}

	return err.Error()
}

func formatGitHubAPIError(err *githubapi.APIError) string {
	if err == nil {
		return "GitHub API request failed"
	}

	lines := []string{"GitHub API request failed", ""}

	if route := strings.TrimSpace(err.Route); route != "" {
		lines = append(lines, fmt.Sprintf("Route: %s", route))
	}

	statusText := strings.TrimSpace(http.StatusText(err.Status))
	statusLine := fmt.Sprintf("Status: %d", err.Status)
	if statusText != "" {
		statusLine += " " + statusText
	}
	lines = append(lines, statusLine)

	message := strings.TrimSpace(err.Message)
	if message != "" && !strings.EqualFold(message, statusText) {
		lines = append(lines, fmt.Sprintf("Message: %s", message))
	}

	if hint := strings.TrimSpace(err.Hint()); hint != "" {
		lines = append(lines, "", fmt.Sprintf("Hint: %s", hint))
	}

	return strings.Join(lines, "\n")
}

func formatGojaException(err *goja.Exception) string {
	if err == nil {
		return "JavaScript execution failed"
	}

	lines := []string{"JavaScript execution failed", ""}

	message := strings.TrimSpace(cleanGojaMessage(err.Value().String()))
	if message == "" {
		message = strings.TrimSpace(cleanGojaMessage(err.Error()))
	}
	if message != "" {
		lines = append(lines, fmt.Sprintf("Message: %s", message))
	}

	stack := err.Stack()
	if len(stack) > 0 {
		frame := stack[0]
		source := strings.TrimSpace(frame.SrcName())
		position := strings.TrimSpace(frame.Position().String())
		location := strings.TrimSpace(strings.TrimSpace(source) + ":" + strings.TrimLeft(position, ":"))
		location = strings.Trim(location, ":")
		if location != "" {
			lines = append(lines, fmt.Sprintf("Location: %s", location))
		}
	}

	return strings.Join(lines, "\n")
}

func cleanGojaMessage(message string) string {
	message = strings.TrimSpace(message)
	prefixes := []string{
		"GoError: ",
		"execute exported function: ",
		"execute exported main function: ",
		"execute exported default function: ",
	}
	for {
		updated := false
		for _, prefix := range prefixes {
			trimmed := strings.TrimPrefix(message, prefix)
			if trimmed != message {
				message = strings.TrimSpace(trimmed)
				updated = true
			}
		}
		if !updated {
			break
		}
	}

	if index := strings.Index(message, " at "); index != -1 {
		tail := message[index+4:]
		if strings.HasSuffix(tail, "(native)") {
			message = strings.TrimSpace(message[:index])
		}
	}

	return strings.TrimSpace(message)
}

func formatWrappedJavaScriptError(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	raw := strings.TrimSpace(err.Error())
	cleaned := cleanGojaMessage(raw)
	if cleaned == "" || cleaned == raw {
		return "", false
	}

	return strings.Join([]string{
		"JavaScript execution failed",
		"",
		fmt.Sprintf("Message: %s", cleaned),
	}, "\n"), true
}
