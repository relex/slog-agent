package bconfig

// BaseConfig contains basic properties required for all Config types
type BaseConfig interface {
	// GetType returns the type name
	GetType() string
}
