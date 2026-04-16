package generator

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
)

type InspectionTypeKind string

const (
	InspectionTypeEnum        InspectionTypeKind = "enum"
	InspectionTypeDomain      InspectionTypeKind = "domain"
	InspectionTypeEnumArray   InspectionTypeKind = "enum_array"
	InspectionTypeDomainArray InspectionTypeKind = "domain_array"
	InspectionTypeCustom      InspectionTypeKind = "custom"
	InspectionTypeCustomArray InspectionTypeKind = "custom_array"
)

type InspectionRecommendation string

const (
	InspectionRecommendationNone     InspectionRecommendation = ""
	InspectionRecommendationTypeMap  InspectionRecommendation = "type_map"
	InspectionRecommendationGenerate InspectionRecommendation = "generate"
	InspectionRecommendationManual   InspectionRecommendation = "manual"
)

const (
	inspectionMappingTypeMap        = "TypeMap"
	inspectionMappingGeneratedTypes = "PostgreSQL.GeneratedTypes.TypeMap"
	defaultGeneratedTypesPackage    = "dbtypes"
)

type InspectionReport struct {
	Dialect  config.DatabaseDialect
	Objects  []InspectionObject
	Findings []InspectionTypeFinding
}

type InspectionObject struct {
	Name string
	Kind string
}

type InspectionTypeFinding struct {
	DBType                 string
	Kind                   InspectionTypeKind
	BaseDBType             string
	CurrentMapping         string
	MappingSource          string
	SuggestedGoType        string
	Recommendation         InspectionRecommendation
	SuggestedImportPath    string
	SuggestedImportPackage string
	DomainBaseType         string
	EnumLabels             []string
	Usages                 []InspectionColumnUsage
	Note                   string
}

type InspectionColumnUsage struct {
	ObjectName string
	ObjectKind string
	ColumnName string
}

type postgresInspectionColumnRow struct {
	ObjectName      string `gorm:"column:object_name"`
	ColumnName      string `gorm:"column:column_name"`
	ColumnType      string `gorm:"column:column_type"`
	TypeSchema      string `gorm:"column:type_schema"`
	TypeName        string `gorm:"column:type_name"`
	TypeKind        string `gorm:"column:type_kind"`
	ElementSchema   string `gorm:"column:element_schema"`
	ElementTypeName string `gorm:"column:element_type_name"`
	ElementTypeKind string `gorm:"column:element_type_kind"`
}

type postgresInspectionTypeInfo struct {
	DBType         string
	Kind           InspectionTypeKind
	BaseDBType     string
	DomainBaseType string
	EnumLabels     []string
}

// Inspect analyzes the configured database and reports type-mapping guidance.
func (s *Service) Inspect(ctx context.Context, cfg config.Config) (InspectionReport, error) {
	if err := ctx.Err(); err != nil {
		return InspectionReport{}, err
	}

	switch cfg.DatabaseDialect {
	case config.PostgreSQL:
		return s.inspectPostgres(ctx, cfg)
	case config.SQLite:
		return InspectionReport{}, fmt.Errorf("inspect currently supports postgresql only")
	default:
		return InspectionReport{}, fmt.Errorf("unsupported database dialect %q", cfg.DatabaseDialect)
	}
}

func (s *Service) inspectPostgres(ctx context.Context, cfg config.Config) (InspectionReport, error) {
	db, err := openPostgresDB(ctx, s.logger, cfg)
	if err != nil {
		return InspectionReport{}, err
	}

	objects, err := postgresObjects(db, cfg)
	if err != nil {
		return InspectionReport{}, err
	}

	columns, err := loadPostgresInspectionColumns(db, objects)
	if err != nil {
		return InspectionReport{}, err
	}

	enumMeta, err := loadPostgresEnumMetadata(db)
	if err != nil {
		return InspectionReport{}, err
	}

	domainMeta, err := loadPostgresDomainMetadata(db)
	if err != nil {
		return InspectionReport{}, err
	}

	report := buildPostgresInspectionReport(cfg, objects, columns, enumMeta, domainMeta)

	imported, err := loadInspectionImportedPackages(ctx, cfg.ImportPackagePaths)
	if err != nil {
		return InspectionReport{}, err
	}
	applyInspectionImportRecommendations(&report, imported)

	return report, nil
}

