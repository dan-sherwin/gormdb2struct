package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAcceptsLegacySQLiteKeyAndAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
OutPath = "./generated"
DatabaseDialect = "sqlite"
Sqlitedbpath = "./legacy.db"
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.SQLiteDBPath != "./legacy.db" {
		t.Fatalf("expected legacy sqlite path to normalize, got %q", cfg.SQLiteDBPath)
	}
	if cfg.TypeMap["jsonb"] != "datatypes.JSONMap" {
		t.Fatalf("expected default jsonb type map, got %q", cfg.TypeMap["jsonb"])
	}
	if cfg.TypeMap["uuid"] != "datatypes.UUID" {
		t.Fatalf("expected default uuid type map, got %q", cfg.TypeMap["uuid"])
	}
	assertContainsImportPath(t, cfg.ImportPackagePaths, "github.com/dan-sherwin/gormdb2struct/pgtypes")
}

func TestLoadAcceptsLegacyDomainTypeMapAndObjectLists(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
OutPath = "./generated"
DatabaseDialect = "postgresql"
DbHost = "localhost"
DbName = "example"
Tables = ["tickets", "ticket_comments"]
MaterializedViews = ["ticket_rollup", "tickets"]

[TypeMap]
"jsonb" = "datatypes.JSONMap"

[DomainTypeMap]
"tenant_number" = "sl_datatypes.SpacelinkTenantNumber"
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Objects == nil {
		t.Fatal("expected legacy object lists to merge into Objects")
	}
	wantObjects := []string{"tickets", "ticket_comments", "ticket_rollup"}
	gotObjects := *cfg.Objects
	if len(gotObjects) != len(wantObjects) {
		t.Fatalf("expected %d objects, got %d: %#v", len(wantObjects), len(gotObjects), gotObjects)
	}
	for idx := range wantObjects {
		if gotObjects[idx] != wantObjects[idx] {
			t.Fatalf("expected objects %v, got %v", wantObjects, gotObjects)
		}
	}
	if cfg.TypeMap["tenant_number"] != "sl_datatypes.SpacelinkTenantNumber" {
		t.Fatalf("expected legacy DomainTypeMap to merge into TypeMap, got %q", cfg.TypeMap["tenant_number"])
	}
}

func TestLoadAcceptsLegacyDbInitSettings(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
OutPath = "./generated"
DatabaseDialect = "postgresql"
DbHost = "localhost"
DbName = "example"
GenerateDbInit = true
IncludeAutoMigrate = true
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.DbInit.Enabled {
		t.Fatal("expected legacy GenerateDbInit to load")
	}
	if !cfg.DbInit.IncludeAutoMigrate {
		t.Fatal("expected legacy IncludeAutoMigrate to load")
	}
}

func TestLoadRejectsLegacyTypeMapConflict(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
OutPath = "./generated"
DatabaseDialect = "postgresql"
DbHost = "localhost"
DbName = "example"

[TypeMap]
"my_text_domain" = "string"

[DomainTypeMap]
"my_text_domain" = "int64"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected conflicting legacy type mappings to be rejected")
	}
	if !strings.Contains(err.Error(), `TypeMap["my_text_domain"] conflicts with DomainTypeMap["my_text_domain"]`) {
		t.Fatalf("expected legacy conflict error, got %v", err)
	}
}

func TestSampleTOMLLoads(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, SampleTOML())

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load sample config: %v", err)
	}

	if cfg.OutPath != "./generated" {
		t.Fatalf("expected sample OutPath to load, got %q", cfg.OutPath)
	}
	if cfg.DbHost != "localhost" {
		t.Fatalf("expected sample Database.PostgreSQL.Host to load, got %q", cfg.DbHost)
	}
	if cfg.SQLiteDBPath != "./schema.db" {
		t.Fatalf("expected sample Database.SQLite.Path to load, got %q", cfg.SQLiteDBPath)
	}
	if cfg.GeneratedTypes.PackageName != "dbtypes" {
		t.Fatalf("expected sample PostgreSQL.GeneratedTypes package name to load, got %q", cfg.GeneratedTypes.PackageName)
	}
	if cfg.DbInit.GenerateAppSettingsRegistration {
		t.Fatal("expected sample DbInit.GenerateAppSettingsRegistration to default to false")
	}
	if cfg.DbInit.UseSlogGormLogger {
		t.Fatal("expected sample DbInit.UseSlogGormLogger to default to false")
	}
	if !cfg.DbInit.Enabled {
		t.Fatal("expected sample DbInit.Enabled to default to true")
	}
}

