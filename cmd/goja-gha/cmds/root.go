package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	ghacli "github.com/go-go-golems/goja-github-actions/pkg/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const appName = "goja-gha"

func newRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:   appName,
		Short: "Run GitHub Actions-oriented JavaScript on top of Goja",
		Long: `goja-gha is a Go/Goja CLI for GitHub Actions-style JavaScript automation.

The current bootstrap build wires the Glazed command surface and shared schema,
so commands can evolve task by task without ad hoc flag parsing.`,
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
	return cobraCommand, nil
}

func Execute() error {
	root := newRootCommand()

	if err := logging.AddLoggingSectionToRootCommand(root, appName); err != nil {
		return errors.Wrap(err, "add logging section to root command")
	}

	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, root)

	runCommand, err := NewRunCommand()
	if err != nil {
		return errors.Wrap(err, "create run command")
	}
	runCobraCommand, err := buildCommand(runCommand)
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
