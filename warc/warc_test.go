package warc_test

import (
	"os"
	"testing"

	"github.com/eikeon/ginger/warc"
)

func TestReader(t *testing.T) {
	if f, err := os.OpenFile("testdata/test.warc.gz", os.O_RDONLY, 0666); err == nil {
		warc.NewWarcReader(f)
	} else {
		t.Error(err)
	}
}
