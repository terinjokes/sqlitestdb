// Copyright 2024 Terin Stock.
// Copyright 2023 Peter Downs.
// SPDX-License-Identifier: MIT

package golangmigrator

import (
	"context"
	"database/sql"
	"io/fs"

	"braces.dev/errtrace"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" // sqlite3 driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // "file://"" source driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/peterldowns/pgtestdb/migrators/common"
	"github.com/terinjokes/sqlitestdb"
)

// Option provides a way to configure the GolangMigrator struct and its behavior.
//
// golang-migrate documentation: https://github.com/golang-migrate/migrate
type Option func(*GolangMigrator)

// WithFS specifies a [fs.FS] from which to read the migration files.
// If not specified as an option to [New], the migrator will read from
// the real filesystem.
func WithFS(dir fs.FS) Option {
	return func(gm *GolangMigrator) {
		gm.FS = dir
	}
}

// GolangMigrator is a [sqlitestdb.Migrator] that uses golang-migrate to perform migrations.
//
// Because [Hash] requires calculating a unique hash based on the contents of
// the migrations, this implementation only supports reading migration files
// from disk or an embedded filesystem.
type GolangMigrator struct {
	MigrationsDir string
	FS            fs.FS
}

// New returns a [GolangMigrator], which implements sqlitestdb.Migrator
// using golang-migrate to perform up migrations.
func New(migrationsDir string, opts ...Option) *GolangMigrator {
	gm := &GolangMigrator{MigrationsDir: migrationsDir}
	for _, opt := range opts {
		opt(gm)
	}

	return gm
}

func (gm *GolangMigrator) Hash() (string, error) {
	return common.HashDirs(gm.FS, "*.sql", gm.MigrationsDir)
}

// Migrate runs migrate.Up() to migrate the template database.
func (gm *GolangMigrator) Migrate(_ context.Context, _ *sql.DB, templateConfig sqlitestdb.Config) error {
	var m *migrate.Migrate
	dsn := "sqlite3://" + templateConfig.Database

	switch {
	case gm.FS != nil:
		d, err := iofs.New(gm.FS, gm.MigrationsDir)
		if err != nil {
			return errtrace.Wrap(err)
		}

		m, err = migrate.NewWithSourceInstance("iofs", d, dsn)
		if err != nil {
			return errtrace.Wrap(err)
		}
	default:
		var err error
		m, err = migrate.New("file://"+gm.MigrationsDir, dsn)
		if err != nil {
			return errtrace.Wrap(err)
		}
	}

	defer m.Close()
	return errtrace.Wrap(m.Up())
}
