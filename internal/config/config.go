package config

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gorm.io/gorm/schema"
)

type DatabaseDialect string

const (
	PostgreSQL DatabaseDialect = "postgresql"
	SQLite     DatabaseDialect = "sqlite"
)

type configSourceFormat uint8

const (
	configSourceFormatUnknown configSourceFormat = iota
	configSourceFormatLegacy
	configSourceFormatVersioned
)

type Config struct {
	DatabaseDialect         DatabaseDialect
	OutPath                 string
	OutPackagePath          string
	ImportPackagePaths      []string
	Objects                 *[]string
	JSONTagOverridesByTable map[string]map[string]string
	ExtraFields             map[string][]ExtraField
	TypeMap                 map[string]string
	GeneratedTypes          GeneratedTypesConfig
	DbInit                  GenerateDbInitConfig
	NamingStrategy          schema.NamingStrategy `toml:"-"`
	CleanUp                 bool
	DbHost                  string
	DbPort                  int
	DbName                  string
	DbUser                  string
	DbPassword              string
	DbSSLMode               bool
	SQLiteDBPath            string
	sourceFormat            configSourceFormat
}

type ExtraField struct {
	StructPropName    string
	StructPropType    string
	FkStructPropName  string
	RefStructPropName string
	HasMany           bool
	Pointer           bool
}

type GeneratedTypesConfig struct {
	PackageName  string
	RelativePath string
	PackagePath  string
	TypeMap      map[string]string
}

type GenerateDbInitConfig struct {
	Enabled                         bool
	IncludeAutoMigrate              bool
	GenerateAppSettingsRegistration bool
	UseSlogGormLogger               bool
}

var (
	legacyDefaultTypeMap = map[string]string{
		"jsonb": "datatypes.JSONMap",
		"uuid":  "datatypes.UUID",
	}

	versionedDefaultTypeMap = map[string]string{
		"jsonb": "datatypes.JSON",
		"uuid":  "datatypes.UUID",
	}

	defaultImportPackagePaths = []string{
		"github.com/dan-sherwin/gormdb2struct/pgtypes",
		"gorm.io/datatypes",
	}
)

func (c *Config) Normalize() {
	if c.TypeMap == nil {
		c.TypeMap = map[string]string{}
	}
	if c.GeneratedTypes.TypeMap == nil {
		c.GeneratedTypes.TypeMap = map[string]string{}
	}
	if c.ExtraFields == nil {
		c.ExtraFields = map[string][]ExtraField{}
	}
	if c.JSONTagOverridesByTable == nil {
		c.JSONTagOverridesByTable = map[string]map[string]string{}
	}

	if c.GeneratedTypes.HasEntries() {
		if strings.TrimSpace(c.GeneratedTypes.RelativePath) == "" {
			c.GeneratedTypes.RelativePath = filepath.Join("models", "dbtypes")
		}
		if strings.TrimSpace(c.GeneratedTypes.PackageName) == "" {
			c.GeneratedTypes.PackageName = filepath.Base(c.GeneratedTypes.RelativePath)
		}
	}

	for key, value := range c.defaultTypeMap() {
		if _, exists := c.TypeMap[key]; !exists {
			c.TypeMap[key] = value
		}
	}

	seen := make(map[string]struct{}, len(c.ImportPackagePaths))
	for _, importPath := range c.ImportPackagePaths {
		seen[importPath] = struct{}{}
	}
	for _, importPath := range defaultImportPackagePaths {
		if _, exists := seen[importPath]; exists {
			continue
		}
		c.ImportPackagePaths = append(c.ImportPackagePaths, importPath)
	}

	if c.DatabaseDialect == PostgreSQL && c.DbPort == 0 {
		c.DbPort = 5432
	}
}

func (c Config) defaultTypeMap() map[string]string {
	switch c.sourceFormat {
	case configSourceFormatLegacy:
		return legacyDefaultTypeMap
	default:
		return versionedDefaultTypeMap
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.OutPath) == "" {
		return fmt.Errorf("OutPath is required")
	}
	if err := validateObjects(c.Objects); err != nil {
		return err
	}

	switch c.DatabaseDialect {
	case PostgreSQL:
		if strings.TrimSpace(c.DbHost) == "" {
			return fmt.Errorf("DbHost is required for postgresql dialect")
		}
		if strings.TrimSpace(c.DbName) == "" {
			return fmt.Errorf("DbName is required for postgresql dialect")
		}
		if err := c.GeneratedTypes.Validate(); err != nil {
			return err
		}
	case SQLite:
		if strings.TrimSpace(c.SQLiteDBPath) == "" {
			return fmt.Errorf("SqliteDbPath is required for sqlite dialect")
		}
		if c.GeneratedTypes.HasEntries() {
			return fmt.Errorf("GeneratedTypes is currently only supported for postgresql dialect")
		}
	default:
		return fmt.Errorf("DatabaseDialect must be %q or %q", PostgreSQL, SQLite)
	}

	return nil
}

func (g GeneratedTypesConfig) HasEntries() bool {
	return len(g.TypeMap) > 0
}

func (g GeneratedTypesConfig) Validate() error {
	if !g.HasEntries() {
		return nil
	}

	if strings.TrimSpace(g.RelativePath) == "" {
		return fmt.Errorf("GeneratedTypes.RelativePath is required when generated types are configured")
	}
	if filepath.IsAbs(g.RelativePath) {
		return fmt.Errorf("GeneratedTypes.RelativePath must be relative to OutPath")
	}
	cleanRelativePath := filepath.Clean(g.RelativePath)
	if cleanRelativePath == "." || cleanRelativePath == ".." || strings.HasPrefix(cleanRelativePath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("GeneratedTypes.RelativePath must stay within OutPath")
	}
	if strings.TrimSpace(g.PackageName) == "" {
		return fmt.Errorf("GeneratedTypes.PackageName is required when generated types are configured")
	}
	if strings.TrimSpace(g.PackagePath) != "" && path.Base(g.PackagePath) != g.PackageName {
		return fmt.Errorf("GeneratedTypes.PackagePath must end with package name %q", g.PackageName)
	}

	for dbType, goType := range g.TypeMap {
		if strings.TrimSpace(dbType) == "" {
			return fmt.Errorf("GeneratedTypes.TypeMap contains an empty database type key")
		}
		if strings.TrimSpace(goType) == "" {
			return fmt.Errorf("GeneratedTypes.TypeMap[%q] must not be empty", dbType)
		}
	}

	return nil
}

func mergeObjectLists(lists ...*[]string) *[]string {
	hasConfiguredObjects := false
	for _, list := range lists {
		if list != nil {
			hasConfiguredObjects = true
			break
		}
	}
	if !hasConfiguredObjects {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, list := range lists {
		if list == nil {
			continue
		}
		for _, objectName := range *list {
			trimmed := strings.TrimSpace(objectName)
			if trimmed == "" {
				out = append(out, trimmed)
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			out = append(out, trimmed)
		}
	}

	return &out
}

func validateObjects(objects *[]string) error {
	if objects == nil {
		return nil
	}
	for _, objectName := range *objects {
		if strings.TrimSpace(objectName) == "" {
			return fmt.Errorf("objects contains an empty object name")
		}
	}
	return nil
}
