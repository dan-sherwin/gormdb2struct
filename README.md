# Gorm Database to Struct
_Generate strongly typed GORM models and query helpers from your database schema (PostgreSQL or SQLite) with a stateless, config-first CLI._

This tool connects to your database, introspects database objects, and produces:
- Models (structs with json and gorm tags)
- A query package (via gorm.io/gen) with helpful typed methods
- Optional db initializer (DbInit) tailored for your dialect

It’s configuration-driven via a TOML file, intentionally stateless, and suitable for CI/CD use.

Current support is focused on PostgreSQL and SQLite.
Support for additional GORM dialects is planned for future releases.

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
- **Flexible type mapping**: override with a single `TypeMap` for standard types, enums, domains, and arrays
- **Relationship helpers**: add has-one / has-many fields via `ExtraFields`
- **Fine-grained JSON control**: override tags per-table/field
- **Optional AutoMigrate** in generated DbInit
- **Safe cleanup** of old generated files
- **Quick-start config generator** (`generate-config-sample`)
- **Modern CLI architecture** with structured logging and clean internal package boundaries

---

## Status

- **Supported today**: PostgreSQL, SQLite
- **Planned future dialects**: MySQL, TiDB, GaussDB, SQL Server, ClickHouse, Oracle
- **Config format**: clean versioned config with `ConfigVersion = 1`, plus legacy unversioned compatibility for older shipped configs

---

## Table of Contents
- [Install](#install)
- [Quick Start](#quick-start)
- [Configuration (TOML)](#configuration-toml)
- [Architecture](#architecture)
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
    - brew tap dan-sherwin/homebrew-tap
    - brew install gormdb2struct

- Or directly:
    - brew install dan-sherwin/homebrew-tap/gormdb2struct

Then verify:
- `gormdb2struct -version`

### Build from source:
Requires Go 1.26+.

You can run directly via `go run` for quick use, or build a binary for reuse:

- Use directly via `go run` from a clone:
  - git clone https://github.com/dan-sherwin/gormdb2struct.git
  - cd gormdb2struct
  - go run ./cmd generate-config-sample
  - Edit the generated `gormdb2struct-sample.toml`
  - go run ./cmd ./path/to/your-config.toml

- Or build the binary:
  - go build -o gormdb2struct ./cmd
  - ./gormdb2struct generate-config-sample
  - ./gormdb2struct ./path/to/your-config.toml

---

## Quick Start

1) Generate a sample config you can edit:

```
$ go run ./cmd generate-config-sample
Sample config written to gormdb2struct-sample.toml
```

2) Edit the TOML to match your environment (see Configuration below).

3) Run the generator:

```
$ go run ./cmd ./gormdb2struct-sample.toml
```

4) Your generated code will appear under `OutPath` (e.g., `./generated`).

_That's it! You now have a generated GORM-ready package tailored to your schema._

---

## Configuration (TOML)

Preferred format uses `ConfigVersion = 1`.

If `ConfigVersion` is omitted, `gormdb2struct` falls back to the legacy unversioned parser for older configs that are already in use.

Main sections in the versioned format:
- `ConfigVersion = 1`
- `[Generator]`: `OutPath`, `OutPackagePath`, `CleanUp`, `ImportPackagePaths`, `Objects`
- `[Database]`: `Dialect`
- `[Database.PostgreSQL]`: `Host`, `Port`, `Name`, `User`, `Password`, `SSLMode`
- `[Database.SQLite]`: `Path`
- `[DbInit]`: `Enabled`, `IncludeAutoMigrate`, `GenerateAppSettingsRegistration`, `UseSlogGormLogger`
- `[TypeMap]`: shared database type overrides
- `[ExtraFields]`: relation/helper fields
- `[JSONTagOverridesByTable]`: per-table JSON tag overrides
- `[PostgreSQL.GeneratedTypes]` and `[PostgreSQL.GeneratedTypes.TypeMap]`: PostgreSQL-only generated wrapper types

Legacy compatibility:
- If `ConfigVersion` is absent, older unversioned configs are still supported.
- That legacy parser continues to accept older keys such as `DomainTypeMap`, `Tables`, `MaterializedViews`, `GenerateDbInit`, `IncludeAutoMigrate`, and `Sqlitedbpath`.
- New configs should use the versioned format below.

Sample config:

