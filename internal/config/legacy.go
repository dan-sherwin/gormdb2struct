package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type legacyFileConfig struct {
	DatabaseDialect         DatabaseDialect
	OutPath                 string
	OutPackagePath          string
	ImportPackagePaths      []string
	Tables                  *[]string `toml:"Tables"`
	MaterializedViews       *[]string `toml:"MaterializedViews"`
	JSONTagOverridesByTable map[string]map[string]string
	ExtraFields             map[string][]ExtraField
	TypeMap                 map[string]string
	DomainTypeMap           map[string]string `toml:"DomainTypeMap"`
	GenerateDbInit          bool
	IncludeAutoMigrate      bool
	CleanUp                 bool
	DbHost                  string
	DbPort                  int
	DbName                  string
	DbUser                  string
	DbPassword              string
	DbSSLMode               bool
	SQLiteDBPath            string `toml:"SqliteDbPath"`
	LegacySQLiteDBPath      string `toml:"Sqlitedbpath"`
}

func loadLegacy(data []byte, path string) (Config, error) {
	var raw legacyFileConfig
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return Config{}, fmt.Errorf("parse TOML config %s: %w", path, err)
	}

	sqlitePath := strings.TrimSpace(raw.SQLiteDBPath)
	if sqlitePath == "" {
		sqlitePath = strings.TrimSpace(raw.LegacySQLiteDBPath)
	}

	typeMap, err := mergeLegacyTypeMaps(raw.TypeMap, raw.DomainTypeMap)
	if err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", path, err)
	}

	cfg := Config{
		DatabaseDialect:         raw.DatabaseDialect,
		OutPath:                 raw.OutPath,
		OutPackagePath:          raw.OutPackagePath,
		ImportPackagePaths:      append([]string(nil), raw.ImportPackagePaths...),
		Objects:                 mergeObjectLists(raw.Tables, raw.MaterializedViews),
		JSONTagOverridesByTable: raw.JSONTagOverridesByTable,
		ExtraFields:             raw.ExtraFields,
		TypeMap:                 typeMap,
		DbInit: GenerateDbInitConfig{
			Enabled:            raw.GenerateDbInit,
			IncludeAutoMigrate: raw.IncludeAutoMigrate,
		},
		CleanUp:      raw.CleanUp,
		DbHost:       raw.DbHost,
		DbPort:       raw.DbPort,
		DbName:       raw.DbName,
		DbUser:       raw.DbUser,
		DbPassword:   raw.DbPassword,
		DbSSLMode:    raw.DbSSLMode,
		SQLiteDBPath: sqlitePath,
		sourceFormat: configSourceFormatLegacy,
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", path, err)
	}

	return cfg, nil
}

func mergeLegacyTypeMaps(typeMap map[string]string, domainTypeMap map[string]string) (map[string]string, error) {
	if len(typeMap) == 0 && len(domainTypeMap) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(typeMap)+len(domainTypeMap))
	for key, value := range typeMap {
		out[key] = value
	}
	for key, value := range domainTypeMap {
		existing, exists := out[key]
		if exists && existing != value {
			return nil, fmt.Errorf("TypeMap[%q] conflicts with DomainTypeMap[%q] in legacy config", key, key)
		}
		out[key] = value
	}

	return out, nil
}
