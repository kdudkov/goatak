package staticfiles

import (
	"embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

//go:embed static
var st embed.FS

func TestWalk(t *testing.T) {
	names := make([]string, 0)
	err := walkEmbed(st, func(fs embed.FS, fname string) {
		names = append(names, fname)
	})

	require.NoError(t, err)
	assert.Len(t, names, 23)
}
