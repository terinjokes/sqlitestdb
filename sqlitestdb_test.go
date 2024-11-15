// Copyright 2024 Terin Stock.
// Copyright 2023 Peter Downs.
// SPDX-License-Identifier: MIT

package sqlitestdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/peterldowns/pgtestdb/migrators/common"
	"github.com/terinjokes/sqlitestdb"
	"gotest.tools/v3/assert"
	_ "modernc.org/sqlite"
)

func New(t *testing.T) *sql.DB {
	t.Helper()
	dbconf := sqlitestdb.Config{
		Driver: "sqlite3",
	}
	m := defaultMigrator()
	return sqlitestdb.New(t, dbconf, m)
}

func TestNew(t *testing.T) {
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

func TestCustom(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbconf := sqlitestdb.Config{
		Driver: "sqlite3",
	}
	m := defaultMigrator()

	config := sqlitestdb.Custom(t, dbconf, m)
	assert.Assert(t, dbconf != *config)

	db, err := sqlx.Connect("sqlite3", config.URI())
	assert.NilError(t, err)
	defer db.Close()

	type Cat struct {
		ID   int
		Name string
	}
	var cat Cat
	assert.NilError(t, db.GetContext(ctx, &cat, "SELECT * FROM cats WHERE id = 1"))
	assert.DeepEqual(t, cat, Cat{ID: 1, Name: "daisy"})
}

func TestParallel1(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			db := New(t)

			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) from cats").Scan(&count)
			assert.NilError(t, err)
			assert.Equal(t, 2, count)
		})
	}
}

func TestParallel2(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_%d", i), func(t *testing.T) {
			t.Parallel()
			db := New(t)

			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) from cats").Scan(&count)
			assert.NilError(t, err)
			assert.Equal(t, 2, count)
		})
	}
}

func TestDifferentHashesAlwaysResultInDifferentDatabases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbconf := sqlitestdb.Config{Driver: "sqlite3"}
	// These two migrators have different hashes and they create databases with different schemas.
	// The xxx schema contains a table xxx, the yyy schema contains a table yyy.
	xxxm := &sqlMigrator{
		migrations: []string{
			"CREATE TABLE xxx (id INTEGER PRIMARY KEY)",
		},
	}
	yyym := &sqlMigrator{
		migrations: []string{
			"CREATE TABLE yyy (id INTEGER PRIMARY KEY)",
		},
	}
	// These two migrators should have different hashes.
	yyyh, err := yyym.Hash()
	assert.NilError(t, err)
	xxxh, err := xxxm.Hash()
	assert.NilError(t, err)
	assert.Assert(t, yyyh != xxxh)

	// Create two databases. They _should_ have different schemas.
	xxxdb := sqlitestdb.New(t, dbconf, xxxm)
	yyydb := sqlitestdb.New(t, dbconf, yyym)

	// But, the bug is that due to use of t.Once(), they will actually have the
	// same schema.  One of these two statements will always fail! Due to
	// ordering in this test, the xxx database gets created first, and the yyy
	// database will re-use that template (mistakenly!).
	//
	// In the case where we're writing a package and have multiple tests in
	// parallel, the order is dependent on whichever test runs first, which is
	// really annoying to debug.
	var countXXX int
	err = xxxdb.QueryRowContext(ctx, "select count(*) from xxx").Scan(&countXXX)
	if err != nil {
		assert.Equal(t, 0, countXXX)
	}
	var countYYY int
	err = yyydb.QueryRowContext(ctx, "select count(*) from yyy").Scan(&countYYY)
	if err != nil {
		assert.Equal(t, 0, countXXX)
	}
}

func TestWithMattnAndModernC(t *testing.T) {
	t.Parallel()
	mattnConfig := sqlitestdb.Config{Driver: "sqlite3"}
	moderncConfig := sqlitestdb.Config{Driver: "sqlite"}

	migrator := defaultMigrator()
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_mattn_%d", i), func(t *testing.T) {
			t.Parallel()
			_ = sqlitestdb.New(t, mattnConfig, migrator)
		})
	}
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("subtest_modernc_%d", i), func(t *testing.T) {
			t.Parallel()
			_ = sqlitestdb.New(t, moderncConfig, migrator)
		})
	}
}

func defaultMigrator() sqlitestdb.Migrator {
	return &sqlMigrator{
		migrations: []string{`
			-- the "migration"
			CREATE TABLE cats (
				id INTEGER PRIMARY KEY,
				name TEXT
			);
			INSERT INTO cats (name) VALUES ('daisy'), ('sunny')
        `},
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
