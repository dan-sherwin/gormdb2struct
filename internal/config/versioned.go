package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type versionedFileConfig struct {
	ConfigVersion           int
	Generator               versionedGeneratorConfig
	Database                versionedDatabaseConfig
	DbInit                  GenerateDbInitConfig
	TypeMap                 map[string]string
	ExtraFields             map[string][]ExtraField
	JSONTagOverridesByTable map[string]map[string]string
	PostgreSQL              versionedPostgreSQLConfig
}

type versionedGeneratorConfig struct {
	OutPath            string
	OutPackagePath     string
	CleanUp            bool
	ImportPackagePaths []string
	Objects            *[]string
}

type versionedDatabaseConfig struct {
	Dialect    DatabaseDialect
	PostgreSQL versionedPostgreSQLConnectionConfig
	SQLite     versionedSQLiteConnectionConfig
}

type versionedPostgreSQLConnectionConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  bool
}

type versionedSQLiteConnectionConfig struct {
	Path string
}

type versionedPostgreSQLConfig struct {
	GeneratedTypes GeneratedTypesConfig
}

func loadVersioned(data []byte, path string) (Config, error) {
	var raw versionedFileConfig
	meta, err := toml.Decode(string(data), &raw)
	if err != nil {
		return Config{}, fmt.Errorf("parse TOML config %s: %w", path, err)
	}

	if undecoded := formatUndecodedKeys(meta.Undecoded()); len(undecoded) > 0 {
		return Config{}, fmt.Errorf("validate config %s: unsupported keys for ConfigVersion = %d: %s", path, raw.ConfigVersion, strings.Join(undecoded, ", "))
	}

	cfg := Config{
		DatabaseDialect:         raw.Database.Dialect,
		OutPath:                 raw.Generator.OutPath,
		OutPackagePath:          raw.Generator.OutPackagePath,
		ImportPackagePaths:      append([]string(nil), raw.Generator.ImportPackagePaths...),
		Objects:                 raw.Generator.Objects,
		JSONTagOverridesByTable: raw.JSONTagOverridesByTable,
		ExtraFields:             raw.ExtraFields,
		TypeMap:                 raw.TypeMap,
		GeneratedTypes:          raw.PostgreSQL.GeneratedTypes,
		DbInit:                  raw.DbInit,
		CleanUp:                 raw.Generator.CleanUp,
		DbHost:                  raw.Database.PostgreSQL.Host,
		DbPort:                  raw.Database.PostgreSQL.Port,
		DbName:                  raw.Database.PostgreSQL.Name,
		DbUser:                  raw.Database.PostgreSQL.User,
		DbPassword:              raw.Database.PostgreSQL.Password,
		DbSSLMode:               raw.Database.PostgreSQL.SSLMode,
		SQLiteDBPath:            raw.Database.SQLite.Path,
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", path, err)
	}

	return cfg, nil
}

func formatUndecodedKeys(keys []toml.Key) []string {
	if len(keys) == 0 {
		return nil
	}

	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key.String())
	}
	sort.Strings(out)
	return out
}
