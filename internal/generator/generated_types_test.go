package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
)

func TestCleanUpRemovesNestedGeneratedFiles(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "generated")
	generatedFiles := []string{
		filepath.Join(outPath, "gen.go"),
		filepath.Join(outPath, "models", "ticket.gen.go"),
		filepath.Join(outPath, "models", "types", "ticket_status.gen.go"),
	}
	keepFile := filepath.Join(outPath, "models", "keep.go")

	for _, file := range append(generatedFiles, keepFile) {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(file), err)
		}
		if err := os.WriteFile(file, []byte("package test\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	if err := cleanUp(outPath); err != nil {
		t.Fatalf("clean up: %v", err)
	}

	for _, file := range generatedFiles {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Fatalf("expected generated file %s to be removed, err=%v", file, err)
		}
	}
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("expected keep file to remain: %v", err)
	}
}

func TestBuildGeneratedTypesPackageWritesFiles(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "generated")
	cfg := config.Config{
		OutPath:        outPath,
		OutPackagePath: "example.com/generated",
		GeneratedTypes: config.GeneratedTypesConfig{
			PackageName:  "types",
			RelativePath: filepath.Join("models", "types"),
			TypeMap: map[string]string{
				"ticket_status":        "TicketStatus",
				"public.ticket_status": "TicketStatus",
				"ticket_status[]":      "TicketStatusArray",
				"tenant_number":        "TenantNumber",
				"public.tenant_number": "TenantNumber",
			},
		},
	}

	enumMeta := map[string]postgresEnumMetadata{
		"ticket_status": {
			SchemaName: "public",
			TypeName:   "ticket_status",
			Labels:     []string{"new", "in_progress", "closed"},
		},
		"public.ticket_status": {
			SchemaName: "public",
			TypeName:   "ticket_status",
			Labels:     []string{"new", "in_progress", "closed"},
		},
	}
	domainMeta := map[string]postgresDomainMetadata{
		"tenant_number": {
			SchemaName:   "public",
			DomainName:   "tenant_number",
			BaseSchema:   "pg_catalog",
			BaseTypeName: "text",
			ConstraintDef: []string{
				"CHECK (VALUE ~ '^[0-9]{4}-[0-9]{4}$')",
			},
		},
		"public.tenant_number": {
			SchemaName:   "public",
			DomainName:   "tenant_number",
			BaseSchema:   "pg_catalog",
			BaseTypeName: "text",
			ConstraintDef: []string{
				"CHECK (VALUE ~ '^[0-9]{4}-[0-9]{4}$')",
			},
		},
	}

	pkg, err := buildGeneratedTypesPackage(cfg, enumMeta, domainMeta)
	if err != nil {
		t.Fatalf("build generated types package: %v", err)
	}

	if pkg.PackagePath != "example.com/generated/models/types" {
		t.Fatalf("unexpected package path: %q", pkg.PackagePath)
	}
	if pkg.TypeMap["ticket_status"] != "types.TicketStatus" {
		t.Fatalf("unexpected enum type map value: %q", pkg.TypeMap["ticket_status"])
	}
	if pkg.TypeMap["public.ticket_status"] != "types.TicketStatus" {
		t.Fatalf("unexpected qualified enum type map value: %q", pkg.TypeMap["public.ticket_status"])
	}
	if pkg.TypeMap["ticket_status[]"] != "types.TicketStatusArray" {
		t.Fatalf("unexpected array type map value: %q", pkg.TypeMap["ticket_status[]"])
	}
	if pkg.TypeMap["tenant_number"] != "types.TenantNumber" {
		t.Fatalf("unexpected domain type map value: %q", pkg.TypeMap["tenant_number"])
	}
	if pkg.TypeMap["public.tenant_number"] != "types.TenantNumber" {
		t.Fatalf("unexpected qualified domain type map value: %q", pkg.TypeMap["public.tenant_number"])
	}
	if len(pkg.Enums) != 1 {
		t.Fatalf("expected one generated enum file, got %d", len(pkg.Enums))
	}
	if len(pkg.Domains) != 1 {
		t.Fatalf("expected one generated domain file, got %d", len(pkg.Domains))
	}

	if err := writeGeneratedTypesPackage(pkg); err != nil {
		t.Fatalf("write generated types package: %v", err)
	}

	assertFileContains(t, filepath.Join(pkg.OutputDir, "zz_generated_helpers.gen.go"), "func generatedScanInto")
	assertFileContains(t, filepath.Join(pkg.OutputDir, "ticket_status.gen.go"), `type TicketStatus string`)
	assertFileContains(t, filepath.Join(pkg.OutputDir, "ticket_status.gen.go"), `TicketStatusInProgress TicketStatus = "in_progress"`)
	assertFileContains(t, filepath.Join(pkg.OutputDir, "ticket_status_array.gen.go"), `type TicketStatusArray []TicketStatus`)
	assertFileContains(t, filepath.Join(pkg.OutputDir, "tenant_number.gen.go"), `type TenantNumber string`)
	assertFileContains(t, filepath.Join(pkg.OutputDir, "tenant_number.gen.go"), `regexp.MustCompile`)
}

func assertFileContains(t *testing.T, path string, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), substring) {
		t.Fatalf("expected %s to contain %q", path, substring)
	}
}
