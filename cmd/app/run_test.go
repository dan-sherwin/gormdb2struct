package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGenerateConfigSampleCommandWritesSampleWithoutDeprecatedAliases(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "sample.toml")

	if err := Run(context.Background(), []string{"generate-config-sample", "--out", outPath}); err != nil {
		t.Fatalf("run generate-config-sample command: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read sample config: %v", err)
	}

	sample := string(content)
	if !strings.Contains(sample, "ConfigVersion = 1\n") {
		t.Fatalf("expected sample config to declare ConfigVersion = 1")
	}
	if !strings.Contains(sample, "\n[Generator]\n") {
		t.Fatalf("expected sample config to show the Generator section")
	}
	if !strings.Contains(sample, "\n[DbInit]\n") {
		t.Fatalf("expected sample config to show the DbInit section")
	}
	if !strings.Contains(sample, "\n[PostgreSQL.GeneratedTypes.TypeMap]\n") {
		t.Fatalf("expected sample config to show the PostgreSQL.GeneratedTypes.TypeMap section")
	}
	if strings.Contains(sample, "\nDatabaseDialect =") ||
		strings.Contains(sample, "\nGenerateDbInit =") ||
		strings.Contains(sample, "\nTables =") ||
		strings.Contains(sample, "MaterializedViews") ||
		strings.Contains(sample, "DomainTypeMap") {
		t.Fatalf("expected sample config to omit legacy config keys")
	}
	if !strings.Contains(sample, "# Objects = [") {
		t.Fatalf("expected sample config to show Generator.Objects")
	}
}

func TestRunLegacyGenerateConfigSampleFlagStillWorks(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	if err := Run(context.Background(), []string{"-generateConfigSample"}); err != nil {
		t.Fatalf("run legacy generateConfigSample flag: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "gormdb2struct-sample.toml"))
	if err != nil {
		t.Fatalf("read legacy sample config: %v", err)
	}

	if !strings.Contains(string(content), "ConfigVersion = 1\n") {
		t.Fatalf("expected legacy sample output to declare ConfigVersion = 1")
	}
	if !strings.Contains(string(content), "\n[DbInit]\n") {
		t.Fatalf("expected legacy sample output to show the DbInit section")
	}
	if !strings.Contains(string(content), "\n[PostgreSQL.GeneratedTypes.TypeMap]\n") {
		t.Fatalf("expected legacy sample output to show PostgreSQL.GeneratedTypes.TypeMap")
	}
	if strings.Contains(string(content), "\nDatabaseDialect =") ||
		strings.Contains(string(content), "\nGenerateDbInit =") ||
		strings.Contains(string(content), "\nTables =") ||
		strings.Contains(string(content), "MaterializedViews") ||
		strings.Contains(string(content), "DomainTypeMap") {
		t.Fatalf("expected legacy sample output to omit legacy config keys")
	}
}

func TestRunTopLevelHelpListsGenerateConfigSampleWithoutRunCommand(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Run(context.Background(), []string{"--help"}); err != nil {
			t.Fatalf("run top-level help: %v", err)
		}
	})

	if !strings.Contains(output, "generate-config-sample") {
		t.Fatalf("expected top-level help to include generate-config-sample command, got: %s", output)
	}
	if !strings.Contains(output, "-version, --version") {
		t.Fatalf("expected top-level help to include version flags, got: %s", output)
	}
	if strings.Contains(output, "\n  run") || strings.Contains(output, "Usage: gormdb2struct <command>") {
		t.Fatalf("expected top-level help to omit run command, got: %s", output)
	}
	if !strings.Contains(output, "Usage: gormdb2struct <config> [flags]") {
		t.Fatalf("expected top-level help to describe config path usage, got: %s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	defer func() {
		os.Stdout = origStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}

	return buf.String()
}
