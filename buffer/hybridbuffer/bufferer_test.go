package hybridbuffer

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/defs"
	"github.com/stretchr/testify/assert"
)

func testMatchChunkID(chunkID string) bool {
	return true
}

func TestBufferer(t *testing.T) {
	root, terr := ioutil.TempDir("", "example")
	if terr != nil {
		t.Fatal(terr)
	}
	defs.BufferMaxNumChunksInQueue = 100
	defs.BufferMaxNumChunksInMemory = 5
	mfactory := base.NewMetricFactory("testbuf_", nil, nil)
	buf := newBufferer(logger.Root(), root, "b1", testMatchChunkID, mfactory, 1048576, false).(*bufferer)
	buf.Launch()
	dir := buf.QueueDirPath()
	// produce logs
	noTimeout := make(chan time.Time)
	for i := 0; i < 50; i++ {
		c := base.LogChunk{
			ID:    fmt.Sprintf("%d", i),
			Data:  []byte(fmt.Sprintf("content-%d", i)),
			Saved: false,
		}
		buf.Accept(c, noTimeout)
	}
	for len(buf.feeder.outputChannel) < 5 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 44, len(buf.inputChannel)) // one chunk holding in loop
	// retrieve logs
	consumerArgs := buf.RegisterNewConsumer()
	t.Run("check unsaved chunks", func(tt *testing.T) {
		for i := 0; i < 5; i++ {
			c := <-buf.feeder.outputChannel
			assert.Equal(t, fmt.Sprintf("%d", i), c.ID, i)
			assert.Equal(t, fmt.Sprintf("content-%d", i), string(c.Data), i)
			assert.False(t, c.Saved, i)
			_, err := os.Stat(dir + "/" + c.ID)
			assert.True(t, os.IsNotExist(err), i)
			consumerArgs.OnChunkLeftover(c) // return chunks for saving
		}
	})
	t.Run("check saved and unsaved chunks", func(tt *testing.T) {
		savedStart := false
		for i := 5; i < 20; i++ {
			c := <-buf.feeder.outputChannel
			assert.Equal(t, fmt.Sprintf("%d", i), c.ID, i)
			assert.Equal(t, fmt.Sprintf("content-%d", i), string(c.Data), i)
			if !savedStart && c.Saved {
				savedStart = true
			}
			if savedStart {
				_, err := os.Stat(dir + "/" + c.ID)
				assert.Nil(t, err, i)
			}
			consumerArgs.OnChunkConsumed(c) // remove files if saved
		}
	})
	consumerArgs.OnFinished()
	buf.Destroy()
	assert.Zero(t, len(buf.inputChannel))
	assert.Zero(t, len(buf.feeder.outputChannel))
	assert.True(t, buf.feeder.outputClosed.Peek())
	assert.True(t, buf.Stopped().Peek())
	t.Run("check skipped/returned chunks", func(tt *testing.T) {
		for i := 0; i < 5; i++ {
			fn := fmt.Sprintf("%d", i)
			_, err := os.Stat(dir + "/" + fn)
			assert.Nil(t, err, i)
			assert.Nil(t, os.Remove(dir+"/"+fn))
		}
	})
	t.Run("check consumed chunks", func(tt *testing.T) {
		for i := 5; i < 20; i++ {
			fn := fmt.Sprintf("%d", i)
			_, err := os.Stat(dir + "/" + fn)
			assert.True(t, os.IsNotExist(err), i)
		}
	})
	t.Run("check unconsumed chunks", func(tt *testing.T) {
		for i := 20; i < 50; i++ {
			fn := fmt.Sprintf("%d", i)
			_, err := os.Stat(dir + "/" + fn)
			assert.Nil(t, err, i)
			assert.Nil(t, os.Remove(dir+"/"+fn))
		}
	})
}

