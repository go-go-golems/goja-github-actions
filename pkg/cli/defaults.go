package ghacli

import "github.com/go-go-golems/glazed/pkg/cmds/schema"

const AppName = "goja-gha"

const (
	SourceRunnerEnv = "runner-env"
	SourceConfig    = "config"
	SourceDefaults  = "app-defaults"
)

type EnvFieldMapping struct {
	SectionSlug string
	FieldName   string
	EnvKeys     []string
}

func DefaultFieldValues() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		schema.DefaultSlug: {
			"cwd": ".",
		},
	}
}

func RunnerEnvMappings() []EnvFieldMapping {
	return []EnvFieldMapping{
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "event-path",
			EnvKeys:     []string{"GITHUB_EVENT_PATH"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "action-path",
			EnvKeys:     []string{"GITHUB_ACTION_PATH"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "runner-env-file",
			EnvKeys:     []string{"GITHUB_ENV"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "runner-output-file",
			EnvKeys:     []string{"GITHUB_OUTPUT"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "runner-path-file",
			EnvKeys:     []string{"GITHUB_PATH"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "runner-summary-file",
			EnvKeys:     []string{"GITHUB_STEP_SUMMARY"},
		},
		{
			SectionSlug: schema.DefaultSlug,
			FieldName:   "debug",
			EnvKeys:     []string{"RUNNER_DEBUG"},
		},
		{
			SectionSlug: GitHubActionsSectionSlug,
			FieldName:   "workspace",
			EnvKeys:     []string{"GITHUB_WORKSPACE"},
		},
		{
			SectionSlug: GitHubActionsSectionSlug,
			FieldName:   "github-token",
			EnvKeys:     []string{"GITHUB_TOKEN", "GH_TOKEN"},
		},
	}
}
