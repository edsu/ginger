package warc_test

import (
	"compress/gzip"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger/warc"
)

func TestReader(t *testing.T) {
	f, _ := os.Open("testdata/test.warc.gz")
	gz, _ := gzip.NewReader(f)

	reader := warc.NewWarcReader(gz)
	rec, err := reader.ReadRecord()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, rec.Version, "WARC/1.0")
	assert.Equal(t, len(rec.Headers), 4)
	//assert.Equal(t, rec.Headers["content-length"], "251")
}
