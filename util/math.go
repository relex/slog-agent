package util

// MinInt returns the smaller one of given integers
func MinInt(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

// MaxInt returns the larger one of given integers
func MaxInt(x int, y int) int {
	if x > y {
		return x
	}
	return y
}