func loadPostgresInspectionColumns(db *gorm.DB, objects []postgresObject) ([]postgresInspectionColumnRow, error) {
	if len(objects) == 0 {
		return nil, nil
	}

	objectNames := make([]string, 0, len(objects))
	for _, object := range objects {
		objectNames = append(objectNames, object.Name)
	}

	var rows []postgresInspectionColumnRow
	if err := db.Raw(`
		SELECT
			c.relname AS object_name,
			a.attname AS column_name,
			format_type(a.atttypid, a.atttypmod) AS column_type,
			tns.nspname AS type_schema,
			t.typname AS type_name,
			t.typtype AS type_kind,
			etns.nspname AS element_schema,
			et.typname AS element_type_name,
			et.typtype AS element_type_kind
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace ns ON ns.oid = c.relnamespace
		JOIN pg_type t ON t.oid = a.atttypid
		JOIN pg_namespace tns ON tns.oid = t.typnamespace
		LEFT JOIN pg_type et ON et.oid = t.typelem AND t.typelem <> 0
		LEFT JOIN pg_namespace etns ON etns.oid = et.typnamespace
		WHERE ns.nspname = 'public'
		  AND c.relname IN ?
		  AND c.relkind IN ('r', 'p', 'v', 'm')
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY c.relname, a.attnum
	`, objectNames).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load PostgreSQL column type usage: %w", err)
	}

	return rows, nil
}

func buildPostgresInspectionReport(
	cfg config.Config,
	objects []postgresObject,
	columns []postgresInspectionColumnRow,
	enumMeta map[string]postgresEnumMetadata,
	domainMeta map[string]postgresDomainMetadata,
) InspectionReport {
	report := InspectionReport{
		Dialect: config.PostgreSQL,
		Objects: make([]InspectionObject, 0, len(objects)),
	}

	objectKinds := make(map[string]postgresObjectKind, len(objects))
	for _, object := range objects {
		objectKinds[object.Name] = object.Kind
		report.Objects = append(report.Objects, InspectionObject{
			Name: object.Name,
			Kind: string(object.Kind),
		})
	}

	findingsByType := make(map[string]*InspectionTypeFinding)
	for _, column := range columns {
		typeInfo, ok := classifyPostgresInspectionType(column, enumMeta, domainMeta)
		if !ok {
			continue
		}

		finding := findingsByType[typeInfo.DBType]
		if finding == nil {
			currentMapping, mappingSource := lookupInspectionMapping(cfg, typeInfo.DBType)
			finding = &InspectionTypeFinding{
				DBType:         typeInfo.DBType,
				Kind:           typeInfo.Kind,
				BaseDBType:     typeInfo.BaseDBType,
				CurrentMapping: currentMapping,
				MappingSource:  mappingSource,
				DomainBaseType: typeInfo.DomainBaseType,
				EnumLabels:     append([]string(nil), typeInfo.EnumLabels...),
			}
			findingsByType[typeInfo.DBType] = finding
		}

		finding.Usages = append(finding.Usages, InspectionColumnUsage{
			ObjectName: column.ObjectName,
			ObjectKind: string(objectKinds[column.ObjectName]),
			ColumnName: column.ColumnName,
		})
	}

	ensureInspectionBaseEnumFindings(cfg, findingsByType)
	assignInspectionRecommendations(cfg, findingsByType)

	keys := make([]string, 0, len(findingsByType))
	for dbType := range findingsByType {
		keys = append(keys, dbType)
	}
	sort.Strings(keys)

	report.Findings = make([]InspectionTypeFinding, 0, len(keys))
	for _, dbType := range keys {
		finding := findingsByType[dbType]
		sortInspectionUsages(finding.Usages)
		report.Findings = append(report.Findings, *finding)
	}

	return report
}

