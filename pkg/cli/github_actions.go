package ghacli

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	glazedschema "github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/pkg/errors"
)

const GitHubActionsSectionSlug = "github-actions"

type GitHubActionsSettings struct {
	Workspace   string `glazed:"workspace"`
	GitHubToken string `glazed:"github-token"`
}

type RunnerSettings struct {
	Script            string `glazed:"script"`
	EventPath         string `glazed:"event-path"`
	ActionPath        string `glazed:"action-path"`
	RunnerEnvFile     string `glazed:"runner-env-file"`
	RunnerOutputFile  string `glazed:"runner-output-file"`
	RunnerPathFile    string `glazed:"runner-path-file"`
	RunnerSummaryFile string `glazed:"runner-summary-file"`
	Cwd               string `glazed:"cwd"`
	Debug             bool   `glazed:"debug"`
	JSONResult        bool   `glazed:"json-result"`
}

func NewRunnerFields() []*fields.Definition {
	return []*fields.Definition{
		fields.New(
			"script",
			fields.TypeString,
			fields.WithHelp("Path to the JavaScript entrypoint"),
		),
		fields.New(
			"event-path",
			fields.TypeString,
			fields.WithHelp("Path to the GitHub event payload JSON"),
		),
		fields.New(
			"action-path",
			fields.TypeString,
			fields.WithHelp("Action directory when running inside GitHub Actions"),
		),
		fields.New(
			"runner-env-file",
			fields.TypeString,
			fields.WithHelp("Override path for the runner env file during local/test runs"),
		),
		fields.New(
			"runner-output-file",
			fields.TypeString,
			fields.WithHelp("Override path for the runner output file during local/test runs"),
		),
		fields.New(
			"runner-path-file",
			fields.TypeString,
			fields.WithHelp("Override path for the runner path file during local/test runs"),
		),
		fields.New(
			"runner-summary-file",
			fields.TypeString,
			fields.WithHelp("Override path for the runner summary file during local/test runs"),
		),
		fields.New(
			"cwd",
			fields.TypeString,
			fields.WithHelp("Working directory for script execution"),
		),
		fields.New(
			"debug",
			fields.TypeBool,
			fields.WithDefault(false),
			fields.WithHelp("Enable verbose runtime logging"),
		),
		fields.New(
			"json-result",
			fields.TypeBool,
			fields.WithDefault(false),
			fields.WithHelp("Emit final script result as JSON once runtime execution exists"),
		),
	}
}

func NewGitHubActionsSection() (glazedschema.Section, error) {
	section, err := glazedschema.NewSection(
		GitHubActionsSectionSlug,
		"Shared GitHub settings",
		glazedschema.WithFields(
			fields.New(
				"workspace",
				fields.TypeString,
				fields.WithHelp("Workspace directory shared by GitHub-oriented commands"),
			),
			fields.New(
				"github-token",
				fields.TypeString,
				fields.WithHelp("GitHub token shared by commands that call the GitHub API"),
			),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create GitHub Actions section")
	}
	return section, nil
}
