// Copyright 2024 Terin Stock.
// Copyright 2023 Peter Downs.
// SPDX-License-Identifier: MIT

package golangmigrator_test

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/terinjokes/sqlitestdb"
	"github.com/terinjokes/sqlitestdb/migrators/golangmigrator"
	"gotest.tools/v3/assert"
)

//go:embed migrations/*.sql
var exampleFS embed.FS

func TestMigrateFromDisk(t *testing.T) {
	t.Parallel()
	gm := golangmigrator.New("migrations")
	db := sqlitestdb.New(t, sqlitestdb.Config{Driver: "sqlite3"}, gm)
	testDB(t, db)
}

func TestMigrateFromEmbeddedFS(t *testing.T) {
	t.Parallel()
	gm := golangmigrator.New("migrations", golangmigrator.WithFS(exampleFS))
	db := sqlitestdb.New(t, sqlitestdb.Config{Driver: "sqlite3"}, gm)
	testDB(t, db)
}

func testDB(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	var version int
	err := db.QueryRowContext(ctx, "SELECT version FROM schema_migrations").Scan(&version)
	assert.NilError(t, err)
	assert.Equal(t, 2, version)

	var dirty bool
	err = db.QueryRowContext(ctx, "SELECT dirty FROM schema_migrations").Scan(&dirty)
	assert.NilError(t, err)
	assert.Assert(t, !dirty)

	var numUsers int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&numUsers)
	assert.NilError(t, err)
	assert.Equal(t, 0, numUsers)

	var numCats int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM cats").Scan(&numCats)
	assert.NilError(t, err)
	assert.Equal(t, 0, numCats)

	var numBlogPosts int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM blog_posts").Scan(&numBlogPosts)
	assert.NilError(t, err)
	assert.Equal(t, 0, numBlogPosts)
}
