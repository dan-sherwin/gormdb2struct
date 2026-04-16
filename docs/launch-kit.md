# Launch Kit

This file is a ready-to-use starting point for talking about `gormdb2struct` in public.

## One-Line Pitch

`gormdb2struct` generates GORM models and typed `gorm.io/gen` query helpers from an existing PostgreSQL or SQLite schema, with first-class support for PostgreSQL enums, domains, and wrapper types.

## Short Repo Blurb

Schema-first generator for Go + GORM that emits model structs, typed `gorm.io/gen` query helpers, optional `DbInit` code, and PostgreSQL wrapper types from an existing database schema.

## Short Launch Post

I just released `gormdb2struct`, a schema-first generator for Go + GORM.

It connects to an existing PostgreSQL or SQLite database and generates:
- GORM model structs
- typed `gorm.io/gen` query helpers
- optional `DbInit` code
- PostgreSQL wrapper types for enums, domains, and enum arrays

The PostgreSQL type story was the main reason I built it. I wanted a workflow that handled enums, domains, generated wrappers, and existing custom Go types without turning the project into a pile of manual plumbing.

Repo: https://github.com/dan-sherwin/gormdb2struct

## Longer Technical Post

I built and open-sourced `gormdb2struct` because I got tired of hand-maintaining the same layers in every GORM project:

- model structs drifting from the actual schema
- `gorm.io/gen` query scaffolding that had to be regenerated and wired up manually
- PostgreSQL enums and domains needing custom Go types
- one-off database init code in every service

`gormdb2struct` is a config-first CLI that points at an existing PostgreSQL or SQLite database and generates:

- GORM models
- typed `gorm.io/gen` query helpers
- optional `DbInit` code
- optional PostgreSQL wrapper types for enums, domains, and enum arrays

The PostgreSQL support is the part I cared most about. It can:

- inspect schema objects and recommend missing type mappings
- generate wrapper types for enums and domains
- reuse existing Go type packages through `TypeMap`
- inspect external import packages and prefer real application types over generated wrappers when the names match

It is intentionally stateless and works well in CI/CD. No persistent app settings database, no hidden local state, just config in and generated code out.

If you work in a schema-first Go + PostgreSQL + GORM environment, especially one that uses `gorm.io/gen`, there is a good chance this will save you some repetitive work.

Repo: https://github.com/dan-sherwin/gormdb2struct

## What To Emphasize

- It is not just a struct generator. It also generates typed `gorm.io/gen` query code.
- The PostgreSQL enum/domain/custom-type handling is one of the real differentiators.
- The inspection commands reduce guesswork when setting up mappings.
- It is config-first and stateless, which makes it easier to use in automation.

## What Not To Over-Explain Up Front

- A deep GORM vs `gorm.io/gen` tutorial
- every edge case in PostgreSQL type handling
- every config option in the first post

Lead with the value, then answer details when people ask.

## Good Places To Share It

- GitHub release notes
- Go / GORM / PostgreSQL Discord or Slack communities
- Reddit: `r/golang`, `r/postgresql`, `r/selfhosted` if the audience fits
- Hacker News "Show HN" if you want broader discovery
- LinkedIn or Mastodon if you want a quieter first signal

## Suggested Demo Flow

1. Show a small schema with one or two related tables.
2. Show the config.
3. Run the generator.
4. Show the generated models and query package.
5. Show a tiny usage snippet with `DbInit()` and a typed query.

Use the SQLite walkthrough in `examples/sqlite-quickstart` if you want a self-contained demo.