func TestBuffererShutdown(t *testing.T) {
	root, terr := ioutil.TempDir("", "example")
	if terr != nil {
		t.Fatal(terr)
	}
	defs.BufferMaxNumChunksInQueue = 100
	defs.BufferMaxNumChunksInMemory = 5
	mfactory := base.NewMetricFactory("testbuf_shutdown_", nil, nil)
	buf := newBufferer(logger.Root(), root, "b2", testMatchChunkID, mfactory, 1048576, false).(*bufferer)
	buf.Launch()
	dir := buf.QueueDirPath()
	noTimeout := make(chan time.Time)
	for i := 0; i < 50; i++ {
		c := base.LogChunk{
			ID:    fmt.Sprintf("%d", i),
			Data:  []byte(fmt.Sprintf("content-%d", i)),
			Saved: false,
		}
		buf.Accept(c, noTimeout)
	}
	buf.Destroy()
	for i := 0; i < 50; i++ {
		fn := fmt.Sprintf("%d", i)
		_, err := os.Stat(dir + "/" + fn)
		assert.Nil(t, err, i)
		assert.Nil(t, os.Remove(dir+"/"+fn))
	}
	assert.Zero(t, len(buf.inputChannel))
	assert.Zero(t, len(buf.feeder.outputChannel))
	if dump, err := mfactory.DumpMetrics(true); assert.Nil(t, err) {
		assert.Equal(t, `testbuf_shutdown_buffer_consumed_chunks_total{storage="hybridBuffer"} 0
testbuf_shutdown_buffer_dropped_chunks_total{storage="hybridBuffer"} 0
testbuf_shutdown_buffer_io_errors_total{storage="hybridBuffer"} 0
testbuf_shutdown_buffer_leftover_chunks_total{storage="hybridBuffer"} 0
testbuf_shutdown_buffer_pending_chunks{storage="hybridBuffer"} 50
testbuf_shutdown_buffer_persistent_chunk_bytes{storage="hybridBuffer"} 490
testbuf_shutdown_buffer_persistent_chunks{storage="hybridBuffer"} 50
`, regexp.MustCompile(`testbuf_shutdown_buffer_.*state=.*\n`).ReplaceAllString(dump, ""))
		assert.Equal(t, uint64(50),
			mfactory.AddOrGetCounter("buffer_input_chunks_total", "", []string{"state", "storage"}, []string{"persistent", "hybridBuffer"}).Get()+
				mfactory.AddOrGetCounter("buffer_input_chunks_total", "", []string{"state", "storage"}, []string{"transient", "hybridBuffer"}).Get(),
		)
	}
}

func TestBuffererSendAllAtEnd(t *testing.T) {
	root, terr := ioutil.TempDir("", "example")
	if terr != nil {
		t.Fatal(terr)
	}
	defs.BufferMaxNumChunksInQueue = 100
	defs.BufferMaxNumChunksInMemory = 5
	mfactory := base.NewMetricFactory("testbuf_sendall_", nil, nil)
	buf := newBufferer(logger.Root(), root, "b3", testMatchChunkID, mfactory, 1048576, true).(*bufferer)
	buf.Launch()
	dir := buf.QueueDirPath()
	// produce logs
	noTimeout := make(chan time.Time)
	for i := 0; i < 50; i++ {
		c := base.LogChunk{
			ID:    fmt.Sprintf("%d", i),
			Data:  []byte(fmt.Sprintf("content-%d", i)),
			Saved: false,
		}
		buf.Accept(c, noTimeout)
	}
	for len(buf.feeder.outputChannel) < 5 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 44, len(buf.inputChannel)) // one chunk holding in loop
	// retrieve logs
	consumerArgs := buf.RegisterNewConsumer()
	t.Run("check unsaved chunks", func(tt *testing.T) {
		for i := 0; i < 5; i++ {
			c := <-buf.feeder.outputChannel
			assert.Equal(t, fmt.Sprintf("%d", i), c.ID, i)
			assert.Equal(t, fmt.Sprintf("content-%d", i), string(c.Data), i)
			assert.False(t, c.Saved, i)
			_, err := os.Stat(dir + "/" + c.ID)
			assert.NotNil(t, err, i)
		}
	})
	go func() {
		buf.Destroy()
		t.Run("check shutdown", func(tt *testing.T) {
			files, err := ioutil.ReadDir(dir)
			assert.Nil(t, err)
			assert.Zero(t, len(files))
		})
	}()
	time.Sleep(10 * time.Millisecond)
	t.Run("read chunks sent during shutdown", func(tt *testing.T) {
		for i := 5; i < 50; i++ {
			c := <-buf.feeder.outputChannel
			assert.Equal(t, fmt.Sprintf("%d", i), c.ID, i)
			assert.Equal(t, fmt.Sprintf("content-%d", i), string(c.Data), i)
			consumerArgs.OnChunkConsumed(c)
		}
		consumerArgs.OnFinished()
	})
	if dump, err := mfactory.DumpMetrics(true); assert.Nil(t, err) {
		assert.Equal(t, `testbuf_sendall_buffer_consumed_chunks_total{storage="hybridBuffer"} 45
testbuf_sendall_buffer_dropped_chunks_total{storage="hybridBuffer"} 0
testbuf_sendall_buffer_io_errors_total{storage="hybridBuffer"} 0
testbuf_sendall_buffer_leftover_chunks_total{storage="hybridBuffer"} 0
testbuf_sendall_buffer_pending_chunks{storage="hybridBuffer"} 5
testbuf_sendall_buffer_persistent_chunk_bytes{storage="hybridBuffer"} 0
testbuf_sendall_buffer_persistent_chunks{storage="hybridBuffer"} 0
`, regexp.MustCompile(`testbuf_sendall_buffer_.*state=.*\n`).ReplaceAllString(dump, ""))
		assert.Equal(t, uint64(50),
			mfactory.AddOrGetCounter("buffer_input_chunks_total", "", []string{"state", "storage"}, []string{"persistent", "hybridBuffer"}).Get()+
				mfactory.AddOrGetCounter("buffer_input_chunks_total", "", []string{"state", "storage"}, []string{"transient", "hybridBuffer"}).Get(),
		)
	}
}

