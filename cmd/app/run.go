package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/cmd/app/consts"
	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/internal/generator"
)

// Run executes the gormdb2struct CLI against the provided process arguments.
func Run(ctx context.Context, args []string) error {
	initLogger("info")

	handled, err := handleTopLevelHelp(args)
	if handled {
		return err
	}

	handled, err = handleLegacyCompat(args)
	if handled {
		return err
	}
	handled, err = handleCommands(ctx, args)
	if handled {
		return err
	}

	cli := CLIConfig{}
	parser := buildParser(&cli)

	if _, err := parser.Parse(args); err != nil {
		return err
	}

	initLogger(cli.Logging.Level)

	if strings.TrimSpace(cli.ConfigPath) == "" {
		return errors.New("a config.toml path is required")
	}

	cfg, err := config.Load(cli.ConfigPath)
	if err != nil {
		return err
	}

	slog.Debug("Loaded configuration",
		slog.String("dialect", string(cfg.DatabaseDialect)),
		slog.String("out_path", cfg.OutPath),
	)

	return generator.New(slog.Default()).Generate(ctx, cfg)
}

func handleTopLevelHelp(args []string) (bool, error) {
	if len(args) != 1 {
		return false, nil
	}
	switch args[0] {
	case "-h", "--help":
		_, _ = fmt.Fprintf(os.Stdout, `Usage: %s <config> [flags]
       %s generate-config-sample [flags]
       %s inspect <config> [flags]
       %s inspect-postgresql [flags]

Generate strongly typed GORM models and query helpers from an existing database schema.

Arguments:
  <config>    Path to the TOML configuration file.

Commands:
  generate-config-sample    Write a sample TOML configuration file.
  inspect                   Inspect a PostgreSQL schema and recommend type mappings.
  inspect-postgresql        Inspect PostgreSQL directly from connection flags and emit starter config guidance.

Flags:
  -h, --help                    Show context-sensitive help.
  -version, --version           Print version information.
      --logging.level="info"    Log level.

Run "%s generate-config-sample --help", "%s inspect --help", or "%s inspect-postgresql --help" for command-specific help.
`, consts.APPNAME, consts.APPNAME, consts.APPNAME, consts.APPNAME, consts.APPNAME, consts.APPNAME, consts.APPNAME)
		return true, nil
	default:
		return false, nil
	}
}

func handleLegacyCompat(args []string) (bool, error) {
	if len(args) != 1 {
		return false, nil
	}

	switch args[0] {
	case "-version", "--version":
		_, _ = fmt.Fprintf(os.Stdout, "version: %s\n", consts.Version)
		return true, nil
	case "-generateConfigSample":
		return true, writeSampleConfig("gormdb2struct-sample.toml")
	default:
		return false, nil
	}
}

func handleCommands(ctx context.Context, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}

	switch args[0] {
	case "generate-config-sample":
		cmd := GenerateConfigSampleCmd{}
		parser := buildGenerateConfigSampleParser(&cmd)
		if _, err := parser.Parse(args[1:]); err != nil {
			return true, err
		}
		return true, writeSampleConfig(cmd.Out)
	case "inspect":
		cmd := InspectCmd{}
		parser := buildInspectParser(&cmd)
		if _, err := parser.Parse(args[1:]); err != nil {
			return true, err
		}

		initLogger(cmd.Logging.Level)

		cfg, err := config.Load(cmd.ConfigPath)
		if err != nil {
			return true, err
		}

		report, err := generator.New(slog.Default()).Inspect(ctx, cfg)
		if err != nil {
			return true, err
		}

		rendered, err := generator.RenderInspectionReport(report, cmd.Format)
		if err != nil {
			return true, err
		}

		if _, err := fmt.Fprint(os.Stdout, rendered); err != nil {
			return true, fmt.Errorf("write inspection report: %w", err)
		}
		if !strings.HasSuffix(rendered, "\n") {
			_, _ = fmt.Fprintln(os.Stdout)
		}

		return true, nil
	case "inspect-postgresql":
		cmd := InspectPostgreSQLCmd{}
		parser := buildInspectPostgreSQLParser(&cmd)
		if _, err := parser.Parse(args[1:]); err != nil {
			return true, err
		}

		return true, runInspectPostgreSQL(ctx, cmd)
	default:
		return false, nil
	}
}

func writeSampleConfig(out string) error {
	if err := os.WriteFile(out, []byte(config.SampleTOML()), 0o644); err != nil {
		return fmt.Errorf("write sample config %s: %w", out, err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Sample config written to %s\n", out)
	return nil
}
