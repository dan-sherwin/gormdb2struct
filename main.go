package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/iancoleman/strcase"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type (
	DatabaseDialect string

	ConversionConfig struct {
		DatabaseDialect         DatabaseDialect
		OutPath                 string
		OutPackagePath          string
		ImportPackagePaths      []string
		Tables                  *[]string
		MaterializedViews       *[]string
		JsonTagOverridesByTable map[string]map[string]string
		ExtraFields             map[string][]ExtraField
		TypeMap                 map[string]string
		DomainTypeMap           map[string]string
		NamingStrategy          schema.NamingStrategy
		CleanUp                 bool
		GenerateDbInit          bool
		IncludeAutoMigrate      bool
		DbHost                  string
		DbPort                  int
		DbName                  string
		DbUser                  string
		DbPassword              string
		DbSSLMode               bool
		Sqlitedbpath            string
	}

	ExtraField struct {
		StructPropName    string //Property name to be added into the table struct
		StructPropType    string //The full path type of the property (e.g. models.MyType)
		FkStructPropName  string //Struct prpoerty name that is used in the foreign key
		RefStructPropName string //Struct property name of the referenced table struct
		HasMany           bool   // A one-one or one-to-many relationship
		Pointer           bool   // Should the added property be a pointer
	}
)

const (
	POSTGRESQL DatabaseDialect = "postgresql"
	SQLITE     DatabaseDialect = "sqlite"
)

var (
	// These variables are set via -ldflags at build time, for example:
	//   go build -ldflags "-X main.version=v0.1.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=2025-09-01T08:38:00"
	version = "dev"
	commit  = "none"
	date    = "unknown"

	conversionConfig = ConversionConfig{
		TypeMap: map[string]string{
			"jsonb": "datatypes.JSONMap",
			"uuid":  "datatypes.UUID",
		},
		ImportPackagePaths: []string{
			"github.com/dan-sherwin/gormdb2struct/pgtypes",
		},
	}
	//extraFields    = map[string][]ExtraField{
	//"ticket_extended": {
	//	{
	//		StructPropName:    "Attachments",
	//		StructPropType:    "models.Attachment",
	//		FkStructPropName:  "TicketID",
	//		RefStructPropName: "TicketID",
	//		HasMany:           true,
	//		Pointer:           true,
	//	},
	//},
	//}
	//jsonTagOverridesByTable = map[string]map[string]string{
	//"ticket_extended": {
	//	"subject_fts": "-",
	//},
	//}
)

func usage(exitCode int, errMsg string) {
	prog := filepath.Base(os.Args[0])
	if strings.TrimSpace(errMsg) != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", errMsg)
	}
	fmt.Fprintf(os.Stderr, "Usage:\n  %s <config.toml>\n  %s -generateConfigSample\n  %s -version | --version\n\n", prog, prog, prog)
	fmt.Fprintln(os.Stderr, "Description:")
	fmt.Fprintln(os.Stderr, "  Generates GORM models and optional DB initializer code from an existing database.")
	fmt.Fprintln(os.Stderr, "  Provide a TOML configuration file describing the database and generation options.")
	fmt.Fprintln(os.Stderr, "  Use -generateConfigSample to write a sample configuration file named 'gormdb2struct-sample.toml' in the current directory.")
	os.Exit(exitCode)
}

func main() {
	// Handle version flags early
	if len(os.Args) == 2 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Fprintf(os.Stdout, "version: %s\ncommit: %s\ndate: %s\n", version, commit, date)
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "-generateConfigSample" {
		out := "gormdb2struct-sample.toml"
		if err := os.WriteFile(out, []byte(sampleConfigTOML()), 0644); err != nil {
			usage(2, fmt.Sprintf("failed to write sample config to %s: %v", out, err))
		}
		fmt.Fprintf(os.Stdout, "Sample config written to %s\n", out)
		return
	}
	if len(os.Args) != 2 {
		usage(2, "exactly one argument is required: path to a TOML config file or -generateConfigSample")
	}
	cfgPath := os.Args[1]
	if _, err := os.Stat(cfgPath); err != nil {
		usage(2, fmt.Sprintf("cannot access config file %s: %v", cfgPath, err))
	}
	var cfg ConversionConfig
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		usage(2, fmt.Sprintf("failed to parse TOML config: %v", err))
	}
	// Ensure defaults for maps to avoid nil-map issues if omitted in TOML
	if cfg.TypeMap == nil {
		cfg.TypeMap = map[string]string{}
	}
	if cfg.DomainTypeMap == nil {
		cfg.DomainTypeMap = map[string]string{}
	}
	if cfg.ExtraFields == nil {
		cfg.ExtraFields = map[string][]ExtraField{}
	}
	if cfg.JsonTagOverridesByTable == nil {
		cfg.JsonTagOverridesByTable = map[string]map[string]string{}
	}
	// Merge defaults from conversionConfig into cfg when not defined in the imported config
	// Merge TypeMap
	for k, v := range conversionConfig.TypeMap {
		if _, exists := cfg.TypeMap[k]; !exists {
			cfg.TypeMap[k] = v
		}
	}
	// Merge DomainTypeMap
	for k, v := range conversionConfig.DomainTypeMap {
		if _, exists := cfg.DomainTypeMap[k]; !exists {
			cfg.DomainTypeMap[k] = v
		}
	}
	// Merge ImportPackagePaths (append missing entries while preserving order)
	existing := map[string]struct{}{}
	for _, p := range cfg.ImportPackagePaths {
		existing[p] = struct{}{}
	}
	for _, p := range conversionConfig.ImportPackagePaths {
		if _, ok := existing[p]; !ok {
			cfg.ImportPackagePaths = append(cfg.ImportPackagePaths, p)
		}
	}

	// Validate imported config
	if strings.TrimSpace(cfg.OutPath) == "" {
		usage(2, "configuration error: OutPath is required")
	}
	if cfg.DatabaseDialect != POSTGRESQL && cfg.DatabaseDialect != SQLITE {
		usage(2, fmt.Sprintf("configuration error: DatabaseDialect must be '%s' or '%s'", POSTGRESQL, SQLITE))
	}
	if cfg.DatabaseDialect == POSTGRESQL {
		if cfg.DbPort == 0 {
			cfg.DbPort = 5432
		}
		if strings.TrimSpace(cfg.DbHost) == "" {
			usage(2, "configuration error: DbHost is required for postgresql dialect")
		}
		if strings.TrimSpace(cfg.DbName) == "" {
			usage(2, "configuration error: DbName is required for postgresql dialect")
		}
	}
	if cfg.DatabaseDialect == SQLITE {
		if strings.TrimSpace(cfg.Sqlitedbpath) == "" {
			usage(2, "configuration error: Sqlitedbpath is required for sqlite dialect")
		}
	}

	switch cfg.DatabaseDialect {
	case POSTGRESQL:
		postgresToGorm(cfg)
	case SQLITE:
		sqliteToGorm(cfg)
	default:
		usage(2, fmt.Sprintf("unknown database dialect: %s (expected '%s' or '%s')", cfg.DatabaseDialect, POSTGRESQL, SQLITE))
	}
}

