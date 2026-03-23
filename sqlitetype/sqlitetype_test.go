// Package sqlitetype provides SQLite-specific type mapping and utility functions for GORM.
package sqlitetype

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type fakeColumnType struct {
	nullable bool
	colType  string
}

func (f fakeColumnType) Name() string                      { return "" }
func (f fakeColumnType) DatabaseTypeName() string          { return "" }
func (f fakeColumnType) ColumnType() (string, bool)        { return f.colType, true }
func (f fakeColumnType) PrimaryKey() (bool, bool)          { return false, false }
func (f fakeColumnType) AutoIncrement() (bool, bool)       { return false, false }
func (f fakeColumnType) Length() (int64, bool)             { return 0, false }
func (f fakeColumnType) DecimalSize() (int64, int64, bool) { return 0, 0, false }
func (f fakeColumnType) Nullable() (bool, bool)            { return f.nullable, true }
func (f fakeColumnType) ScanType() reflect.Type            { return reflect.TypeOf(int(0)) }
func (f fakeColumnType) Unique() (bool, bool)              { return false, false }
func (f fakeColumnType) Comment() (string, bool)           { return "", false }
func (f fakeColumnType) DefaultValue() (string, bool)      { return "", false }

func TestNullablePtr(t *testing.T) {
	// when nullable, known scalar types should be pointer
	if got := nullablePtr(true, "bool"); got != "*bool" {
		t.Fatalf("expected *bool, got %s", got)
	}
	if got := nullablePtr(true, "time.Time"); got != "*time.Time" {
		t.Fatalf("expected *time.Time, got %s", got)
	}
	// when not nullable, keep base
	if got := nullablePtr(false, "int64"); got != "int64" {
		t.Fatalf("expected int64, got %s", got)
	}
	// types not in the list should remain unchanged
	if got := nullablePtr(true, "[]byte"); got != "[]byte" {
		t.Fatalf("expected []byte unchanged, got %s", got)
	}
}

func TestTypeMap_BooleansAndTinyInt(t *testing.T) {
	// BOOLEAN
	if fn, ok := TypeMap["BOOLEAN"]; !ok {
		t.Fatalf("BOOLEAN mapping missing")
	} else {
		got := fn(fakeColumnType{nullable: true})
		if got != "*bool" {
			t.Fatalf("BOOLEAN nullable -> *bool, got %s", got)
		}
		got = fn(fakeColumnType{nullable: false})
		if got != "bool" {
			t.Fatalf("BOOLEAN not nullable -> bool, got %s", got)
		}
	}
	// TINYINT(1) should be bool
	fn := TypeMap["TINYINT"]
	got := fn(fakeColumnType{nullable: true, colType: "TINYINT(1)"})
	if got != "*bool" {
		t.Fatalf("TINYINT(1) nullable -> *bool, got %s", got)
	}
	// other TINYINT should be int8
	got = fn(fakeColumnType{nullable: false, colType: "TINYINT(2)"})
	if got != "int8" {
		t.Fatalf("TINYINT(2) -> int8, got %s", got)
	}
	// case-insensitive prefix
	got = fn(fakeColumnType{nullable: false, colType: strings.ToLower("tinyint(1)")})
	if got != "bool" {
		t.Fatalf("tinyint(1) -> bool, got %s", got)
	}
}

func TestTypeMap_IntegerAndTextAndDate(t *testing.T) {
	if fn := TypeMap["INTEGER"]; fn == nil {
		t.Fatal("INTEGER mapping missing")
	} else {
		if got := fn(fakeColumnType{nullable: true}); got != "*int64" {
			t.Fatalf("INTEGER nullable -> *int64, got %s", got)
		}
		if got := fn(fakeColumnType{nullable: false}); got != "int64" {
			t.Fatalf("INTEGER -> int64, got %s", got)
		}
	}
	if fn := TypeMap["TEXT"]; fn == nil {
		t.Fatal("TEXT mapping missing")
	} else {
		if got := fn(fakeColumnType{nullable: true}); got != "*string" {
			t.Fatalf("TEXT -> *string, got %s", got)
		}
	}
	if fn := TypeMap["DATE"]; fn == nil {
		t.Fatal("DATE mapping missing")
	} else {
		if got := fn(fakeColumnType{nullable: true}); got != "*time.Time" {
			t.Fatalf("DATE -> *time.Time, got %s", got)
		}
	}
}

// Ensure the package compiles references for gorm.DB in signatures (unused import fix)
var _ = gorm.DB{}
