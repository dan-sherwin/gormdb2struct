// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"strconv"
	"strings"
)

type Float64Array []float64

func (a *Float64Array) Scan(src interface{}) error {
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

func (a Float64Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64(a))
}

func (a *Float64Array) UnmarshalJSON(data []byte) error {
	var tmp []float64
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Float64Array(tmp)
	return nil
}

func (a Float64Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return []byte(strings.Join(strs, ",")), nil
}

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

func (Float64Array) GormDataType() string {
	return "double precision[]"
}

func (Float64Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "double precision[]"
	}
	return ""
}

func (Float64Array) FromSlice(s []float64) Float64Array {
	return Float64Array(s)
}

func (a Float64Array) AsSlice() []float64 {
	return []float64(a)
}

func (a Float64Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(strs, ",")
}

func (a Float64Array) Len() int           { return len(a) }
func (a Float64Array) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Float64Array) Less(i, j int) bool { return a[i] < a[j] }

func (a Float64Array) Contains(val float64) bool {
	for _, x := range a {
		if x == val {
			return true
		}
	}
	return false
}

func (a Float64Array) IndexOf(val float64) int {
	for i, x := range a {
		if x == val {
			return i
		}
	}
	return -1
}

func (a Float64Array) IsEmpty() bool {
	return len(a) == 0
}

func (a Float64Array) Unique() Float64Array {
	seen := make(map[float64]struct{}, len(a))
	var out Float64Array
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (a Float64Array) Filter(f func(float64) bool) Float64Array {
	var out Float64Array
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a Float64Array) Append(vals ...float64) Float64Array {
	return append(a, vals...)
}

func (a Float64Array) Equals(b Float64Array) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