func genRelationField(ef *ExtraField, fld gen.Field) {
	baseType := ef.StructPropType
	if lastDotIndex := strings.LastIndex(ef.StructPropType, "."); lastDotIndex != -1 {
		baseType = ef.StructPropType[lastDotIndex+1:]
	}
	if ef.Pointer {
		baseType = "*" + baseType
	}
	if ef.HasMany {
		baseType = "[]" + baseType
	}
	fld.Name = ef.StructPropName
	fld.Type = baseType
	t := field.Tag{}
	t.Set("json", strcase.ToLowerCamel(ef.StructPropName))
	fld.Tag = t
	fld.GORMTag = field.GormTag{}
	fld.GORMTag.Set("foreignKey", ef.FkStructPropName)
	fld.GORMTag.Set("references", ef.RefStructPropName)
	r := field.HasOne
	if ef.HasMany {
		r = field.HasMany
	}
	fld.Relation = field.NewRelationWithType(
		r,
		ef.StructPropName,
		ef.StructPropType,
	)
}

func sampleConfigTOML() string {
	return `# gormdb2struct configuration
# OutPath: directory where generated files are written (models, query, db init)
OutPath = "./generated"

# OutPackagePath: package path to the out path for use in the DbInit file (e.g. github.com/username/my_app/generated) (optional)
OutPackagePath = ""

# DatabaseDialect: "postgresql" or "sqlite"
DatabaseDialect = "postgresql"

# GenerateDbInit: also generate a db initialization file (db.go or db_sqlite.go)
GenerateDbInit = true

# IncludeAutoMigrate: if true, generated DbInit will run AutoMigrate for all models
IncludeAutoMigrate = false

# CleanUp: remove previous *gen.go files in OutPath before generating
CleanUp = true

# ImportPackagePaths: extra imports to include in generated code (optional)
ImportPackagePaths = [
  "github.com/dan-sherwin/gormdb2struct/pgtypes",
]

# Tables (optional. defaults to all tables)
#Tables = ["foo", 'bar']

# Materialized Views (optional. defaults to all)
#MaterializedViews = ["foo","bar"]

# TypeMap: database column type overrides (optional)
[TypeMap]
# "jsonb" = "datatypes.JSONMap"
# "uuid"  = "datatypes.UUID"

# DomainTypeMap: map database domain names to Go types (optional)
[DomainTypeMap]
# "my_text_domain" = "string"

# ExtraFields: add relation fields to specific models (optional)
[ExtraFields]
# [ExtraFields."ticket_extended"]
#   [[ExtraFields."ticket_extended"]]
#   StructPropName = "Attachments"
#   StructPropType = "models.Attachment"  # fully-qualified type
#   FkStructPropName = "TicketID"
#   RefStructPropName = "TicketID"
#   HasMany = true
#   Pointer = true

# JsonTagOverridesByTable: override json tags for fields (optional)
[JsonTagOverridesByTable]
# [JsonTagOverridesByTable."ticket_extended"]
#   subject_fts = "-"  # omit from JSON

# --- PostgreSQL specific options ---
# Required when DatabaseDialect = "postgresql"
DbHost = "localhost"     # required
DbPort = 5432             # optional, defaults to 5432
DbName = "my_database"    # required
DbUser = "my_user"        # optional
DbPassword = "secret"     # optional
DbSSLMode = false         # optional: true to enable sslmode=require in DSN

# --- SQLite specific option ---
# Required when DatabaseDialect = "sqlite"
Sqlitedbpath = "./schema.db"
`
}

func cleanUp(outPath string) {
	//Cleanup
	genFiles, err := filepath.Glob(outPath + "/*gen.go")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, genFile := range genFiles {
		os.Remove(genFile)
	}
	genFiles, err = filepath.Glob(outPath + "/models/*gen.go")
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, genFile := range genFiles {
		os.Remove(genFile)
	}

}