func classifyPostgresInspectionType(
	row postgresInspectionColumnRow,
	enumMeta map[string]postgresEnumMetadata,
	domainMeta map[string]postgresDomainMetadata,
) (postgresInspectionTypeInfo, bool) {
	switch {
	case row.TypeKind == "e":
		dbType := preferQualifiedPostgresTypeName(row.TypeSchema, row.TypeName)
		if meta, ok := lookupPostgresEnumMetadata(enumMeta, dbType); ok {
			return postgresInspectionTypeInfo{
				DBType:     meta.canonicalName(),
				Kind:       InspectionTypeEnum,
				EnumLabels: append([]string(nil), meta.Labels...),
			}, true
		}
		return postgresInspectionTypeInfo{
			DBType: dbType,
			Kind:   InspectionTypeEnum,
		}, true
	case row.TypeKind == "d":
		dbType := preferQualifiedPostgresTypeName(row.TypeSchema, row.TypeName)
		if meta, ok := lookupPostgresDomainMetadata(domainMeta, dbType); ok {
			return postgresInspectionTypeInfo{
				DBType:         meta.canonicalName(),
				Kind:           InspectionTypeDomain,
				DomainBaseType: preferQualifiedPostgresTypeName(meta.BaseSchema, meta.BaseTypeName),
			}, true
		}
		return postgresInspectionTypeInfo{
			DBType: dbType,
			Kind:   InspectionTypeDomain,
		}, true
	case row.ElementTypeKind == "e":
		baseType := preferQualifiedPostgresTypeName(row.ElementSchema, row.ElementTypeName)
		labels := []string(nil)
		if meta, ok := lookupPostgresEnumMetadata(enumMeta, baseType); ok {
			baseType = meta.canonicalName()
			labels = append(labels, meta.Labels...)
		}
		return postgresInspectionTypeInfo{
			DBType:     baseType + "[]",
			Kind:       InspectionTypeEnumArray,
			BaseDBType: baseType,
			EnumLabels: labels,
		}, true
	case row.ElementTypeKind == "d":
		baseType := preferQualifiedPostgresTypeName(row.ElementSchema, row.ElementTypeName)
		domainBaseType := ""
		if meta, ok := lookupPostgresDomainMetadata(domainMeta, baseType); ok {
			baseType = meta.canonicalName()
			domainBaseType = preferQualifiedPostgresTypeName(meta.BaseSchema, meta.BaseTypeName)
		}
		return postgresInspectionTypeInfo{
			DBType:         baseType + "[]",
			Kind:           InspectionTypeDomainArray,
			BaseDBType:     baseType,
			DomainBaseType: domainBaseType,
		}, true
	case isInspectableCustomSchema(row.TypeSchema):
		return postgresInspectionTypeInfo{
			DBType: preferQualifiedPostgresTypeName(row.TypeSchema, row.TypeName),
			Kind:   InspectionTypeCustom,
		}, true
	case isInspectableCustomSchema(row.ElementSchema):
		return postgresInspectionTypeInfo{
			DBType:     preferQualifiedPostgresTypeName(row.ElementSchema, row.ElementTypeName) + "[]",
			Kind:       InspectionTypeCustomArray,
			BaseDBType: preferQualifiedPostgresTypeName(row.ElementSchema, row.ElementTypeName),
		}, true
	default:
		return postgresInspectionTypeInfo{}, false
	}
}

func lookupInspectionMapping(cfg config.Config, dbType string) (string, string) {
	if mapped, ok := lookupConfiguredType(cfg.GeneratedTypes.TypeMap, dbType); ok {
		return mapped, inspectionMappingGeneratedTypes
	}
	if mapped, ok := lookupConfiguredType(cfg.TypeMap, dbType); ok {
		return mapped, inspectionMappingTypeMap
	}
	return "", ""
}

