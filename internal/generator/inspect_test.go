package generator

import (
	"context"
	"strings"
	"testing"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
)

func TestBuildPostgresInspectionReportRecommendsGeneratedTypes(t *testing.T) {
	cfg := config.Config{
		DatabaseDialect: config.PostgreSQL,
		TypeMap:         map[string]string{},
		GeneratedTypes: config.GeneratedTypesConfig{
			TypeMap: map[string]string{},
		},
	}
	cfg.Normalize()

	report := buildPostgresInspectionReport(
		cfg,
		[]postgresObject{{Name: "tickets", Kind: postgresObjectTable}},
		[]postgresInspectionColumnRow{
			{
				ObjectName: "tickets",
				ColumnName: "status",
				ColumnType: "ticket_status",
				TypeSchema: "public",
				TypeName:   "ticket_status",
				TypeKind:   "e",
			},
			{
				ObjectName: "tickets",
				ColumnName: "tenant_si",
				ColumnType: "spacelink_identifier",
				TypeSchema: "public",
				TypeName:   "spacelink_identifier",
				TypeKind:   "d",
			},
			{
				ObjectName:      "tickets",
				ColumnName:      "allowed_types",
				ColumnType:      "ticket_type[]",
				TypeSchema:      "pg_catalog",
				TypeName:        "_ticket_type",
				TypeKind:        "b",
				ElementSchema:   "public",
				ElementTypeName: "ticket_type",
				ElementTypeKind: "e",
			},
		},
		map[string]postgresEnumMetadata{
			"ticket_status": {SchemaName: "public", TypeName: "ticket_status", Labels: []string{"new", "closed"}},
			"ticket_type":   {SchemaName: "public", TypeName: "ticket_type", Labels: []string{"incident", "request"}},
		},
		map[string]postgresDomainMetadata{
			"spacelink_identifier": {SchemaName: "public", DomainName: "spacelink_identifier", BaseSchema: "pg_catalog", BaseTypeName: "text"},
		},
	)

	status := findInspectionFinding(t, report.Findings, "ticket_status")
	if status.Recommendation != InspectionRecommendationGenerate || status.SuggestedGoType != "TicketStatus" {
		t.Fatalf("expected ticket_status to recommend generated TicketStatus, got %#v", status)
	}

	domain := findInspectionFinding(t, report.Findings, "spacelink_identifier")
	if domain.Recommendation != InspectionRecommendationGenerate || domain.SuggestedGoType != "SpacelinkIdentifier" {
		t.Fatalf("expected spacelink_identifier to recommend generated SpacelinkIdentifier, got %#v", domain)
	}

	enumArray := findInspectionFinding(t, report.Findings, "ticket_type[]")
	if enumArray.Recommendation != InspectionRecommendationGenerate || enumArray.SuggestedGoType != "TicketTypeArray" {
		t.Fatalf("expected ticket_type[] to recommend generated TicketTypeArray, got %#v", enumArray)
	}

	rendered, err := RenderInspectionReport(report, "toml")
	if err != nil {
		t.Fatalf("render toml inspection report: %v", err)
	}
	if !strings.Contains(rendered, "[PostgreSQL.GeneratedTypes.TypeMap]") {
		t.Fatalf("expected generated types section in rendered output, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"ticket_status\" = \"TicketStatus\"") {
		t.Fatalf("expected ticket_status recommendation in rendered output, got:\n%s", rendered)
	}
}

func TestBuildPostgresInspectionReportMarksEnumArrayManualWhenBaseEnumIsManual(t *testing.T) {
	cfg := config.Config{
		DatabaseDialect: config.PostgreSQL,
		TypeMap: map[string]string{
			"ticket_type": "mytypes.TicketType",
		},
		GeneratedTypes: config.GeneratedTypesConfig{
			TypeMap: map[string]string{},
		},
	}
	cfg.Normalize()

	report := buildPostgresInspectionReport(
		cfg,
		[]postgresObject{{Name: "tickets", Kind: postgresObjectTable}},
		[]postgresInspectionColumnRow{
			{
				ObjectName:      "tickets",
				ColumnName:      "allowed_types",
				ColumnType:      "ticket_type[]",
				TypeSchema:      "pg_catalog",
				TypeName:        "_ticket_type",
				TypeKind:        "b",
				ElementSchema:   "public",
				ElementTypeName: "ticket_type",
				ElementTypeKind: "e",
			},
		},
		map[string]postgresEnumMetadata{
			"ticket_type": {SchemaName: "public", TypeName: "ticket_type", Labels: []string{"incident", "request"}},
		},
		nil,
	)

	enumArray := findInspectionFinding(t, report.Findings, "ticket_type[]")
	if enumArray.Recommendation != InspectionRecommendationManual {
		t.Fatalf("expected ticket_type[] to require manual review, got %#v", enumArray)
	}
	if enumArray.SuggestedGoType != "mytypes.TicketTypeArray" {
		t.Fatalf("expected manual array suggestion to reuse mapped package, got %#v", enumArray)
	}
	if !strings.Contains(enumArray.Note, "base enum is generated") {
		t.Fatalf("expected manual enum array note, got %#v", enumArray)
	}
}

func TestApplyInspectionImportRecommendationsPrefersImportedTypeMap(t *testing.T) {
	report := InspectionReport{
		Findings: []InspectionTypeFinding{
			{
				DBType:          "spacelink_identifier",
				Kind:            InspectionTypeDomain,
				SuggestedGoType: "SpacelinkIdentifier",
				Recommendation:  InspectionRecommendationGenerate,
			},
			{
				DBType:          "ticket_type[]",
				Kind:            InspectionTypeEnumArray,
				SuggestedGoType: "TicketTypeArray",
				Recommendation:  InspectionRecommendationGenerate,
			},
		},
	}

	applyInspectionImportRecommendations(&report, []inspectionImportedPackage{
		{
			ImportPath:  "go.corp.spacelink.com/sdks/go/sl_datatypes",
			PackageName: "sl_datatypes",
			ExportedTypes: map[string]struct{}{
				"SpacelinkIdentifier": {},
				"TicketTypeArray":     {},
			},
		},
	})

	domain := report.Findings[0]
	if domain.Recommendation != InspectionRecommendationTypeMap {
		t.Fatalf("expected imported TypeMap recommendation, got %#v", domain)
	}
	if domain.SuggestedGoType != "sl_datatypes.SpacelinkIdentifier" {
		t.Fatalf("unexpected imported Go type suggestion: %#v", domain)
	}
	if domain.SuggestedImportPath != "go.corp.spacelink.com/sdks/go/sl_datatypes" {
		t.Fatalf("unexpected import path suggestion: %#v", domain)
	}

	array := report.Findings[1]
	if array.Recommendation != InspectionRecommendationTypeMap {
		t.Fatalf("expected imported TypeMap recommendation for array, got %#v", array)
	}
	if array.SuggestedGoType != "sl_datatypes.TicketTypeArray" {
		t.Fatalf("unexpected imported array Go type suggestion: %#v", array)
	}
}

func TestLoadInspectionImportedPackagesFindsExportedTypes(t *testing.T) {
	imported, err := loadInspectionImportedPackages(context.Background(), []string{
		"github.com/dan-sherwin/gormdb2struct/internal/testfixtures/sl_datatypes",
	})
	if err != nil {
		t.Fatalf("load inspection imported packages: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected one imported package, got %#v", imported)
	}
	if imported[0].PackageName != "sl_datatypes" {
		t.Fatalf("unexpected package name: %#v", imported[0])
	}
	if _, exists := imported[0].ExportedTypes["SpacelinkIdentifier"]; !exists {
		t.Fatalf("expected exported SpacelinkIdentifier type, got %#v", imported[0].ExportedTypes)
	}
	if _, exists := imported[0].ExportedTypes["TicketTypeArray"]; !exists {
		t.Fatalf("expected exported TicketTypeArray type, got %#v", imported[0].ExportedTypes)
	}
}

func TestLoadInspectionImportedPackagesCanDriveTypeMapRecommendations(t *testing.T) {
	report := InspectionReport{
		Findings: []InspectionTypeFinding{
			{
				DBType:          "spacelink_identifier",
				Kind:            InspectionTypeDomain,
				SuggestedGoType: "SpacelinkIdentifier",
				Recommendation:  InspectionRecommendationGenerate,
			},
		},
	}

	imported, err := loadInspectionImportedPackages(context.Background(), []string{
		"github.com/dan-sherwin/gormdb2struct/internal/testfixtures/sl_datatypes",
	})
	if err != nil {
		t.Fatalf("load inspection imported packages: %v", err)
	}

	applyInspectionImportRecommendations(&report, imported)
	if report.Findings[0].Recommendation != InspectionRecommendationTypeMap {
		t.Fatalf("expected imported TypeMap recommendation, got %#v", report.Findings[0])
	}
	if report.Findings[0].SuggestedGoType != "sl_datatypes.SpacelinkIdentifier" {
		t.Fatalf("unexpected imported Go type suggestion: %#v", report.Findings[0])
	}
}

func findInspectionFinding(t *testing.T, findings []InspectionTypeFinding, dbType string) InspectionTypeFinding {
	t.Helper()

	for _, finding := range findings {
		if finding.DBType == dbType {
			return finding
		}
	}

	t.Fatalf("could not find inspection finding for %q in %#v", dbType, findings)
	return InspectionTypeFinding{}
}
