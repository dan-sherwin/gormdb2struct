// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// BoolArray is a slice of booleans that supports PostgreSQL's boolean array type.
type BoolArray []bool

// Scan implements the sql.Scanner interface.
func (a *BoolArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var input string
	switch t := src.(type) {
	case []byte:
		input = string(t)
	case string:
		input = t
	default:
		return fmt.Errorf("cannot scan type %T into BoolArray", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = BoolArray{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(BoolArray, len(parts))
	for i, p := range parts {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "t", "true":
			result[i] = true
		case "f", "false":
			result[i] = false
		default:
			return fmt.Errorf("invalid boolean value: %s", p)
		}
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a BoolArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		if v {
			strs[i] = "t"
		} else {
			strs[i] = "f"
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a BoolArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]bool(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *BoolArray) UnmarshalJSON(data []byte) error {
	var tmp []bool
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = BoolArray(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a BoolArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		if v {
			strs[i] = "true"
		} else {
			strs[i] = "false"
		}
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *BoolArray) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = BoolArray{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(BoolArray, len(parts))
	for i, s := range parts {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "t":
			out[i] = true
		case "false", "f":
			out[i] = false
		default:
			return fmt.Errorf("invalid boolean value: %s", s)
		}
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (BoolArray) GormDataType() string {
	return "boolean[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (BoolArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "boolean[]"
	}
	return ""
}

// FromSlice converts a bool slice to a BoolArray.
func (BoolArray) FromSlice(s []bool) BoolArray {
	return BoolArray(s)
}

// AsSlice converts the BoolArray to a bool slice.
func (a BoolArray) AsSlice() []bool {
	return []bool(a)
}

// String returns the string representation of the BoolArray.
func (a BoolArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		if v {
			strs[i] = "true"
		} else {
			strs[i] = "false"
		}
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a BoolArray) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a BoolArray) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a BoolArray) Less(i, j int) bool { return !a[i] && a[j] }

// Contains returns true if the BoolArray contains the given value.
func (a BoolArray) Contains(val bool) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the first occurrence of the given value, or -1 if not found.
func (a BoolArray) IndexOf(val bool) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the BoolArray is empty.
func (a BoolArray) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new BoolArray with duplicate values removed.
func (a BoolArray) Unique() BoolArray {
	var out BoolArray
	seen := map[bool]bool{}
	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// Filter returns a new BoolArray containing only elements that satisfy the given predicate.
func (a BoolArray) Filter(f func(bool) bool) BoolArray {
	var out BoolArray
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

// Append returns a new BoolArray with the given values appended.
func (a BoolArray) Append(vals ...bool) BoolArray {
	return append(a, vals...)
}

// Equals returns true if the BoolArray is equal to another BoolArray.
func (a BoolArray) Equals(b BoolArray) bool {
	return slices.Equal(a, b)
}
