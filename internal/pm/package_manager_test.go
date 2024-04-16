package pm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByHash(t *testing.T) {
	pm := NewPackageManager("aaa")
	pm.noSave = true

	data := []byte{1, 2, 3, 4, 5}
	hash := Hash(data)

	pi := &PackageInfo{
		UID:                "uid1",
		SubmissionDateTime: time.Time{},
		Keywords:           nil,
		MIMEType:           "application/octet-stream",
		Size:               "",
		SubmissionUser:     "",
		PrimaryKey:         0,
		Hash:               "",
		CreatorUID:         "",
		Scope:              "",
		Name:               "file.name",
		Tool:               "",
	}

	err := pm.SaveFile(pi, data)

	require.NoError(t, err)
	assert.Equal(t, hash, pi.Hash)
	assert.Equal(t, "uid1", pi.UID)

	assert.Len(t, pm.GetByHash(hash), 1)

	pi2 := pm.GetByHash("aaa")
	assert.Nil(t, pi2)
}
