// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// UUIDArray represents a PostgreSQL uuid array ([]uuid).
type UUIDArray []uuid.UUID

// Scan implements the sql.Scanner interface.
func (a *UUIDArray) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into UUIDArray", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(UUIDArray, len(parts))
	for i, p := range parts {
		parsed, err := uuid.Parse(strings.Trim(p, `"`))
		if err != nil {
			return err
		}
		result[i] = parsed
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = fmt.Sprintf("\"%s\"", v.String())
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a UUIDArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]uuid.UUID(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *UUIDArray) UnmarshalJSON(data []byte) error {
	var tmp []uuid.UUID
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = UUIDArray(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a UUIDArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *UUIDArray) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(UUIDArray, len(parts))
	for i, s := range parts {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		out[i] = id
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (UUIDArray) GormDataType() string {
	return "uuid[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (UUIDArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "uuid[]"
	}
	return ""
}

// FromSlice converts a uuid slice to a UUIDArray.
func (UUIDArray) FromSlice(s []uuid.UUID) UUIDArray {
	return UUIDArray(s)
}

// AsSlice converts the UUIDArray to a uuid slice.
func (a UUIDArray) AsSlice() []uuid.UUID {
	return []uuid.UUID(a)
}

// String returns the string representation of the UUIDArray.
func (a UUIDArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a UUIDArray) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a UUIDArray) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a UUIDArray) Less(i, j int) bool { return strings.Compare(a[i].String(), a[j].String()) < 0 }

// Contains returns true if the value exists in the array.
func (a UUIDArray) Contains(val uuid.UUID) bool {
	return slices.Contains(a, val)
}

// IndexOf returns the index of the value, or -1 if not found.
func (a UUIDArray) IndexOf(val uuid.UUID) int {
	return slices.Index(a, val)
}

// IsEmpty returns true if the array has no elements.
func (a UUIDArray) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new UUIDArray with duplicate values removed.
func (a UUIDArray) Unique() UUIDArray {
	seen := make(map[uuid.UUID]struct{}, len(a))
	var out UUIDArray
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			out = append(out, val)
		}
	}
	return out
}

// Filter returns a new UUIDArray with elements matching the filter.
func (a UUIDArray) Filter(f func(uuid.UUID) bool) UUIDArray {
	var out UUIDArray
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new UUIDArray with the specified values added.
func (a UUIDArray) Append(vals ...uuid.UUID) UUIDArray {
	return append(a, vals...)
}

// Equals returns true if the other UUIDArray has the same values in order.
func (a UUIDArray) Equals(b UUIDArray) bool {
	return slices.Equal(a, b)
}
