package pm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetByHash(t *testing.T) {
	pm := NewPackageManager("aaa")
	pm.noSave = true

	data := []byte{1, 2, 3, 4, 5}
	hash := Hash(data)

	pi, err := pm.SaveData("uid1", "", "file.name", "application/octet-stream", data, nil)

	require.NoError(t, err)
	assert.Equal(t, hash, pi.Hash)
	assert.Equal(t, "uid1", pi.UID)

	pi1 := pm.GetByHash(hash)
	assert.NotNil(t, pi1)
	assert.Equal(t, "uid1", pi1.UID)

	pi2 := pm.GetByHash("aaa")
	assert.Nil(t, pi2)
}
