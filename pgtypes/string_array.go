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

// StringArray represents a PostgreSQL text array ([]text).
type StringArray []string

// Scan implements the sql.Scanner interface.
func (a *StringArray) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into StringArray", src)
	}

	input = strings.Trim(input, "{}")
	if input == "" {
		*a = []string{}
		return nil
	}

	parts := strings.Split(input, ",")
	out := make([]string, len(parts))
	for i, s := range parts {
		out[i] = strings.Trim(s, `"`)
	}
	*a = out
	return nil
}

// Value implements the driver.Valuer interface.
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	quoted := make([]string, len(a))
	for i, s := range a {
		quoted[i] = fmt.Sprintf(`"%s"`, s)
	}
	return fmt.Sprintf("{%s}", strings.Join(quoted, ",")), nil
}

// MarshalJSON implements json.Marshaler
func (a StringArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(a))
}

// UnmarshalJSON implements json.Unmarshaler
func (a *StringArray) UnmarshalJSON(data []byte) error {
	var tmp []string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = StringArray(tmp)
	return nil
}

// MarshalText implements encoding.TextMarshaler
func (a StringArray) MarshalText() ([]byte, error) {
	return []byte(strings.Join(a, ",")), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (a *StringArray) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = StringArray{}
		return nil
	}
	*a = strings.Split(string(data), ",")
	return nil
}

// GormDataType returns the general data type
func (StringArray) GormDataType() string {
	return "text[]"
}

// GormDBDataType returns the database data type for a specific dialect
func (StringArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "text[]"
	}
	return ""
}

// FromSlice creates a new StringArray from a []string
func (StringArray) FromSlice(s []string) StringArray {
	return StringArray(s)
}

// AsStringSlice returns the StringArray as a []string
func (a StringArray) AsStringSlice() []string {
	return []string(a)
}

// String implements fmt.Stringer
func (a StringArray) String() string {
	return strings.Join(a, ",")
}

// Len implements sort.Interface
func (a StringArray) Len() int {
	return len(a)
}

// Swap implements sort.Interface
func (a StringArray) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less implements sort.Interface
func (a StringArray) Less(i, j int) bool {
	return a[i] < a[j]
}

// Contains returns true if the value exists in the array.
func (a StringArray) Contains(val string) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the value, or -1 if not found.
func (a StringArray) IndexOf(val string) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the array has no elements
func (a StringArray) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new StringArray with duplicate values removed
func (a StringArray) Unique() StringArray {
	seen := make(map[string]struct{}, len(a))
	var out StringArray
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out
}

// Filter returns a new StringArray with elements matching the filter
func (a StringArray) Filter(f func(string) bool) StringArray {
	var out StringArray
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new StringArray with the specified values added
func (a StringArray) Append(vals ...string) StringArray {
	return append(a, vals...)
}

// Equals returns true if the other StringArray has the same values in order.
func (a StringArray) Equals(b StringArray) bool {
	return slices.Equal(a, b)
}
