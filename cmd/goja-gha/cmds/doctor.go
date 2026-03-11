package cmds

import (
	"context"
	"strings"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	"github.com/pkg/errors"
)

type DoctorCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*DoctorCommand)(nil)

func NewDoctorCommand() (*DoctorCommand, error) {
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
		"doctor",
		cmds.WithShort("Inspect resolved GitHub Actions settings"),
		cmds.WithLong(`Inspect the resolved Glazed settings for goja-gha.

This command is intended for bootstrap and validation work. It reports the
resolved values without exposing the raw GitHub token.

Examples:
  goja-gha doctor --script ./examples/permissions-audit.js
  goja-gha doctor --script ./examples/permissions-audit.js --output json
  goja-gha doctor --script ./examples/permissions-audit.js --print-schema
`),
		cmds.WithFlags(ghacli.NewRunnerFields()...),
		cmds.WithSections(glazedSection, commandSettingsSection, githubActionsSection),
	)

	return &DoctorCommand{CommandDescription: description}, nil
}

func (c *DoctorCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	githubSettings := &ghacli.GitHubActionsSettings{}
	if err := vals.DecodeSectionInto(ghacli.GitHubActionsSectionSlug, githubSettings); err != nil {
		return errors.Wrap(err, "decode GitHub Actions settings")
	}

	runnerSettings := &ghacli.RunnerSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, runnerSettings); err != nil {
		return errors.Wrap(err, "decode runner settings")
	}

	row := types.NewRow(
		types.MRP("script", runnerSettings.Script),
		types.MRP("script_exists", fileExists(runnerSettings.Script)),
		types.MRP("event_path", runnerSettings.EventPath),
		types.MRP("workspace", githubSettings.Workspace),
		types.MRP("action_path", runnerSettings.ActionPath),
		types.MRP("github_token_present", strings.TrimSpace(githubSettings.GitHubToken) != ""),
		types.MRP("runner_env_file", runnerSettings.RunnerEnvFile),
		types.MRP("runner_output_file", runnerSettings.RunnerOutputFile),
		types.MRP("runner_path_file", runnerSettings.RunnerPathFile),
		types.MRP("runner_summary_file", runnerSettings.RunnerSummaryFile),
		types.MRP("cwd", runnerSettings.Cwd),
		types.MRP("debug", runnerSettings.Debug),
		types.MRP("json_result", runnerSettings.JSONResult),
	)

	return gp.AddRow(ctx, row)
}
