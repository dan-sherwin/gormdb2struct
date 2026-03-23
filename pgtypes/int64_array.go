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

// Int64Array represents a PostgreSQL bigint array ([]bigint).
type Int64Array []int64

// Scan implements the sql.Scanner interface.
func (a *Int64Array) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into Int64Array", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = Int64Array{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(Int64Array, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return err
		}
		result[i] = val
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a Int64Array) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a Int64Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int64(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *Int64Array) UnmarshalJSON(data []byte) error {
	var tmp []int64
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Int64Array(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a Int64Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *Int64Array) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = Int64Array{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(Int64Array, len(parts))
	for i, s := range parts {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return err
		}
		out[i] = v
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (Int64Array) GormDataType() string {
	return "bigint[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (Int64Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "bigint[]"
	}
	return ""
}

// FromSlice converts an int64 slice to an Int64Array.
func (Int64Array) FromSlice(s []int64) Int64Array {
	return Int64Array(s)
}

// AsSlice converts the Int64Array to an int64 slice.
func (a Int64Array) AsSlice() []int64 {
	return []int64(a)
}

// String returns the string representation of the Int64Array.
func (a Int64Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a Int64Array) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a Int64Array) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a Int64Array) Less(i, j int) bool { return a[i] < a[j] }

// Contains returns true if the value exists in the array.
func (a Int64Array) Contains(val int64) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the value, or -1 if not found.
func (a Int64Array) IndexOf(val int64) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the array has no elements.
func (a Int64Array) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new Int64Array with duplicate values removed.
func (a Int64Array) Unique() Int64Array {
	seen := make(map[int64]struct{}, len(a))
	var out Int64Array
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out
}

// Filter returns a new Int64Array with elements matching the filter.
func (a Int64Array) Filter(f func(int64) bool) Int64Array {
	var out Int64Array
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new Int64Array with the specified values added.
func (a Int64Array) Append(vals ...int64) Int64Array {
	return append(a, vals...)
}

// Equals returns true if the other Int64Array has the same values in order.
func (a Int64Array) Equals(b Int64Array) bool {
	return slices.Equal(a, b)
}
