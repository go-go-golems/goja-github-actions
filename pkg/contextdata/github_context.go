package contextdata

import (
	"strings"

	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
)

type RepoContext struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

type GitHubContext struct {
	Actor      string                 `json:"actor,omitempty"`
	EventName  string                 `json:"event_name,omitempty"`
	EventPath  string                 `json:"event_path,omitempty"`
	Ref        string                 `json:"ref,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	SHA        string                 `json:"sha,omitempty"`
	Workspace  string                 `json:"workspace,omitempty"`
	Repo       RepoContext            `json:"repo"`
	Payload    map[string]interface{} `json:"payload"`
}

func BuildGitHubContext(settings *gharuntime.Settings) (*GitHubContext, error) {
	payload, err := LoadEventPayload(settings.EventPath)
	if err != nil {
		return nil, err
	}

	env := settings.ProcessEnv()
	repository := firstNonEmpty(
		env["GITHUB_REPOSITORY"],
		nestedString(payload, "repository", "full_name"),
	)
	owner, repo := splitRepository(repository)
	if owner == "" {
		owner = firstNonEmpty(
			nestedString(payload, "repository", "owner", "login"),
			nestedString(payload, "repository", "owner", "name"),
		)
	}
	if repo == "" {
		repo = nestedString(payload, "repository", "name")
	}

	return &GitHubContext{
		Actor:      env["GITHUB_ACTOR"],
		EventName:  env["GITHUB_EVENT_NAME"],
		EventPath:  settings.EventPath,
		Ref:        env["GITHUB_REF"],
		Repository: repository,
		SHA:        env["GITHUB_SHA"],
		Workspace:  settings.Workspace,
		Repo: RepoContext{
			Owner: owner,
			Repo:  repo,
		},
		Payload: payload,
	}, nil
}

func splitRepository(repository string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(repository), "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nestedString(payload map[string]interface{}, path ...string) string {
	current := interface{}(payload)
	for _, key := range path {
		next, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = next[key]
		if !ok {
			return ""
		}
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}
	return value
}
