package sqlitestdb

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/peterldowns/pgtestdb/migrators/common"
	"gotest.tools/v3/assert"
)

func TestRemovingTemplateDatabaseOnError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errm := &sqlMigrator{
		migrations: []string{
			"SELECT x FROM nothing",
		},
	}

	errh, err := errm.Hash()
	assert.NilError(t, err)

	dbconf := Config{Driver: "sqlite3", Database: "/tmp/sqlitestdb_tpl_" + errh + ".sqlite"}

	errdb, err := getOrCreateTemplate(ctx, dbconf, errm)
	assert.Assert(t, err != nil)
	assert.Assert(t, errdb == nil)

	_, err = os.Stat(dbconf.Database)
	assert.Assert(t, errors.Is(err, os.ErrNotExist))
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

func (m *sqlMigrator) Migrate(ctx context.Context, db *sql.DB, _ Config) error {
	for _, migration := range m.migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}
