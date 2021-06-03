package tcplistener

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mlrHelper struct {
	reader   *multiLineReader
	output   []string
	fetchEnd int
}

func newMultiLineReaderHelper(input chan string) *mlrHelper {
	helper := &mlrHelper{
		reader:   nil,
		output:   make([]string, 0, 100),
		fetchEnd: 0,
	}
	read := func(p []byte) (n int, err error) {
		select {
		case block := <-input:
			if len(p) < len(block) {
				return -1, fmt.Errorf("overflow")
			}
			n := copy(p, block)
			return n, nil
		default:
			return 0, io.EOF
		}
	}
	test := func(s []byte) bool {
		return len(s) > 0 && s[0] == '>'
	}
	consume := func(s []byte) {
		helper.output = append(helper.output, string(s)) // force copy
	}
	helper.reader = newMultiLineReader(read, test, 0, 20, consume)
	return helper
}

func (h *mlrHelper) nextOutput() int {
	h.fetchEnd++
	return h.fetchEnd
}

func (h *mlrHelper) fetchOutput() string {
	return h.output[h.fetchEnd-1]
}

func TestMultiLineReader(t *testing.T) {
	input := make(chan string, 100)
	h := newMultiLineReaderHelper(input)
	{
		input <- "> Something\n"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, len(h.output))
		input <- "> Some"
		assert.Nil(t, h.reader.Read())
		input <- "Thing"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, len(h.output))
		input <- "\nElse"
		assert.Nil(t, h.reader.Read())
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> Something", h.fetchOutput())
		}
	}
	{
		input <- "\n> Line 1\n."
		assert.Nil(t, h.reader.Read())
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> SomeThing\nElse", h.fetchOutput())
		}
		input <- "Line 2\n."
		assert.Nil(t, h.reader.Read())
		input <- "Line 3\n"
		assert.Nil(t, h.reader.Read())
		input <- "x\n> MLine 1\nMLine 2\n> A\nB\nC\n> Next\n"
		assert.Nil(t, h.reader.Read())
		if assert.Equal(t, h.nextOutput(), len(h.output)-2) {
			assert.Equal(t, "> Line 1\n.Line 2\n.Line 3\nx", h.fetchOutput())
		}
		if assert.Equal(t, h.nextOutput(), len(h.output)-1) {
			assert.Equal(t, "> MLine 1\nMLine 2", h.fetchOutput())
		}
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> A\nB\nC", h.fetchOutput())
		}
		input <- "End\n"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, h.fetchEnd, len(h.output))
		h.reader.Flush()
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> Next\nEnd", h.fetchOutput())
		}
	}
	testMultiLineReaderOverflow(t, input, h)
	testMultiLineReaderFlush(t, input, h)
}

func testMultiLineReaderOverflow(t *testing.T, input chan<- string, h *mlrHelper) {
	t.Run("overflow all garbage", func(tt *testing.T) {
		input <- "01234567890123456789"
		assert.Nil(t, h.reader.Read())
		input <- "01234567890123456789"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, 40, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		input <- "0123456789"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		assert.Equal(t, h.fetchEnd, len(h.output))
	})
	t.Run("overflow single record", func(tt *testing.T) {
		input <- "> abcdefgh0123456789"
		assert.Nil(t, h.reader.Read())
		input <- "01234567890123456789"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, 40, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		input <- "ABCDEFGHIJKLMNOP"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> abcdefgh012345678901234567890123456789ABCDEFGHIJKLMNOP", h.fetchOutput())
		}
	})
	t.Run("overflow mid record", func(tt *testing.T) {
		input <- "012345678\n> abcdefgh"
		assert.Nil(t, h.reader.Read())
		input <- "01234567890123456789"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, 40, h.reader.offsetAppend)
		assert.Equal(t, 10, h.reader.offsetSearch)
		input <- "ABCDEFGHIJKLMNOP"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> abcdefgh01234567890123456789ABCDEFGHIJKLMNOP", h.fetchOutput())
		}
	})
	t.Run("overflow two records", func(tt *testing.T) {
		input <- ">12345678\n> abcdefgh"
		assert.Nil(t, h.reader.Read())
		input <- "01234567890123456789"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, 40, h.reader.offsetAppend)
		assert.Equal(t, 10, h.reader.offsetSearch)
		assert.Equal(t, h.fetchEnd, len(h.output))
		input <- "ABCDEFGHIJKLMNOP"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		if assert.Equal(t, h.nextOutput(), len(h.output)-1) {
			assert.Equal(t, ">12345678", h.fetchOutput())
		}
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> abcdefgh01234567890123456789ABCDEFGHIJKLMNOP", h.fetchOutput())
		}
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
	})
	t.Run("overflow all newlines", func(tt *testing.T) {
		input <- "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"
		assert.Nil(t, h.reader.Read())
		input <- "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, 39, h.reader.offsetAppend)
		assert.Equal(t, 39, h.reader.offsetSearch)
		assert.Equal(t, h.fetchEnd, len(h.output))
		input <- "\n\n\n\n\n\n\n\n\n> helloxxx"
		assert.Nil(t, h.reader.Read())
		assert.Zero(t, h.reader.offsetAppend)
		assert.Zero(t, h.reader.offsetSearch)
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> helloxxx", h.fetchOutput())
		}
	})
}

func testMultiLineReaderFlush(t *testing.T, input chan<- string, h *mlrHelper) {
	t.Run("flush last remained", func(tt *testing.T) {
		input <- "> shut \ndown 1"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, h.fetchEnd, len(h.output))
		h.reader.Flush()
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> shut ", h.fetchOutput())
		}
		assert.Equal(t, "down 1", string(h.reader.buffer[:h.reader.offsetAppend]))
		assert.Zero(t, h.reader.offsetSearch)
		h.reader.FlushAll()
		assert.Equal(t, h.fetchEnd, len(h.output))
		assert.Zero(t, h.reader.offsetAppend)

		input <- "> shut \ndown 2"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, h.fetchEnd, len(h.output))
		h.reader.FlushAll()
		assert.Zero(t, h.reader.offsetAppend)
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> shut \ndown 2", h.fetchOutput())
		}

		input <- "> shut down 3\n"
		assert.Nil(t, h.reader.Read())
		assert.Equal(t, h.fetchEnd, len(h.output))
		h.reader.FlushAll()
		assert.Zero(t, h.reader.offsetAppend)
		if assert.Equal(t, h.nextOutput(), len(h.output)) {
			assert.Equal(t, "> shut down 3", h.fetchOutput())
		}
	})
}
