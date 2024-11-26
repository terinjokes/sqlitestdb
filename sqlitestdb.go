// Copyright 2024 Terin Stock.
// Copyright 2023 Peter Downs.
// SPDX-License-Identifier: MIT

package sqlitestdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"braces.dev/errtrace"
	"github.com/terinjokes/sqlitestdb/once"
	"golang.org/x/mod/semver"
)

// minVersion is the minimium version of SQLite that can be used with sqlitestdb,
// and is the version where "VACUUM INTO" was added.
const minVersion = "v3.27.0"

type Migrator interface {
	Hash() (string, error)
	Migrate(context.Context, *sql.DB, Config) error
}

// Config contains the details needed to handle a SQLite database.
type Config struct {
	Driver   string // The driver name used in sql.Open(). "sqlite3" (mattn/go-sqlite3), "sqlite" (modernc), or "libsql" (LibSQL)
	Database string // The path to the database file.
}

// URI returns a URI string needed to open the SQLite database.
//
//	"file:/path/to/database.sql?options=..."
//
// This should be a subset of the URIs defined by [SQLite URIs], but may contain
// driver-specific options.
//
// [SQLite URIs]: https://www.sqlite.org/uri.html
func (c Config) URI() string {
	return fmt.Sprintf("file:%s", c.Database)
}

// Connect calls [sql.Open] and connects to the database.
func (c Config) Connect() (*sql.DB, error) {
	db, err := sql.Open(c.Driver, c.URI())
	if err != nil {
		return nil, errtrace.Wrap(err)
	}

	return db, nil
}

// New creates a fresh SQLite database and connects. This database is created by
// cloning a database migrated by the provided migrator. It is safe to call
// concurrently, but running the same migrations across multiple packages at the
// same time may race. If there is an error creating the database, the test will
// be immediately failed with [testing.TB.Fatalf].
//
// The [Config.Database] field may be left blank, as a new database will be created.
//
// If this methods succeeds, it will call [testing.TB.Log] with the SQLite URI of
// the test database, so that you may open the database manually and see what failed.
//
// If this method succeeds and your test succeeds, the database will be removed
// as part of the test cleanup process.
func New(t testing.TB, config Config, migrator Migrator) *sql.DB {
	t.Helper()
	_, db := create(t, config, migrator)
	return db
}

// Custom is like [New] but after creating the new database instance, it closes
// any connections and returns the configuration details od the test database,
// so that you can connect to it explicitly, potentnially via a different SQL
// interface.
func Custom(t testing.TB, config Config, migrator Migrator) *Config {
	t.Helper()
	c, db := create(t, config, migrator)
	if err := db.Close(); err != nil {
		t.Fatalf("could not close test database %q: %+v", config.Database, err)
	}

	return c
}

// create contains the implementation of [New] and [Custom], and is responsible
// for actually creating the instance database to be used by a testcase.
func create(t testing.TB, config Config, migrator Migrator) (*Config, *sql.DB) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tpl, err := getOrCreateTemplate(ctx, config, migrator)
	if err != nil {
		t.Fatalf("could not create template database: %+v", err)
	}

	tplDB, err := tpl.config.Connect()
	if err != nil {
		t.Fatalf("could not open template datbase: %+v", err)
	}

	var version string
	if err := tplDB.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		t.Fatalf("could not determine SQLite version: %+v", err)
	}

	if semver.Compare("v"+version, minVersion) < 0 {
		t.Fatalf("SQLite version too old (found v%s, minimium required %s)", version, minVersion)
	}

	instance, err := createInstance(ctx, tplDB, *tpl)
	if err != nil {
		t.Fatalf("could not create instance: %+v", err)
	}

	t.Logf("sqlitestdb: %s", instance.URI())

	if err := tplDB.Close(); err != nil {
		t.Fatalf("could not close template DB: %+v", err)
	}

	db, err := instance.Connect()
	if err != nil {
		t.Fatalf("could not connect to instance database: %+v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("could not close instance database %q: %+v", instance.Database, err)
		}

		if t.Failed() {
			return
		}

		os.Remove(instance.Database)
	})

	return instance, db
}

// templateState keeps the state of a single template, so that each program only
// attempts to create and migrate the template at most once.
type templateState struct {
	config Config
	hash   string
}

var templates = once.NewMap[string, templateState]()

