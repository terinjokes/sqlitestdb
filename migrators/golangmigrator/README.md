golangmigrator provides a sqlitestdb.Migrator that can be used to migrate the template database using [golang-migrate](https://github.com/golang-migrate/migrate).

Because `Hash()` requires calculating a unique hash based on the contents of the migrations, this implementation only supports reading migration files from disk or from an embedded filesystem.

```go
package db_test

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/terinjokes/sqlitestdb"
	"github.com/terinjokes/sqlitestdb/migrators/golangmigrator"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func TestMigrateFromDisk(t *testing.T) {
	gm := golangmigrator.New("migrations")
	db := sqlitestdb.New(t, sqlitestdb.Config{Driver: "sqlite3"}, gm)

	var version string
	err := db.QueryRowContext("sqlite_version()").Scan(&version)
	if err != nil {
		t.Fatalf("could not read from SQLite: %+v\n", err)
	}
}

func TestMigrateFromEmbeddedFS(t *testing.T) {
	gm := golangmigrator.New("migrations", golangmigrator.WithFS(migrationsFS))
	db := sqlitestdb.New(t, sqlitestdb.Config{Driver: "sqlite3"}, gm)

	var version string
	err := db.QueryRowContext("sqlite_version()").Scan(&version)
	if err != nil {
		t.Fatalf("could not read from SQLite: %+v\n", err)
	}
}
```
