package ghacli

import (
	"os"
	"strings"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewParserConfig() glazedcli.CobraParserConfig {
	return glazedcli.CobraParserConfig{
		AppName:           AppName,
		MiddlewaresFunc:   NewMiddlewaresFunc(os.LookupEnv),
		ShortHelpSections: []string{},
	}
}

func NewMiddlewaresFunc(
	lookupEnv func(string) (string, bool),
) glazedcli.CobraMiddlewaresFunc {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	return func(
		parsedCommandSections *values.Values,
		cmd *cobra.Command,
		args []string,
	) ([]sources.Middleware, error) {
		configFiles, err := ResolveConfigFiles(parsedCommandSections)
		if err != nil {
			return nil, err
		}

		return []sources.Middleware{
			sources.FromCobra(cmd, fields.WithSource("cobra")),
			sources.FromArgs(args, fields.WithSource("arguments")),
			sources.FromFiles(configFiles,
				sources.WithParseOptions(fields.WithSource(SourceConfig)),
			),
			sources.FromMap(
				RunnerEnvValuesFromLookup(lookupEnv),
				fields.WithSource(SourceRunnerEnv),
			),
			sources.FromMapAsDefault(
				DefaultFieldValues(),
				fields.WithSource(SourceDefaults),
			),
			sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
		}, nil
	}
}

func ResolveConfigFiles(parsedCommandSections *values.Values) ([]string, error) {
	commandSettings := &glazedcli.CommandSettings{}
	if err := parsedCommandSections.DecodeSectionInto(glazedcli.CommandSettingsSlug, commandSettings); err != nil {
		return nil, errors.Wrap(err, "decode command settings")
	}

	configPath, err := glazedconfig.ResolveAppConfigPath(AppName, commandSettings.ConfigFile)
	if err != nil {
		return nil, errors.Wrap(err, "resolve config path")
	}
	if configPath == "" {
		return []string{}, nil
	}

	return []string{configPath}, nil
}

func RunnerEnvValuesFromLookup(
	lookupEnv func(string) (string, bool),
) map[string]map[string]interface{} {
	valuesBySection := map[string]map[string]interface{}{}

	for _, mapping := range RunnerEnvMappings() {
		for _, envKey := range mapping.EnvKeys {
			value, ok := lookupEnv(envKey)
			if !ok || strings.TrimSpace(value) == "" {
				continue
			}

			if _, ok := valuesBySection[mapping.SectionSlug]; !ok {
				valuesBySection[mapping.SectionSlug] = map[string]interface{}{}
			}
			valuesBySection[mapping.SectionSlug][mapping.FieldName] = normalizeEnvValue(mapping.FieldName, value)
			break
		}
	}

	return valuesBySection
}

func normalizeEnvValue(fieldName string, value string) interface{} {
	switch fieldName {
	case "debug":
		normalized := strings.TrimSpace(strings.ToLower(value))
		return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
	default:
		return value
	}
}
