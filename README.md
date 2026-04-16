# gormdb2struct

`gormdb2struct` is a schema-first code generator for Go + GORM that emits typed `gorm.io/gen` query code.

It connects to an existing PostgreSQL or SQLite database and generates:
- GORM model structs
- typed `gorm.io/gen` query helpers
- an optional `DbInit` file for the chosen dialect
- optional PostgreSQL wrapper types for enums, domains, and enum arrays

If your database schema is the source of truth and you are tired of hand-maintaining structs, `gorm.io/gen` query scaffolding, and custom type plumbing, this tool is built for that workflow.

## Why This Exists

Many Go teams using GORM still end up doing a lot of repetitive work by hand:
- rebuilding model structs every time the schema changes
- writing boilerplate around PostgreSQL enums and domains
- keeping generated query code and manual model code in sync
- stitching database-specific initialization code into each project

`gormdb2struct` exists to take that work off the critical path without introducing hidden state or a heavyweight framework.

## Best Fit

This tool is a strong fit if you:
- already have a PostgreSQL or SQLite schema
- use GORM and want typed query helpers from `gorm.io/gen`
- want a config-first CLI that works well in CI/CD
- have PostgreSQL enums, domains, arrays, or custom wrapper types that need to stay honest

It is especially useful for PostgreSQL-heavy codebases where enums and domains are part of the real application contract, not just database decoration.

## Highlights

- PostgreSQL and SQLite support today
- Typed query generation via `gorm.io/gen`
- Versioned, human-editable TOML configuration
- Optional `DbInit` generation with app-settings and `slog-gorm` support
- Unified `TypeMap` for standard types, enums, domains, and arrays
- PostgreSQL generated wrapper types for enums, domains, and enum arrays
- PostgreSQL inspection commands that recommend missing mappings and can emit a starter config
- Support for importing existing Go type packages so the inspector can recommend real `TypeMap` entries instead of generated wrappers
- Built-in `pgtypes` package for PostgreSQL arrays and intervals
- Stateless CLI design with no persistent local settings

## Install

### Homebrew

```bash
brew tap dan-sherwin/homebrew-tap
brew install gormdb2struct
```

Or:

```bash
brew install dan-sherwin/homebrew-tap/gormdb2struct
```

### Binary release

