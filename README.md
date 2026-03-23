# Gorm Database to Struct
_Generate strongly typed GORM models and query helpers from your database schema (PostgreSQL or SQLite)._

This tool connects to your database, introspects tables (and Postgres materialized views), and produces:
- Models (structs with json and gorm tags)
- A query package (via gorm.io/gen) with helpful typed methods
- Optional db initializer (DbInit) tailored for your dialect

It’s configuration-driven via a TOML file and suitable for CI/CD use.

---

## Why use this?

- **Save time** by automatically generating GORM models and query helpers from your existing database schema.
- **Enforce type safety** with strongly typed structs and queries tailored to your database.
- **Seamless integration** with GORM gen for powerful and type-safe query building.
- **Supports CI/CD workflows** with configuration-driven generation and safe cleanup of old files.

---

## Features
- **Multi-database support**: PostgreSQL and SQLite
- **Customizable JSON tags**: lowerCamel via strcase
- **Flexible type mapping**: override with `TypeMap` or `DomainTypeMap`
- **Relationship helpers**: add has-one / has-many fields via `ExtraFields`
- **Fine-grained JSON control**: override tags per-table/field
- **Optional AutoMigrate** in generated DbInit
- **Safe cleanup** of old generated files
- **Quick-start config generator** (`-generateConfigSample`)

---

