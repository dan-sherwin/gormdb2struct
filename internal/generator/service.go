package generator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/iancoleman/strcase"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type Service struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{logger: logger}
}

func (s *Service) Generate(ctx context.Context, cfg config.Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	switch cfg.DatabaseDialect {
	case config.PostgreSQL:
		return s.generatePostgres(ctx, cfg)
	case config.SQLite:
		return s.generateSQLite(ctx, cfg)
	default:
		return fmt.Errorf("unsupported database dialect %q", cfg.DatabaseDialect)
	}
}

func newGenerator(outPath string) *gen.Generator {
	return gen.NewGenerator(gen.Config{
		OutPath:           outPath,
		ModelPkgPath:      filepath.Join(outPath, "models"),
		WithUnitTest:      false,
		FieldNullable:     true,
		FieldCoverable:    true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		Mode:              gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
}

func configureJSONTags(g *gen.Generator) {
	g.WithJSONTagNameStrategy(func(col string) string {
		return strcase.ToLowerCamel(col)
	})
}

func cleanUp(outPath string) error {
	if _, err := os.Stat(outPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat output path %s: %w", outPath, err)
	}

	return filepath.WalkDir(outPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), "gen.go") {
			return nil
		}
		if err := osRemove(path); err != nil {
			return fmt.Errorf("remove generated file %s: %w", path, err)
		}
		return nil
	})
}

func genRelationField(ef config.ExtraField, fld gen.Field) {
	baseType := ef.StructPropType
	if idx := strings.LastIndex(ef.StructPropType, "."); idx != -1 {
		baseType = ef.StructPropType[idx+1:]
	}
	if ef.Pointer {
		baseType = "*" + baseType
	}
	if ef.HasMany {
		baseType = "[]" + baseType
	}

	fld.Name = ef.StructPropName
	fld.Type = baseType
	fld.Tag = field.Tag{}
	fld.Tag.Set("json", strcase.ToLowerCamel(ef.StructPropName))
	fld.GORMTag = field.GormTag{}
	fld.GORMTag.Set("foreignKey", ef.FkStructPropName)
	fld.GORMTag.Set("references", ef.RefStructPropName)

	relationType := field.HasOne
	if ef.HasMany {
		relationType = field.HasMany
	}
	fld.Relation = field.NewRelationWithType(relationType, ef.StructPropName, ef.StructPropType)
}
