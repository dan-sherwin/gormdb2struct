# SQLite Quickstart

This example is the fastest way to see what `gormdb2struct` does without setting up PostgreSQL first.

It creates a tiny helpdesk-style SQLite schema with:
- `tickets`
- `ticket_comments`
- a one-to-many relation wired in through `ExtraFields`

## Prerequisites

- `gormdb2struct` installed
- `sqlite3` CLI available

## Files

- `schema.sql`
  Creates the sample SQLite schema
- `gormdb2struct.toml`
  Example config for this schema

## Run It

From this directory:

```bash
rm -f helpdesk.db
sqlite3 helpdesk.db < schema.sql
gormdb2struct ./gormdb2struct.toml
```

That will generate code into `./generated`.

## What You Get

After running the generator, you should see:

- `generated/models`
  Generated GORM model structs
- `generated/db_sqlite.go`
  Optional SQLite `DbInit`
- additional generated files under `generated/`
  Typed `gorm.io/gen` query helpers

## Why This Example Is Useful

This example shows a realistic but compact workflow:

- start from an existing schema
- keep configuration in TOML
- generate models and typed query helpers
- add relationship helper fields without hand-editing generated files

## Example Config

The included config is intentionally small:

- SQLite dialect
- explicit object list
- generated SQLite `DbInit`
- one `ExtraFields` relationship so `Ticket` gets a `Comments` collection

## Example Usage

Once generated, code in the same module can use the package like this:

```go
package main

import (
	"log"

	g "github.com/dan-sherwin/gormdb2struct/examples/sqlite-quickstart/generated"
	m "github.com/dan-sherwin/gormdb2struct/examples/sqlite-quickstart/generated/models"
)

func main() {
	if err := g.DbInit(); err != nil {
		log.Fatal(err)
	}

	subject := "Login broken"
	status := "open"
	priority := "high"
	requester := "user@example.com"

	ticket := m.Ticket{
		Subject:        &subject,
		Status:         &status,
		Priority:       &priority,
		RequesterEmail: &requester,
	}

	if err := g.DB.Create(&ticket).Error; err != nil {
		log.Fatal(err)
	}

	openTickets, err := g.Ticket.Where(g.Ticket.Status.Eq("open")).Find()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("loaded %d open ticket(s)", len(openTickets))
}
```

The important part there is that `DbInit()` calls `SetDefault(...)`, so you can use both:

- `g.DB` for plain GORM access
- package-level typed query objects like `g.Ticket` and `g.TicketComment`

## Clean Up

If you want to rerun the example from scratch:

```bash
rm -rf generated helpdesk.db
```
