package util

// All returns true if all items in collection are matched
//     list := []string{"Bob", "David", "April"}
//     All(len(list), func(i) bool { return list[i] != "" })
func All(length int, test func(index int) bool) bool {
	for i := 0; i < length; i++ {
		if !test(i) {
			return false
		}
	}
	return true
}

// Any returns true if any item in collection is matched
//     list := []string{"Bob", "David", "April"}
//     Any(len(list), func(i) bool { return list[i] == "David" })
func Any(length int, test func(index int) bool) bool {
	for i := 0; i < length; i++ {
		if test(i) {
			return true
		}
	}
	return false
}

// Each calls the given func for [0 ... length - 1]
//     list := []string{"Bob", "David", "April"}
//     Each(len(list), func(i) { fmt.Println(list[i]) })
func Each(length int, action func(index int)) {
	for i := 0; i < length; i++ {
		action(i)
	}
}

// IndexOfString returns the index of target in the given string slice, or -1 if not found
func IndexOfString(slice []string, target string) int {
	for i, item := range slice {
		if item == target {
			return i
		}
	}
	return -1
}

// CopyByteSlice copies the slice to a newly-allocated one
func CopyByteSlice(slice []byte) []byte { // xx:inline
	return append([]byte(nil), slice...)
}

// ResetStringBuffer resets all elements in the given string slice and return [:0]
func ResetStringBuffer(buffer []string) []string {
	for i := range buffer {
		buffer[i] = ""
	}
	return buffer[:0]
}
