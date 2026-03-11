package ghacli

import (
	"os"
	"path/filepath"
	"testing"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

func TestRunnerEnvValuesFromLookup(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"GITHUB_EVENT_PATH":  "/tmp/event.json",
		"GITHUB_OUTPUT":      "/tmp/output.txt",
		"GITHUB_WORKSPACE":   "/workspace",
		"GITHUB_TOKEN":       "token-from-github",
		"GH_TOKEN":           "token-from-gh",
		"RUNNER_DEBUG":       "1",
		"GITHUB_ACTION_PATH": "/action",
	}

	actual := RunnerEnvValuesFromLookup(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})

	defaultValues := actual[schema.DefaultSlug]
	githubValues := actual[GitHubActionsSectionSlug]

	if got, want := defaultValues["event-path"], "/tmp/event.json"; got != want {
		t.Fatalf("event-path = %v, want %v", got, want)
	}
	if got, want := defaultValues["runner-output-file"], "/tmp/output.txt"; got != want {
		t.Fatalf("runner-output-file = %v, want %v", got, want)
	}
	if got, want := defaultValues["action-path"], "/action"; got != want {
		t.Fatalf("action-path = %v, want %v", got, want)
	}
	if got, want := defaultValues["debug"], true; got != want {
		t.Fatalf("debug = %v, want %v", got, want)
	}
	if got, want := githubValues["workspace"], "/workspace"; got != want {
		t.Fatalf("workspace = %v, want %v", got, want)
	}
	if got, want := githubValues["github-token"], "token-from-github"; got != want {
		t.Fatalf("github-token = %v, want %v", got, want)
	}
}

func TestResolveConfigFilesUsesExplicitCommandSetting(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("default:\n  cwd: /tmp\n"), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	commandSettingsSection, err := glazedcli.NewCommandSettingsSection()
	if err != nil {
		t.Fatalf("new command settings section: %v", err)
	}

	fieldValues, err := commandSettingsSection.GetDefinitions().GatherFieldsFromMap(
		map[string]interface{}{"config-file": configPath},
		true,
		fields.WithSource("test"),
	)
	if err != nil {
		t.Fatalf("gather command settings: %v", err)
	}

	sectionValues, err := values.NewSectionValues(commandSettingsSection, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("new section values: %v", err)
	}

	parsed := values.New(values.WithSectionValues(glazedcli.CommandSettingsSlug, sectionValues))
	files, err := ResolveConfigFiles(parsed)
	if err != nil {
		t.Fatalf("resolve config files: %v", err)
	}

	if len(files) != 1 || files[0] != configPath {
		t.Fatalf("files = %v, want [%s]", files, configPath)
	}
}

func TestSettingsPrecedence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configYAML := []byte(`
default:
  event-path: /config/event.json
  runner-output-file: /config/output.txt
github-actions:
  workspace: /config/workspace
`)
	if err := os.WriteFile(configPath, configYAML, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	schema_ := testSchema(t)
	parsed := values.New()

	err := sources.Execute(
		schema_,
		parsed,
		sources.FromMap(
			map[string]map[string]interface{}{
				schema.DefaultSlug: {
					"script":      "from-flags.js",
					"event-path":  "/flag/event.json",
					"json-result": true,
				},
				GitHubActionsSectionSlug: {
					"workspace": "/flag/workspace",
				},
			},
			fields.WithSource("cobra"),
		),
		sources.FromFiles([]string{configPath},
			sources.WithParseOptions(fields.WithSource(SourceConfig)),
		),
		sources.FromMap(
			RunnerEnvValuesFromLookup(func(key string) (string, bool) {
				env := map[string]string{
					"GITHUB_EVENT_PATH":  "/env/event.json",
					"GITHUB_OUTPUT":      "/env/output.txt",
					"GITHUB_ACTION_PATH": "/env/action",
					"GITHUB_WORKSPACE":   "/env/workspace",
					"GITHUB_TOKEN":       "env-token",
					"RUNNER_DEBUG":       "true",
				}
				value, ok := env[key]
				return value, ok
			}),
			fields.WithSource(SourceRunnerEnv),
		),
		sources.FromMapAsDefault(
			DefaultFieldValues(),
			fields.WithSource(SourceDefaults),
		),
		sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
	)
	if err != nil {
		t.Fatalf("execute sources: %v", err)
	}

	runnerSettings, githubSettings, err := DecodeSettings(parsed)
	if err != nil {
		t.Fatalf("decode settings: %v", err)
	}

	if got, want := runnerSettings.Script, "from-flags.js"; got != want {
		t.Fatalf("script = %q, want %q", got, want)
	}
	if got, want := runnerSettings.EventPath, "/flag/event.json"; got != want {
		t.Fatalf("event-path = %q, want %q", got, want)
	}
	if got, want := runnerSettings.RunnerOutputFile, "/config/output.txt"; got != want {
		t.Fatalf("runner-output-file = %q, want %q", got, want)
	}
	if got, want := runnerSettings.ActionPath, "/env/action"; got != want {
		t.Fatalf("action-path = %q, want %q", got, want)
	}
	if got, want := runnerSettings.Cwd, "."; got != want {
		t.Fatalf("cwd = %q, want %q", got, want)
	}
	if got, want := runnerSettings.Debug, true; got != want {
		t.Fatalf("debug = %v, want %v", got, want)
	}
	if got, want := runnerSettings.JSONResult, true; got != want {
		t.Fatalf("json-result = %v, want %v", got, want)
	}
	if got, want := githubSettings.Workspace, "/flag/workspace"; got != want {
		t.Fatalf("workspace = %q, want %q", got, want)
	}
	if got, want := githubSettings.GitHubToken, "env-token"; got != want {
		t.Fatalf("github-token = %q, want %q", got, want)
	}
}

func TestValidateRunSettingsRequiresScript(t *testing.T) {
	t.Parallel()

	validation := ValidateRunSettings(&RunnerSettings{}, &GitHubActionsSettings{})
	if validation.IsOK() {
		t.Fatal("expected validation to fail without script")
	}
	if got, want := validation.Error(), "--script is required"; got != want {
		t.Fatalf("validation error = %q, want %q", got, want)
	}
}

func testSchema(t *testing.T) *schema.Schema {
	t.Helper()

	defaultSection, err := schema.NewSection(
		schema.DefaultSlug,
		"Runner settings",
		schema.WithFields(NewRunnerFields()...),
	)
	if err != nil {
		t.Fatalf("new default section: %v", err)
	}

	githubSection, err := NewGitHubActionsSection()
	if err != nil {
		t.Fatalf("new github section: %v", err)
	}

	return schema.NewSchema(schema.WithSections(defaultSection, githubSection))
}
