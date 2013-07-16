package warc

import (
	"bufio"
	"errors"
	"io"
)

type Record struct {
	Version string
}

func NewWarcReader(reader io.Reader) *Reader {
	return &Reader{bufio.NewReader(reader)}
}

type Reader struct {
	reader *bufio.Reader
}

func (wr *Reader) ReadRecord() (*Record, error) {
	r := &Record{}
	version, isPrefix, err := wr.reader.ReadLine()
	if isPrefix == true {
		return nil, errors.New("buffer not big enough to read version line")
	}
	if err != nil {
		return nil, errors.New(err.Error())
	}
	r.Version = string(version)
	return r, nil
}
