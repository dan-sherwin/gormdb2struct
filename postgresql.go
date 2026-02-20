package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/dan-sherwin/go-utilities"
	"github.com/dan-sherwin/gormdb2struct/pgtypes"
	"github.com/iancoleman/strcase"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func postgresToGorm(cfg ConversionConfig) {
	var db *gorm.DB
	var err error
	if cfg.DbHost == "" {
		cfg.DbHost = os.Getenv("DB_HOST")
		if cfg.DbHost == "" {
			cfg.DbHost = "localhost"
		}
	}
	if cfg.DbPort == 0 {
		cfg.DbPort = 5432
		port := os.Getenv("DB_PORT")
		if port != "" {
			cfg.DbPort, err = strconv.Atoi(port)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
	}
	if cfg.DbName == "" {
		cfg.DbName = os.Getenv("DB_NAME")
		if cfg.DbName == "" {
			log.Fatal("no database name provided. Please set DB_NAME environment variable or pass it as a command line argument")
		}
	}
	if cfg.DbUser == "" {
		cfg.DbUser = os.Getenv("DB_USER")
	}
	if cfg.DbPassword == "" {
		cfg.DbPassword = os.Getenv("DB_PASSWORD")
	}
	dsn := utilities.DbDSN(utilities.DbDSNConfig{
		Server:   cfg.DbHost,
		Port:     cfg.DbPort,
		Name:     cfg.DbName,
		User:     cfg.DbUser,
		Password: cfg.DbPassword,
		SSLMode:  cfg.DbSSLMode,
	})
	db, err = gorm.Open(postgres.Open(dsn))
	if err != nil {
		log.Fatal(err.Error())
	}
	sqldb, _ := db.DB()
	err = sqldb.Ping()
	if err != nil {
		log.Fatal("Unable to ping database: " + err.Error())
	}

	if cfg.CleanUp {
		cleanUp(cfg.OutPath)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:           cfg.OutPath,
		ModelPkgPath:      cfg.OutPath + "/models",
		WithUnitTest:      false,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		Mode:              gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	tables := []string{}
	if cfg.Tables != nil {
		tables = *cfg.Tables
	} else {
		err = db.Raw("select table_name from information_schema.tables where table_schema = 'public'").Scan(&tables).Error
		if err != nil {
			log.Fatal(err.Error())
		}
	}
	materializedViews := []string{}
	if cfg.MaterializedViews != nil {
		materializedViews = *cfg.MaterializedViews
	} else {
		err = db.Raw("select matviewname from pg_matviews where schemaname='public'").Scan(&materializedViews).Error
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	g.WithJSONTagNameStrategy(func(col string) (tag string) { return strcase.ToLowerCamel(col) })
	g.WithImportPkgPath(cfg.ImportPackagePaths...)

	var dtMaps = map[string]func(columnType gorm.ColumnType) (dataType string){}
	f := func(def string) func(columnType gorm.ColumnType) (dataType string) {
		return func(columnType gorm.ColumnType) string {
			if colType, ok := columnType.ColumnType(); ok {
				if domain, ok := cfg.DomainTypeMap[colType]; ok {
					return domain
				}
				if pt, ok := cfg.TypeMap[colType]; ok {
					return pt
				}
			}
			return def
		}
	}
	for pgTypeSTr, goTypeStr := range pgtypes.PgTypeMap {
		dtMaps[pgTypeSTr] = f(goTypeStr)
	}

	g.WithDataTypeMap(dtMaps)
	g.UseDB(db)
	modelsMap := map[string]any{}
	for _, tableName := range tables {
		model := g.GenerateModel(tableName)
		if ef, ok := cfg.ExtraFields[tableName]; ok {
			for _, ef := range ef {
				a := gen.FieldNew("", "", nil)
				f := a(nil)
				genRelationField(&ef, gen.Field(f))
				model.Fields = append(model.Fields, f)
			}
		}
		if jsonTagOverrides, ok := cfg.JsonTagOverridesByTable[tableName]; ok {
			for _, f := range model.Fields {
				if jsonTag, ok := jsonTagOverrides[f.ColumnName]; ok {
					f.Tag.Set("json", jsonTag)
				} else if jsonTag, ok := jsonTagOverrides[f.Name]; ok {
					f.Tag.Set("json", jsonTag)
				}
			}
		}
		modelsMap[tableName] = model
	}

	for _, viewName := range materializedViews {
		tmpViewName := viewName + "_temp"
		_, _ = sqldb.Query("drop view if exists " + tmpViewName)
		_, err = sqldb.Query("create view " + tmpViewName + " as select * from " + viewName)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer sqldb.Query("drop view " + tmpViewName)
		modelName := cfg.NamingStrategy.SchemaName(viewName)
		model := g.GenerateModelAs(tmpViewName, modelName)

		if ef, ok := cfg.ExtraFields[viewName]; ok {
			for _, ef := range ef {
				a := gen.FieldNew("", "", nil)
				f := a(nil)
				genRelationField(&ef, gen.Field(f))
				model.Fields = append(model.Fields, f)
			}
		}
		model.FileName = viewName
		model.TableName = viewName
		if jsonTagOverrides, ok := cfg.JsonTagOverridesByTable[viewName]; ok {
			for _, f := range model.Fields {
				if jsonTag, ok := jsonTagOverrides[f.ColumnName]; ok {
					f.Tag.Set("json", jsonTag)
				} else if jsonTag, ok := jsonTagOverrides[f.Name]; ok {
					f.Tag.Set("json", jsonTag)
				}
			}
		}
		modelsMap[viewName] = model
	}

	models := []any{}
	for _, model := range modelsMap {
		models = append(models, model)
	}
	g.ApplyBasic(models...)
	g.Execute()
	if cfg.GenerateDbInit {
		generatePostgresDbInit(cfg, g)
	}
}

func generatePostgresDbInit(cfg ConversionConfig, g *gen.Generator) {
	outPath := g.OutPath
	fullPackageName := filepath.Base(outPath)
	if cfg.OutPackagePath != "" {
		fullPackageName = cfg.OutPackagePath
	}
	packageName := filepath.Base(fullPackageName)
	modelStructNames := []string{}
	for modelName := range g.Data {
		modelStructNames = append(modelStructNames, modelName)
	}

	// Prepare data for the template
	data := struct {
		PackageName        string
		FullPackageName    string
		DbHost             string
		DbPort             int
		DbName             string
		DbUser             string
		DbPassword         string
		DbSSLMode          bool
		IncludeAutoMigrate bool
		ModelStructNames   []string
	}{
		PackageName:        packageName,
		FullPackageName:    fullPackageName,
		DbHost:             cfg.DbHost,
		DbPort:             cfg.DbPort,
		DbName:             cfg.DbName,
		DbUser:             cfg.DbUser,
		DbPassword:         cfg.DbPassword,
		DbSSLMode:          cfg.DbSSLMode,
		IncludeAutoMigrate: cfg.IncludeAutoMigrate,
		ModelStructNames:   modelStructNames,
	}

	tmpl, err := template.New("pgDbInit").Parse(pgDbInitTemplate)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}

	// Write to db.go in the output path
	outFile := filepath.Join(outPath, "db.go")
	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
}

var pgDbInitTemplate = `
// Code generated by gormdb2struct; DO NOT EDIT.
// This file was generated automatically to initialize DB connections.
// Warning: Manual edits may be overwritten by the generator and IDEs like GoLand may mark this as generated code.
package {{.PackageName}}

import (
	"fmt"
	"log/slog"
	"os"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	{{if .IncludeAutoMigrate}}
	"{{.FullPackageName}}/models"
	{{end}}
)

var (
	DbHost     = "{{.DbHost}}"
	DbPort     = {{.DbPort}}
	DbName     = "{{.DbName}}"
	DbUser     = "{{.DbUser}}"
	DbPassword = "{{.DbPassword}}"
	DbSSLMode  = {{.DbSSLMode}}
	DB         *gorm.DB
)

func DbInit(optionalDSN ...string) {
	var dsn string
	if len(optionalDSN) > 0 && optionalDSN[0] != "" {
		dsn = optionalDSN[0]
	} else {
		dsn = DbDSN(DbDSNConfig{
		Server:   DbHost,
		Port:     DbPort,
		Name:     DbName,
		User:     DbUser,
		Password: DbPassword,
		SSLMode:  DbSSLMode,
	})
	}
	slog.Info("Connecting to database", slog.String("host", DbHost), slog.Int("port", DbPort), slog.String("db", DbName), slog.String("user", DbUser))
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: slogGorm.New()})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	sqldb, _ := gormDB.DB()
	if err = sqldb.Ping(); err != nil {
		slog.Error("Unable to ping database: ", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("Database connection established")

	{{if .IncludeAutoMigrate}}
	// Ensure schema exists (idempotent). Uses GORM AutoMigrate to create tables and indexes.
	slog.Debug("Ensuring database schema via AutoMigrate")
	if err = gormDB.AutoMigrate(
		{{- range .ModelStructNames}}
		&models.{{.}}{},
		{{- end}}
	); err != nil {
		slog.Error("Unable to ensure database schema", slog.String("error", err.Error()))
		os.Exit(1)
	}
	{{end}}

	// Expose the query objects for use elsewhere in the app.
	SetDefault(gormDB)
	DB = gormDB
	slog.Debug("GORM query objects initialized")
}

type (
	DbDSNConfig struct {
		Server   string
		Port     int
		Name     string
		User     string
		Password string
		SSLMode  bool
		TimeZone string
	}
)

// DbDSN generates a database connection string (DSN) based on the provided configuration structure, including server, port, database name, user, password, SSL mode, and timezone. The SSL mode defaults to "enable" or "disable" based on the cfg.SSLMode flag.
func DbDSN(cfg DbDSNConfig) string {
	var sm string
	if cfg.SSLMode {
		sm = "enable"
	} else {
		sm = "disable"
	}
	connstr := fmt.Sprintf("host=%s dbname=%s sslmode=%s", cfg.Server, cfg.Name, sm)
	if cfg.Port != 0 {
		connstr = fmt.Sprintf("%s port=%d", connstr, cfg.Port)
	}
	if len(cfg.User) > 0 {
		connstr = fmt.Sprintf("%s user=%s", connstr, cfg.User)
	}
	if len(cfg.Password) > 0 {
		connstr = fmt.Sprintf("%s password=%s", connstr, cfg.Password)
	}
	if len(cfg.TimeZone) > 0 {
		connstr = fmt.Sprintf("%s TimeZone=%s", connstr, cfg.TimeZone)
	}
	return connstr
}

`
