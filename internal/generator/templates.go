package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"gorm.io/gen"
)

func WritePostgresDBInit(cfg config.Config, g *gen.Generator) error {
	outPath := g.OutPath
	fullPackageName := resolveOutPackagePath(cfg.OutPackagePath, outPath)
	packageName := filepath.Base(outPath)
	modelStructNames := sortedModelStructNames(g)

	data := struct {
		PackageName                     string
		FullPackageName                 string
		DbHost                          string
		DbPort                          int
		DbName                          string
		DbUser                          string
		DbPassword                      string
		DbSSLMode                       bool
		IncludeAutoMigrate              bool
		GenerateAppSettingsRegistration bool
		UseSlogGormLogger               bool
		ModelStructNames                []string
	}{
		PackageName:                     packageName,
		FullPackageName:                 fullPackageName,
		DbHost:                          cfg.DbHost,
		DbPort:                          cfg.DbPort,
		DbName:                          cfg.DbName,
		DbUser:                          cfg.DbUser,
		DbPassword:                      cfg.DbPassword,
		DbSSLMode:                       cfg.DbSSLMode,
		IncludeAutoMigrate:              cfg.DbInit.IncludeAutoMigrate,
		GenerateAppSettingsRegistration: cfg.DbInit.GenerateAppSettingsRegistration,
		UseSlogGormLogger:               cfg.DbInit.UseSlogGormLogger,
		ModelStructNames:                modelStructNames,
	}

	tmpl, err := template.New("postgres_db_init").Parse(postgresDBInitTemplate)
	if err != nil {
		return fmt.Errorf("parse postgres DbInit template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render postgres DbInit template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format postgres DbInit template: %w", err)
	}

	outFile := filepath.Join(outPath, "db.go")
	if err := os.WriteFile(outFile, formatted, 0o644); err != nil {
		return fmt.Errorf("write postgres DbInit file %s: %w", outFile, err)
	}

	return nil
}

func WriteSQLiteDBInit(cfg config.Config, g *gen.Generator) error {
	outPath := g.OutPath
	fullPackageName := resolveOutPackagePath(cfg.OutPackagePath, outPath)
	packageName := filepath.Base(outPath)
	modelStructNames := sortedModelStructNames(g)

	data := struct {
		PackageName                     string
		FullPackageName                 string
		DbPath                          string
		IncludeAutoMigrate              bool
		GenerateAppSettingsRegistration bool
		UseSlogGormLogger               bool
		ModelStructNames                []string
	}{
		PackageName:                     packageName,
		FullPackageName:                 fullPackageName,
		DbPath:                          cfg.SQLiteDBPath,
		IncludeAutoMigrate:              cfg.DbInit.IncludeAutoMigrate,
		GenerateAppSettingsRegistration: cfg.DbInit.GenerateAppSettingsRegistration,
		UseSlogGormLogger:               cfg.DbInit.UseSlogGormLogger,
		ModelStructNames:                modelStructNames,
	}

	tmpl, err := template.New("sqlite_db_init").Parse(sqliteDBInitTemplate)
	if err != nil {
		return fmt.Errorf("parse sqlite DbInit template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render sqlite DbInit template: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format sqlite DbInit template: %w", err)
	}

	outFile := filepath.Join(outPath, "db_sqlite.go")
	if err := os.WriteFile(outFile, formatted, 0o644); err != nil {
		return fmt.Errorf("write sqlite DbInit file %s: %w", outFile, err)
	}

	return nil
}

func sortedModelStructNames(g *gen.Generator) []string {
	modelNames := make([]string, 0, len(g.Data))
	for modelName := range g.Data {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)
	return modelNames
}

func resolveOutPackagePath(explicit, outPath string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}

	wd, err := os.Getwd()
	if err != nil {
		return filepath.Base(outPath)
	}

	moduleRoot, modulePath, err := findModuleInfo(wd)
	if err != nil {
		return filepath.Base(outPath)
	}

	absOutPath, err := filepath.Abs(outPath)
	if err != nil {
		return filepath.Base(outPath)
	}

	rel, err := filepath.Rel(moduleRoot, absOutPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.Base(outPath)
	}

	return path.Join(modulePath, filepath.ToSlash(rel))
}

func findModuleInfo(startDir string) (string, string, error) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return dir, strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
				}
			}
			return "", "", fmt.Errorf("module statement not found in %s", goModPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", "", fmt.Errorf("go.mod not found from %s upward", startDir)
}

