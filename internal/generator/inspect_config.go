package generator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
)

// RenderInspectionStarterConfig renders a versioned TOML starter config using
// the provided PostgreSQL connection settings and inspection recommendations.
func RenderInspectionStarterConfig(cfg config.Config, report InspectionReport) string {
	var builder strings.Builder

	builder.WriteString("# gormdb2struct configuration\n")
	builder.WriteString("ConfigVersion = 1\n\n")

	builder.WriteString("# ----------------------------------------------------------------------\n")
	builder.WriteString("# Generator\n")
	builder.WriteString("# ----------------------------------------------------------------------\n\n")
	builder.WriteString("[Generator]\n")
	_, _ = fmt.Fprintf(&builder, "OutPath = %s\n", strconv.Quote(cfg.OutPath))
	_, _ = fmt.Fprintf(&builder, "OutPackagePath = %s\n", strconv.Quote(cfg.OutPackagePath))
	_, _ = fmt.Fprintf(&builder, "CleanUp = %t\n", cfg.CleanUp)
	builder.WriteString("ImportPackagePaths = [\n")
	for _, importPath := range inspectionStarterImportPaths(report.Findings) {
		_, _ = fmt.Fprintf(&builder, "  %q,\n", importPath)
	}
	builder.WriteString("]\n")
	if len(report.Objects) > 0 {
		builder.WriteString("Objects = [\n")
		for _, object := range report.Objects {
			_, _ = fmt.Fprintf(&builder, "  %s,\n", strconv.Quote(object.Name))
		}
		builder.WriteString("]\n")
	} else {
		builder.WriteString("# Objects = [\"tickets\", \"ticket_rollup\"]\n")
	}

	builder.WriteString("\n# ----------------------------------------------------------------------\n")
	builder.WriteString("# Database\n")
	builder.WriteString("# Keep only the database subsection that matches Database.Dialect.\n")
	builder.WriteString("# ----------------------------------------------------------------------\n\n")
	builder.WriteString("[Database]\n")
	builder.WriteString("Dialect = \"postgresql\"\n\n")
	builder.WriteString("[Database.PostgreSQL]\n")
	_, _ = fmt.Fprintf(&builder, "Host = %s\n", strconv.Quote(cfg.DbHost))
	_, _ = fmt.Fprintf(&builder, "Port = %d\n", cfg.DbPort)
	_, _ = fmt.Fprintf(&builder, "Name = %s\n", strconv.Quote(cfg.DbName))
	_, _ = fmt.Fprintf(&builder, "User = %s\n", strconv.Quote(cfg.DbUser))
	_, _ = fmt.Fprintf(&builder, "Password = %s\n", strconv.Quote(cfg.DbPassword))
	_, _ = fmt.Fprintf(&builder, "SSLMode = %t\n\n", cfg.DbSSLMode)
	builder.WriteString("[Database.SQLite]\n")
	builder.WriteString("# Path = \"./schema.db\"\n")

	builder.WriteString("\n# ----------------------------------------------------------------------\n")
	builder.WriteString("# Optional generation sections\n")
	builder.WriteString("# ----------------------------------------------------------------------\n\n")
	builder.WriteString("[DbInit]\n")
	builder.WriteString("Enabled = true\n")
	builder.WriteString("IncludeAutoMigrate = false\n")
	builder.WriteString("GenerateAppSettingsRegistration = false\n")
	builder.WriteString("UseSlogGormLogger = false\n\n")

	builder.WriteString("# TypeMap: shared database type overrides (optional).\n")
	builder.WriteString("# PostgreSQL: standard types, enums, domains, arrays.\n")
	builder.WriteString("# SQLite: declared column types.\n")
	builder.WriteString("[TypeMap]\n")
	typeMappedFindings := inspectionTypeMappedFindings(report.Findings)
	manualFindings := inspectionManualFindings(report.Findings)
	if len(typeMappedFindings) == 0 && len(manualFindings) == 0 {
		builder.WriteString("# \"jsonb\" = \"datatypes.JSON\"\n")
		builder.WriteString("# \"uuid\" = \"datatypes.UUID\"\n")
		builder.WriteString("# \"my_text_domain\" = \"string\"\n")
	} else {
		for _, finding := range typeMappedFindings {
			_, _ = fmt.Fprintf(&builder, "%q = %q\n", finding.DBType, finding.SuggestedGoType)
		}
		for _, finding := range manualFindings {
			line := fmt.Sprintf("# %q = %q", finding.DBType, finding.SuggestedGoType)
			if finding.Note != "" {
				line += "  # " + finding.Note
			}
			builder.WriteString(line + "\n")
		}
	}

	builder.WriteString("\n# ExtraFields: add relation fields to specific models (optional)\n")
	builder.WriteString("[ExtraFields]\n")
	builder.WriteString("# [[ExtraFields.\"ticket_extended\"]]\n")
	builder.WriteString("# StructPropName = \"Attachments\"\n")
	builder.WriteString("# StructPropType = \"models.Attachment\"\n")
	builder.WriteString("# FkStructPropName = \"TicketID\"\n")
	builder.WriteString("# RefStructPropName = \"TicketID\"\n")
	builder.WriteString("# HasMany = true\n")
	builder.WriteString("# Pointer = true\n\n")

	builder.WriteString("# JSONTagOverridesByTable: override json tags for fields (optional)\n")
	builder.WriteString("[JSONTagOverridesByTable]\n")
	builder.WriteString("# [JSONTagOverridesByTable.\"ticket_extended\"]\n")
	builder.WriteString("# subject_fts = \"-\"\n")

	builder.WriteString("\n# ----------------------------------------------------------------------\n")
	builder.WriteString("# PostgreSQL-only sections\n")
	builder.WriteString("# ----------------------------------------------------------------------\n\n")
	builder.WriteString("[PostgreSQL.GeneratedTypes]\n")
	_, _ = fmt.Fprintf(&builder, "PackageName = %s\n", strconv.Quote(inspectionGeneratedPackageName(cfg)))
	_, _ = fmt.Fprintf(&builder, "RelativePath = %s\n", strconv.Quote(inspectionGeneratedTypesRelativePath(cfg)))
	builder.WriteString("PackagePath = \"\"\n\n")

	builder.WriteString("[PostgreSQL.GeneratedTypes.TypeMap]\n")
	generatedFindings := inspectionGeneratedFindings(report.Findings)
	if len(generatedFindings) == 0 {
		builder.WriteString("# \"ticket_status\" = \"TicketStatus\"\n")
		builder.WriteString("# \"ticket_type\" = \"TicketType\"\n")
		builder.WriteString("# \"ticket_type[]\" = \"TicketTypeArray\"\n")
		builder.WriteString("# \"my_text_domain\" = \"MyTextDomain\"\n")
	} else {
		for _, finding := range generatedFindings {
			_, _ = fmt.Fprintf(&builder, "%q = %q\n", finding.DBType, finding.SuggestedGoType)
		}
	}

	return builder.String()
}

