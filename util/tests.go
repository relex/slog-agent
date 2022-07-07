package util

import (
	"os"

	"golang.org/x/exp/slices"
)

// IsTestGenerationMode returns true if we're running in "go test -args gen" to generate expected test outputs
func IsTestGenerationMode() bool {
	return slices.Index(os.Args, "gen") != -1
}
