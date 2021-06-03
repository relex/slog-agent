package fluentdforward

import (
	"fmt"
	"sync"
	"time"
)

const chunkIDSuffix = ".ff"

var chunkIDLock = &sync.Mutex{}
var chunkIDEpochNano int64
var chunkIDSequence int32

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
