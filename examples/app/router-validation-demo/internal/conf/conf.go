package conf

// Bootstrap is the root configuration loaded from YAML.
type Bootstrap struct {
	Server ServerConfig `json:"server" yaml:"server"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	HTTP HTTPConfig `json:"http" yaml:"http"`
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Addr string `json:"addr" yaml:"addr"`
}
