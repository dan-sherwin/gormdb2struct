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
	handled, err = handleCommands(args)
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

Generate strongly typed GORM models and query helpers from an existing database schema.

Arguments:
  <config>    Path to the TOML configuration file.

Commands:
  generate-config-sample    Write a sample TOML configuration file.

Flags:
  -h, --help                    Show context-sensitive help.
  -version, --version           Print version information.
      --logging.level="info"    Log level.

Run "%s generate-config-sample --help" for more information on the sample-config command.
`, consts.APPNAME, consts.APPNAME, consts.APPNAME)
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

func handleCommands(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] != "generate-config-sample" {
		return false, nil
	}

	cmd := GenerateConfigSampleCmd{}
	parser := buildGenerateConfigSampleParser(&cmd)
	if _, err := parser.Parse(args[1:]); err != nil {
		return true, err
	}

	return true, writeSampleConfig(cmd.Out)
}

func writeSampleConfig(out string) error {
	if err := os.WriteFile(out, []byte(config.SampleTOML()), 0o644); err != nil {
		return fmt.Errorf("write sample config %s: %w", out, err)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Sample config written to %s\n", out)
	return nil
}
