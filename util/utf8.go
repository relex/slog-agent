package util

func CleanUTF8(s string) string {
	if s == "" {
		return s
	}
	// correct encoding is https://en.wikipedia.org/wiki/UTF-8
	// too much work; just cut everything after the last ASCII byte
	end := findLastEndOfASCII(s)
	return s[:end]
}

func findLastEndOfASCII(s string) int {
	n := len(s)
	for i := n - 1; i >= 0; i-- {
		b := s[i]

		if b <= 0x7F {
			return i + 1
		}
	}
	return 0
}
