package cmds

import (
	"context"
	"encoding/json"
	"os"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	coremodule "github.com/go-go-golems/goja-github-actions/pkg/modules/core"
	execmodule "github.com/go-go-golems/goja-github-actions/pkg/modules/exec"
	githubmodule "github.com/go-go-golems/goja-github-actions/pkg/modules/github"
	iomodule "github.com/go-go-golems/goja-github-actions/pkg/modules/io"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
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

This command resolves the Glazed/GitHub settings, initializes a Goja runtime,
and executes the entrypoint as a CommonJS module.

Examples:
  goja-gha run --script ./examples/permissions-audit.js
  goja-gha run --script ./examples/permissions-audit.js --event-path ./testdata/events/workflow_dispatch.json
  goja-gha run --script ./examples/trivial.js --json-result
`),
		cmds.WithFlags(ghacli.NewRunnerFields()...),
		cmds.WithSections(glazedSection, commandSettingsSection, githubActionsSection),
	)

	return &RunCommand{CommandDescription: description}, nil
}

func (c *RunCommand) Run(_ context.Context, vals *values.Values) error {
	runnerSettings, githubSettings, err := ghacli.DecodeSettings(vals)
	if err != nil {
		return err
	}

	if validation := ghacli.ValidateRunSettings(runnerSettings, githubSettings); !validation.IsOK() {
		return validation
	}

	settings := gharuntime.NewSettings(runnerSettings, githubSettings, environmentSnapshot())
	result, err := gharuntime.RunScriptWithModules(
		context.Background(),
		settings,
		coremodule.Spec(coremodule.NewDependencies(settings)),
		iomodule.Spec(&iomodule.Dependencies{Settings: settings}),
		execmodule.Spec(&execmodule.Dependencies{Settings: settings}),
		githubmodule.Spec(&githubmodule.Dependencies{Settings: settings}),
	)
	if err != nil {
		return err
	}

	if settings.State != nil && settings.State.ExitCode != 0 {
		if settings.State.FailureMessage != "" {
			return errors.New(settings.State.FailureMessage)
		}
		return errors.Errorf("script requested exit code %d", settings.State.ExitCode)
	}

	if runnerSettings.JSONResult {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result.Export()); err != nil {
			return errors.Wrap(err, "encode script result")
		}
	}

	return nil
}
