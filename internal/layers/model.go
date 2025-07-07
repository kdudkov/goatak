package layers

type LayerDescription struct {
	Name        string   `yaml:"name" json:"name" koanf:"name"`
	URL         string   `yaml:"url" json:"url" koanf:"url"`
	MinZoom     int      `yaml:"min_zoom" json:"min_zoom,omitempty" koanf:"min_zoom"`
	MaxZoom     int      `yaml:"max_zoom" json:"max_zoom,omitempty" koanf:"max_zoom"`
	Tms         bool     `yaml:"tms" json:"tms,omitempty" koanf:"tms"`
	TileType    string   `yaml:"tile_type" json:"tile_type,omitempty" koanf:"tile_type"`
	ServerParts []string `yaml:"server_parts" json:"server_parts,omitempty" koanf:"server_parts"`
}

func GetDefaultLayers() []*LayerDescription {
	return []*LayerDescription{
		{
			Name:        "Google Hybrid",
			URL:         "https://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}&s=Galileo&scale=2",
			MaxZoom:     20,
			ServerParts: []string{"0", "1", "2", "3"},
		},
	}
}
