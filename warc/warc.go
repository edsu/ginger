package warc

import (
	"bufio"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
)

// NewWarcReader constructs a WARC Reader for a given input stream.
func NewWarcReader(reader io.Reader) *WarcReader {
	return &WarcReader{reader, bufio.NewReader(reader), 0}
}

// WarcReader lets you iterate through the records in a WARC file.
type WarcReader struct {
	reader    io.Reader
	bufreader *bufio.Reader
	pos       int64
}

// WarcRecord represents a complete WARC Record.
type WarcRecord struct {
	Version           string
	Headers           map[string]string
	ContentBlockStart int64
	ContentBlockEnd   int64
}

// ReadRecord is method that retuns the next WARC Record available.
func (wr *WarcReader) ReadRecord() (*WarcRecord, error) {
	rec := &WarcRecord{
		Headers: make(map[string]string),
	}

	parts := []partReader{
		wr.readVersion,
		wr.readHeaders,
		wr.readContentBlock,
		wr.readEndOfRecord,
	}

	for _, f := range parts {
		err := f(rec)
		if err == io.EOF {
			return nil, nil
		} else if err != nil {
			log.Fatal(err)
		}
	}

	return rec, nil
}

func (wr *WarcReader) readVersion(rec *WarcRecord) (err error) {
	rec.Version, err = wr.readLine()
	return
}

func (wr *WarcReader) readHeaders(rec *WarcRecord) error {
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

func (wr *WarcReader) readContentBlock(rec *WarcRecord) error {
	rec.ContentBlockStart = wr.pos
	contentLength, err := strconv.Atoi(rec.Headers["content-length"])
	if err != nil {
		return err
	}
	remaining := contentLength
	buffSize := 1024
	for remaining > 0 {
		if remaining < buffSize {
			buffSize = remaining
		}
		buff := make([]byte, buffSize)
		n, err := wr.bufreader.Read(buff)
		wr.pos += int64(n)
		if err != nil {
			return err
		}
		remaining -= n
	}
	rec.ContentBlockEnd = rec.ContentBlockStart + int64(contentLength)
	return err
}

func (wr *WarcReader) readEndOfRecord(rec *WarcRecord) error {
	for i := 0; i < 2; i++ {
		line, err := wr.readLine()
		if err != nil {
			return err
		}
		if line != "" {
			return errors.New("expected newline got " + line)
		}
	}
	return nil
}

func (wr *WarcReader) readLine() (string, error) {
	bytes, isPrefix, err := wr.bufreader.ReadLine()
	if isPrefix == true {
		return "", errors.New("unable to read entire buffer")
	} else if err != nil {
		return "", err
	}

	wr.pos += int64(len(bytes))
	return string(bytes), nil
}

func parseHeader(line string, headers map[string]string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		name := strings.ToLower(parts[0])
		value := parts[1]
		headers[name] = strings.TrimSpace(value)
	}
}

type partReader func(*WarcRecord) error
