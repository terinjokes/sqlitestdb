sqlitestdb is a Go library that helps you write efficient SQLite-backed tests. It clones a template database to give each test a fully prepared and migrated SQLite database. Migrations are only ran once and each test gets its own database. A port of [pgtestdb](https://github.com/peterldowns/pgtestdb) to SQLite.


# How It Works

Each time you call `sqlitestdb.New` in your tests, sqlitestdb will check to see if a template database already exists. If not, it creates a new database and runs your migrations on it. Once the template exists, it then creates a test-specific database from that template.

Creating a new database from a template is very fast, on the order of milliseconds. And because sqlitestdb hashes your migrations to determine which template database to use, your migrations only end up being ran one time, regardless of how many tests or separate packages you have. This is true even across test runs; sqlitestdb will only run your migrations again if you change them in some way.

When a test succeeds the database it used is automatically deleted. When a test fails, the database it created is left behind, and test logs will indicate a SQLite URI you can use to open with `sqlite3` and explore what happened.

sqlitestdb is concurrency-safe, because each of your test gets its own database, you can and should run your tests in parallel.


# Install

```shell
go get github.com/terinjokes/sqlitestdb@latest
```


# Quickstart


## Example Test

Here&rsquo;s how to use `sqlitestdb.New` in a test to get a database.

```go
package sqlitestdb_test

// sqlitestdb uses the "database/sql" interface to interact with SQLite, you
// just have to bring your own driver. Here we're using the CGO-base driver,
// which registers a driver with the name "sqlite3"
import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/terinjokes/sqlitestdb"
)

// testNew should be called "TestNew" in your code, but is unexported here for GoDoc.
func testNew(t *testing.T) {
	// sqlitestdb is concurrency safe, enjoy yourself, run a lot of tests at once.
	t.Parallel()
	// You do not need to provide a database name when calling [New] or [Custom].
	conf := sqlitestdb.Config{Driver: "sqlite3"}

	// You'll want to use a real migrator, this is just an example.
	migrator := sqlitestdb.NoopMigrator{}
	db := sqlitestdb.New(t, conf, migrator)

	// If there was any error creating a template or instance database the
	// test would have failed with [testing.TB.Fatalf].
	var message string
	err := db.QueryRow("SELECT 'hellorld!'").Scan(&message)
	if err != nil {
		t.Fatalf("expected nil error: %+v\n", err)
	}

	if message != "hellord!" {
		t.Fatalf("expected message to be 'hellord!'")
	}
}
```


## Defining a Test Helper

The above example as a bit of boilerplate, you can define a test helper that calls `sqlitestdb.New` with the same settings and `sqlitestdb.Migrator` each time.

```go
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	conf := sqlitestdb.Config{Driver: "sqlite3"}
	migrator := sqlitestdb.NoopMigrator{}

	return sqlitestdb.New(t, conf, migrator)
}
```

Your test can then call the helper to get a valid `*sql.DB`.

```go
func TestExample(t *testing.T) {
	t.Parallel()
	db := NewDB(t)

	var message string
	err := db.QueryRow("SELECT 'hellorld!'").Scan(&message)
	if err != nil {
		t.Fatalf("expected nil error: %+v\n", err)
	}

	if message != "hellord!" {
		t.Fatalf("expected message to be 'hellord!'")
	}
}
```


## Choosing a Driver

As part of creating, migrating, and cloning for a new test database, sqlitestdb will need to use a SQLite implementation via the &ldquo;database/sql&rdquo; interface. In order to do so you must choose, register, and pass the name of your SQL driver. sqlitestdb is tested against [go-sqlite3](https://github.com/mattn/go-sqlite3), [sqlite](https://modernc.org/sqlite), and [libsql](https://github.com/tursodatabase/go-libsql). Other database/sql drivers for SQLite-like things may work.


## Using another database adapter

You can still use sqlitestdb even if you don&rsquo;t use the &ldquo;database/sql&rdquo; interface, such as if you&rsquo;re using an ORM-like database access layer, by calling `sqlitestdb.Custom`. You still need to register a driver for &ldquo;database/sql&rdquo; for sqlitestdb&rsquo;s internal behavior.

```go
package sqlitestdb_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/terinjokes/sqlitestdb"
)

// testCustom should be called "TestCustom" in your code, but is unexported here for GoDoc.
func testCustom(t *testing.T) {
	ctx := context.Background()
	conf := sqlitestdb.Custom(t, sqlitestdb.Config{Driver: "sqlite3"}, sqlitestdb.NoopMigrator{})

	db, err := sqlx.Connect("sqlite3", conf.URI())
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	defer db.Close()

	var message string
	if err = db.GetContext(ctx, &message, "SELECT 'hellord!'"); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	if message != "hellord!" {
		t.Fatalf("expected message to be 'hellord!'")
	}
}
```
