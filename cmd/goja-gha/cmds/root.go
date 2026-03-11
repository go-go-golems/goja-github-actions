package cmds

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	helpdoc "github.com/go-go-golems/goja-github-actions/pkg/helpdoc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const appName = "goja-gha"

func newRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:           appName,
		Short:         "Run GitHub Actions-oriented JavaScript on top of Goja",
		SilenceErrors: true,
		Long: `goja-gha is a Go/Goja CLI for GitHub Actions-style JavaScript automation.

It includes a native @actions/*-style module surface, runner-file support,
and embedded long-form help pages for users and developers.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
}

func buildCommand(command cmds.Command) (*cobra.Command, error) {
	parserConfig := ghacli.NewParserConfig()
	parserConfig.ShortHelpSections = []string{schema.DefaultSlug, ghacli.GitHubActionsSectionSlug}

	cobraCommand, err := cli.BuildCobraCommandFromCommand(
		command,
		cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "build cobra command %q", command.Description().Name)
	}
	cobraCommand.SilenceErrors = true
	cobraCommand.SilenceUsage = true
	return cobraCommand, nil
}

func buildBareCommand(command cmds.Command) (*cobra.Command, error) {
	bareCommand, ok := command.(cmds.BareCommand)
	if !ok {
		return nil, errors.Errorf("command %T does not implement cmds.BareCommand", command)
	}

	description := command.Description()
	parserConfig := ghacli.NewParserConfig()
	parserConfig.ShortHelpSections = []string{schema.DefaultSlug, ghacli.GitHubActionsSectionSlug}

	cobraCommand := cli.NewCobraCommandFromCommandDescription(description)
	cobraCommand.SilenceErrors = true
	cobraCommand.SilenceUsage = true

	parser, err := cli.NewCobraParserFromSections(description.Schema, &parserConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "build cobra parser %q", description.Name)
	}

	if err := parser.AddToCobraCommand(cobraCommand); err != nil {
		return nil, errors.Wrapf(err, "add parser to cobra command %q", description.Name)
	}

	cobraCommand.RunE = func(cmd *cobra.Command, args []string) error {
		parsedValues, err := parser.Parse(cmd, args)
		if err != nil {
			return err
		}

		commandSettings := &cli.CommandSettings{}
		if commandSettingsValues, ok := parsedValues.Get(cli.CommandSettingsSlug); ok {
			if err := commandSettingsValues.DecodeInto(commandSettings); err != nil {
				return err
			}

			if commandSettings.PrintParsedFields {
				printParsedFields(parsedValues)
				return nil
			}
			if commandSettings.PrintYAML {
				return command.ToYAML(os.Stdout)
			}
			if commandSettings.PrintSchema {
				schema_, err := command.Description().ToJsonSchema()
				if err != nil {
					return err
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(schema_)
			}
		}

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		return bareCommand.Run(ctx, parsedValues)
	}

	return cobraCommand, nil
}

func printParsedFields(parsedValues *values.Values) {
	sectionsMap := map[string]map[string]interface{}{}
	parsedValues.ForEach(func(sectionName string, sectionValues *values.SectionValues) {
		fieldValues := map[string]interface{}{}
		sectionValues.Fields.ForEach(func(name string, fieldValue *fields.FieldValue) {
			fieldMap := map[string]interface{}{
				"value": fieldValue.Value,
			}
			logs := make([]map[string]interface{}, 0, len(fieldValue.Log))
			for _, entry := range fieldValue.Log {
				logEntry := map[string]interface{}{
					"source": entry.Source,
					"value":  entry.Value,
				}
				if len(entry.Metadata) > 0 {
					logEntry["metadata"] = entry.Metadata
				}
				logs = append(logs, logEntry)
			}
			if len(logs) > 0 {
				fieldMap["log"] = logs
			}
			fieldValues[name] = fieldMap
		})
		sectionsMap[sectionName] = fieldValues
	})

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	_ = encoder.Encode(sectionsMap)
}

func Execute() error {
	root := newRootCommand()

	if err := logging.AddLoggingSectionToRootCommand(root, appName); err != nil {
		return errors.Wrap(err, "add logging section to root command")
	}

	helpSystem := help.NewHelpSystem()
	if err := helpdoc.AddDocToHelpSystem(helpSystem); err != nil {
		return errors.Wrap(err, "load embedded help docs")
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	runCommand, err := NewRunCommand()
	if err != nil {
		return errors.Wrap(err, "create run command")
	}
	runCobraCommand, err := buildBareCommand(runCommand)
	if err != nil {
		return err
	}

	doctorCommand, err := NewDoctorCommand()
	if err != nil {
		return errors.Wrap(err, "create doctor command")
	}
	doctorCobraCommand, err := buildCommand(doctorCommand)
	if err != nil {
		return err
	}

	root.AddCommand(runCobraCommand)
	root.AddCommand(doctorCobraCommand)

	return root.Execute()
}
