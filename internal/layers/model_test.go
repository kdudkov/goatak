package layers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYaml(t *testing.T) {
	s := `
- name: OSM
  url: "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
  max_zoom: 19
  server_parts: ["a", "b", "c"]
`

	l := make([]*LayerDescription, 0)

	require.NoError(t, yaml.Unmarshal([]byte(s), &l))
	require.Len(t, l, 1)
	require.Len(t, l[0].ServerParts, 3)
}
