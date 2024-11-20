// Copyright 2024 Terin Stock.
// Copyright 2023 Peter Downs.
// SPDX-License-Identifier: MIT

// libsql has C symbol conflicts with go-sqlite3, so it is being tested separately here.

package libsql_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/pgtestdb/migrators/common"
	"github.com/terinjokes/sqlitestdb"
	_ "github.com/tursodatabase/go-libsql"
	"gotest.tools/v3/assert"
)

func New(t *testing.T) *sql.DB {
	t.Helper()
	dbconf := sqlitestdb.Config{
		Driver: "libsql",
	}
	m := defaultMigrator()
	return sqlitestdb.New(t, dbconf, m)
}

func TestLibSQL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := New(t)

	rows, err := db.QueryContext(ctx, "SELECT name FROM cats ORDER BY name ASC")
	assert.NilError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		assert.NilError(t, rows.Scan(&name))
		names = append(names, name)
	}

	assert.DeepEqual(t, names, []string{"daisy", "sunny"})
}

func defaultMigrator() sqlitestdb.Migrator {
	// Separate the table creation and insertion into two separate steps
	// as libsql has [a bug] where only the first statement in a
	// [sql.DB.ExecContext] is ran.
	//
	// [a bug]: https://github.com/tursodatabase/go-libsql/issues/22
	return &sqlMigrator{
		migrations: []string{`
			-- the "migration"
			CREATE TABLE cats (
				id INTEGER PRIMARY KEY,
				name TEXT
			);
        `, `
			INSERT INTO cats (name) VALUES ('daisy'), ('sunny');
        `,
		},
	}
}

type sqlMigrator struct {
	migrations []string
}

func (m *sqlMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash()
	for _, migration := range m.migrations {
		hash.Add([]byte(migration))
	}
	return hash.String(), nil
}

func (m *sqlMigrator) Migrate(ctx context.Context, db *sql.DB, _ sqlitestdb.Config) error {
	for _, migration := range m.migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}
