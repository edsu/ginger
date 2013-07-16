package warc

import (
	"io"
)

func NewWarcReader(reader io.Reader) *Reader {
	return &Reader{reader}
}

type Reader struct {
	reader io.Reader
}