## Table of Contents
- [Install](#install)
- [Quick Start](#quick-start)
- [Configuration (TOML)](#configuration-toml)
- [Generated Code Layout](#generated-code-layout)
- [PostgreSQL Custom Types (pgtypes)](#postgresql-custom-types-pgtypes)
- [Advanced: Type Mapping](#advanced-type-mapping)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [License](#license)
- [Contributing](#contributing)

---

## Install

### Using DNF/YUM (RPM):
- Ensure config-manager is available (dnf-plugins-core provides it):
  - `sudo dnf -y install dnf-plugins-core`  (on older yum-based systems: `sudo yum -y install yum-utils`)
- Add the repo file to your system:
  - `sudo dnf config-manager --add-repo https://dan-sherwin.github.io/dan-sherwin.repo`
- Install the package:
  - `sudo dnf install gormdb2struct`

### Using a binary release:
- Download the latest release from the [Releases page](https://github.com/dan-sherwin/gormdb2struct/releases).
- Extract the archive and run the binary.

### Install via Homebrew (macOS):

- Using the tap:
    - brew tap dan-sherwin/tap
    - brew install gormdb2struct

- Or directly:
    - brew install dan-sherwin/tap/gormdb2struct

Then verify:
- `gormdb2struct -version`

### Go Install:
Requires Go 1.25+.

- Install directly into your GOPATH/bin (or GOBIN):
  - `go install github.com/dan-sherwin/gormdb2struct@latest`

You can also run directly via `go run` for quick use, or build a binary for reuse:

- Use directly via `go run` from a clone:
  - git clone https://github.com/dan-sherwin/gormdb2struct.git
  - cd gormdb2struct
  - go run . -generateConfigSample
  - Edit the generated `gormdb2struct-sample.toml`
  - go run . ./path/to/your-config.toml

- Or build the binary:
  - go build -o gormdb2struct .
  - ./gormdb2struct -generateConfigSample
  - ./gormdb2struct ./path/to/your-config.toml

---

## Quick Start

1) Generate a sample config you can edit:

```
$ go run . -generateConfigSample
Sample config written to gormdb2struct-sample.toml
```

2) Edit the TOML to match your environment (see Configuration below).

3) Run the generator:

```
$ go run . ./gormdb2struct-sample.toml
```

4) Your generated code will appear under `OutPath` (e.g., `./generated`).

_That's it! You now have a generated GORM-ready package tailored to your schema._

---

## Configuration (TOML)

Minimal required keys depend on the selected `DatabaseDialect`.

- Shared
  - OutPath: directory where generated files are written
  - OutPackagePath: package path to the out path for use in the DbInit file
  - DatabaseDialect: "postgresql" or "sqlite"
  - GenerateDbInit: set true to also generate db initializer
  - IncludeAutoMigrate: if true, DbInit runs GORM AutoMigrate for all models
  - CleanUp: when true, remove old `*gen.go` files in OutPath before generating
- PostgreSQL
  - DbHost (required), DbName (required), DbPort (optional, defaults 5432)
  - DbUser (optional), DbPassword (optional), DbSSLMode (optional)
- SQLite
  - Sqlitedbpath (required): path to your sqlite database file

Advanced options:
- ImportPackagePaths: extra import paths for generated code
- TypeMap: override database column type -> Go type mapping (per column type)
- DomainTypeMap: override PostgreSQL domain name -> Go type mapping
- ExtraFields: add relation fields to specific models (has-one/has-many)
- JSONTagOverridesByTable: override json tags per-table per-field

Sample config:

```
# gormdb2struct configuration
# OutPath: directory where generated files are written (models, query, db init)
OutPath = "./generated"

# OutPackagePath: package path to the out path for use in the DbInit file (e.g. github.com/username/my_app/generated) (optional)
OutPackagePath = ""

# DatabaseDialect: "postgresql" or "sqlite"
DatabaseDialect = "postgresql"

# GenerateDbInit: also generate a db initialization file (db.go or db_sqlite.go)
GenerateDbInit = true

# IncludeAutoMigrate: if true, generated DbInit will run AutoMigrate for all models
IncludeAutoMigrate = false

# CleanUp: remove previous *gen.go files in OutPath before generating
CleanUp = true

# ImportPackagePaths: extra imports to include in generated code (optional)
ImportPackagePaths = [
  "github.com/dan-sherwin/gormdb2struct/pgtypes",
]

# TypeMap: database column type overrides (optional)
[TypeMap]
# "jsonb" = "datatypes.JSONMap"
# "uuid"  = "datatypes.UUID"

# DomainTypeMap: map database domain names to Go types (optional)
[DomainTypeMap]
# "my_text_domain" = "string"

# ExtraFields: add relation fields to specific models (optional)
[ExtraFields]
# [ExtraFields."ticket_extended"]
#   [[ExtraFields."ticket_extended"]]
#   StructPropName = "Attachments"
#   StructPropType = "models.Attachment"  # fully-qualified type
#   FkStructPropName = "TicketID"
#   RefStructPropName = "TicketID"
#   HasMany = true
#   Pointer = true

# JSONTagOverridesByTable: override json tags for fields (optional)
[JSONTagOverridesByTable]
# [JSONTagOverridesByTable."ticket_extended"]
#   subject_fts = "-"  # omit from JSON

# --- PostgreSQL specific options ---
# Required when DatabaseDialect = "postgresql"
DbHost = "localhost"      # required
DbPort = 5432               # optional, defaults to 5432
DbName = "my_database"     # required
DbUser = "my_user"         # optional
DbPassword = "secret"      # optional
DbSSLMode = false           # optional: true to enable sslmode=require in DSN

# --- SQLite specific option ---
# Required when DatabaseDialect = "sqlite"
Sqlitedbpath = "./schema.db"
```

Validation rules enforced by the tool:
- OutPath is required
- DatabaseDialect must be "postgresql" or "sqlite"
- For postgresql: DbHost and DbName required; DbPort defaults to 5432 if omitted
- For sqlite: Sqlitedbpath required

---

## Generated Code Layout

Given OutPath = "./generated":
- ./generated/models: structs for your tables with tags
- ./generated: query code via gorm.io/gen and a `db.go`/`db_sqlite.go` initializer when enabled
- The package name equals the base directory of OutPath (e.g., "generated")

### Using the generated package

- Import the package in your app (module path depends on your project):

```
import (
  g "your/module/path/generated"
  m "your/module/path/generated/models"
)
```

- Initialize the database:
  - PostgreSQL: `g.DbInit()` accepts an optional DSN override string; if omitted, a DSN is built from DbHost/DbPort/DbName/DbUser/DbPassword/DbSSLMode.
  - SQLite: `g.DbInit()` accepts an optional file path override string; if omitted, DbPath from the generated file is used.

- Perform operations with GORM using `g.DB`:

```
var rec m.YourModel
if err := g.DB.First(&rec).Error; err != nil { /* handle */ }
```

- If IncludeAutoMigrate = true, the generated DbInit will call AutoMigrate for all models.

---

## PostgreSQL Custom Types (pgtypes)

The `pgtypes` package provides GORM-compatible custom types for PostgreSQL features that are not natively handled by GORM or standard Go types. These are especially useful when working with PostgreSQL arrays and intervals.

### Implemented Types

The following Go types are implemented in the `pgtypes` package:

- `pgtypes.StringArray`: A slice of `string` for PostgreSQL `text[]`, `varchar[]`.
- `pgtypes.BoolArray`: A slice of `bool` for PostgreSQL `boolean[]`.
- `pgtypes.Int32Array`: A slice of `int32` for PostgreSQL `integer[]`, `int4[]`.
- `pgtypes.Int64Array`: A slice of `int64` for PostgreSQL `bigint[]`, `int8[]`.
- `pgtypes.Float64Array`: A slice of `float64` for PostgreSQL `float8[]`, `double precision[]`.
- `pgtypes.UUIDArray`: A slice of `uuid.UUID` for PostgreSQL `uuid[]` (using `github.com/google/uuid`).
- `pgtypes.TimeArray`: A slice of `time.Time` for PostgreSQL `timestamptz[]`, `timestamp[]`.
- `pgtypes.Duration`: A wrapper around `time.Duration` for PostgreSQL `interval`.
- `pgtypes.DurationArray`: A slice of `pgtypes.Duration` for PostgreSQL `interval[]`.

### Supported PostgreSQL Mappings

| PostgreSQL Type | `pgtypes` Go Type | Description |
|-----------------|-------------------|-------------|
| `text[]`, `varchar[]` | `pgtypes.StringArray` | Array of strings |
| `boolean[]` | `pgtypes.BoolArray` | Array of booleans |
| `integer[]`, `int4[]` | `pgtypes.Int32Array` | Array of 32-bit integers |
| `bigint[]`, `int8[]` | `pgtypes.Int64Array` | Array of 64-bit integers |
| `float8[]`, `double precision[]` | `pgtypes.Float64Array` | Array of 64-bit floats |
| `uuid[]` | `pgtypes.UUIDArray` | Array of UUIDs |
| `timestamptz[]`, `timestamp[]` | `pgtypes.TimeArray` | Array of timestamps |
| `interval` | `pgtypes.Duration` | Postgres interval string to `time.Duration` |
| `interval[]` | `pgtypes.DurationArray` | Array of intervals |

### Using pgtypes as a Separate Dependency

You can use the `pgtypes` package in your own Go projects even if you are not using the `gormdb2struct` generator.

1. Add the module to your project:
   ```bash
   go get github.com/dan-sherwin/gormdb2struct/pgtypes
   ```

2. Import and use the types in your structs:
   ```go
   import "github.com/dan-sherwin/gormdb2struct/pgtypes"

   type User struct {
       ID    uint
       Tags  pgtypes.StringArray `gorm:"type:text[]"`
       Roles pgtypes.StringArray `gorm:"type:text[]"`
   }
   ```

These types implement the `sql.Scanner` and `driver.Valuer` interfaces, as well as `json.Marshaler` and `json.Unmarshaler`.

---

## Advanced: Type Mapping

- TypeMap: maps a database column type (e.g., "jsonb", "uuid") to a Go type string used in the generated struct.
- DomainTypeMap (Postgres): if a column’s domain matches a configured key, the mapped Go type is used.
- SQLite type handling is provided in `sqlitetype/TypeMap`.

---

## Testing

The repository includes an end-to-end SQLite test `TestEndToEndSQLite` which:
- Creates a temporary SQLite DB with a schema that exercises the type map and relations
- Runs the generator
- Builds and runs a small program using the generated package to exercise CRUD
- Cleans up after itself

Run tests:

```
go test ./...
```

To skip the E2E test in quick runs:

```
go test -short ./...
```

---

## Troubleshooting
- Ensure your database is reachable and credentials are correct.
- For PostgreSQL, if you pass a custom DSN to `DbInit(dsn)`, it will override the default constructed DSN.
- For SQLite, pass the database file path to `DbInit(path)` to override.
- OutPath’s package name is derived from the directory name; ensure your imports use the correct module path.

---

## License

This project is licensed under the terms of the LICENSE file included in this repository.

---

## Contributing

Issues and pull requests are welcome. Please include clear reproduction steps or tests where possible.

---

## Screenshots / Examples

Contributions of example outputs or screenshots of generated structs and query helpers are welcome! Please share examples to help others understand the generated code structure and usage.


---

## Releases and Packages

This project uses GoReleaser to publish binaries for Linux, macOS (darwin), and Windows, for both amd64 and arm64, whenever you push a tag (vX.Y.Z).

- Download prebuilt archives from the GitHub Releases page for your OS/arch.
- Checksums (checksums.txt) are attached to each release.
- Version metadata (version/commit/date) is embedded and visible via `gormdb2struct -version`.

### Install via dnf/yum (RPM)

Releases also include RPM packages built with nfpm, and a YUM/DNF repository is published to GitHub Pages.

To enable `dnf install gormdb2struct` on your machine, add this repo file:

Create /etc/yum.repos.d/gormdb2struct.repo with:

```
[gormdb2struct]
name=gormdb2struct
baseurl=https://dan-sherwin.github.io/yum/rpm/$basearch/
enabled=1
gpgcheck=0
```

Then run:

```
sudo dnf clean all
sudo dnf makecache
sudo dnf install gormdb2struct
```

Notes:
- `$basearch` resolves to x86_64 or aarch64 automatically.
- If you prefer to install directly from a downloaded RPM, you can still do:
  - `sudo dnf install ./gormdb2struct_<version>_linux_amd64.rpm`

Optional (recommended) GPG signing:
- If you later enable RPM signing and publish your public key at `https://dan-sherwin.github.io/gormdb2struct/public.key`, change the repo to:
```
[gormdb2struct]
name=gormdb2struct
baseurl=https://dan-sherwin.github.io/gormdb2struct/rpm/$basearch/
enabled=1
gpgcheck=1
gpgkey=https://dan-sherwin.github.io/public.key
```