```
# gormdb2struct configuration
ConfigVersion = 1

# ----------------------------------------------------------------------
# Generator
# ----------------------------------------------------------------------

[Generator]
OutPath = "./generated"
OutPackagePath = ""
CleanUp = true
ImportPackagePaths = [
  "github.com/dan-sherwin/gormdb2struct/pgtypes",
]
# Objects = ["tickets", "ticket_rollup"]

# ----------------------------------------------------------------------
# Database
# Keep only the database subsection that matches Database.Dialect.
# ----------------------------------------------------------------------

[Database]
Dialect = "postgresql"

[Database.PostgreSQL]
Host = "localhost"
Port = 5432
Name = "my_database"
User = "my_user"
Password = "secret"
SSLMode = false

[Database.SQLite]
Path = "./schema.db"

# ----------------------------------------------------------------------
# Optional generation sections
# ----------------------------------------------------------------------

[DbInit]
Enabled = true
IncludeAutoMigrate = false
GenerateAppSettingsRegistration = false
UseSlogGormLogger = false

# TypeMap: shared database type overrides (optional).
# PostgreSQL: standard types, enums, domains, arrays.
# SQLite: declared column types.
[TypeMap]
# "jsonb" = "datatypes.JSONMap"
# "uuid" = "datatypes.UUID"
# "my_text_domain" = "string"

# ExtraFields: add relation fields to specific models (optional)
[ExtraFields]
# [[ExtraFields."ticket_extended"]]
# StructPropName = "Attachments"
# StructPropType = "models.Attachment"
# FkStructPropName = "TicketID"
# RefStructPropName = "TicketID"
# HasMany = true
# Pointer = true

# JSONTagOverridesByTable: override json tags for fields (optional)
[JSONTagOverridesByTable]
# [JSONTagOverridesByTable."ticket_extended"]
# subject_fts = "-"

# ----------------------------------------------------------------------
# PostgreSQL-only sections
# ----------------------------------------------------------------------

[PostgreSQL.GeneratedTypes]
PackageName = "dbtypes"
RelativePath = "models/dbtypes"
PackagePath = ""

[PostgreSQL.GeneratedTypes.TypeMap]
# "ticket_status" = "TicketStatus"
# "ticket_type" = "TicketType"
# "ticket_type[]" = "TicketTypeArray"
# "my_text_domain" = "MyTextDomain"
```

Validation rules enforced by the tool:
- `ConfigVersion = 1` is the current explicit format
- For versioned configs, unknown or misplaced keys are rejected
- `[Generator].OutPath` is required
- `[Database].Dialect` must be `postgresql` or `sqlite`
- For PostgreSQL: `[Database.PostgreSQL].Host` and `[Database.PostgreSQL].Name` are required, and `Port` defaults to `5432` if omitted
- For SQLite: `[Database.SQLite].Path` is required
- PostgreSQL generated types are only supported when `[Database].Dialect = "postgresql"`

---

## Architecture

- `cmd/main.go` is the single CLI entrypoint and release build target
- `cmd/app` owns CLI parsing, version/build metadata, and structured logging
- `internal/config` owns TOML decoding, normalization, defaults, and validation
- `internal/generator` owns generation orchestration, dialect handling, cleanup, and DbInit emission
- `pgtypes` remains the public PostgreSQL custom type package for generated projects
- The CLI is intentionally stateless: no persistent settings, no hidden local app database, and no working-directory side effects

---

## Generated Code Layout

Given OutPath = "./generated":
- ./generated/models: structs for your tables with tags
- ./generated: query code via gorm.io/gen and a `db.go`/`db_sqlite.go` initializer when enabled
- The package name equals the base directory of OutPath (e.g., "generated")
- If `OutPackagePath` is omitted, the generator will try to derive the import path from the current Go module before writing `DbInit`

### Using the generated package

- Import the package in your app (module path depends on your project):

```
import (
  g "your/module/path/generated"
  m "your/module/path/generated/models"
)
```

- Initialize the database:
  - PostgreSQL: `g.DbInit()` returns an error and accepts an optional DSN override string; if omitted, a DSN is built from the configured PostgreSQL host/port/name/user/password/SSL mode via `github.com/dan-sherwin/go-utilities`.
  - SQLite: `g.DbInit()` returns an error and accepts an optional file path override string; if omitted, DbPath from the generated file is used.
  - If `DbInit.GenerateAppSettingsRegistration = true`, the generated init file also registers DB settings with `github.com/dan-sherwin/go-app-settings`.
  - If `DbInit.UseSlogGormLogger = true`, the generated init file configures GORM with `github.com/orandin/slog-gorm`.

- Perform operations with GORM using `g.DB`:

```
if err := g.DbInit(); err != nil { /* handle */ }

var rec m.YourModel
if err := g.DB.First(&rec).Error; err != nil { /* handle */ }
```

- If `DbInit.IncludeAutoMigrate = true`, the generated DbInit will call AutoMigrate for all models.

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

- `Generator.Objects` is the preferred list of database objects to generate from. In legacy unversioned configs, `Tables` and `MaterializedViews` are still accepted and merged into the canonical object list.
- `TypeMap` maps a database type (for example `jsonb`, `uuid`, `ticket_status`, `my_domain`, `ticket_type[]`) to the Go type string used in the generated struct.
- In legacy unversioned configs, `DomainTypeMap` is still accepted and merged into the canonical type map.
- SQLite type handling is provided in `sqlitetype/TypeMap`.
- `PostgreSQL.GeneratedTypes` is PostgreSQL-only; SQLite uses `TypeMap` overrides but does not support generated enum/domain wrapper types.

---

## Testing

The repository includes an end-to-end SQLite test `TestEndToEndSQLite` which:
- Creates a temporary SQLite DB with a schema that exercises the type map and relations
- Runs the generator
- Builds and runs a small program using the generated package to exercise CRUD
- Cleans up after itself

There is also a template-generation test for PostgreSQL `DbInit` output so the generated initializer stays compile-oriented and deterministic.

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
- `DbInit` returns an error instead of panicking or exiting, so the parent app stays in control of error handling.
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
