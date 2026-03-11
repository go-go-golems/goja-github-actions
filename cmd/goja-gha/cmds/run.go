package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	"github.com/pkg/errors"
)

type RunCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*RunCommand)(nil)

func NewRunCommand() (*RunCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, errors.Wrap(err, "create glazed section")
	}

	commandSettingsSection, err := glazedcli.NewCommandSettingsSection()
	if err != nil {
		return nil, errors.Wrap(err, "create command settings section")
	}

	githubActionsSection, err := ghacli.NewGitHubActionsSection()
	if err != nil {
		return nil, errors.Wrap(err, "create GitHub Actions section")
	}

	description := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Validate run settings and prepare for script execution"),
		cmds.WithLong(`Validate the Glazed/GitHub Actions settings required to run a JavaScript entrypoint.

This bootstrap version does not execute the Goja runtime yet. It resolves and
prints the settings, then exits with a clear not-implemented error so we can
build the runtime task by task without hidden configuration behavior.

Examples:
  goja-gha run --script ./examples/permissions-audit.js
  goja-gha run --script ./examples/permissions-audit.js --event-path ./testdata/events/workflow_dispatch.json
  goja-gha run --script ./examples/permissions-audit.js --print-parsed-fields
`),
		cmds.WithFlags(ghacli.NewRunnerFields()...),
		cmds.WithSections(glazedSection, commandSettingsSection, githubActionsSection),
	)

	return &RunCommand{CommandDescription: description}, nil
}

func (c *RunCommand) Run(_ context.Context, vals *values.Values) error {
	githubSettings := &ghacli.GitHubActionsSettings{}
	if err := vals.DecodeSectionInto(ghacli.GitHubActionsSectionSlug, githubSettings); err != nil {
		return errors.Wrap(err, "decode GitHub Actions settings")
	}

	runnerSettings := &ghacli.RunnerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, runnerSettings); err != nil {
		return errors.Wrap(err, "decode runner settings")
	}

	if strings.TrimSpace(runnerSettings.Script) == "" {
		return errors.New("--script is required")
	}

	payload := map[string]any{
		"action_path":          runnerSettings.ActionPath,
		"cwd":                  runnerSettings.Cwd,
		"debug":                runnerSettings.Debug,
		"runner_env_file":      runnerSettings.RunnerEnvFile,
		"event_path":           runnerSettings.EventPath,
		"github_token_present": strings.TrimSpace(githubSettings.GitHubToken) != "",
		"json_result":          runnerSettings.JSONResult,
		"runner_output_file":   runnerSettings.RunnerOutputFile,
		"runner_path_file":     runnerSettings.RunnerPathFile,
		"runner_summary_file":  runnerSettings.RunnerSummaryFile,
		"script":               runnerSettings.Script,
		"script_exists":        fileExists(runnerSettings.Script),
		"workspace":            githubSettings.Workspace,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return errors.Wrap(err, "encode bootstrap payload")
	}

	return fmt.Errorf("goja-gha run bootstrap complete: runtime execution is not implemented yet")
}
