package generator

import (
	"strings"
	"testing"

	"github.com/dan-sherwin/gormdb2struct/internal/config"
)

func TestRenderInspectionStarterConfigIncludesConnectionDetailsAndRecommendations(t *testing.T) {
	objects := []string{"tickets", "ticket_extended"}
	cfg := config.Config{
		DatabaseDialect: config.PostgreSQL,
		OutPath:         "./generated",
		OutPackagePath:  "example.com/foo/generated",
		CleanUp:         true,
		DbHost:          "invictus",
		DbPort:          5432,
		DbName:          "ticket_data_core",
		DbUser:          "ticket_management_service",
		DbPassword:      "secret",
		DbSSLMode:       false,
		GeneratedTypes: config.GeneratedTypesConfig{
			PackageName:  "types",
			RelativePath: "models/types",
		},
		Objects: &objects,
	}

	report := InspectionReport{
		Dialect: config.PostgreSQL,
		Objects: []InspectionObject{
			{Name: "tickets", Kind: "table"},
			{Name: "ticket_extended", Kind: "materialized view"},
		},
		Findings: []InspectionTypeFinding{
			{
				DBType:          "ticket_status",
				Recommendation:  InspectionRecommendationGenerate,
				SuggestedGoType: "TicketStatus",
			},
			{
				DBType:                 "spacelink_identifier",
				Recommendation:         InspectionRecommendationTypeMap,
				SuggestedGoType:        "sl_datatypes.SpacelinkIdentifier",
				SuggestedImportPath:    "go.corp.spacelink.com/sdks/go/sl_datatypes",
				SuggestedImportPackage: "sl_datatypes",
			},
			{
				DBType:          "custom_domain[]",
				Recommendation:  InspectionRecommendationManual,
				SuggestedGoType: "yourpkg.CustomDomainArray",
				Note:            "domain array wrappers are not auto-generated yet",
			},
		},
	}

	rendered := RenderInspectionStarterConfig(cfg, report)
	if !strings.Contains(rendered, "ConfigVersion = 1") {
		t.Fatalf("expected config version in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Host = \"invictus\"") || !strings.Contains(rendered, "Name = \"ticket_data_core\"") {
		t.Fatalf("expected PostgreSQL connection details in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"tickets\"") || !strings.Contains(rendered, "\"ticket_extended\"") {
		t.Fatalf("expected inspected objects in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"go.corp.spacelink.com/sdks/go/sl_datatypes\"") {
		t.Fatalf("expected imported package path in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"spacelink_identifier\" = \"sl_datatypes.SpacelinkIdentifier\"") {
		t.Fatalf("expected imported TypeMap recommendation in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "[PostgreSQL.GeneratedTypes.TypeMap]\n\"ticket_status\" = \"TicketStatus\"") {
		t.Fatalf("expected generated type recommendation in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "# \"custom_domain[]\" = \"yourpkg.CustomDomainArray\"") {
		t.Fatalf("expected manual placeholder in rendered config, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "PackageName = \"types\"") || !strings.Contains(rendered, "RelativePath = \"models/types\"") {
		t.Fatalf("expected generated type package settings in rendered config, got:\n%s", rendered)
	}
}
