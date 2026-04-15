// Package sqlitetype provides SQLite-specific type mapping and utility functions for GORM.
package sqlitetype

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// TypeMap exposes the SQLite data type mapping used by the generator.
// It maps SQLite type names to functions that return the corresponding Go type as a string.
var TypeMap = map[string]func(gorm.ColumnType) string{
	// ---- booleans ----
	"BOOLEAN": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "bool")
	},
	"BOOL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "bool")
	},
	"TINYINT": func(ct gorm.ColumnType) string {
		// Treat TINYINT(1) as bool; otherwise int8
		col, _ := ct.ColumnType()
		n, _ := ct.Nullable()
		if strings.HasPrefix(strings.ToUpper(col), "TINYINT(1)") {
			return nullablePtr(n, "bool")
		}
		return nullablePtr(n, "int8")
	},

	// ---- integers ----
	"SMALLINT":         func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int16") },
	"INTEGER":          func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") }, // SQLite INTEGER is 64-bit
	"INT":              func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },
	"INT2":             func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int16") },
	"INT8":             func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },
	"MEDIUMINT":        func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int32") },
	"UNSIGNED BIG INT": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "uint64") },
	"BIGINT":           func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "int64") },

	// ---- floats ----
	"REAL":   func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float64") },
	"DOUBLE": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float64") },
	"FLOAT":  func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "float32") },

	// ---- strings ----
	"TEXT":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"VARCHAR": func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"CHAR":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"CLOB":    func(ct gorm.ColumnType) string { n, _ := ct.Nullable(); return nullablePtr(n, "string") },
	"UUID":    func(_ gorm.ColumnType) string { return "datatypes.UUID" },
	"JSON":    func(_ gorm.ColumnType) string { return "datatypes.JSONMap" },
	"JSONB":   func(_ gorm.ColumnType) string { return "datatypes.JSONMap" },

	// ---- bytes ----
	"BLOB": func(_ gorm.ColumnType) string { return "[]byte" },

	// ---- dates/times ----
	"DATE": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	"DATETIME": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	"TIMESTAMP": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Time")
	},
	// Duration-like types (custom schemas often use these as declared types)
	"DURATION": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Duration")
	},
	"INTERVAL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "time.Duration")
	},

	// ---- decimals / numerics ----
	// If you need exact decimals, map to shopspring/decimal and import it.
	"NUMERIC": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "float64")
	},
	"DECIMAL": func(ct gorm.ColumnType) string {
		n, _ := ct.Nullable()
		return nullablePtr(n, "float64")
	},
}

// TableNames returns user-defined (non-internal) tables for SQLite.
func TableNames(db *gorm.DB) (tableNames []string) {
	tableNames, err := LoadTableNames(db)
	if err != nil {
		panic(err)
	}
	return
}

// LoadTableNames returns user-defined (non-internal) SQLite table names in deterministic order.
func LoadTableNames(db *gorm.DB) ([]string, error) {
	tableNames := []string{}
	err := db.Raw(`
		SELECT name
		FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`).Scan(&tableNames).Error
	if err != nil {
		return nil, fmt.Errorf("load sqlite table names: %w", err)
	}
	return tableNames, nil
}

// CloneTypeMap returns a shallow copy of the default SQLite type map so callers
// can override mappings without mutating package-level defaults.
func CloneTypeMap() map[string]func(gorm.ColumnType) string {
	cloned := make(map[string]func(gorm.ColumnType) string, len(TypeMap))
	for k, v := range TypeMap {
		cloned[k] = v
	}
	return cloned
}

func nullablePtr(yes bool, base string) string {
	if yes {
		// make a pointer for nullable scalar types
		switch base {
		case "bool", "int", "int8", "int16", "int32", "int64", "uint64", "float32", "float64", "string", "time.Time", "time.Duration":
			return "*" + base
		}
	}
	return base
}
