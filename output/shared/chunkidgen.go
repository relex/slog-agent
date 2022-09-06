package shared

import (
	"fmt"
	"sync"
	"time"
)

type chunkIDGenerator struct {
	sync.Mutex
	epochNano int64
	sequence  int32
	suffix    string
}

func newChunkIDGenerator(suffix string) *chunkIDGenerator {
	return &chunkIDGenerator{
		Mutex:     sync.Mutex{},
		epochNano: 0,
		sequence:  0,
		suffix:    suffix,
	}
}

// generate returns the next chunk ID, which consists of a nanosecond timestamp and a sequence number
// The sequence number is incremented by one every time until the time is changed
func (generator *chunkIDGenerator) Generate() string {
	generator.Lock()
	nextTimestamp := time.Now().UnixNano()
	if nextTimestamp > generator.epochNano {
		generator.epochNano = nextTimestamp
		generator.sequence = 0
	} else {
		generator.sequence++
	}
	nextSequence := generator.sequence
	generator.Unlock()
	return fmt.Sprintf("%019d-%08d"+generator.suffix, nextTimestamp, nextSequence)
}