const postgresDBInitTemplate = `
// Code generated by gormdb2struct; DO NOT EDIT.
// This file was generated automatically to initialize DB connections.
package {{.PackageName}}

import (
	utilities "github.com/dan-sherwin/go-utilities"
	{{- if .GenerateAppSettingsRegistration}}
	app_settings "github.com/dan-sherwin/go-app-settings"
	{{- end}}
	{{- if .UseSlogGormLogger}}
	slogGorm "github.com/orandin/slog-gorm"
	{{- end}}
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	{{- if .IncludeAutoMigrate}}
	"{{.FullPackageName}}/models"
	{{- end}}
)

var (
	DbHost     = {{printf "%q" .DbHost}}
	DbPort     = {{.DbPort}}
	DbName     = {{printf "%q" .DbName}}
	DbUser     = {{printf "%q" .DbUser}}
	DbPassword = {{printf "%q" .DbPassword}}
	DbSSLMode  = {{.DbSSLMode}}
	DB         *gorm.DB
)

{{- if .GenerateAppSettingsRegistration}}
func init() {
	app_settings.RegisterStringSetting("dbHost", "Hostname of the database", &DbHost)
	app_settings.RegisterIntSetting("dbPort", "Port of the database", &DbPort)
	app_settings.RegisterStringSetting("dbName", "Name of the database", &DbName)
	app_settings.RegisterStringSetting("dbUser", "Username of the database", &DbUser)
	app_settings.RegisterStringSetting("dbPassword", "Password of the database", &DbPassword)
	app_settings.RegisterBoolSetting("dbSSLMode", "Whether to require SSL for the database connection", &DbSSLMode)
}

{{- end}}
// DbInit opens the PostgreSQL database. If optionalDSN is provided, it overrides the generated connection string.
func DbInit(optionalDSN ...string) error {
	var dsn string
	if len(optionalDSN) > 0 && optionalDSN[0] != "" {
		dsn = optionalDSN[0]
	} else {
		dsn = utilities.DbDSN(utilities.DbDSNConfig{
			Server:   DbHost,
			Port:     DbPort,
			Name:     DbName,
			User:     DbUser,
			Password: DbPassword,
			SSLMode:  DbSSLMode,
		})
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		{{- if .UseSlogGormLogger}}
		Logger: slogGorm.New(),
		{{- end}}
	})
	if err != nil {
		return err
	}

	sqldb, err := gormDB.DB()
	if err != nil {
		return err
	}
	if err = sqldb.Ping(); err != nil {
		return err
	}

	{{if .IncludeAutoMigrate}}
	if err = gormDB.AutoMigrate(
		{{- range .ModelStructNames}}
		&models.{{.}}{},
		{{- end}}
	); err != nil {
		return err
	}
	{{end}}

	SetDefault(gormDB)
	DB = gormDB
	return nil
}
`

const sqliteDBInitTemplate = `
// Code generated by gormdb2struct; DO NOT EDIT.
// This file was generated automatically to initialize SQLite DB connections.
package {{.PackageName}}

import (
	{{- if .GenerateAppSettingsRegistration}}
	app_settings "github.com/dan-sherwin/go-app-settings"
	{{- end}}
	"github.com/glebarez/sqlite"
	{{- if .UseSlogGormLogger}}
	slogGorm "github.com/orandin/slog-gorm"
	{{- end}}
	"gorm.io/gorm"
	{{- if .IncludeAutoMigrate}}
	"{{.FullPackageName}}/models"
	{{- end}}
)

var (
	DbPath = {{printf "%q" .DbPath}}
	DB     *gorm.DB
)

{{- if .GenerateAppSettingsRegistration}}
func init() {
	app_settings.RegisterStringSetting("dbPath", "Path of the database", &DbPath)
}

{{- end}}
// DbInit opens the SQLite database. If optionalFilePath is provided, it overrides the generated DbPath.
func DbInit(optionalFilePath ...string) error {
	filePath := DbPath
	if len(optionalFilePath) > 0 && optionalFilePath[0] != "" {
		filePath = optionalFilePath[0]
	}

	gormDB, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{
		{{- if .UseSlogGormLogger}}
		Logger: slogGorm.New(),
		{{- end}}
	})
	if err != nil {
		return err
	}

	sqldb, err := gormDB.DB()
	if err != nil {
		return err
	}
	if err = sqldb.Ping(); err != nil {
		return err
	}

	{{if .IncludeAutoMigrate}}
	if err = gormDB.AutoMigrate(
		{{- range .ModelStructNames}}
		&models.{{.}}{},
		{{- end}}
	); err != nil {
		return err
	}
	{{end}}

	SetDefault(gormDB)
	DB = gormDB
	return nil
}
`
