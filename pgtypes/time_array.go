// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TimeArray represents a PostgreSQL timestamp with time zone array ([]timestamptz).
type TimeArray []time.Time

// Scan implements the sql.Scanner interface.
func (a *TimeArray) Scan(src any) error {
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
		return fmt.Errorf("cannot scan type %T into TimeArray", src)
	}
	input = strings.Trim(input, "{}")
	if input == "" {
		*a = TimeArray{}
		return nil
	}
	parts := strings.Split(input, ",")
	result := make(TimeArray, len(parts))
	for i, p := range parts {
		ts := strings.Trim(p, `"`)
		// Try parsing with timezone first
		t, err := time.Parse("2006-01-02 15:04:05.999999-07", ts)
		if err != nil {
			// Fallback to without timezone
			t, err = time.Parse("2006-01-02 15:04:05.999999", ts)
			if err != nil {
				return fmt.Errorf("parsing time %q failed: %w", ts, err)
			}
		}
		result[i] = t
	}
	*a = result
	return nil
}

// Value implements the driver.Valuer interface.
func (a TimeArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = `"` + v.Format(time.RFC3339Nano) + `"`
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ",")), nil
}

// MarshalJSON implements the json.Marshaler interface.
func (a TimeArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]time.Time(a))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *TimeArray) UnmarshalJSON(data []byte) error {
	var tmp []time.Time
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = TimeArray(tmp)
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a TimeArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.Format(time.RFC3339Nano)
	}
	return []byte(strings.Join(strs, ",")), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *TimeArray) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*a = TimeArray{}
		return nil
	}
	parts := strings.Split(string(data), ",")
	out := make(TimeArray, len(parts))
	for i, s := range parts {
		t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
		if err != nil {
			return err
		}
		out[i] = t
	}
	*a = out
	return nil
}

// GormDataType implements the gorm.DataTypeInterface.
func (TimeArray) GormDataType() string {
	return "timestamptz[]"
}

// GormDBDataType implements the gorm.DBDataTypeInterface.
func (TimeArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "timestamptz[]"
	}
	return ""
}

// FromSlice converts a time.Time slice to a TimeArray.
func (TimeArray) FromSlice(s []time.Time) TimeArray {
	return TimeArray(s)
}

// AsSlice converts the TimeArray to a time.Time slice.
func (a TimeArray) AsSlice() []time.Time {
	return []time.Time(a)
}

// String returns the string representation of the TimeArray.
func (a TimeArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.Format(time.RFC3339Nano)
	}
	return strings.Join(strs, ",")
}

// Len implements sort.Interface.
func (a TimeArray) Len() int { return len(a) }

// Swap implements sort.Interface.
func (a TimeArray) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements sort.Interface.
func (a TimeArray) Less(i, j int) bool { return a[i].Before(a[j]) }

// Contains returns true if the value exists in the array.
func (a TimeArray) Contains(val time.Time) bool {
	return slices.ContainsFunc(a, func(t time.Time) bool { return t.Equal(val) })
}

// IndexOf returns the index of the value, or -1 if not found.
func (a TimeArray) IndexOf(val time.Time) int {
	return slices.IndexFunc(a, func(t time.Time) bool { return t.Equal(val) })
}

// IsEmpty returns true if the array has no elements.
func (a TimeArray) IsEmpty() bool {
	return len(a) == 0
}

// Unique returns a new TimeArray with duplicate values removed.
func (a TimeArray) Unique() TimeArray {
	seen := make(map[string]struct{}, len(a))
	var out TimeArray
	for _, v := range a {
		key := v.Format(time.RFC3339Nano)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Filter returns a new TimeArray with elements matching the filter.
func (a TimeArray) Filter(f func(time.Time) bool) TimeArray {
	var out TimeArray
	for _, val := range a {
		if f(val) {
			out = append(out, val)
		}
	}
	return out
}

// Append returns a new TimeArray with the specified values added.
func (a TimeArray) Append(vals ...time.Time) TimeArray {
	return append(a, vals...)
}

// Equals returns true if the other TimeArray has the same values in order.
func (a TimeArray) Equals(b TimeArray) bool {
	return slices.EqualFunc(a, b, func(t1, t2 time.Time) bool { return t1.Equal(t2) })
}
