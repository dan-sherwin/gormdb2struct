package config

import (
	"strings"
	"testing"
)

func TestRenderVersionedTOMLOmitsLegacyAliasesAndImplicitDefaults(t *testing.T) {
	objects := []string{"tickets", "ticket_extended"}
	cfg := Config{
		DatabaseDialect:    PostgreSQL,
		OutPath:            "./generated",
		OutPackagePath:     "example/internal/db",
		CleanUp:            true,
		ImportPackagePaths: []string{"go.corp.spacelink.com/sdks/go/sl_datatypes"},
		Objects:            &objects,
		TypeMap: map[string]string{
			"jsonb":                "datatypes.JSON",
			"uuid":                 "datatypes.UUID",
			"spacelink_identifier": "sl_datatypes.SpacelinkIdentifier",
		},
		DbInit: GenerateDbInitConfig{
			Enabled:            false,
			IncludeAutoMigrate: false,
		},
		DbHost:     "invictus",
		DbPort:     5432,
		DbName:     "ticket_data_core",
		DbUser:     "ticket_management_service",
		DbPassword: "secret",
		DbSSLMode:  false,
	}

	rendered := RenderVersionedTOML(cfg)

	if !strings.Contains(rendered, "ConfigVersion = 1\n") {
		t.Fatalf("expected rendered config to declare version 1:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\n[Generator]\n") || !strings.Contains(rendered, "\n[Database]\n") {
		t.Fatalf("expected rendered config to include structured sections:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"spacelink_identifier\" = \"sl_datatypes.SpacelinkIdentifier\"") {
		t.Fatalf("expected rendered config to retain explicit type map entries:\n%s", rendered)
	}
	if strings.Contains(rendered, "\"jsonb\" =") || strings.Contains(rendered, "\"uuid\" =") {
		t.Fatalf("expected rendered config to omit implicit default type map entries:\n%s", rendered)
	}
	if strings.Contains(rendered, "gorm.io/datatypes") {
		t.Fatalf("expected rendered config to omit implicit datatypes import path:\n%s", rendered)
	}
	if strings.Contains(rendered, "DomainTypeMap") || strings.Contains(rendered, "DatabaseDialect") {
		t.Fatalf("expected rendered config to omit legacy keys:\n%s", rendered)
	}
}

func TestRenderVersionedTOMLIncludesGeneratedTypesWhenConfigured(t *testing.T) {
	cfg := Config{
		DatabaseDialect: PostgreSQL,
		OutPath:         "./generated",
		CleanUp:         true,
		DbHost:          "invictus",
		DbPort:          5432,
		DbName:          "billing_core",
		GeneratedTypes: GeneratedTypesConfig{
			PackageName:  "types",
			RelativePath: "models/types",
			TypeMap: map[string]string{
				"ledger_application_strategy": "LedgerApplicationStrategy",
			},
		},
	}

	rendered := RenderVersionedTOML(cfg)

	if !strings.Contains(rendered, "[PostgreSQL.GeneratedTypes]\n") {
		t.Fatalf("expected generated types section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "[PostgreSQL.GeneratedTypes.TypeMap]\n") {
		t.Fatalf("expected generated type map section:\n%s", rendered)
	}
	if !strings.Contains(rendered, "\"ledger_application_strategy\" = \"LedgerApplicationStrategy\"") {
		t.Fatalf("expected rendered generated type entry:\n%s", rendered)
	}
}
