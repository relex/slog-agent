package util

import (
	"runtime/debug"
)

// Stack returns current stack trace as string
func Stack() string {
	return StringFromBytes(debug.Stack())
}
