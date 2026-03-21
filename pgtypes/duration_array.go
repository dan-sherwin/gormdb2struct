// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DurationArray is a slice of Durations that supports PostgreSQL's interval array type.
type DurationArray []Duration

// Scan implements the sql.Scanner interface.
func (a *DurationArray) Scan(src interface{}) error {
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
		return fmt.Errorf("cannot scan type %T into DurationArray", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = DurationArray{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(DurationArray, len(parts))
	for i, p := range parts {
		dur, err := parsePostgresInterval(strings.Trim(p, `"`))
		if err != nil {
			return err
		}
		result[i] = Duration{dur}
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a DurationArray) Value() (driver.Value, error) {
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
func (a DurationArray) MarshalJSON() ([]byte, error) {
	raw := make([]string, len(a))
	for i, v := range a {
		raw[i] = v.String()
	}
	return json.Marshal(raw)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *DurationArray) UnmarshalJSON(data []byte) error {
	var tmp []string
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	result := make(DurationArray, len(tmp))
	for i, s := range tmp {
		d, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		result[i] = Duration{d}
	}
	*a = result
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a DurationArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *DurationArray) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = DurationArray{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(DurationArray, len(parts))
	for i, s := range parts {
		d, err := time.ParseDuration(strings.TrimSpace(s))
		if err != nil {
			return err
		}
		out[i] = Duration{d}
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (DurationArray) GormDataType() string {
	return "interval[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (DurationArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "interval[]"
	}
	return ""
}

// FromSlice converts a time.Duration slice to a DurationArray.
func (DurationArray) FromSlice(s []time.Duration) DurationArray {
	out := make(DurationArray, len(s))
	for i, v := range s {
		out[i] = Duration{v}
	}
	return out
}

// AsSlice converts the DurationArray to a time.Duration slice.
func (a DurationArray) AsSlice() []time.Duration {
	out := make([]time.Duration, len(a))
	for i, v := range a {
		out[i] = v.Duration
	}
	return out
}

func (a DurationArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return strings.Join(strs, ",")
}

func (a DurationArray) Len() int           { return len(a) }
func (a DurationArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DurationArray) Less(i, j int) bool { return a[i].Duration < a[j].Duration }

// Contains returns true if the DurationArray contains the given value.
func (a DurationArray) Contains(val time.Duration) bool {
	for _, x := range a {
		if x.Duration == val {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of the given value, or -1 if not found.
func (a DurationArray) IndexOf(val time.Duration) int {
	for i, x := range a {
		if x.Duration == val {
			return i
		}
	}
	return -1
}

// IsEmpty returns true if the DurationArray is empty.
func (a DurationArray) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new DurationArray with duplicate values removed.
func (a DurationArray) Unique() DurationArray {
	seen := make(map[time.Duration]struct{}, len(a))
	var out DurationArray
	for _, v := range a {
		if _, ok := seen[v.Duration]; !ok {
			seen[v.Duration] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Filter returns a new DurationArray containing only elements that satisfy the given predicate.
func (a DurationArray) Filter(f func(time.Duration) bool) DurationArray {
	var out DurationArray
	for _, v := range a {
		if f(v.Duration) {
			out = append(out, v)
		}
	}
	return out
}

// Append returns a new DurationArray with the given values appended.
func (a DurationArray) Append(vals ...time.Duration) DurationArray {
	for _, v := range vals {
		a = append(a, Duration{v})
	}
	return a
}

// Equals returns true if the DurationArray is equal to another DurationArray.
func (a DurationArray) Equals(b DurationArray) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Duration != b[i].Duration {
			return false
		}
	}
	return true
}