func TestBuffererSpaceLimit(t *testing.T) {
	root, terr := ioutil.TempDir("", "example")
	if terr != nil {
		t.Fatal(terr)
	}
	defs.BufferMaxNumChunksInQueue = 100
	defs.BufferMaxNumChunksInMemory = 0          // force all chunks to be persisted
	defs.BufferShutDownTimeout = 1 * time.Second // quick shutdown in case of unread chunks
	mfactory := base.NewMetricFactory("testbuf_space_", nil, nil)
	buf := newBufferer(logger.Root(), root, "bspace", testMatchChunkID, mfactory, 100, true).(*bufferer)
	buf.Launch()
	dir := buf.QueueDirPath()
	noTimeout := make(chan time.Time)
	for i := 0; i < 50; i++ {
		c := base.LogChunk{
			ID:    fmt.Sprintf("%d", i),
			Data:  []byte(fmt.Sprintf("%010d", i)),
			Saved: false,
		}
		buf.Accept(c, noTimeout)
	}
	time.Sleep(100 * time.Millisecond)
	// start retriving chunks
	t.Log("start retriving from output")
	consumerArgs := buf.RegisterNewConsumer()
	t.Run("check unsaved and saved chunks", func(tt *testing.T) {
		for i := 0; i < 10; i++ {
			c := <-buf.feeder.outputChannel
			assert.Equal(t, fmt.Sprintf("%d", i), c.ID, i)
			assert.Equal(t, fmt.Sprintf("%010d", i), string(c.Data), i)
			if assert.True(t, c.Saved, i) {
				_, err := os.Stat(dir + "/" + c.ID)
				assert.Nil(t, err, i)
			}
			consumerArgs.OnChunkConsumed(c) // remove files if saved
		}
	})
	consumerArgs.OnFinished()
	// the rest should have been dropped
	assert.Zero(t, len(buf.inputChannel))
	assert.Zero(t, len(buf.feeder.outputChannel))
	buf.Destroy()
	if dump, err := mfactory.DumpMetrics(true); assert.Nil(t, err) {
		assert.Equal(t, `testbuf_space_buffer_consumed_chunks_total{storage="hybridBuffer"} 10
testbuf_space_buffer_dropped_chunks_total{storage="hybridBuffer"} 40
testbuf_space_buffer_input_chunks_total{state="persistent",storage="hybridBuffer"} 50
testbuf_space_buffer_input_chunks_total{state="transient",storage="hybridBuffer"} 0
testbuf_space_buffer_io_errors_total{storage="hybridBuffer"} 0
testbuf_space_buffer_leftover_chunks_total{storage="hybridBuffer"} 0
testbuf_space_buffer_pending_chunks{storage="hybridBuffer"} 0
testbuf_space_buffer_persistent_chunk_bytes{storage="hybridBuffer"} 0
testbuf_space_buffer_persistent_chunks{storage="hybridBuffer"} 0
testbuf_space_buffer_queued_chunks{state="persistent",storage="hybridBuffer"} 0
testbuf_space_buffer_queued_chunks{state="transient",storage="hybridBuffer"} 0
`, dump)
	}
}
