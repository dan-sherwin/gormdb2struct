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

type Int32Array []int32

func (a *Int32Array) Scan(src interface{}) error {
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

func (a Int32Array) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int32(a))
}

func (a *Int32Array) UnmarshalJSON(data []byte) error {
	var tmp []int32
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = Int32Array(tmp)
	return nil
}

func (a Int32Array) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(int64(v), 10)
	}
	return []byte(strings.Join(strs, ",")), nil
}

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

func (Int32Array) GormDataType() string {
	return "integer[]"
}

func (Int32Array) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "integer[]"
	}
	return ""
}

func (Int32Array) FromSlice(s []int32) Int32Array {
	return Int32Array(s)
}

func (a Int32Array) AsSlice() []int32 {
	return []int32(a)
}

func (a Int32Array) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = strconv.FormatInt(int64(v), 10)
	}
	return strings.Join(strs, ",")
}

func (a Int32Array) Len() int           { return len(a) }
func (a Int32Array) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Int32Array) Less(i, j int) bool { return a[i] < a[j] }

func (a Int32Array) Contains(val int32) bool {
	for _, x := range a {
		if x == val {
			return true
		}
	}
	return false
}

func (a Int32Array) IndexOf(val int32) int {
	for i, x := range a {
		if x == val {
			return i
		}
	}
	return -1
}

func (a Int32Array) IsEmpty() bool {
	return len(a) == 0
}

func (a Int32Array) Unique() Int32Array {
	seen := make(map[int32]struct{}, len(a))
	var out Int32Array
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (a Int32Array) Filter(f func(int32) bool) Int32Array {
	var out Int32Array
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a Int32Array) Append(vals ...int32) Int32Array {
	return append(a, vals...)
}

func (a Int32Array) Equals(b Int32Array) bool {
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
