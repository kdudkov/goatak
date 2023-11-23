package staticfiles

import (
	"embed"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed static
var st embed.FS

func TestWalk(t *testing.T) {
	names := make([]string, 0)
	err := walkEmbed(st, func(fs embed.FS, fname string) {
		names = append(names, fname)
	})

	assert.NoError(t, err)
	assert.Equal(t, len(names), 23)
}
