package shared

import (
	"compress/gzip"
	"io"

	"github.com/relex/gotils/logger"
)

func InitGZIPCompessor(log logger.Logger, w io.Writer) io.WriteCloser {
	gzipper, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		log.Errorf("couldn't initialize gzip compressor; compression disabled")
		return nil
	}

	return gzipper
}
