package config

import (
	"fmt"
	"sort"
	"strings"
)

const implicitImportPackagePath = "gorm.io/datatypes"

// RenderVersionedTOML renders a canonical ConfigVersion=1 TOML representation
// of the effective config. It is intended for migration and normalization, not
// for emitting the commented sample config.
func RenderVersionedTOML(cfg Config) string {
	cfg.Normalize()

	var b strings.Builder

	writeLine(&b, "# gormdb2struct configuration")
	writeLine(&b, fmt.Sprintf("ConfigVersion = %d", CurrentConfigVersion))
	writeBlankLine(&b)

	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "# Generator")
	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "[Generator]")
	writeLine(&b, fmt.Sprintf("OutPath = %q", cfg.OutPath))
	writeLine(&b, fmt.Sprintf("OutPackagePath = %q", cfg.OutPackagePath))
	writeLine(&b, fmt.Sprintf("CleanUp = %t", cfg.CleanUp))
	writeStringArray(&b, "ImportPackagePaths", renderedImportPackagePaths(cfg.ImportPackagePaths))
	if cfg.Objects != nil {
		writeStringArray(&b, "Objects", append([]string(nil), (*cfg.Objects)...))
	}
	writeBlankLine(&b)

	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "# Database")
	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "[Database]")
	writeLine(&b, fmt.Sprintf("Dialect = %q", cfg.DatabaseDialect))
	writeBlankLine(&b)

	switch cfg.DatabaseDialect {
	case PostgreSQL:
		writeLine(&b, "[Database.PostgreSQL]")
		writeLine(&b, fmt.Sprintf("Host = %q", cfg.DbHost))
		writeLine(&b, fmt.Sprintf("Port = %d", cfg.DbPort))
		writeLine(&b, fmt.Sprintf("Name = %q", cfg.DbName))
		writeLine(&b, fmt.Sprintf("User = %q", cfg.DbUser))
		writeLine(&b, fmt.Sprintf("Password = %q", cfg.DbPassword))
		writeLine(&b, fmt.Sprintf("SSLMode = %t", cfg.DbSSLMode))
	case SQLite:
		writeLine(&b, "[Database.SQLite]")
		writeLine(&b, fmt.Sprintf("Path = %q", cfg.SQLiteDBPath))
	}
	writeBlankLine(&b)

	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "# Optional generation sections")
	writeLine(&b, "# ----------------------------------------------------------------------")
	writeLine(&b, "[DbInit]")
	writeLine(&b, fmt.Sprintf("Enabled = %t", cfg.DbInit.Enabled))
	writeLine(&b, fmt.Sprintf("IncludeAutoMigrate = %t", cfg.DbInit.IncludeAutoMigrate))
	writeLine(&b, fmt.Sprintf("GenerateAppSettingsRegistration = %t", cfg.DbInit.GenerateAppSettingsRegistration))
	writeLine(&b, fmt.Sprintf("UseSlogGormLogger = %t", cfg.DbInit.UseSlogGormLogger))

	if filteredTypeMap := renderedTypeMap(cfg.TypeMap, versionedDefaultTypeMap); len(filteredTypeMap) > 0 {
		writeBlankLine(&b)
		writeLine(&b, "[TypeMap]")
		writeStringMap(&b, filteredTypeMap)
	}

	if len(cfg.ExtraFields) > 0 {
		writeBlankLine(&b)
		writeLine(&b, "[ExtraFields]")
		writeExtraFields(&b, cfg.ExtraFields)
	}

	if len(cfg.JSONTagOverridesByTable) > 0 {
		writeBlankLine(&b)
		writeJSONTagOverrides(&b, cfg.JSONTagOverridesByTable)
	}

	if cfg.DatabaseDialect == PostgreSQL && cfg.GeneratedTypes.HasEntries() {
		writeBlankLine(&b)
		writeBlankLine(&b)
		writeLine(&b, "# ----------------------------------------------------------------------")
		writeLine(&b, "# PostgreSQL-only sections")
		writeLine(&b, "# ----------------------------------------------------------------------")
		writeLine(&b, "[PostgreSQL.GeneratedTypes]")
		writeLine(&b, fmt.Sprintf("PackageName = %q", cfg.GeneratedTypes.PackageName))
		writeLine(&b, fmt.Sprintf("RelativePath = %q", cfg.GeneratedTypes.RelativePath))
		writeLine(&b, fmt.Sprintf("PackagePath = %q", cfg.GeneratedTypes.PackagePath))
		writeBlankLine(&b)
		writeLine(&b, "[PostgreSQL.GeneratedTypes.TypeMap]")
		writeStringMap(&b, cfg.GeneratedTypes.TypeMap)
	}

	return b.String()
}

func renderedImportPackagePaths(imports []string) []string {
	seen := make(map[string]struct{}, len(imports))
	out := make([]string, 0, len(imports))
	for _, importPath := range imports {
		trimmed := strings.TrimSpace(importPath)
		if trimmed == "" || trimmed == implicitImportPackagePath {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func renderedTypeMap(typeMap map[string]string, implicitDefaults map[string]string) map[string]string {
	if len(typeMap) == 0 {
		return nil
	}

	out := make(map[string]string, len(typeMap))
	for key, value := range typeMap {
		if defaultValue, isDefault := implicitDefaults[key]; isDefault && defaultValue == value {
			continue
		}
		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func writeLine(b *strings.Builder, line string) {
	b.WriteString(line)
	b.WriteByte('\n')
}

func writeBlankLine(b *strings.Builder) {
	b.WriteByte('\n')
}

func writeStringArray(b *strings.Builder, name string, values []string) {
	writeLine(b, fmt.Sprintf("%s = [", name))
	for _, value := range values {
		writeLine(b, fmt.Sprintf("  %q,", value))
	}
	writeLine(b, "]")
}

func writeStringMap(b *strings.Builder, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeLine(b, fmt.Sprintf("%q"+" = "+"%q", key, values[key]))
	}
}

func writeExtraFields(b *strings.Builder, values map[string][]ExtraField) {
	tables := make([]string, 0, len(values))
	for table := range values {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	for _, table := range tables {
		for _, field := range values[table] {
			writeLine(b, fmt.Sprintf("[[ExtraFields.%q]]", table))
			writeLine(b, fmt.Sprintf("StructPropName = %q", field.StructPropName))
			writeLine(b, fmt.Sprintf("StructPropType = %q", field.StructPropType))
			writeLine(b, fmt.Sprintf("FkStructPropName = %q", field.FkStructPropName))
			writeLine(b, fmt.Sprintf("RefStructPropName = %q", field.RefStructPropName))
			writeLine(b, fmt.Sprintf("HasMany = %t", field.HasMany))
			writeLine(b, fmt.Sprintf("Pointer = %t", field.Pointer))
			writeBlankLine(b)
		}
	}
}

func writeJSONTagOverrides(b *strings.Builder, values map[string]map[string]string) {
	writeLine(b, "[JSONTagOverridesByTable]")

	tables := make([]string, 0, len(values))
	for table := range values {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	for _, table := range tables {
		writeBlankLine(b)
		writeLine(b, fmt.Sprintf("[JSONTagOverridesByTable.%q]", table))
		writeStringMap(b, values[table])
	}
}
