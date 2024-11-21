//go:build cgo

package sqlitestdb

/*
#include <sqlite3.h>
int sqlitestdb_vfs_delete(sqlite3_vfs* vfs, const char *zName) {
   return vfs->xDelete(vfs, zName, 0);
}
*/
import "C"
import (
	"context"
	"database/sql"

	"braces.dev/errtrace"
	"github.com/mattn/go-sqlite3"
)

func vfsDBFilename(db *sql.DB) string {
	conn, _ := db.Conn(context.Background())
	defer conn.Close()

	var filename string
	conn.Raw(func(driverConn any) error {
		switch t := driverConn.(type) {
		case *sqlite3.SQLiteConn:
			filename = t.GetFilename("")
		}
		return nil
	})
	return filename
}

func vfsRemove(filename string) error {
	vfs := C.sqlite3_vfs_find(nil)
	if vfs == nil {
		return errtrace.New("unable to find VFS")
	}

	ok := C.sqlitestdb_vfs_delete(vfs, C.CString(filename))
	if ok != C.SQLITE_OK {
		return errtrace.New("unable to delete file")
	}

	return nil
}
