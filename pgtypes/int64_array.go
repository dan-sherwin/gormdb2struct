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

type Int64Array []int64

func (a *Int64Array) Scan(src interface{}) error {
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

func (a Int64Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int64(a))
}

func (a *Int64Array) UnmarshalJSON(data []byte) error {
	var tmp []int64
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Int64Array(tmp)
	return nil
}

func (a Int64Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return []byte(strings.Join(strs, ",")), nil
}

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

func (Int64Array) GormDataType() string {
	return "bigint[]"
}

func (Int64Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "bigint[]"
	}
	return ""
}

func (Int64Array) FromSlice(s []int64) Int64Array {
	return Int64Array(s)
}

func (a Int64Array) AsSlice() []int64 {
	return []int64(a)
}

func (a Int64Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(strs, ",")
}

func (a Int64Array) Len() int           { return len(a) }
func (a Int64Array) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Int64Array) Less(i, j int) bool { return a[i] < a[j] }

func (a Int64Array) Contains(val int64) bool {
	for _, x := range a {
		if x == val {
			return true
		}
	}
	return false
}

func (a Int64Array) IndexOf(val int64) int {
	for i, x := range a {
		if x == val {
			return i
		}
	}
	return -1
}

func (a Int64Array) IsEmpty() bool {
	return len(a) == 0
}

func (a Int64Array) Unique() Int64Array {
	seen := make(map[int64]struct{}, len(a))
	var out Int64Array
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (a Int64Array) Filter(f func(int64) bool) Int64Array {
	var out Int64Array
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a Int64Array) Append(vals ...int64) Int64Array {
	return append(a, vals...)
}

func (a Int64Array) Equals(b Int64Array) bool {
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
