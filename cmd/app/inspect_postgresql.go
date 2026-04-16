package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/internal/generator"
)

func runInspectPostgreSQL(ctx context.Context, cmd InspectPostgreSQLCmd) error {
	initLogger(cmd.Logging.Level)

	cfg, err := buildInspectPostgreSQLConfig(cmd)
	if err != nil {
		return err
	}

	report, err := generator.New(slog.Default()).Inspect(ctx, cfg)
	if err != nil {
		return err
	}

	renderedReport, err := generator.RenderInspectionReport(report, "text")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(os.Stdout, renderedReport); err != nil {
		return fmt.Errorf("write inspection report: %w", err)
	}
	if !endsWithNewline(renderedReport) {
		if _, err := fmt.Fprintln(os.Stdout); err != nil {
			return fmt.Errorf("terminate inspection report: %w", err)
		}
	}

	starterConfig := generator.RenderInspectionStarterConfig(cfg, report)
	switch outputMode, outputPath := inspectPostgreSQLOutputMode(cmd.Out); outputMode {
	case inspectPostgreSQLOutputNone:
		return nil
	case inspectPostgreSQLOutputStdout:
		if _, err := fmt.Fprintln(os.Stdout); err != nil {
			return fmt.Errorf("separate starter config output: %w", err)
		}
		if _, err := fmt.Fprint(os.Stdout, starterConfig); err != nil {
			return fmt.Errorf("write starter config: %w", err)
		}
		if !endsWithNewline(starterConfig) {
			if _, err := fmt.Fprintln(os.Stdout); err != nil {
				return fmt.Errorf("terminate starter config: %w", err)
			}
		}
		return nil
	case inspectPostgreSQLOutputFile:
		if err := os.WriteFile(outputPath, []byte(starterConfig), 0o644); err != nil {
			return fmt.Errorf("write starter config %s: %w", outputPath, err)
		}

		_, err = fmt.Fprintf(os.Stdout, "Starter config written to %s\n", outputPath)
		return err
	default:
		return fmt.Errorf("unsupported inspect-postgresql output mode for %q", cmd.Out)
	}
}

func endsWithNewline(content string) bool {
	return len(content) > 0 && content[len(content)-1] == '\n'
}

type inspectPostgreSQLOutputKind string

const (
	inspectPostgreSQLOutputNone   inspectPostgreSQLOutputKind = "none"
	inspectPostgreSQLOutputStdout inspectPostgreSQLOutputKind = "stdout"
	inspectPostgreSQLOutputFile   inspectPostgreSQLOutputKind = "file"
)

func inspectPostgreSQLOutputMode(out string) (inspectPostgreSQLOutputKind, string) {
	cleaned := strings.TrimSpace(out)
	switch cleaned {
	case "":
		return inspectPostgreSQLOutputNone, ""
	case "stdout":
		return inspectPostgreSQLOutputStdout, ""
	default:
		return inspectPostgreSQLOutputFile, cleaned
	}
}

func buildInspectPostgreSQLConfig(cmd InspectPostgreSQLCmd) (config.Config, error) {
	password, err := resolveInspectPostgreSQLPassword(cmd)
	if err != nil {
		return config.Config{}, err
	}

	cfg := config.Config{
		DatabaseDialect:    config.PostgreSQL,
		OutPath:            cmd.OutPath,
		OutPackagePath:     cmd.OutPackagePath,
		ImportPackagePaths: append([]string(nil), cmd.ImportPackagePaths...),
		GeneratedTypes: config.GeneratedTypesConfig{
			PackageName:  cmd.GeneratedTypesPackage,
			RelativePath: cmd.GeneratedTypesPath,
		},
		DbHost:     cmd.Host,
		DbPort:     cmd.Port,
		DbName:     cmd.Database,
		DbUser:     cmd.User,
		DbPassword: password,
		DbSSLMode:  cmd.SSLMode,
		CleanUp:    true,
	}

	if len(cmd.Objects) > 0 {
		objects := append([]string(nil), cmd.Objects...)
		cfg.Objects = &objects
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}
