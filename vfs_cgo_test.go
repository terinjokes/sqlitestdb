//go:build cgo

package sqlitestdb

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestVFSDelete(t *testing.T) {
	file, err := os.CreateTemp("", "sqlitestdb_vfs")
	assert.NilError(t, err)

	assert.NilError(t, vfsRemove(file.Name()))
}
