// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Float64Array represents a PostgreSQL double precision array ([]double precision).
type Float64Array []float64

// Scan implements the sql.Scanner interface.
func (a *Float64Array) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into Float64Array", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = Float64Array{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(Float64Array, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return err
		}
		result[i] = val
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a Float64Array) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a Float64Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *Float64Array) UnmarshalJSON(data []byte) error {
	var tmp []float64
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Float64Array(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a Float64Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *Float64Array) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = Float64Array{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(Float64Array, len(parts))
	for i, s := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return err
		}
		out[i] = v
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (Float64Array) GormDataType() string {
	return "double precision[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (Float64Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "double precision[]"
	}
	return ""
}

// FromSlice converts a float64 slice to a Float64Array.
func (Float64Array) FromSlice(s []float64) Float64Array {
	return Float64Array(s)
}

// AsSlice converts the Float64Array to a float64 slice.
func (a Float64Array) AsSlice() []float64 {
	return []float64(a)
}

// String returns the string representation of the Float64Array.
func (a Float64Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a Float64Array) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a Float64Array) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a Float64Array) Less(i, j int) bool { return a[i] < a[j] }

// Contains returns true if the value exists in the array.
func (a Float64Array) Contains(val float64) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the value, or -1 if not found.
func (a Float64Array) IndexOf(val float64) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the array has no elements.
func (a Float64Array) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new Float64Array with duplicate values removed.
func (a Float64Array) Unique() Float64Array {
	seen := make(map[float64]struct{}, len(a))
	var out Float64Array
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out
}

// Filter returns a new Float64Array with elements matching the filter.
func (a Float64Array) Filter(f func(float64) bool) Float64Array {
	var out Float64Array
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new Float64Array with the specified values added.
func (a Float64Array) Append(vals ...float64) Float64Array {
	return append(a, vals...)
}

// Equals returns true if the other Float64Array has the same values in order.
func (a Float64Array) Equals(b Float64Array) bool {
	return slices.Equal(a, b)
}
