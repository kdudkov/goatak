package staticfiles

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed static
var st embed.FS

func TestWalk(t *testing.T) {
	names := make([]string, 0)
	err := walkEmbed(st, func(fs embed.FS, fname string) error {
		names = append(names, fname)

		return nil
	})

	require.NoError(t, err)
	assert.Len(t, names, 23)
}