func assignInspectionRecommendations(cfg config.Config, findings map[string]*InspectionTypeFinding) {
	generatedPackageName := inspectionGeneratedPackageName(cfg)
	usedGeneratedNames := make(map[string]struct{}, len(cfg.GeneratedTypes.TypeMap))
	generatedBaseNames := make(map[string]string)

	for dbType, configuredGoType := range cfg.GeneratedTypes.TypeMap {
		_, name, err := normalizeGeneratedGoType(generatedPackageName, configuredGoType)
		if err != nil {
			continue
		}
		usedGeneratedNames[name] = struct{}{}
		if baseType := strings.TrimSuffix(canonicalDBType(dbType), "[]"); !strings.HasSuffix(canonicalDBType(dbType), "[]") {
			generatedBaseNames[baseType] = name
		}
	}

	keys := make([]string, 0, len(findings))
	for dbType := range findings {
		keys = append(keys, dbType)
	}
	sort.Strings(keys)

	for _, dbType := range keys {
		finding := findings[dbType]
		if finding.CurrentMapping != "" {
			continue
		}

		switch finding.Kind {
		case InspectionTypeEnum, InspectionTypeDomain:
			finding.Recommendation = InspectionRecommendationGenerate
			finding.SuggestedGoType = allocateGeneratedSuggestionName(
				suggestGeneratedTypeName(dbType, false),
				usedGeneratedNames,
			)
			generatedBaseNames[finding.DBType] = finding.SuggestedGoType
		}
	}

	for _, dbType := range keys {
		finding := findings[dbType]
		if finding.CurrentMapping != "" {
			continue
		}

		switch finding.Kind {
		case InspectionTypeEnumArray:
			if baseName, ok := generatedBaseNames[finding.BaseDBType]; ok {
				finding.Recommendation = InspectionRecommendationGenerate
				finding.SuggestedGoType = allocateGeneratedSuggestionName(baseName+"Array", usedGeneratedNames)
				continue
			}
			finding.Recommendation = InspectionRecommendationManual
			finding.SuggestedGoType = suggestManualArrayGoType(cfg, finding.BaseDBType)
			finding.Note = "enum arrays can only be auto-generated when the base enum is generated in PostgreSQL.GeneratedTypes.TypeMap"
		case InspectionTypeDomainArray:
			finding.Recommendation = InspectionRecommendationManual
			finding.SuggestedGoType = manualPlaceholderGoType(finding)
			finding.Note = "domain array wrappers are not auto-generated yet"
		case InspectionTypeCustom, InspectionTypeCustomArray:
			finding.Recommendation = InspectionRecommendationManual
			finding.SuggestedGoType = manualPlaceholderGoType(finding)
		}
	}
}

func ensureInspectionBaseEnumFindings(cfg config.Config, findings map[string]*InspectionTypeFinding) {
	for _, finding := range findings {
		if finding.Kind != InspectionTypeEnumArray {
			continue
		}
		if _, exists := findings[finding.BaseDBType]; exists {
			continue
		}

		currentMapping, mappingSource := lookupInspectionMapping(cfg, finding.BaseDBType)
		findings[finding.BaseDBType] = &InspectionTypeFinding{
			DBType:         finding.BaseDBType,
			Kind:           InspectionTypeEnum,
			CurrentMapping: currentMapping,
			MappingSource:  mappingSource,
			EnumLabels:     append([]string(nil), finding.EnumLabels...),
			Note:           fmt.Sprintf("required to generate %s", finding.DBType),
		}
	}
}

// RenderInspectionReport renders the inspection report in the requested format.
func RenderInspectionReport(report InspectionReport, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return renderInspectionText(report), nil
	case "toml":
		return renderInspectionTOML(report), nil
	default:
		return "", fmt.Errorf("unsupported inspection format %q", format)
	}
}

