package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/internal/generator"
	"gorm.io/gen"
)

func TestSQLiteDbInitTemplateOptionalAppSettingsAndSlogGorm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sqlite template test in short mode")
	}

	outPath := filepath.Join(projectRoot(t), "generated_sqlite_nodb_opts")
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outPath) })

	g := gen.NewGenerator(gen.Config{
		OutPath:      outPath,
		ModelPkgPath: filepath.Join(outPath, "models"),
	})
	g.Data["Foo"] = nil

	cfg := config.Config{
		SQLiteDBPath: "./example.db",
		DbInit: config.GenerateDbInitConfig{
			GenerateAppSettingsRegistration: true,
			UseSlogGormLogger:               true,
		},
	}

	if err := generator.WriteSQLiteDBInit(cfg, g); err != nil {
		t.Fatalf("write sqlite DbInit with optional features: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(outPath, "db_sqlite.go"))
	if err != nil {
		t.Fatalf("reading generated db_sqlite.go: %v", err)
	}
	content := string(b)

	mustContain(t, content, `app_settings.RegisterStringSetting("dbPath", "Path of the database", &DbPath)`)
	mustContain(t, content, `"github.com/dan-sherwin/go-app-settings"`)
	mustContain(t, content, `"github.com/orandin/slog-gorm"`)
	mustContain(t, content, "Logger: slogGorm.New(),")
	mustNotContain(t, content, "slog.")
}