// getOrCreateTemplate will get-or-create a template, synchronizing calls using the
// templates map, so that each template is get-or-created at most once.
//
// If there was an error during template creation an error will be returned by
// the inner function, which will cause the error to be returned to all callers
// during this program's execution.
func getOrCreateTemplate(ctx context.Context, config Config, migrator Migrator) (*templateState, error) {
	mhash, err := migrator.Hash()
	if err != nil {
		return nil, err
	}

	return errtrace.Wrap2(templates.Set(mhash, func() (*templateState, error) {
		tpl := templateState{}
		tpl.config = config
		tpl.config.Database = filepath.Join(os.TempDir(), "sqlitestdb_tpl_"+mhash+".sqlite")
		tpl.hash = mhash

		if _, err := os.Stat(tpl.config.Database); err == nil {
			return &tpl, nil
		}

		if err := ensureTemplate(ctx, tpl.config, migrator); err != nil {
			os.Remove(tpl.config.Database)
			return nil, errtrace.Wrap(err)
		}

		return &tpl, nil
	}))
}

// ensureTemplate creates a template database using the config and migrator. If there
// was an error during creation it will be returned.
func ensureTemplate(ctx context.Context, config Config, migrator Migrator) error {
	db, err := config.Connect()
	if err != nil {
		return errtrace.Wrap(err)
	}
	defer db.Close()

	// As SQLite doesn't have advisory locks, the best we can do is enable
	// exclusive [locking-mode], which will prevent reads and writes from other
	// processes.
	//
	// Note that taking the exclusive lock requires a write, so this still allows
	// migrations which exec another program to succeed.
	//
	// This uses [sql.DB.QueryRowContext] instead of [sql.DB.ExecContext] due to libsql
	// returning an error, instead of non-nil [sql.Result], when Exec is used for any
	// statements that return rows. See [tursodatabase/go-libsql#28].
	//
	// [locking-mode]: https://www.sqlite.org/pragma.html#pragma_locking_mode
	// [tursodatabase/go-libsql#28]: https://github.com/tursodatabase/go-libsql/issues/28
	var ignored string
	row := db.QueryRowContext(ctx, "PRAGMA main.locking_mode=EXCLUSIVE")
	err = row.Scan(&ignored)
	if err != nil {
		return errtrace.Wrap(err)
	}

	if err := migrator.Migrate(ctx, db, config); err != nil {
		return errtrace.Wrap(err)
	}

	return nil
}

// createInstance creates a new test database by cloning a template.
func createInstance(ctx context.Context, baseDB *sql.DB, template templateState) (*Config, error) {
	baseConn, err := baseDB.Conn(ctx)
	if err != nil {
		return nil, errtrace.Wrap(err)
	}
	defer baseConn.Close()

	id, err := randomID()
	if err != nil {
		return nil, errtrace.Wrap(err)
	}

	testConfig := template.config
	testConfig.Database = filepath.Join(os.TempDir(), "sqlitestdb_tpl_"+template.hash+"_inst_"+id+".sqlite")

	// Since we can be reasonably sure the template database is free of any transactions
	// at this point, we can use the "VACUUM INTO" statement to create a new database.
	// This allows us to avoid the Online Backup API, which would require separate
	// implementations for github.com/mattn/go-sqlite3 and modernc.org/sqlite, as the
	// backup API requires acquiring the raw driver connection.
	if _, err := baseDB.ExecContext(ctx, "VACUUM INTO ?", testConfig.URI()); err != nil {
		return nil, errtrace.Wrap(err)
	}

	return &testConfig, nil
}

// randomID is a helper for coming up with the names of the instance databases.
// It uses 32 random bits in the name, which means collisions are unlikely.
func randomID() (string, error) {
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", errtrace.Wrap(err)
	}
	return hex.EncodeToString(bytes), nil
}

// NoopMigrator fulfills the [Migrator] interface, but it does absolutely
// nothing. You can use this to get empty databases in your tests, or if
// you're trying out sqlitestdb (hello!) and aren't sure which migrator
// to use yet.
type NoopMigrator struct{}

func (m NoopMigrator) Hash() (string, error) {
	return "noop", nil
}

func (m NoopMigrator) Migrate(ctx context.Context, _ *sql.DB, _ Config) error {
	return nil
}
