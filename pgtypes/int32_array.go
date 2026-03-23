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

// Int32Array represents a PostgreSQL integer array ([]integer).
type Int32Array []int32

// Scan implements the sql.Scanner interface.
func (a *Int32Array) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into Int32Array", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = Int32Array{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(Int32Array, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseInt(strings.TrimSpace(p), 10, 32)
		if err != nil {
			return err
		}
		result[i] = int32(val)
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a Int32Array) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(int64(v), 10)
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a Int32Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int32(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *Int32Array) UnmarshalJSON(data []byte) error {
	var tmp []int32
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Int32Array(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a Int32Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(int64(v), 10)
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *Int32Array) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = Int32Array{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(Int32Array, len(parts))
	for i, s := range parts {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
		if err != nil {
			return err
		}
		out[i] = int32(v)
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (Int32Array) GormDataType() string {
	return "integer[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (Int32Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "integer[]"
	}
	return ""
}

// FromSlice converts an int32 slice to an Int32Array.
func (Int32Array) FromSlice(s []int32) Int32Array {
	return Int32Array(s)
}

// AsSlice converts the Int32Array to an int32 slice.
func (a Int32Array) AsSlice() []int32 {
	return []int32(a)
}

// String returns the string representation of the Int32Array.
func (a Int32Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(int64(v), 10)
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a Int32Array) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a Int32Array) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a Int32Array) Less(i, j int) bool { return a[i] < a[j] }

// Contains returns true if the value exists in the array.
func (a Int32Array) Contains(val int32) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the value, or -1 if not found.
func (a Int32Array) IndexOf(val int32) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the array has no elements.
func (a Int32Array) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new Int32Array with duplicate values removed.
func (a Int32Array) Unique() Int32Array {
	seen := make(map[int32]struct{}, len(a))
	var out Int32Array
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out
}

// Filter returns a new Int32Array with elements matching the filter.
func (a Int32Array) Filter(f func(int32) bool) Int32Array {
	var out Int32Array
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new Int32Array with the specified values added.
func (a Int32Array) Append(vals ...int32) Int32Array {
	return append(a, vals...)
}

// Equals returns true if the other Int32Array has the same values in order.
func (a Int32Array) Equals(b Int32Array) bool {
	return slices.Equal(a, b)
}
