// Package main provides end-to-end tests for the gormdb2struct tool.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/glebarez/go-sqlite"
)

// TestEndToEndSQLite generates models from a temp SQLite DB and then builds and runs
// a tiny program that uses the generated package to insert, read, and update rows.
func TestEndToEndSQLite(t *testing.T) {
	// Skip on short to keep CI fast if needed; this is an E2E test.
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create sqlite schema with various data types and a relation to test ExtraFields.
	dsn := fmt.Sprintf("file:%s?cache=shared&_fk=1", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := []string{
		// main table covering many type mappings
		`CREATE TABLE IF NOT EXISTS all_types (
			id INTEGER PRIMARY KEY,
			bool_col BOOLEAN,
			tiny1 TINYINT(1),
			int_col INTEGER NOT NULL DEFAULT 0,
			big_col BIGINT,
			real_col REAL,
			double_col DOUBLE,
			float_col FLOAT,
			text_col TEXT,
			varchar_col VARCHAR(255),
			char_col CHAR(10),
			blob_col BLOB,
			date_col DATE,
			datetime_col DATETIME,
			ts_col TIMESTAMP,
			numeric_col NUMERIC,
			decimal_col DECIMAL,
			duration_col DURATION,
			json_col JSONB
		);`,
		// second table to exercise relation via ExtraFields (one-to-many)
		`CREATE TABLE IF NOT EXISTS child (
			id INTEGER PRIMARY KEY,
			all_types_id INTEGER NOT NULL,
			name TEXT,
			FOREIGN KEY(all_types_id) REFERENCES all_types(id)
		);`,
	}
	for _, q := range schema {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("create schema: %v for query %s", err, q)
		}
	}

	// Create an OutPath under the project root so imports resolve within the same module.
	outPath := filepath.Join(projectRoot(t), "generated_e2e")
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(outPath) })

	// Write TOML config for generator.
	cfgToml := fmt.Sprintf(`
ConfigVersion = 1

[Generator]
OutPath = %q
CleanUp = true

[Database]
Dialect = "sqlite"

[Database.SQLite]
Path = %q

[DbInit]
Enabled = true
IncludeAutoMigrate = true

[ExtraFields]
  [[ExtraFields."all_types"]]
  StructPropName = "Children"
  StructPropType = "models.Child"
  FkStructPropName = "AllTypesID"
  RefStructPropName = "ID"
  HasMany = true
  Pointer = false
`, outPath, dbPath)
	cfgPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(cfgToml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run generator: go run ./cmd <config>
	cmd := exec.CommandContext(context.Background(), "go", "run", "./cmd", cfgPath)
	cmd.Dir = projectRoot(t)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generator failed: %v\nOutput:\n%s", err, string(out))
	}

	// Verify expected generated files exist
	mustExist(t, filepath.Join(outPath, "models"))
	mustExist(t, filepath.Join(outPath, "db_sqlite.go"))

	// Determine the generated struct name for the all_types table by reading its model file
	modelsDir := filepath.Join(outPath, "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		t.Fatal(err)
	}
	var allTypesFile string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "all_types") && strings.HasSuffix(e.Name(), ".go") {
			allTypesFile = filepath.Join(modelsDir, e.Name())
			break
		}
	}
	if allTypesFile == "" {
		t.Fatalf("could not find all_types model file in %s", modelsDir)
	}
	mb, err := os.ReadFile(allTypesFile)
	if err != nil {
		t.Fatal(err)
	}
	// very simple parse: find `type <Name> struct {`
	var modelType string
	for _, ln := range strings.Split(string(mb), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "type ") && strings.Contains(ln, " struct {") {
			parts := strings.Split(ln, " ")
			if len(parts) >= 3 {
				modelType = parts[1]
				break
			}
		}
	}
	if modelType == "" {
		t.Fatalf("unable to determine model type name from %s", allTypesFile)
	}

	pkgBase := filepath.Base(outPath)

	// Create a small program under this module that uses the generated package
	cmdDir := filepath.Join(projectRoot(t), "cmd_e2e")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cmdDir) })
	mainGo := fmt.Sprintf(`package main
import (
  "fmt"
  "time"
  "gorm.io/datatypes"
  g "%s/%s"
  m "%s/%s/models"
)
func main(){
  if err := g.DbInit(%q); err != nil { panic(err) }
  // Insert
  js := datatypes.JSON([]byte(`+"`"+`{"a":1,"b":2}`+"`"+`))
  a := &m.%s{BoolCol: ptrBool(true), Tiny1: ptrStr("1"), IntCol: ptrI64(42), BigCol: ptrI64(4200), RealCol: ptrF64(1.5), DoubleCol: ptrF64(2.5), FloatCol: ptrF32(3.5), TextCol: ptrStr("hello"), VarcharCol: ptrStr("v"), CharCol: ptrStr("c"), BlobCol: ptrBytes([]byte{1,2,3}), DateCol: ptrTime(1700000000), DatetimeCol: ptrTime(1700000100), TsCol: ptrTime(1700000200), NumericCol: ptrF64(10.5), DecimalCol: ptrF64(20.5), DurationCol: ptrDur(1234567890), JSONCol: &js}
  if err := g.DB.Create(a).Error; err != nil { panic(err) }
  // Read
  var got m.%s
  if err := g.DB.First(&got, a.ID).Error; err != nil { panic(err) }
  // Update each field
  b := false
  jsu := datatypes.JSON([]byte(`+"`"+`"scalar"`+"`"+`))
  if err := g.DB.Model(&got).Updates(map[string]any{
    "bool_col": &b,
    "tiny1": ptrStr("0"),
    "int_col": ptrI64(43),
    "big_col": ptrI64(4300),
    "real_col": ptrF64(9.5),
    "double_col": ptrF64(8.5),
    "float_col": ptrF32(7.5),
    "text_col": ptrStr("world"),
    "varchar_col": ptrStr("vv"),
    "char_col": ptrStr("cc"),
    "blob_col": ptrBytes([]byte{9,8,7}),
    "date_col": ptrTime(1700001000),
    "datetime_col": ptrTime(1700001100),
    "ts_col": ptrTime(1700001200),
    "numeric_col": ptrF64(11.5),
    "decimal_col": ptrF64(21.5),
    "duration_col": ptrDur(987654321),
    "json_col": &jsu,
  }).Error; err != nil { panic(err) }
  var after m.%s
  if err := g.DB.First(&after, a.ID).Error; err != nil { panic(err) }
  if after.TextCol == nil || *after.TextCol != "world" { panic(fmt.Sprintf("unexpected text: %%v", after.TextCol)) }
  if after.JSONCol == nil || string(*after.JSONCol) != "\"scalar\"" { panic(fmt.Sprintf("unexpected json: %%v", after.JSONCol)) }
  fmt.Print("OK")
}
func ptrStr(s string)*string{ return &s }
func ptrI64(v int64)*int64{ return &v }
func ptrF64(v float64)*float64{ return &v }
func ptrF32(v float32)*float32{ return &v }
func ptrBool(v bool)*bool{ return &v }
func ptrBytes(b []byte)*[]byte{ return &b }
func ptrTime(sec int64)*time.Time{ t:=time.Unix(sec,0); return &t }
func ptrDur(n int64)*time.Duration{ d:=time.Duration(n); return &d }
`, modulePath(t), pkgBase, modulePath(t), pkgBase, dbPath, modelType, modelType, modelType)
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build and run the small program from the repo root
	run := exec.Command("go", "run", "./cmd_e2e")
	run.Dir = projectRoot(t)
	run.Env = os.Environ()
	progOut, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("use app failed: %v\nOutput:\n%s", err, string(progOut))
	}
	if !strings.Contains(string(progOut), "OK") {
		t.Fatalf("unexpected output: %s", string(progOut))
	}
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected exists: %s: %v", p, err)
	}
}

// projectRoot returns the repo root directory (where this test file lives).
func projectRoot(_ *testing.T) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func modulePath(t *testing.T) string {
	b, err := os.ReadFile(filepath.Join(projectRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "module "))
		}
	}
	t.Fatal("module path not found")
	return ""
}
