// Package pgtypes provides GORM-compatible custom PostgreSQL types.
package pgtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"strings"
)

type UUIDArray []uuid.UUID

func (a *UUIDArray) Scan(src interface{}) error {
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

func (a UUIDArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]uuid.UUID(a))
}

func (a *UUIDArray) UnmarshalJSON(data []byte) error {
	var tmp []uuid.UUID
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*a = UUIDArray(tmp)
	return nil
}

func (a UUIDArray) MarshalText() ([]byte, error) {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return []byte(strings.Join(strs, ",")), nil
}

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

func (UUIDArray) GormDataType() string {
	return "uuid[]"
}

func (UUIDArray) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	if db.Name() == "postgres" {
		return "uuid[]"
	}
	return ""
}

func (UUIDArray) FromSlice(s []uuid.UUID) UUIDArray {
	return UUIDArray(s)
}

func (a UUIDArray) AsSlice() []uuid.UUID {
	return []uuid.UUID(a)
}

func (a UUIDArray) String() string {
	strs := make([]string, len(a))
	for i, v := range a {
		strs[i] = v.String()
	}
	return strings.Join(strs, ",")
}

func (a UUIDArray) Len() int           { return len(a) }
func (a UUIDArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UUIDArray) Less(i, j int) bool { return strings.Compare(a[i].String(), a[j].String()) < 0 }

func (a UUIDArray) Contains(val uuid.UUID) bool {
	for _, x := range a {
		if x == val {
			return true
		}
	}
	return false
}

func (a UUIDArray) IndexOf(val uuid.UUID) int {
	for i, x := range a {
		if x == val {
			return i
		}
	}
	return -1
}

func (a UUIDArray) IsEmpty() bool {
	return len(a) == 0
}

func (a UUIDArray) Unique() UUIDArray {
	seen := make(map[uuid.UUID]struct{}, len(a))
	var out UUIDArray
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (a UUIDArray) Filter(f func(uuid.UUID) bool) UUIDArray {
	var out UUIDArray
	for _, v := range a {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

func (a UUIDArray) Append(vals ...uuid.UUID) UUIDArray {
	return append(a, vals...)
}

func (a UUIDArray) Equals(b UUIDArray) bool {
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
