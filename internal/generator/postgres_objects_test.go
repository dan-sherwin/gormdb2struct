package generator

import (
	"strings"
	"testing"
)

func TestDefaultPostgresObjectsIncludesTablesThenMaterializedViews(t *testing.T) {
	t.Parallel()

	relations := []postgresObject{
		{Name: "alpha_view", Kind: postgresObjectView},
		{Name: "alpha_table", Kind: postgresObjectTable},
		{Name: "beta_matview", Kind: postgresObjectMaterializedView},
		{Name: "beta_table", Kind: postgresObjectTable},
	}

	got := defaultPostgresObjects(relations)
	want := []postgresObject{
		{Name: "alpha_table", Kind: postgresObjectTable},
		{Name: "beta_table", Kind: postgresObjectTable},
		{Name: "beta_matview", Kind: postgresObjectMaterializedView},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d default objects, got %d: %#v", len(want), len(got), got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected default objects %v, got %v", want, got)
		}
	}
}

func TestResolveConfiguredPostgresObjectsSupportsViewsAndQualifiedPublicNames(t *testing.T) {
	t.Parallel()

	relations := []postgresObject{
		{Name: "tickets", Kind: postgresObjectTable},
		{Name: "ticket_search", Kind: postgresObjectView},
		{Name: "ticket_rollup", Kind: postgresObjectMaterializedView},
	}

	got, err := resolveConfiguredPostgresObjects(
		[]string{"public.tickets", "ticket_search", "ticket_rollup", "tickets"},
		relations,
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("resolve configured postgres objects: %v", err)
	}

	want := []postgresObject{
		{Name: "tickets", Kind: postgresObjectTable},
		{Name: "ticket_search", Kind: postgresObjectView},
		{Name: "ticket_rollup", Kind: postgresObjectMaterializedView},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d resolved objects, got %d: %#v", len(want), len(got), got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected resolved objects %v, got %v", want, got)
		}
	}
}

func TestResolveConfiguredPostgresObjectsRejectsProcedures(t *testing.T) {
	t.Parallel()

	_, err := resolveConfiguredPostgresObjects(
		[]string{"rebuild_ticket_rollup"},
		nil,
		map[string]string{"rebuild_ticket_rollup": "procedure"},
	)
	if err == nil {
		t.Fatal("expected procedure objects to be rejected")
	}
	if !strings.Contains(err.Error(), "currently supports tables, views, and materialized views only") {
		t.Fatalf("expected unsupported object kind error, got %v", err)
	}
}
