package tdrop

import (
	"math/rand"
	"testing"

	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bmatch"
	"github.com/stretchr/testify/assert"
)

type dropTransformByRand struct {
	dropTransform
}

func (tf *dropTransformByRand) Transform(record *base.LogRecord) base.FilterResult {
	if !tf.matcher.Match(record) {
		return base.PASS
	}

	if tf.targetRate == 100 {
		tf.countDropped(record.RawLength)
		return base.DROP
	}

	if rand.Int31n(100) >= int32(tf.targetRate) {
		tf.totalMatched++
		tf.totalDropped++
		tf.countDropped(record.RawLength)
		return base.DROP
	}

	tf.totalMatched++
	tf.countRetained(record.RawLength)
	return base.PASS
}

func BenchmarkDropTransform1(b *testing.B) {
	numDropped := int64(0)
	numRetained := int64(0)

	tf := &dropTransform{
		matcher:       bmatch.LogMatcher{},
		countDropped:  func(length int) { numDropped++ },
		countRetained: func(length int) { numRetained++ },
		targetRate:    50,
		totalMatched:  0,
		totalDropped:  0,
	}

	passed, dropped := benchmarkDropTransform(b, tf)
	assert.Equal(b, numDropped, dropped)
	assert.Equal(b, numRetained, passed)
}

func BenchmarkDropTransform2(b *testing.B) {
	numDropped := int64(0)
	numRetained := int64(0)

	tf := &dropTransformByRand{dropTransform{
		matcher:       bmatch.LogMatcher{},
		countDropped:  func(length int) { numDropped++ },
		countRetained: func(length int) { numRetained++ },
		targetRate:    50,
		totalMatched:  0,
		totalDropped:  0,
	}}

	passed, dropped := benchmarkDropTransform(b, tf)
	assert.Equal(b, numDropped, dropped)
	assert.Equal(b, numRetained, passed)
}

func benchmarkDropTransform(b *testing.B, tf base.LogTransform) (int64, int64) {
	schema := base.MustNewLogSchema([]string{"msg"})
	record := schema.NewTestRecord1(base.LogFields{"hello"})
	record.RawLength = 5

	finalPassed := int64(0)
	finalDropped := int64(0)

	for iter := 0; iter < b.N; iter++ {
		if tf.Transform(record) == base.PASS {
			finalPassed++
		} else {
			finalDropped++
		}
	}

	b.Logf("result: pass=%d, drop=%d", finalPassed, finalDropped)
	return finalPassed, finalDropped
}
