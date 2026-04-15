package generator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/sqlitetype"
	"github.com/glebarez/sqlite"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func (s *Service) generateSQLite(ctx context.Context, cfg config.Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.logger.Info("Connecting to SQLite", slog.String("path", cfg.SQLiteDBPath))

	db, err := gorm.Open(sqlite.Open(cfg.SQLiteDBPath), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open SQLite database: %w", err)
	}

	sqldb, err := db.DB()
	if err != nil {
		return fmt.Errorf("get SQLite sql.DB handle: %w", err)
	}
	if err := sqldb.PingContext(ctx); err != nil {
		return fmt.Errorf("ping SQLite database: %w", err)
	}

	if cfg.CleanUp {
		if err := cleanUp(cfg.OutPath); err != nil {
			return err
		}
	}

	g := newGenerator(cfg.OutPath)
	configureJSONTags(g)

	dataTypeMap := sqlitetype.CloneTypeMap()
	for columnType, goType := range cfg.TypeMap {
		mappedType := goType
		dataTypeMap[columnType] = func(gorm.ColumnType) string { return mappedType }
	}
	g.WithDataTypeMap(dataTypeMap)
	g.WithImportPkgPath(mergeImportPaths(cfg.ImportPackagePaths, []string{"gorm.io/datatypes"})...)
	g.UseDB(db)

	objects, err := sqliteObjectNames(db, cfg)
	if err != nil {
		return err
	}

	models := make([]any, 0, len(objects))
	for _, objectName := range objects {
		model := g.GenerateModel(objectName)
		if extraFields, ok := cfg.ExtraFields[objectName]; ok {
			for _, extraField := range extraFields {
				fieldFactory := gen.FieldNew("", "", nil)
				fld := fieldFactory(nil)
				genRelationField(extraField, gen.Field(fld))
				model.Fields = append(model.Fields, fld)
			}
		}
		if jsonOverrides, ok := cfg.JSONTagOverridesByTable[objectName]; ok {
			for _, fld := range model.Fields {
				if jsonTag, exists := jsonOverrides[fld.ColumnName]; exists {
					fld.Tag.Set("json", jsonTag)
					continue
				}
				if jsonTag, exists := jsonOverrides[fld.Name]; exists {
					fld.Tag.Set("json", jsonTag)
				}
			}
		}
		models = append(models, model)
	}

	g.ApplyBasic(models...)
	g.Execute()

	if cfg.DbInit.Enabled {
		if err := WriteSQLiteDBInit(cfg, g); err != nil {
			return err
		}
	}

	return nil
}

func sqliteObjectNames(db *gorm.DB, cfg config.Config) ([]string, error) {
	if cfg.Objects != nil {
		return append([]string(nil), (*cfg.Objects)...), nil
	}
	return sqlitetype.LoadTableNames(db)
}

func mergeImportPaths(existing []string, required []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(required))
	out := make([]string, 0, len(existing)+len(required))

	for _, path := range existing {
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	for _, path := range required {
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}

	return out
}
