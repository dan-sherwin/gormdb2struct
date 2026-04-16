package app

import (
	"os"
	"testing"
)

func TestResolveInspectPostgreSQLPasswordFromEnv(t *testing.T) {
	t.Setenv("GORMDB2STRUCT_TEST_PASSWORD", "from-env")

	password, err := resolveInspectPostgreSQLPassword(InspectPostgreSQLCmd{
		PasswordEnv: "GORMDB2STRUCT_TEST_PASSWORD",
	})
	if err != nil {
		t.Fatalf("resolve password from env: %v", err)
	}
	if password != "from-env" {
		t.Fatalf("expected env password, got %q", password)
	}
}

func TestResolveInspectPostgreSQLPasswordRejectsMultipleSources(t *testing.T) {
	_, err := resolveInspectPostgreSQLPassword(InspectPostgreSQLCmd{
		Password:    "inline",
		PasswordEnv: "DB_PASSWORD",
	})
	if err == nil {
		t.Fatal("expected multiple password sources to fail")
	}
}

func TestResolveInspectPostgreSQLPasswordFromStdin(t *testing.T) {
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = origStdin
	}()

	go func() {
		_, _ = w.WriteString("from-stdin\n")
		_ = w.Close()
	}()

	password, err := resolveInspectPostgreSQLPassword(InspectPostgreSQLCmd{
		PasswordStdin: true,
	})
	if err != nil {
		t.Fatalf("resolve password from stdin: %v", err)
	}
	if password != "from-stdin" {
		t.Fatalf("expected stdin password, got %q", password)
	}
}

func TestBuildInspectPostgreSQLParserCollectsImportPackages(t *testing.T) {
	cmd := InspectPostgreSQLCmd{}
	parser := buildInspectPostgreSQLParser(&cmd)

	if _, err := parser.Parse([]string{
		"--host", "localhost",
		"--database", "mydb",
		"--user", "myuser",
		"--import-package", "pkg/one",
		"--import-package", "pkg/two",
	}); err != nil {
		t.Fatalf("parse inspect-postgresql args: %v", err)
	}

	if len(cmd.ImportPackagePaths) != 2 || cmd.ImportPackagePaths[0] != "pkg/one" || cmd.ImportPackagePaths[1] != "pkg/two" {
		t.Fatalf("unexpected import package paths: %#v", cmd.ImportPackagePaths)
	}
}
