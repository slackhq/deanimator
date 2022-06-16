package png

import (
	"bytes"
	"fmt"
	gopng "image/png"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/slackhq/deanimator/goldentest"
)

var (
	animatedPNG []byte
	regularPNG  []byte
)

func TestMain(m *testing.M) {
	goldentest.TestMain(m)
	resetPNGs()
	os.Exit(m.Run())
}

func resetPNGs() {
	var err error
	animatedPNG, err = ioutil.ReadFile("../testdata/animated.png")
	if err != nil {
		panic(err.Error())
	}

	regularPNG, err = ioutil.ReadFile("../testdata/emoji-smile.png")
	if err != nil {
		panic(err.Error())
	}
}

func TestIsAnimated(t *testing.T) {
	testCases := []struct {
		data                 []byte
		expectAnimated       bool
		expectUnderflowError bool
	}{
		{animatedPNG, true, false},
		{animatedPNG[:4600], true, false}, // truncated, but enough data to tell it's animated
		{animatedPNG[:20], false, true},   // truncated, too little data to tell it's animated

		{regularPNG, false, false},
		{regularPNG[:2000], false, false}, // truncated, but enough data to tell it's not animated
		{regularPNG[:900], false, true},   // truncated, too little data to tell it's not animated
	}

	for idx, tc := range testCases {
		data := tc.data
		expectAnimated := tc.expectAnimated
		expectUnderflowError := tc.expectUnderflowError
		idx := idx

		t.Run(fmt.Sprintf("animated: %v, underflow: %v, len: %d", expectAnimated, expectUnderflowError, len(data)), func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			result, err := IsAnimated(r)
			if result != expectAnimated {
				t.Errorf("%d: expected IsAnimated == %v, got %v", idx, expectAnimated, result)
			}
			if err == nil && expectUnderflowError {
				t.Errorf("%d: expected underflow error, got nil", idx)
			} else if err != nil {
				if !expectUnderflowError {
					t.Errorf("%d: expected no error, got %v", idx, err)
				} else if err != errUnderflow && err != io.EOF {
					t.Errorf("%d: expected underflow error, got %v", idx, err)
				}
			}
		})
	}
}

func TestRenderFirstFrame(t *testing.T) {
	defer resetPNGs()

	r := bytes.NewReader(animatedPNG)
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Errorf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.png", w.Bytes())

	i, err := gopng.Decode(w)
	if err != nil {
		t.Errorf("first frame buffer invalid: %v", err)
	}

	bounds := i.Bounds()
	if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 100 || bounds.Max.Y != 100 {
		t.Errorf("expected bounds (0,0)-(100,100), got %s", bounds)
	}
}

func TestRenderFirstFramePartialDownload(t *testing.T) {
	defer resetPNGs()

	// Exclude last byte of trailing "IEND" chunk to create a truncated PNG.
	r := bytes.NewReader(animatedPNG[:len(animatedPNG)-1])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Errorf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.png", w.Bytes())

	i, err := gopng.Decode(w)
	if err != nil {
		t.Errorf("first frame buffer invalid: %v", err)
	}

	if i != nil {
		bounds := i.Bounds()
		if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 100 || bounds.Max.Y != 100 {
			t.Errorf("expected bounds (0,0)-(100,100), got %s", bounds)
		}
	}
}

func TestRenderFirstFramePartialDownloadError(t *testing.T) {
	defer resetPNGs()

	// PNG to the end of the "IDAT" -- we cannot tell if we saw all the "IDAT" chunks, so expect an
	// error.
	r := bytes.NewReader(animatedPNG[:4728])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err == nil {
		t.Errorf("expected error rendering first frame, got nil")
	}
}

func TestRenderFirstFrameMinimumDownload(t *testing.T) {
	defer resetPNGs()

	// PNG to the first 8 bytes of the "fcTL" chunk following the IDAT -- just enough data to
	// determine we have all "IDAT" chunks.
	r := bytes.NewReader(animatedPNG[:4736])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Errorf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.png", w.Bytes())

	i, err := gopng.Decode(w)
	if err != nil {
		t.Errorf("first frame buffer invalid: %v", err)
	}

	if i != nil {
		bounds := i.Bounds()
		if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 100 || bounds.Max.Y != 100 {
			t.Errorf("expected bounds (0,0)-(100,100), got %s", bounds)
		}
	}
}