func renderInspectionText(report InspectionReport) string {
	var buf bytes.Buffer

	mapped := make([]InspectionTypeFinding, 0)
	typeMapped := make([]InspectionTypeFinding, 0)
	generated := make([]InspectionTypeFinding, 0)
	manual := make([]InspectionTypeFinding, 0)
	for _, finding := range report.Findings {
		switch {
		case finding.CurrentMapping != "":
			mapped = append(mapped, finding)
		case finding.Recommendation == InspectionRecommendationTypeMap:
			typeMapped = append(typeMapped, finding)
		case finding.Recommendation == InspectionRecommendationGenerate:
			generated = append(generated, finding)
		case finding.Recommendation == InspectionRecommendationManual:
			manual = append(manual, finding)
		}
	}

	buf.WriteString("PostgreSQL inspection report\n\n")
	_, _ = fmt.Fprintf(&buf,
		"Objects scanned (%d): %s\n",
		len(report.Objects),
		formatInspectionObjects(report.Objects),
	)
	_, _ = fmt.Fprintf(&buf,
		"Findings: %d custom types (%d mapped, %d recommended TypeMap, %d recommended generated, %d manual review)\n",
		len(report.Findings),
		len(mapped),
		len(typeMapped),
		len(generated),
		len(manual),
	)

	if len(report.Findings) == 0 {
		buf.WriteString("\nNo PostgreSQL enums, domains, or other custom types were found in the selected objects.\n")
		return buf.String()
	}

	if len(mapped) > 0 {
		buf.WriteString("\nAlready mapped:\n")
		for _, finding := range mapped {
			_, _ = fmt.Fprintf(&buf,
				"- %s (%s) -> %s [%s]\n",
				finding.DBType,
				describeInspectionFinding(finding),
				finding.CurrentMapping,
				finding.MappingSource,
			)
			if len(finding.Usages) > 0 {
				_, _ = fmt.Fprintf(&buf, "  Used by: %s\n", formatInspectionUsages(finding.Usages))
			}
			if finding.Note != "" {
				_, _ = fmt.Fprintf(&buf, "  Note: %s\n", finding.Note)
			}
		}
	}

	if len(typeMapped) > 0 {
		buf.WriteString("\nRecommended TypeMap mappings:\n")
		for _, finding := range typeMapped {
			_, _ = fmt.Fprintf(&buf,
				"- %s (%s) -> %s\n",
				finding.DBType,
				describeInspectionFinding(finding),
				finding.SuggestedGoType,
			)
			if finding.SuggestedImportPath != "" {
				_, _ = fmt.Fprintf(&buf, "  Import package: %s\n", finding.SuggestedImportPath)
			}
			if len(finding.Usages) > 0 {
				_, _ = fmt.Fprintf(&buf, "  Used by: %s\n", formatInspectionUsages(finding.Usages))
			}
			if finding.Note != "" {
				_, _ = fmt.Fprintf(&buf, "  Note: %s\n", finding.Note)
			}
		}
	}

	if len(generated) > 0 {
		buf.WriteString("\nRecommended generated types:\n")
		for _, finding := range generated {
			_, _ = fmt.Fprintf(&buf,
				"- %s (%s) -> %s\n",
				finding.DBType,
				describeInspectionFinding(finding),
				finding.SuggestedGoType,
			)
			if len(finding.Usages) > 0 {
				_, _ = fmt.Fprintf(&buf, "  Used by: %s\n", formatInspectionUsages(finding.Usages))
			}
			if finding.Note != "" {
				_, _ = fmt.Fprintf(&buf, "  Note: %s\n", finding.Note)
			}
		}
	}

	if len(manual) > 0 {
		buf.WriteString("\nManual review:\n")
		for _, finding := range manual {
			_, _ = fmt.Fprintf(&buf,
				"- %s (%s) -> %s\n",
				finding.DBType,
				describeInspectionFinding(finding),
				finding.SuggestedGoType,
			)
			if len(finding.Usages) > 0 {
				_, _ = fmt.Fprintf(&buf, "  Used by: %s\n", formatInspectionUsages(finding.Usages))
			}
			if finding.Note != "" {
				_, _ = fmt.Fprintf(&buf, "  Note: %s\n", finding.Note)
			}
		}
	}

	buf.WriteString("\nRecommended config snippet:\n")
	buf.WriteString(renderInspectionTOML(report))

	return buf.String()
}

func renderInspectionTOML(report InspectionReport) string {
	var buf bytes.Buffer

	generated := make([]InspectionTypeFinding, 0)
	typeMapped := make([]InspectionTypeFinding, 0)
	manual := make([]InspectionTypeFinding, 0)
	for _, finding := range report.Findings {
		switch finding.Recommendation {
		case InspectionRecommendationTypeMap:
			typeMapped = append(typeMapped, finding)
		case InspectionRecommendationGenerate:
			generated = append(generated, finding)
		case InspectionRecommendationManual:
			manual = append(manual, finding)
		}
	}

	if len(typeMapped) == 0 && len(generated) == 0 && len(manual) == 0 {
		buf.WriteString("# No additional type mappings recommended.\n")
		return buf.String()
	}

	if len(typeMapped) > 0 || len(manual) > 0 {
		importPaths := inspectionRecommendedImportPaths(typeMapped)
		if len(importPaths) > 0 {
			buf.WriteString("# Add these import paths under [Generator].ImportPackagePaths:\n")
			for _, importPath := range importPaths {
				_, _ = fmt.Fprintf(&buf, "# %q\n", importPath)
			}
		}
		buf.WriteString("[TypeMap]\n")
		for _, finding := range typeMapped {
			_, _ = fmt.Fprintf(&buf, "%q = %q\n", finding.DBType, finding.SuggestedGoType)
		}
		for _, finding := range manual {
			line := fmt.Sprintf("# %q = %q", finding.DBType, finding.SuggestedGoType)
			if finding.Note != "" {
				line += "  # " + finding.Note
			}
			buf.WriteString(line + "\n")
		}
	}

	if len(generated) > 0 {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("[PostgreSQL.GeneratedTypes.TypeMap]\n")
		for _, finding := range generated {
			_, _ = fmt.Fprintf(&buf, "%q = %q\n", finding.DBType, finding.SuggestedGoType)
		}
	}

	return buf.String()
}

