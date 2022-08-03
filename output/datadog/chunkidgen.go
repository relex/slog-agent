package datadog

import (
	"fmt"
	"sync"
	"time"
)

// TODO: Share this stuff between outputs, since they are duplicates

const chunkIDSuffix = ".dd"

var (
	chunkIDLock      = &sync.Mutex{}
	chunkIDEpochNano int64
	chunkIDSequence  int32
)

// nextChunkID returns the next chunk ID, which consists of a nanosecond timestamp and a sequence number
// The sequence number is incremented by one every time until the time is changed
func nextChunkID() string {
	chunkIDLock.Lock()
	nextTimestamp := time.Now().UnixNano()
	if nextTimestamp > chunkIDEpochNano {
		chunkIDEpochNano = nextTimestamp
		chunkIDSequence = 0
	} else {
		chunkIDSequence++
	}
	nextSequence := chunkIDSequence
	chunkIDLock.Unlock()
	return fmt.Sprintf("%019d-%08d"+chunkIDSuffix, nextTimestamp, nextSequence)
}
