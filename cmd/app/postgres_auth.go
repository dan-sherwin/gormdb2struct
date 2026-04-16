package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func resolveInspectPostgreSQLPassword(cmd InspectPostgreSQLCmd) (string, error) {
	sources := 0
	if cmd.Password != "" {
		sources++
	}
	if strings.TrimSpace(cmd.PasswordEnv) != "" {
		sources++
	}
	if cmd.PasswordStdin {
		sources++
	}
	if cmd.PasswordPrompt {
		sources++
	}

	if sources > 1 {
		return "", fmt.Errorf("--password, --password-env, --password-stdin, and --password-prompt are mutually exclusive")
	}

	switch {
	case strings.TrimSpace(cmd.PasswordEnv) != "":
		value, exists := os.LookupEnv(strings.TrimSpace(cmd.PasswordEnv))
		if !exists {
			return "", fmt.Errorf("environment variable %q is not set", strings.TrimSpace(cmd.PasswordEnv))
		}
		return value, nil
	case cmd.PasswordStdin:
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read PostgreSQL password from stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	case cmd.PasswordPrompt:
		if _, err := fmt.Fprint(os.Stderr, "PostgreSQL password: "); err != nil {
			return "", fmt.Errorf("prompt for PostgreSQL password: %w", err)
		}
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if _, newlineErr := fmt.Fprintln(os.Stderr); newlineErr != nil && err == nil {
			err = newlineErr
		}
		if err != nil {
			return "", fmt.Errorf("read PostgreSQL password from prompt: %w", err)
		}
		return string(password), nil
	default:
		return cmd.Password, nil
	}
}
