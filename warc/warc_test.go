package warc_test

import (
	"compress/gzip"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/eikeon/ginger/warc"
)

func TestReader(t *testing.T) {
	f, err := os.OpenFile("testdata/test.warc.gz", os.O_RDONLY, 0666)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	gr, err := gzip.NewReader(f)
	assert.Equal(t, err, nil)

	wr := warc.NewWarcReader(gr)
	vr, err := wr.ReadRecord()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, vr.Version, "WARC/1.0")
}
