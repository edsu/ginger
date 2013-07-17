package warc_test

import (
	"compress/gzip"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger/warc"
)

func TestReader(t *testing.T) {
	f, _ := os.OpenFile("testdata/test.warc.gz", os.O_RDONLY, 0666)
	gz, _ := gzip.NewReader(f)

	wr := warc.NewWarcReader(gz)
	vr, err := wr.ReadRecord()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, vr.Version, "WARC/1.0")
}
