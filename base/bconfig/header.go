package bconfig

// Header defines the common parts of *Config implementations
type Header struct {
	Type string `yaml:"type"`
}

// GetType returns the type name
func (header *Header) GetType() string {
	return header.Type
}
