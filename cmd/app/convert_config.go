package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
)

func runConvertConfig(cmd ConvertConfigCmd) error {
	if cmd.InPlace && strings.TrimSpace(cmd.Out) != "" {
		return fmt.Errorf("--in-place and --out cannot be used together")
	}

	cfg, err := config.Load(cmd.ConfigPath)
	if err != nil {
		return err
	}

	rendered := config.RenderVersionedTOML(cfg)

	switch {
	case cmd.InPlace:
		if err := os.WriteFile(cmd.ConfigPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("write converted config %s: %w", cmd.ConfigPath, err)
		}
		_, err = fmt.Fprintf(os.Stdout, "Converted config written to %s\n", cmd.ConfigPath)
		return err
	case strings.TrimSpace(cmd.Out) != "":
		outPath := strings.TrimSpace(cmd.Out)
		if err := os.WriteFile(outPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("write converted config %s: %w", outPath, err)
		}
		_, err = fmt.Fprintf(os.Stdout, "Converted config written to %s\n", outPath)
		return err
	default:
		if _, err := fmt.Fprint(os.Stdout, rendered); err != nil {
			return fmt.Errorf("write converted config: %w", err)
		}
		if !strings.HasSuffix(rendered, "\n") {
			_, _ = fmt.Fprintln(os.Stdout)
		}
		return nil
	}
}
