package goldentest

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var (
	updateGolden = flag.Bool("update", false, "update the golden files of this test")
)

func TestMain(m *testing.M) {
	flag.Parse()
}

func Equals(t *testing.T, goldenFile string, actualData []byte) {
	t.Helper()
	goldenPath := filepath.Join("../testdata/", goldenFile)

	f, err := os.OpenFile(goldenPath, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("unable to open golden file: %s", err)
	}
	defer f.Close()

	if *updateGolden {
		_, err := f.Write(actualData)
		if err != nil {
			t.Fatalf("unable to write golden file: %s", err)
		}

		err = f.Truncate(int64(len(actualData)))
		if err != nil {
			t.Fatalf("unable to truncate golden file: %s", err)
		}

		// nothing to assert, just return
		return
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("unable to read golden file: %s", err)
	}
	if !reflect.DeepEqual(content, actualData) {
		t.Fatalf("actual output (%d bytes) does not match golden file (%q, %d bytes)", len(actualData), goldenFile, len(content))
	}
}
