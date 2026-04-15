package generator

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/dan-sherwin/gormdb2struct/pgtypes"
	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
)

type generatedTypesPackage struct {
	PackageName string
	PackagePath string
	OutputDir   string
	TypeMap     map[string]string
	Enums       []generatedEnumType
	Domains     []generatedDomainType
	Arrays      []generatedArrayType
}

type generatedEnumType struct {
	DBType    string
	GoType    string
	FileName  string
	Labels    []string
	Constants []generatedEnumConstant
}

type generatedEnumConstant struct {
	Name  string
	Value string
}

type generatedDomainType struct {
	DBType           string
	GoType           string
	FileName         string
	UnderlyingGoType string
	Imports          []string
	Constraints      []string
	RegexPattern     string
}

type generatedArrayType struct {
	DBType        string
	GoType        string
	FileName      string
	ElementGoType string
}

func (t generatedEnumType) getGoType() string {
	return t.GoType
}

func (t generatedDomainType) getGoType() string {
	return t.GoType
}

func (t generatedArrayType) getGoType() string {
	return t.GoType
}

var (
	pgConstraintRegexPattern = regexp.MustCompile(`~\*?\s*'((?:[^'\\]|\\.)*)'`)
	qualifiedTypePattern     = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z_][A-Za-z0-9_]*`)
	knownTypeImportPaths     = map[string]string{
		"datatypes": "gorm.io/datatypes",
		"json":      "encoding/json",
		"net":       "net",
		"pgtypes":   "github.com/dan-sherwin/gormdb2struct/pgtypes",
		"time":      "time",
		"uuid":      "github.com/google/uuid",
	}
)

func preparePostgresGeneratedTypes(cfg config.Config, db *gorm.DB) (config.Config, error) {
	if !cfg.GeneratedTypes.HasEntries() {
		return cfg, nil
	}

	enumMeta, err := loadPostgresEnumMetadata(db)
	if err != nil {
		return cfg, err
	}
	domainMeta, err := loadPostgresDomainMetadata(db)
	if err != nil {
		return cfg, err
	}

	pkg, err := buildGeneratedTypesPackage(cfg, enumMeta, domainMeta)
	if err != nil {
		return cfg, err
	}
	if err := writeGeneratedTypesPackage(pkg); err != nil {
		return cfg, err
	}

	effective := cfg
	effective.TypeMap = cloneStringMap(cfg.TypeMap)
	for dbType, goType := range pkg.TypeMap {
		effective.TypeMap[dbType] = goType
	}

	filteredImports := filterImportPathsByAlias(cfg.ImportPackagePaths, pkg.PackageName, pkg.PackagePath)
	effective.ImportPackagePaths = mergeImportPaths(filteredImports, []string{pkg.PackagePath})

	return effective, nil
}

func buildGeneratedTypesPackage(cfg config.Config, enumMeta map[string]postgresEnumMetadata, domainMeta map[string]postgresDomainMetadata) (generatedTypesPackage, error) {
	pkg := generatedTypesPackage{
		PackageName: cfg.GeneratedTypes.PackageName,
		PackagePath: resolveGeneratedTypesPackagePath(cfg),
		OutputDir:   filepath.Join(cfg.OutPath, cfg.GeneratedTypes.RelativePath),
		TypeMap:     make(map[string]string, len(cfg.GeneratedTypes.TypeMap)),
	}

	seenTypeNames := make(map[string]string, len(cfg.GeneratedTypes.TypeMap))

	for dbType, configuredGoType := range cfg.GeneratedTypes.TypeMap {
		qualifiedGoType, goTypeName, err := normalizeGeneratedGoType(pkg.PackageName, configuredGoType)
		if err != nil {
			return generatedTypesPackage{}, fmt.Errorf("GeneratedTypes.TypeMap[%q]: %w", dbType, err)
		}
		pkg.TypeMap[dbType] = qualifiedGoType

		if strings.HasSuffix(strings.TrimSpace(dbType), "[]") {
			baseDBType := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(dbType), "[]"))
			baseMapping, ok := lookupConfiguredType(cfg.GeneratedTypes.TypeMap, baseDBType)
			if !ok {
				return generatedTypesPackage{}, fmt.Errorf("GeneratedTypes.TypeMap[%q] requires a generated base type mapping for %q", dbType, baseDBType)
			}
			_, elementGoType, err := normalizeGeneratedGoType(pkg.PackageName, baseMapping)
			if err != nil {
				return generatedTypesPackage{}, fmt.Errorf("GeneratedTypes.TypeMap[%q]: %w", baseDBType, err)
			}
			if _, ok := lookupPostgresEnumMetadata(enumMeta, baseDBType); !ok {
				return generatedTypesPackage{}, fmt.Errorf("GeneratedTypes.TypeMap[%q] requires PostgreSQL enum metadata for %q", dbType, baseDBType)
			}
			targetKey := "array:" + canonicalDBType(dbType)
			if err := registerGeneratedTypeName(seenTypeNames, goTypeName, targetKey); err != nil {
				return generatedTypesPackage{}, err
			}
			if generatedTypeExists(pkg.Arrays, goTypeName) {
				continue
			}

			pkg.Arrays = append(pkg.Arrays, generatedArrayType{
				DBType:        canonicalDBType(dbType),
				GoType:        goTypeName,
				FileName:      generatedTypeFileName(goTypeName),
				ElementGoType: elementGoType,
			})
			continue
		}

		meta, ok := lookupPostgresEnumMetadata(enumMeta, dbType)
		if !ok {
			meta, ok := lookupPostgresDomainMetadata(domainMeta, dbType)
			if !ok {
				return generatedTypesPackage{}, fmt.Errorf("GeneratedTypes.TypeMap[%q] could not be resolved to a PostgreSQL enum or domain", dbType)
			}
			targetKey := "domain:" + meta.canonicalName()
			if err := registerGeneratedTypeName(seenTypeNames, goTypeName, targetKey); err != nil {
				return generatedTypesPackage{}, err
			}

			underlyingGoType, err := resolveDomainUnderlyingGoType(meta)
			if err != nil {
				return generatedTypesPackage{}, err
			}
			imports, err := typeImportPaths(underlyingGoType)
			if err != nil {
				return generatedTypesPackage{}, err
			}

			if generatedTypeExists(pkg.Domains, goTypeName) {
				continue
			}
			pkg.Domains = append(pkg.Domains, generatedDomainType{
				DBType:           meta.canonicalName(),
				GoType:           goTypeName,
				FileName:         generatedTypeFileName(goTypeName),
				UnderlyingGoType: underlyingGoType,
				Imports:          imports,
				Constraints:      append([]string(nil), meta.ConstraintDef...),
				RegexPattern:     extractConstraintRegex(meta.ConstraintDef),
			})
			continue
		}
		targetKey := "enum:" + meta.canonicalName()
		if err := registerGeneratedTypeName(seenTypeNames, goTypeName, targetKey); err != nil {
			return generatedTypesPackage{}, err
		}
		if generatedTypeExists(pkg.Enums, goTypeName) {
			continue
		}
		pkg.Enums = append(pkg.Enums, generatedEnumType{
			DBType:    meta.canonicalName(),
			GoType:    goTypeName,
			FileName:  generatedTypeFileName(goTypeName),
			Labels:    append([]string(nil), meta.Labels...),
			Constants: buildEnumConstants(goTypeName, meta.Labels),
		})
	}

	sort.Slice(pkg.Enums, func(i, j int) bool { return pkg.Enums[i].FileName < pkg.Enums[j].FileName })
	sort.Slice(pkg.Domains, func(i, j int) bool { return pkg.Domains[i].FileName < pkg.Domains[j].FileName })
	sort.Slice(pkg.Arrays, func(i, j int) bool { return pkg.Arrays[i].FileName < pkg.Arrays[j].FileName })

	return pkg, nil
}

func resolveGeneratedTypesPackagePath(cfg config.Config) string {
	if strings.TrimSpace(cfg.GeneratedTypes.PackagePath) != "" {
		return cfg.GeneratedTypes.PackagePath
	}
	rootPackagePath := resolveOutPackagePath(cfg.OutPackagePath, cfg.OutPath)
	return path.Join(rootPackagePath, filepath.ToSlash(filepath.Clean(cfg.GeneratedTypes.RelativePath)))
}

func cloneStringMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func filterImportPathsByAlias(importPaths []string, alias, keepPath string) []string {
	out := make([]string, 0, len(importPaths))
	for _, importPath := range importPaths {
		if importPath == keepPath {
			out = append(out, importPath)
			continue
		}
		if path.Base(importPath) == alias {
			continue
		}
		out = append(out, importPath)
	}
	return out
}

func normalizeGeneratedGoType(packageName, configured string) (qualified string, name string, err error) {
	cleaned := strings.TrimSpace(configured)
	if cleaned == "" {
		return "", "", fmt.Errorf("type name must not be empty")
	}
	if strings.ContainsAny(cleaned, " *[]") {
		return "", "", fmt.Errorf("type name must be a named Go type, got %q", configured)
	}

	switch strings.Count(cleaned, ".") {
	case 0:
		name = cleaned
		qualified = packageName + "." + cleaned
	case 1:
		parts := strings.SplitN(cleaned, ".", 2)
		if parts[0] != packageName {
			return "", "", fmt.Errorf("type %q must use generated package name %q", cleaned, packageName)
		}
		name = parts[1]
		qualified = cleaned
	default:
		return "", "", fmt.Errorf("type %q must be either %q or %q", cleaned, "TypeName", packageName+".TypeName")
	}

	if !token.IsIdentifier(name) {
		return "", "", fmt.Errorf("type %q is not a valid Go identifier", name)
	}
	if !ast.IsExported(name) {
		return "", "", fmt.Errorf("type %q must be exported so generated models can reference it", name)
	}

	return qualified, name, nil
}

func registerGeneratedTypeName(seen map[string]string, goTypeName, dbType string) error {
	if existing, exists := seen[goTypeName]; exists {
		if existing == dbType {
			return nil
		}
		return fmt.Errorf("generated Go type %q is already assigned to %q and cannot also map %q", goTypeName, existing, dbType)
	}
	seen[goTypeName] = dbType
	return nil
}

func generatedTypeExists[T interface{ getGoType() string }](types []T, goTypeName string) bool {
	for _, generatedType := range types {
		if generatedType.getGoType() == goTypeName {
			return true
		}
	}
	return false
}

func lookupConfiguredType(typeMap map[string]string, dbType string) (string, bool) {
	for _, key := range postgresTypeLookupKeys(dbType) {
		if mapped, ok := typeMap[key]; ok {
			return mapped, true
		}
	}
	return "", false
}

func lookupPostgresEnumMetadata(meta map[string]postgresEnumMetadata, dbType string) (postgresEnumMetadata, bool) {
	for _, key := range postgresTypeLookupKeys(dbType) {
		if found, ok := meta[key]; ok {
			return found, true
		}
	}
	return postgresEnumMetadata{}, false
}

func lookupPostgresDomainMetadata(meta map[string]postgresDomainMetadata, dbType string) (postgresDomainMetadata, bool) {
	for _, key := range postgresTypeLookupKeys(dbType) {
		if found, ok := meta[key]; ok {
			return found, true
		}
	}
	return postgresDomainMetadata{}, false
}

func postgresTypeLookupKeys(dbType string) []string {
	cleaned := canonicalDBType(dbType)
	keys := []string{cleaned}
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		keys = append(keys, cleaned[idx+1:])
	}
	return uniqueStrings(keys)
}

func canonicalDBType(dbType string) string {
	return strings.TrimSpace(normalizeColumnType(dbType))
}

func resolveDomainUnderlyingGoType(meta postgresDomainMetadata) (string, error) {
	candidates := []string{
		meta.BaseTypeName,
		meta.BaseSchema + "." + meta.BaseTypeName,
	}
	for _, candidate := range uniqueStrings(candidates) {
		if goType, ok := pgtypes.PgTypeMap[candidate]; ok {
			return goType, nil
		}
	}
	return "", fmt.Errorf("GeneratedTypes.TypeMap[%q] uses unsupported PostgreSQL base type %q", meta.canonicalName(), meta.BaseTypeName)
}

func typeImportPaths(goType string) ([]string, error) {
	matches := qualifiedTypePattern.FindAllStringSubmatch(goType, -1)
	imports := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		importPath, ok := knownTypeImportPaths[match[1]]
		if !ok {
			return nil, fmt.Errorf("could not determine import path for generated underlying type %q", goType)
		}
		if _, exists := seen[importPath]; exists {
			continue
		}
		seen[importPath] = struct{}{}
		imports = append(imports, importPath)
	}
	sort.Strings(imports)
	return imports, nil
}

func extractConstraintRegex(constraints []string) string {
	for _, constraint := range constraints {
		match := pgConstraintRegexPattern.FindStringSubmatch(constraint)
		if len(match) == 2 {
			return match[1]
		}
	}
	return ""
}

func buildEnumConstants(goType string, labels []string) []generatedEnumConstant {
	constants := make([]generatedEnumConstant, 0, len(labels))
	usedNames := make(map[string]int, len(labels))

	for _, label := range labels {
		suffix := strcase.ToCamel(sanitizeIdentifier(label))
		if suffix == "" {
			suffix = "Value"
		}
		if suffix[0] >= '0' && suffix[0] <= '9' {
			suffix = "Value" + suffix
		}

		name := goType + suffix
		if count := usedNames[name]; count > 0 {
			name = fmt.Sprintf("%s%d", name, count+1)
		}
		usedNames[name]++

		constants = append(constants, generatedEnumConstant{
			Name:  name,
			Value: label,
		})
	}

	return constants
}

func sanitizeIdentifier(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	lastUnderscore := false
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if lastUnderscore {
				continue
			}
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

func generatedTypeFileName(goType string) string {
	return strcase.ToSnake(goType) + ".gen.go"
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
