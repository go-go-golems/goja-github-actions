package ghacli

import (
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
)

type ValidationResult struct {
	Errors []string `json:"errors"`
}

func (r ValidationResult) Error() string {
	return strings.Join(r.Errors, "; ")
}

func (r ValidationResult) IsOK() bool {
	return len(r.Errors) == 0
}

func DecodeSettings(vals *values.Values) (*RunnerSettings, *GitHubActionsSettings, error) {
	githubSettings := &GitHubActionsSettings{}
	if err := vals.DecodeSectionInto(GitHubActionsSectionSlug, githubSettings); err != nil {
		return nil, nil, errors.Wrap(err, "decode GitHub settings")
	}

	runnerSettings := &RunnerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, runnerSettings); err != nil {
		return nil, nil, errors.Wrap(err, "decode runner settings")
	}

	return runnerSettings, githubSettings, nil
}

func ValidateRunSettings(runner *RunnerSettings, _ *GitHubActionsSettings) ValidationResult {
	errors_ := []string{}

	if strings.TrimSpace(runner.Script) == "" {
		errors_ = append(errors_, "--script is required")
	}

	return ValidationResult{Errors: errors_}
}
