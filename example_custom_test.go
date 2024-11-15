// [[file:README.org::*Using another database adapter][Using another database adapter:1]]
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
// Using another database adapter:1 ends here
