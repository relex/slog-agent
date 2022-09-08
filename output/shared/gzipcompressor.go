package shared

import (
	"io"

	"github.com/klauspost/compress/gzip"
	"github.com/relex/gotils/logger"
)

func InitGzipCompessor(log logger.Logger, w io.Writer) io.WriteCloser {
	gzipper, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		log.Errorf("couldn't initialize gzip compressor; compression disabled")
		return nil
	}

	return gzipper
}