Download the latest archive from the [Releases page](https://github.com/dan-sherwin/gormdb2struct/releases), extract it, and place `gormdb2struct` somewhere on your `PATH`.

### DNF / YUM

```bash
sudo dnf config-manager --add-repo https://dan-sherwin.github.io/dan-sherwin.repo
sudo dnf install gormdb2struct
```

On older yum-based systems, install the repo-management package first if needed.

### Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/dan-sherwin/gormdb2struct.git
cd gormdb2struct
go build -o gormdb2struct ./cmd
./gormdb2struct --help
```

## Quick Start

Generate a starter config:

```bash
gormdb2struct generate-config-sample
```

If you are working against PostgreSQL and want help discovering enums, domains, and custom types before you write a config by hand:

```bash
gormdb2struct inspect-postgresql \
  --host localhost \
  --database my_database \
  --user my_user \
  --password-env DB_PASSWORD \
  -o starter.toml
```

If you already know you want to reuse types from an existing Go package, include it:

```bash
gormdb2struct inspect-postgresql \
  --host localhost \
  --database my_database \
  --user my_user \
  --password-env DB_PASSWORD \
  --import-package go.corp.spacelink.com/sdks/go/sl_datatypes \
  -o starter.toml
```

Then generate the code:

```bash
gormdb2struct ./starter.toml
```

If you already have a config and just want a recommendation pass over the objects it references:

```bash
gormdb2struct inspect ./gormdb2struct.toml
```

For a paste-ready TOML recommendation snippet from an existing config:

```bash
gormdb2struct inspect ./gormdb2struct.toml --format toml
```

## What Makes It Different

The differentiator is not just model generation. It is the PostgreSQL type story.

`gormdb2struct` can help with cases like:
- PostgreSQL enums that should become real Go enum wrapper types
- PostgreSQL domains that should map to existing application types
- enum arrays that need a real wrapper type instead of a raw slice
- projects that already have packages like `sl_datatypes` and want those reused automatically in generated config recommendations

That means you can mix and match:
- generated wrappers for schema-defined enums and domains
- explicit `TypeMap` entries for hand-written domain types you already trust
- default built-in support from `pgtypes` for common PostgreSQL arrays and intervals

## Commands

`gormdb2struct` supports four main entry points:

- `gormdb2struct <config.toml>`
  Generate code from a config file.
- `gormdb2struct generate-config-sample`
  Write a full commented starter config.
- `gormdb2struct inspect <config.toml>`
  Inspect the PostgreSQL objects referenced by an existing config and recommend type mappings.
- `gormdb2struct inspect-postgresql`
  Connect directly to PostgreSQL from CLI flags, print an inspection report, and optionally emit a starter config.

`inspect-postgresql` password input options:
- `--password`
- `--password-env`
- `--password-stdin`
- `--password-prompt`

If `-o/--out` is omitted, `inspect-postgresql` prints only the human-readable inspection report.

If `-o stdout` is used, it prints the starter TOML after the report.

If `-o <path>` is used, it writes only the starter TOML to that file.

## Configuration

The preferred config format is explicit and versioned:

```toml
ConfigVersion = 1
```

If `ConfigVersion` is omitted, `gormdb2struct` falls back to a legacy compatibility parser for older shipped configs.

Main sections in the versioned format:
- `[Generator]`
- `[Database]`
- `[Database.PostgreSQL]`
- `[Database.SQLite]`
- `[DbInit]`
- `[TypeMap]`
- `[ExtraFields]`
- `[JSONTagOverridesByTable]`
- `[PostgreSQL.GeneratedTypes]`
- `[PostgreSQL.GeneratedTypes.TypeMap]`

Use `gormdb2struct generate-config-sample` for the full commented example. The sample is structured for hand editing and grouped so dialect-specific settings are easy to find.

Minimal PostgreSQL example:

```toml
ConfigVersion = 1

[Generator]
OutPath = "./generated"
Objects = ["tickets", "ticket_extended"]
ImportPackagePaths = ["github.com/dan-sherwin/gormdb2struct/pgtypes"]

[Database]
Dialect = "postgresql"

[Database.PostgreSQL]
Host = "localhost"
Port = 5432
Name = "ticket_data_core"
User = "ticket_service"
Password = "secret"
SSLMode = false

[DbInit]
Enabled = true
IncludeAutoMigrate = false

[TypeMap]
"spacelink_identifier" = "sl_datatypes.SpacelinkIdentifier"

[PostgreSQL.GeneratedTypes]
PackageName = "types"
RelativePath = "models/types"

[PostgreSQL.GeneratedTypes.TypeMap]
"ticket_status" = "TicketStatus"
"ticket_type" = "TicketType"
```

Validation highlights:
- `[Generator].OutPath` is required
- `[Database].Dialect` must be `postgresql` or `sqlite`
- PostgreSQL requires host and database name
- SQLite requires a database file path
- PostgreSQL generated types are only available when the dialect is PostgreSQL

## Generated Output

Given `OutPath = "./generated"`:

- `./generated/models`
  Generated model structs with `gorm` and `json` tags
- `./generated`
  `gorm.io/gen` query helpers and optional `db.go` / `db_sqlite.go`
- `./generated/models/dbtypes` or your configured generated-types path
  PostgreSQL wrapper types when `PostgreSQL.GeneratedTypes` is enabled

If `OutPackagePath` is omitted, `gormdb2struct` will try to derive it from the current Go module when it needs to emit importable generated files like `DbInit`.

## Generated `DbInit`

`DbInit` generation is optional and controlled by the `[DbInit]` section.

When enabled, generated init code can:
- open the database connection for the selected dialect
- register default query objects
- optionally run `AutoMigrate`
- optionally register database settings with `github.com/dan-sherwin/go-app-settings`
- optionally use `github.com/orandin/slog-gorm` as the GORM logger

`DbInit` returns an error instead of panicking or exiting, so the parent application stays in control.

## PostgreSQL `pgtypes`

The repo also ships a reusable `pgtypes` package for PostgreSQL array and interval handling.

It includes support for types like:
- `pgtypes.StringArray`
- `pgtypes.BoolArray`
- `pgtypes.Int32Array`
- `pgtypes.Int64Array`
- `pgtypes.Float64Array`
- `pgtypes.UUIDArray`
- `pgtypes.TimeArray`
- `pgtypes.Duration`
- `pgtypes.DurationArray`

This package is useful even outside the generator if you want GORM-friendly wrappers for PostgreSQL array and interval columns.

## Architecture

- `cmd/main.go`
  Single executable entrypoint
- `cmd/app`
  CLI parsing, top-level command dispatch, logging, and version/build metadata
- `internal/config`
  Config loading, normalization, validation, and sample generation
- `internal/generator`
  Schema inspection, generation orchestration, dialect handling, and emitted-file logic
- `pgtypes`
  Public PostgreSQL helper types for generated projects

The tool is intentionally stateless. There are no persistent settings and no hidden local project database.

## Development

The local quality gate lives at:

```bash
./dev/ci-local.sh
```

That script runs formatting checks, build, vet, tests, lint, and `govulncheck`.

## Roadmap

Current support is focused on PostgreSQL and SQLite.

Additional GORM dialects are planned for future releases, with likely expansion in this order:
- MySQL
- TiDB
- GaussDB
- SQL Server
- ClickHouse
- Oracle

The long-term goal is not just more dialectors, but dialect-specific generation that stays honest about what each database can actually support.

## Contributing

Issues and pull requests are welcome. Clear reproduction steps, config samples, and tests are especially helpful.

## License

This project is licensed under the terms of the `LICENSE` file in this repository.
