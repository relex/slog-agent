package util

import (
	"os"
)

// IsTestGenerationMode returns true if we're running in "go test -args gen" to generate expected test outputs
func IsTestGenerationMode() bool {
	return IndexOfString(os.Args, "gen") != -1
}
