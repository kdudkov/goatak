package layers

type LayerDescription struct {
	Name        string   `yaml:"name" json:"name"`
	URL         string   `yaml:"url" json:"url"`
	MinZoom     int      `yaml:"min_zoom" json:"min_zoom,omitempty"`
	MaxZoom     int      `yaml:"max_zoom" json:"max_zoom,omitempty"`
	Tms         bool     `yaml:"tms" json:"tms,omitempty"`
	TileType    string   `yaml:"tile_type" json:"tile_type,omitempty"`
	ServerParts []string `yaml:"server_parts" json:"server_parts,omitempty"`
}

func GetDefaultLayers() []*LayerDescription {
	return []*LayerDescription{
		{
			Name:        "Google Hybrid",
			URL:         "http://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo&scale=2",
			MaxZoom:     20,
			ServerParts: []string{"0", "1", "2", "3"},
		},
	}
}
