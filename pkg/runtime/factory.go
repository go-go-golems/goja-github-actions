package runtime

import (
	"context"
	"path/filepath"

	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	"github.com/pkg/errors"
)

type Settings struct {
	ScriptPath         string
	WorkingDirectory   string
	Workspace          string
	EventPath          string
	ActionPath         string
	RunnerEnvFile      string
	RunnerOutputFile   string
	RunnerPathFile     string
	RunnerSummaryFile  string
	GitHubToken        string
	Debug              bool
	JSONResult         bool
	AmbientEnvironment map[string]string
	State              *State
}

type State struct {
	Environment    map[string]string
	ExitCode       int
	FailureMessage string
	HumanOutputRendered bool
}

func NewSettings(
	runnerSettings *ghacli.RunnerSettings,
	githubSettings *ghacli.GitHubActionsSettings,
	ambientEnv map[string]string,
) *Settings {
	settings := &Settings{
		ScriptPath:         runnerSettings.Script,
		WorkingDirectory:   runnerSettings.Cwd,
		Workspace:          githubSettings.Workspace,
		EventPath:          runnerSettings.EventPath,
		ActionPath:         runnerSettings.ActionPath,
		RunnerEnvFile:      runnerSettings.RunnerEnvFile,
		RunnerOutputFile:   runnerSettings.RunnerOutputFile,
		RunnerPathFile:     runnerSettings.RunnerPathFile,
		RunnerSummaryFile:  runnerSettings.RunnerSummaryFile,
		GitHubToken:        githubSettings.GitHubToken,
		Debug:              runnerSettings.Debug,
		JSONResult:         runnerSettings.JSONResult,
		AmbientEnvironment: ambientEnv,
	}
	settings.State = &State{
		Environment: settings.ProcessEnv(),
	}
	return settings
}

func BuildFactory(settings *Settings, modules ...ggjengine.ModuleSpec) (*ggjengine.Factory, error) {
	if settings == nil {
		return nil, errors.New("runtime settings are nil")
	}

	factory, err := ggjengine.NewBuilder(
		ggjengine.WithModuleRootsFromScript(
			settings.ScriptPath,
			ggjengine.DefaultModuleRootsOptions(),
		),
	).WithModules(
		modules...,
	).WithRuntimeInitializers(
		NewBindingsInitializer(settings),
		NewProcessInitializer(settings),
	).Build()
	if err != nil {
		return nil, errors.Wrap(err, "build goja runtime factory")
	}

	return factory, nil
}

func CreateRuntime(ctx context.Context, settings *Settings, modules ...ggjengine.ModuleSpec) (*ggjengine.Runtime, error) {
	factory, err := BuildFactory(settings, modules...)
	if err != nil {
		return nil, err
	}

	rt, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create goja runtime")
	}

	return rt, nil
}

func ModuleEntrypoint(settings *Settings) string {
	return filepath.Base(settings.ScriptPath)
}
