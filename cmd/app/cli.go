// Package app provides CLI wiring and process setup for gormdb2struct.
package app

import (
	"github.com/alecthomas/kong"
	"github.com/dan-sherwin/gormdb2struct/cmd/app/consts"
)

type (
	// LoggingConfig defines CLI-controlled logging options.
	LoggingConfig struct {
		Level string `name:"logging.level" enum:"debug,info,warn,error" default:"info" help:"Log level." group:"logging"`
	}

	// GenerateConfigSampleCmd writes a starter TOML configuration file.
	GenerateConfigSampleCmd struct {
		Out string `name:"out" short:"o" default:"gormdb2struct-sample.toml" help:"Path to write the sample TOML config." type:"path"`
	}

	// CLIConfig defines the top-level command-line contract.
	CLIConfig struct {
		Logging    LoggingConfig `embed:""`
		ConfigPath string        `arg:"" optional:"" name:"config" help:"Path to the TOML configuration file." type:"path"`
	}
)

func buildParser(cli *CLIConfig) *kong.Kong {
	parser := kong.Must(cli,
		kong.Name(consts.APPNAME),
		kong.Description("Generate strongly typed GORM models and query helpers from an existing database schema."),
		kong.ShortUsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
	return parser
}

func buildGenerateConfigSampleParser(cmd *GenerateConfigSampleCmd) *kong.Kong {
	return kong.Must(cmd,
		kong.Name(consts.APPNAME+" generate-config-sample"),
		kong.Description("Write a sample TOML configuration file."),
		kong.ShortUsageOnError(),
	)
}
