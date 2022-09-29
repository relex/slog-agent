package util

// CollectFromChannel collect remaining items from a CLOSED channel
func CollectFromChannel[T any](closedChan <-chan T) []T {
	collected := make([]T, 0, len(closedChan)+20) // FIXME: Why was extra capacity needed? Was cross-thread len() inaccurate after close()?
	for item := range closedChan {
		collected = append(collected, item)
	}
	return collected
}
