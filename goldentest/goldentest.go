package goldentest

import (
	"bytes"
	"flag"
	"image"
	"io"
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

// EqualsImage compares the decoded pixels of actualData with the golden file,
// instead of the encoded bytes. Encoders, such as image/gif, can write
// different bytes for the same image between Go releases.
func EqualsImage(t *testing.T, goldenFile string, actualData []byte, decode func(io.Reader) (image.Image, error)) {
	t.Helper()
	goldenPath := filepath.Join("../testdata/", goldenFile)

	if *updateGolden {
		Equals(t, goldenFile, actualData)
		return
	}

	content, err := ioutil.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("unable to read golden file: %s", err)
	}
	expected, err := decode(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("unable to decode golden file %q: %s", goldenFile, err)
	}
	actual, err := decode(bytes.NewReader(actualData))
	if err != nil {
		t.Fatalf("unable to decode actual output: %s", err)
	}

	if expected.Bounds() != actual.Bounds() {
		t.Fatalf("actual bounds %v do not match golden file (%q) bounds %v", actual.Bounds(), goldenFile, expected.Bounds())
	}
	b := expected.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			er, eg, eb, ea := expected.At(x, y).RGBA()
			ar, ag, ab, aa := actual.At(x, y).RGBA()
			if er != ar || eg != ag || eb != ab || ea != aa {
				t.Fatalf("actual pixel at (%d, %d) does not match golden file (%q)", x, y, goldenFile)
			}
		}
	}
}