func inspectionGeneratedTypesRelativePath(cfg config.Config) string {
	if strings.TrimSpace(cfg.GeneratedTypes.RelativePath) != "" {
		return cfg.GeneratedTypes.RelativePath
	}
	return "models/" + inspectionGeneratedPackageName(cfg)
}

func inspectionGeneratedFindings(findings []InspectionTypeFinding) []InspectionTypeFinding {
	out := make([]InspectionTypeFinding, 0)
	for _, finding := range findings {
		if finding.Recommendation != InspectionRecommendationGenerate {
			continue
		}
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DBType < out[j].DBType })
	return out
}

func inspectionTypeMappedFindings(findings []InspectionTypeFinding) []InspectionTypeFinding {
	out := make([]InspectionTypeFinding, 0)
	for _, finding := range findings {
		if finding.Recommendation != InspectionRecommendationTypeMap {
			continue
		}
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DBType < out[j].DBType })
	return out
}

func inspectionManualFindings(findings []InspectionTypeFinding) []InspectionTypeFinding {
	out := make([]InspectionTypeFinding, 0)
	for _, finding := range findings {
		if finding.Recommendation != InspectionRecommendationManual {
			continue
		}
		out = append(out, finding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DBType < out[j].DBType })
	return out
}

func inspectionRecommendedImportPaths(typeMapped []InspectionTypeFinding) []string {
	seen := map[string]struct{}{
		"github.com/dan-sherwin/gormdb2struct/pgtypes": {},
	}
	out := []string{"github.com/dan-sherwin/gormdb2struct/pgtypes"}
	for _, finding := range typeMapped {
		cleaned := strings.TrimSpace(finding.SuggestedImportPath)
		if cleaned == "" {
			continue
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

func inspectionStarterImportPaths(findings []InspectionTypeFinding) []string {
	typeMapped := inspectionTypeMappedFindings(findings)
	if len(typeMapped) == 0 {
		return []string{"github.com/dan-sherwin/gormdb2struct/pgtypes"}
	}
	return inspectionRecommendedImportPaths(typeMapped)
}