func inspectionGeneratedPackageName(cfg config.Config) string {
	if strings.TrimSpace(cfg.GeneratedTypes.PackageName) != "" {
		return cfg.GeneratedTypes.PackageName
	}
	return defaultGeneratedTypesPackage
}

func isInspectableCustomSchema(schemaName string) bool {
	schemaName = strings.TrimSpace(schemaName)
	return schemaName != "" && schemaName != "pg_catalog" && schemaName != "information_schema"
}

func preferQualifiedPostgresTypeName(schemaName, typeName string) string {
	cleanType := strings.TrimSpace(typeName)
	if cleanType == "" {
		return ""
	}
	cleanSchema := strings.TrimSpace(schemaName)
	if cleanSchema == "" || cleanSchema == "public" {
		return cleanType
	}
	return cleanSchema + "." + cleanType
}

func suggestGeneratedTypeName(dbType string, array bool) string {
	baseType := canonicalDBType(dbType)
	if array {
		baseType = strings.TrimSuffix(baseType, "[]")
	}
	if idx := strings.LastIndex(baseType, "."); idx != -1 {
		baseType = baseType[idx+1:]
	}

	name := strcase.ToCamel(sanitizeIdentifier(baseType))
	if name == "" {
		name = "CustomType"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "Type" + name
	}
	if array {
		name += "Array"
	}
	return name
}

func allocateGeneratedSuggestionName(base string, used map[string]struct{}) string {
	if _, exists := used[base]; !exists {
		used[base] = struct{}{}
		return base
	}

	for idx := 2; ; idx++ {
		candidate := fmt.Sprintf("%s%d", base, idx)
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

func suggestManualArrayGoType(cfg config.Config, baseDBType string) string {
	if mapped, ok := lookupConfiguredType(cfg.TypeMap, baseDBType); ok {
		return appendArraySuffixToGoType(mapped)
	}
	return "yourpkg." + suggestGeneratedTypeName(baseDBType, true)
}

func appendArraySuffixToGoType(goType string) string {
	cleaned := strings.TrimSpace(goType)
	if cleaned == "" {
		return "yourpkg.CustomTypeArray"
	}
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		return cleaned[:idx+1] + cleaned[idx+1:] + "Array"
	}
	return cleaned + "Array"
}

func manualPlaceholderGoType(finding *InspectionTypeFinding) string {
	switch finding.Kind {
	case InspectionTypeDomainArray, InspectionTypeCustomArray:
		return "yourpkg." + suggestGeneratedTypeName(finding.DBType, true)
	default:
		return "yourpkg." + suggestGeneratedTypeName(finding.DBType, false)
	}
}

func sortInspectionUsages(usages []InspectionColumnUsage) {
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].ObjectName != usages[j].ObjectName {
			return usages[i].ObjectName < usages[j].ObjectName
		}
		return usages[i].ColumnName < usages[j].ColumnName
	})
}

func formatInspectionObjects(objects []InspectionObject) string {
	if len(objects) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(objects))
	for _, object := range objects {
		parts = append(parts, fmt.Sprintf("%s (%s)", object.Name, object.Kind))
	}
	return strings.Join(parts, ", ")
}

func formatInspectionUsages(usages []InspectionColumnUsage) string {
	if len(usages) == 0 {
		return "none"
	}

	seen := make(map[string]struct{}, len(usages))
	parts := make([]string, 0, len(usages))
	for _, usage := range usages {
		key := usage.ObjectName + "." + usage.ColumnName
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key)
	}
	return strings.Join(parts, ", ")
}

func describeInspectionFinding(finding InspectionTypeFinding) string {
	switch finding.Kind {
	case InspectionTypeEnum:
		return "enum"
	case InspectionTypeDomain:
		if finding.DomainBaseType != "" {
			return "domain over " + finding.DomainBaseType
		}
		return "domain"
	case InspectionTypeEnumArray:
		return "enum array"
	case InspectionTypeDomainArray:
		if finding.DomainBaseType != "" {
			return "domain array over " + finding.DomainBaseType
		}
		return "domain array"
	case InspectionTypeCustom:
		return "custom type"
	case InspectionTypeCustomArray:
		return "custom array"
	default:
		return string(finding.Kind)
	}
}
