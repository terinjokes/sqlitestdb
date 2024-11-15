// [[file:README.org::*Example Test][Example Test:1]]
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
// Example Test:1 ends here
