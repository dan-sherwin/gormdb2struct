// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestStringArray_ScanAndValue(t *testing.T) {
	var a StringArray
	if err := a.Scan([]byte(`{"a","b","c"}`)); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if got, want := []string(a), []string{"a", "b", "c"}; len(got) != len(want) || got[1] != want[1] {
		t.Fatalf("unexpected scan result: %v", a)
	}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("value failed: %v", err)
	}
	if v == nil {
		t.Fatalf("value should not be nil")
	}
	if vs, ok := v.(string); !ok || vs != `{"a","b","c"}` {
		t.Fatalf("unexpected value: %v", v)
	}
}

func TestStringArray_JSON(t *testing.T) {
	a := StringArray{"x", "y"}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `["x","y"]` {
		t.Fatalf("unexpected json: %s", string(b))
	}
	var out StringArray
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Equals(a) {
		t.Fatalf("roundtrip mismatch: %v vs %v", out, a)
	}
}

func TestInt64Array_ScanAndValue_EmptyAndNil(t *testing.T) {
	var a Int64Array
	// nil from DB
	if err := a.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if a != nil {
		t.Fatalf("expected nil slice on nil scan, got %v", a)
	}
	// empty array text
	if err := a.Scan("{}"); err != nil {
		t.Fatalf("scan empty: %v", err)
	}
	if a == nil || len(a) != 0 {
		t.Fatalf("expected empty slice, got %v", a)
	}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if vs, ok := v.(string); !ok || vs != "{}" {
		t.Fatalf("unexpected value: %v", v)
	}
}

func TestUUIDArray_ScanAndValue(t *testing.T) {
	u1 := uuid.New()
	u2 := uuid.New()
	input := "{\"" + u1.String() + "\",\"" + u2.String() + "\"}"
	var a UUIDArray
	if err := a.Scan(input); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(a) != 2 || a[0] != u1 || a[1] != u2 {
		t.Fatalf("unexpected content: %v", a)
	}
	v, err := a.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if vs, ok := v.(string); !ok || vs != input {
		t.Fatalf("unexpected value: %v", v)
	}
}
