// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"strings"
	"time"
)

type TimeArray []time.Time

func (a *TimeArray) Scan(src interface{}) error {
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

func (a TimeArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]time.Time(a))
}

func (a *TimeArray) UnmarshalJSON(data []byte) error {
	var tmp []time.Time
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = TimeArray(tmp)
	return nil
}

func (a TimeArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.Format(time.RFC3339Nano)
	}
	return []byte(strings.Join(strs, ",")), nil
}

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

func (TimeArray) GormDataType() string {
	return "timestamptz[]"
}

func (TimeArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "timestamptz[]"
	}
	return ""
}

func (TimeArray) FromSlice(s []time.Time) TimeArray {
	return TimeArray(s)
}

func (a TimeArray) AsSlice() []time.Time {
	return []time.Time(a)
}

func (a TimeArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.Format(time.RFC3339Nano)
	}
	return strings.Join(strs, ",")
}

func (a TimeArray) Len() int           { return len(a) }
func (a TimeArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TimeArray) Less(i, j int) bool { return a[i].Before(a[j]) }

func (a TimeArray) Contains(val time.Time) bool {
	for _, x := range a {
		if x.Equal(val) {
			return true
		}
	}
	return false
}

func (a TimeArray) IndexOf(val time.Time) int {
	for i, x := range a {
		if x.Equal(val) {
			return i
		}
	}
	return -1
}

func (a TimeArray) IsEmpty() bool {
	return len(a) == 0
}

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

func (a TimeArray) Filter(f func(time.Time) bool) TimeArray {
	var out TimeArray
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a TimeArray) Append(vals ...time.Time) TimeArray {
	return append(a, vals...)
}

func (a TimeArray) Equals(b TimeArray) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}
