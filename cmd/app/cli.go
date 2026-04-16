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

	// InspectCmd analyzes a schema and recommends missing type mappings.
	InspectCmd struct {
		Logging    LoggingConfig `embed:""`
		ConfigPath string        `arg:"" name:"config" help:"Path to the TOML configuration file." type:"path"`
		Format     string        `name:"format" enum:"text,toml" default:"text" help:"Output format for the inspection report."`
	}

	// InspectPostgreSQLCmd analyzes a PostgreSQL schema directly from CLI connection flags.
	InspectPostgreSQLCmd struct {
		Logging               LoggingConfig `embed:""`
		Host                  string        `name:"host" required:"" help:"PostgreSQL host."`
		Port                  int           `name:"port" default:"5432" help:"PostgreSQL port."`
		Database              string        `name:"database" required:"" help:"PostgreSQL database name."`
		User                  string        `name:"user" required:"" help:"PostgreSQL user."`
		Password              string        `name:"password" help:"PostgreSQL password."`
		PasswordEnv           string        `name:"password-env" help:"Environment variable that contains the PostgreSQL password."`
		PasswordStdin         bool          `name:"password-stdin" help:"Read the PostgreSQL password from stdin."`
		PasswordPrompt        bool          `name:"password-prompt" help:"Prompt securely for the PostgreSQL password."`
		SSLMode               bool          `name:"sslmode" help:"Require SSL for the PostgreSQL connection."`
		Objects               []string      `name:"object" help:"Database object to inspect. Repeat to limit the inspection scope."`
		ImportPackagePaths    []string      `name:"import-package" help:"Import package path to inspect for exported Go types that should be preferred in TypeMap recommendations. Repeat as needed."`
		Out                   string        `name:"out" short:"o" default:"" help:"Starter config destination: omit to suppress it, use 'stdout' to print it, or provide a file path to write it."`
		OutPath               string        `name:"out-path" default:"./generated" help:"OutPath to use in TOML output."`
		OutPackagePath        string        `name:"out-package-path" default:"" help:"OutPackagePath to use in TOML output."`
		GeneratedTypesPackage string        `name:"generated-types-package" default:"dbtypes" help:"PackageName to use for generated PostgreSQL wrapper types in TOML output."`
		GeneratedTypesPath    string        `name:"generated-types-path" default:"models/dbtypes" help:"RelativePath to use for generated PostgreSQL wrapper types in TOML output."`
	}

	// ConvertConfigCmd loads any supported config and emits the canonical
	// ConfigVersion=1 TOML format. This command is intentionally hidden from
	// normal help output and exists as a migration utility.
	ConvertConfigCmd struct {
		ConfigPath string `arg:"" name:"config" help:"Path to the config file to convert." type:"path"`
		Out        string `name:"out" short:"o" default:"" help:"Write the converted config to this path instead of stdout." type:"path"`
		InPlace    bool   `name:"in-place" help:"Overwrite the input config file with the converted format."`
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
		kong.Description("Generate GORM models, query helpers, and optional PostgreSQL wrapper types from an existing schema."),
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
		kong.Description("Write a commented starter TOML configuration file."),
		kong.ShortUsageOnError(),
	)
}

func buildInspectParser(cmd *InspectCmd) *kong.Kong {
	return kong.Must(cmd,
		kong.Name(consts.APPNAME+" inspect"),
		kong.Description("Inspect the PostgreSQL objects referenced by a config and recommend missing type mappings."),
		kong.ShortUsageOnError(),
	)
}

func buildInspectPostgreSQLParser(cmd *InspectPostgreSQLCmd) *kong.Kong {
	return kong.Must(cmd,
		kong.Name(consts.APPNAME+" inspect-postgresql"),
		kong.Description("Inspect PostgreSQL directly from connection flags, recommend mappings, and optionally emit a starter config."),
		kong.ShortUsageOnError(),
	)
}

func buildConvertConfigParser(cmd *ConvertConfigCmd) *kong.Kong {
	return kong.Must(cmd,
		kong.Name(consts.APPNAME+" convert-config"),
		kong.Description("Convert a config file to the canonical ConfigVersion=1 TOML format."),
		kong.ShortUsageOnError(),
	)
}
