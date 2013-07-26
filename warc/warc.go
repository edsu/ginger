package warc

import (
	"bufio"
	"errors"
	"io"
	"log"
	"strings"
)

type Record struct {
	Version string
	Headers map[string]string
}

func NewWarcReader(reader io.Reader) *Reader {
	return &Reader{bufio.NewReader(reader)}
}

type Reader struct {
	reader *bufio.Reader
}

type partReader func(*Record) error

func (wr *Reader) ReadRecord() (*Record, error) {
	rec := &Record{Headers: make(map[string]string)}
	for _, f := range []partReader{wr.readVersion, wr.readHeaders, wr.readContentBlock} {
		err := f(rec)
		if err != nil {
			log.Fatal(err)
		}
	}
	return rec, nil
}

func (wr *Reader) readVersion(rec *Record) (err error) {
	rec.Version, err = wr.readLine()
	return
}

func (wr *Reader) readHeaders(rec *Record) error {
	for {
		line, err := wr.readLine()
		if err != nil {
			return err
		}
		if line == "" {
			break
		}
		parseHeader(line, rec.Headers)
	}

	return nil
}

func (wr *Reader) readContentBlock(*Record) error {
	return nil
}

func (wr *Reader) readLine() (string, error) {
	bytes, isPrefix, err := wr.reader.ReadLine()
	if isPrefix == true {
		return "", errors.New("unable to read entire buffer")
	} else if err == io.EOF {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func parseHeader(line string, headers map[string]string) {
	parts := strings.Split(line, ":")
	if len(parts) == 2 {
		name := strings.ToLower(parts[0])
		value := parts[1]
		headers[name] = value
	}
}
