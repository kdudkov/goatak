package pm

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByHash(t *testing.T) {
	pm := NewPackageManager(os.TempDir())
	pm.noSave = true

	data := []byte{1, 2, 3, 4, 5}

	r := bytes.NewReader(data)

	hash := Hash(data)

	pi := &PackageInfo{
		UID:                "uid1",
		SubmissionDateTime: time.Time{},
		Keywords:           nil,
		MIMEType:           "application/octet-stream",
		Size:               0,
		SubmissionUser:     "",
		PrimaryKey:         0,
		Hash:               "",
		CreatorUID:         "",
		Scope:              "",
		Name:               "file.name",
		Tool:               "",
	}

	err := pm.SaveFile(pi, r)

	require.NoError(t, err)
	assert.Equal(t, hash, pi.Hash)
	assert.Equal(t, "uid1", pi.UID)

	assert.Len(t, pm.GetByHash(hash), 1)

	pi2 := pm.GetByHash("aaa")
	assert.Nil(t, pi2)
}

func Hash(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
