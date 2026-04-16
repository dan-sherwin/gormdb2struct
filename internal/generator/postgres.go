package generator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/pgtypes"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type postgresObjectKind string

const (
	postgresObjectTable            postgresObjectKind = "table"
	postgresObjectView             postgresObjectKind = "view"
	postgresObjectMaterializedView postgresObjectKind = "materialized view"
)

type postgresObject struct {
	Name string
	Kind postgresObjectKind
}

func (s *Service) generatePostgres(ctx context.Context, cfg config.Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	db, err := openPostgresDB(ctx, s.logger, cfg)
	if err != nil {
		return err
	}

	sqldb, err := db.DB()
	if err != nil {
		return fmt.Errorf("get PostgreSQL sql.DB handle: %w", err)
	}

	if cfg.CleanUp {
		if err := cleanUp(cfg.OutPath); err != nil {
			return err
		}
	}

	effectiveCfg, err := preparePostgresGeneratedTypes(cfg, db)
	if err != nil {
		return err
	}

	g := newGenerator(effectiveCfg.OutPath)
	configureJSONTags(g)
	g.WithImportPkgPath(effectiveCfg.ImportPackagePaths...)
	g.WithDataTypeMap(buildPostgresDataTypeMap(effectiveCfg))
	g.UseDB(db)

	objects, err := postgresObjects(db, effectiveCfg)
	if err != nil {
		return err
	}

	models := make([]any, 0, len(objects))
	for _, object := range objects {
		switch object.Kind {
		case postgresObjectTable:
			model := g.GenerateModel(object.Name)
			if extraFields, ok := effectiveCfg.ExtraFields[object.Name]; ok {
				for _, extraField := range extraFields {
					fieldFactory := gen.FieldNew("", "", nil)
					fld := fieldFactory(nil)
					genRelationField(extraField, gen.Field(fld))
					model.Fields = append(model.Fields, fld)
				}
			}
			if jsonOverrides, ok := effectiveCfg.JSONTagOverridesByTable[object.Name]; ok {
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
		case postgresObjectView, postgresObjectMaterializedView:
			tmpViewName := object.Name + "_temp"
			if err := createTempView(sqldb, tmpViewName, object.Name); err != nil {
				return err
			}
			defer func(name string) {
				_ = dropView(sqldb, name)
			}(tmpViewName)

			model := g.GenerateModelAs(tmpViewName, effectiveCfg.NamingStrategy.SchemaName(object.Name))
			if extraFields, ok := effectiveCfg.ExtraFields[object.Name]; ok {
				for _, extraField := range extraFields {
					fieldFactory := gen.FieldNew("", "", nil)
					fld := fieldFactory(nil)
					genRelationField(extraField, gen.Field(fld))
					model.Fields = append(model.Fields, fld)
				}
			}
			model.FileName = object.Name
			model.TableName = object.Name
			if jsonOverrides, ok := effectiveCfg.JSONTagOverridesByTable[object.Name]; ok {
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
		default:
			return fmt.Errorf("unsupported PostgreSQL object kind %q for %q", object.Kind, object.Name)
		}
	}

	g.ApplyBasic(models...)
	g.Execute()

	if effectiveCfg.DbInit.Enabled {
		if err := WritePostgresDBInit(effectiveCfg, g); err != nil {
			return err
		}
	}

	return nil
}

func postgresObjects(db *gorm.DB, cfg config.Config) ([]postgresObject, error) {
	relations, err := loadPostgresRelations(db)
	if err != nil {
		return nil, err
	}
	if cfg.Objects == nil {
		return defaultPostgresObjects(relations), nil
	}

	routines, err := loadPostgresRoutines(db)
	if err != nil {
		return nil, err
	}
	return resolveConfiguredPostgresObjects(*cfg.Objects, relations, routines)
}

func loadPostgresRelations(db *gorm.DB) ([]postgresObject, error) {
	type relationRow struct {
		Name string
		Kind string
	}

	var rows []relationRow
	if err := db.Raw(`
		SELECT c.relname AS name,
		       CASE c.relkind
		         WHEN 'r' THEN 'table'
		         WHEN 'p' THEN 'table'
		         WHEN 'v' THEN 'view'
		         WHEN 'm' THEN 'materialized view'
		       END AS kind
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'p', 'v', 'm')
		ORDER BY c.relname
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load PostgreSQL objects: %w", err)
	}

	relations := make([]postgresObject, 0, len(rows))
	for _, row := range rows {
		switch row.Kind {
		case string(postgresObjectTable):
			relations = append(relations, postgresObject{Name: row.Name, Kind: postgresObjectTable})
		case string(postgresObjectView):
			relations = append(relations, postgresObject{Name: row.Name, Kind: postgresObjectView})
		case string(postgresObjectMaterializedView):
			relations = append(relations, postgresObject{Name: row.Name, Kind: postgresObjectMaterializedView})
		default:
			return nil, fmt.Errorf("unsupported PostgreSQL relation kind %q for %q", row.Kind, row.Name)
		}
	}

	return relations, nil
}

func loadPostgresRoutines(db *gorm.DB) (map[string]string, error) {
	type routineRow struct {
		Name string
		Kind string
	}

	var rows []routineRow
	if err := db.Raw(`
		SELECT p.proname AS name,
		       CASE p.prokind
		         WHEN 'p' THEN 'procedure'
		         ELSE 'function'
		       END AS kind
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public'
		  AND p.prokind IN ('f', 'p')
		ORDER BY p.proname
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load PostgreSQL routines: %w", err)
	}

	routines := make(map[string]string, len(rows))
	for _, row := range rows {
		routines[row.Name] = row.Kind
	}
	return routines, nil
}

func defaultPostgresObjects(relations []postgresObject) []postgresObject {
	objects := make([]postgresObject, 0, len(relations))
	for _, relation := range relations {
		if relation.Kind == postgresObjectTable {
			objects = append(objects, relation)
		}
	}
	for _, relation := range relations {
		if relation.Kind == postgresObjectView {
			objects = append(objects, relation)
		}
	}
	for _, relation := range relations {
		if relation.Kind == postgresObjectMaterializedView {
			objects = append(objects, relation)
		}
	}
	return objects
}

func resolveConfiguredPostgresObjects(configured []string, relations []postgresObject, routines map[string]string) ([]postgresObject, error) {
	relationByName := make(map[string]postgresObject, len(relations))
	for _, relation := range relations {
		relationByName[relation.Name] = relation
	}

	seen := map[string]struct{}{}
	objects := make([]postgresObject, 0, len(configured))
	for _, configuredObject := range configured {
		objectName, err := normalizePostgresObjectName(configuredObject)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[objectName]; exists {
			continue
		}
		seen[objectName] = struct{}{}

		if relation, exists := relationByName[objectName]; exists {
			objects = append(objects, relation)
			continue
		}
		if routineKind, exists := routines[objectName]; exists {
			return nil, fmt.Errorf("PostgreSQL object %q is a %s; gormdb2struct currently supports tables, views, and materialized views only", configuredObject, routineKind)
		}
		return nil, fmt.Errorf("PostgreSQL object %q was not found in schema public", configuredObject)
	}

	return objects, nil
}

func normalizePostgresObjectName(name string) (string, error) {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return "", fmt.Errorf("PostgreSQL Objects contains an empty object name")
	}
	cleaned = strings.TrimPrefix(cleaned, "public.")
	if strings.Contains(cleaned, ".") {
		return "", fmt.Errorf("PostgreSQL object %q must be unqualified or use the public schema", name)
	}
	return cleaned, nil
}

func buildPostgresDataTypeMap(cfg config.Config) map[string]func(gorm.ColumnType) string {
	dataTypeMap := make(map[string]func(gorm.ColumnType) string, len(pgtypes.PgTypeMap)+len(cfg.TypeMap))

	resolver := func(defaultType string) func(gorm.ColumnType) string {
		return func(columnType gorm.ColumnType) string {
			if columnDefinition, ok := columnType.ColumnType(); ok {
				cleaned := normalizeColumnType(columnDefinition)
				if mapped, exists := cfg.TypeMap[cleaned]; exists {
					return mapped
				}
			}
			return defaultType
		}
	}

	for pgType, goType := range pgtypes.PgTypeMap {
		dataTypeMap[pgType] = resolver(goType)
	}
	for pgType, goType := range cfg.TypeMap {
		dataTypeMap[pgType] = resolver(goType)
	}

	return dataTypeMap
}

func normalizeColumnType(columnType string) string {
	cleaned := strings.TrimSpace(columnType)
	if idx := strings.IndexByte(cleaned, '('); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return cleaned
}

func createTempView(db *sql.DB, tmpViewName, sourceView string) error {
	if err := dropView(db, tmpViewName); err != nil {
		return err
	}
	query := fmt.Sprintf(`CREATE VIEW %s AS SELECT * FROM %s`, quoteIdent(tmpViewName), quoteIdent(sourceView))
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("create temp view for %s: %w", sourceView, err)
	}
	return nil
}

func dropView(db *sql.DB, viewName string) error {
	query := fmt.Sprintf(`DROP VIEW IF EXISTS %s`, quoteIdent(viewName))
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("drop temp view %s: %w", viewName, err)
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func postgresDSN(cfg config.Config) string {
	parts := []string{
		fmt.Sprintf("host=%s", cfg.DbHost),
		fmt.Sprintf("dbname=%s", cfg.DbName),
	}

	if cfg.DbPort != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", cfg.DbPort))
	}
	if cfg.DbUser != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.DbUser))
	}
	if cfg.DbPassword != "" {
		parts = append(parts, fmt.Sprintf("password=%s", cfg.DbPassword))
	}
	if cfg.DbSSLMode {
		parts = append(parts, "sslmode=require")
	} else {
		parts = append(parts, "sslmode=disable")
	}

	return strings.Join(parts, " ")
}