func TestLoadAcceptsVersionedGeneratedTypesDefaults(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 1

[Generator]
OutPath = "./generated"

[Database]
Dialect = "postgresql"

[Database.PostgreSQL]
Host = "localhost"
Name = "example"

[PostgreSQL.GeneratedTypes.TypeMap]
"ticket_status" = "TicketStatus"
`)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.GeneratedTypes.PackageName != "dbtypes" {
		t.Fatalf("expected default generated package name dbtypes, got %q", cfg.GeneratedTypes.PackageName)
	}
	if cfg.GeneratedTypes.RelativePath != filepath.Join("models", "dbtypes") {
		t.Fatalf("expected default generated relative path models/dbtypes, got %q", cfg.GeneratedTypes.RelativePath)
	}
	if cfg.GeneratedTypes.TypeMap["ticket_status"] != "TicketStatus" {
		t.Fatalf("expected generated type map to load, got %q", cfg.GeneratedTypes.TypeMap["ticket_status"])
	}
}

func TestLoadRejectsUnsupportedConfigVersion(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 99
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected unsupported ConfigVersion to be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported ConfigVersion 99") {
		t.Fatalf("expected unsupported ConfigVersion error, got %v", err)
	}
}

func TestLoadRejectsNonIntegerConfigVersion(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = "1"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected non-integer ConfigVersion to be rejected")
	}
	if !strings.Contains(err.Error(), "ConfigVersion must be an integer") {
		t.Fatalf("expected non-integer ConfigVersion error, got %v", err)
	}
}

func TestLoadRejectsUnknownKeysInVersionedConfig(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 1

[Generator]
OutPath = "./generated"
UnknownField = true

[Database]
Dialect = "sqlite"

[Database.SQLite]
Path = "./test.db"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected unknown ConfigVersion=1 keys to be rejected")
	}
	if !strings.Contains(err.Error(), "Generator.UnknownField") {
		t.Fatalf("expected unknown key in error, got %v", err)
	}
}

func TestLoadRejectsEmptyObjectNameInVersionedConfig(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 1

[Generator]
OutPath = "./generated"
Objects = ["tickets", "   "]

[Database]
Dialect = "sqlite"

[Database.SQLite]
Path = "./test.db"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected empty object names to be rejected")
	}
	if !strings.Contains(err.Error(), "objects contains an empty object name") {
		t.Fatalf("expected object validation error, got %v", err)
	}
}

func TestLoadRejectsGeneratedTypesForSQLiteInVersionedConfig(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 1

[Generator]
OutPath = "./generated"

[Database]
Dialect = "sqlite"

[Database.SQLite]
Path = "./test.db"

[PostgreSQL.GeneratedTypes.TypeMap]
"ticket_status" = "TicketStatus"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected sqlite generated types to be rejected")
	}
	if !strings.Contains(err.Error(), "only supported for postgresql") {
		t.Fatalf("expected sqlite generated types error, got %v", err)
	}
}

func TestLoadRejectsGeneratedTypesPackagePathAliasMismatch(t *testing.T) {
	t.Parallel()

	cfgPath := writeConfig(t, `
ConfigVersion = 1

[Generator]
OutPath = "./generated"

[Database]
Dialect = "postgresql"

[Database.PostgreSQL]
Host = "localhost"
Name = "example"

[PostgreSQL.GeneratedTypes]
PackageName = "dbtypes"
RelativePath = "models/dbtypes"
PackagePath = "example.com/generated/types"

[PostgreSQL.GeneratedTypes.TypeMap]
"ticket_status" = "TicketStatus"
`)

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected generated package path alias mismatch to be rejected")
	}
	if !strings.Contains(err.Error(), `must end with package name "dbtypes"`) {
		t.Fatalf("expected generated package path mismatch error, got %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return cfgPath
}

func assertContainsImportPath(t *testing.T, importPaths []string, want string) {
	t.Helper()

	for _, importPath := range importPaths {
		if importPath == want {
			return
		}
	}

	t.Fatalf("expected import path %q to be present in %#v", want, importPaths)
}
