package webp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	gowebp "golang.org/x/image/webp"

	"github.com/slackhq/deanimator/goldentest"
)

var (
	animatedWEBP   []byte
	regularWEBP    []byte
	losslessWEBP   []byte
	lossyAlphaWEBP []byte
)

func TestMain(m *testing.M) {
	goldentest.TestMain(m)
	resetWEBPs()
	os.Exit(m.Run())
}

func resetWEBPs() {
	var err error
	animatedWEBP, err = ioutil.ReadFile("../testdata/animated.webp")
	if err != nil {
		panic(err.Error())
	}

	regularWEBP, err = ioutil.ReadFile("../testdata/house.webp")
	if err != nil {
		panic(err.Error())
	}

	losslessWEBP, err = ioutil.ReadFile("../testdata/yellowrose-lossless.webp")
	if err != nil {
		panic(err.Error())
	}

	lossyAlphaWEBP, err = ioutil.ReadFile("../testdata/yellowrose-lossy-alpha.webp")
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
		{animatedWEBP, true, false},
		{animatedWEBP[:4600], true, false}, // truncated, but enough data to tell it's animated
		{animatedWEBP[:12], false, true},   // truncated, too little data to tell it's animated

		{regularWEBP, false, false},
		{regularWEBP[:2000], false, false}, // truncated, but enough data to tell it's not animated
		{regularWEBP[:12], false, true},    // truncated, too little data to tell it's not animated

		{losslessWEBP, false, false},
		{losslessWEBP[:2000], false, false}, // truncated, but enough data to tell it's not animated
		{losslessWEBP[:12], false, true},

		{lossyAlphaWEBP, false, false},
		{lossyAlphaWEBP[:2000], false, false}, // truncated, but enough data to tell it's not animated
		{lossyAlphaWEBP[:12], false, true},
	}

	for idx, tc := range testCases {
		data := tc.data
		expectAnimated := tc.expectAnimated
		expectUnderflowError := tc.expectUnderflowError
		idx := idx

		t.Run(fmt.Sprintf("animated: %v, underflow: %v, len: %d", expectAnimated, expectUnderflowError, len(data)), func(t *testing.T) {
			r := bytes.NewReader(data)
			result, err := IsAnimated(r)
			if result != expectAnimated {
				t.Errorf("%d: expected IsAnimated == %v, got %v", idx, expectAnimated, result)
			}
			if err == nil && expectUnderflowError {
				t.Errorf("%d: expected underflow error, got nil", idx)
			} else if err != nil {
				if !expectUnderflowError {
					t.Errorf("%d: expected no error, got %v", idx, err)
				} else if err.Error() != "riff: short chunk header" && err != io.EOF {
					// this probably shouldn't be hardcoded, but just for testing...
					t.Errorf("%d: expected underflow error, got %v", idx, err)
				}
			}
		})
	}
}

func TestRenderFirstFrame(t *testing.T) {
	defer resetWEBPs()

	r := bytes.NewReader(animatedWEBP)
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Fatalf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.webp", w.Bytes())

	i, err := gowebp.Decode(w)
	if err != nil {
		t.Fatalf("first frame buffer invalid: %v", err)
	}

	if bounds := i.Bounds(); bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 400 || bounds.Max.Y != 400 {
		t.Fatalf("expected bounds (0,0)-(400,400), got %s", bounds)
	}
}

func TestRenderFirstFramePartialDownload(t *testing.T) {
	defer resetWEBPs()

	// Exclude last byte of trailing frames to create a truncated PNG.
	r := bytes.NewReader(animatedWEBP[:len(animatedWEBP)-1])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Fatalf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.webp", w.Bytes())

	i, err := gowebp.Decode(w)
	if err != nil {
		t.Fatalf("first frame buffer invalid: %v", err)
	}

	if i == nil {
		t.Fatalf("decode is nil")
	}

	if bounds := i.Bounds(); bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 400 || bounds.Max.Y != 400 {
		t.Fatalf("expected bounds (0,0)-(400,400), got %s", bounds)
	}
}

func TestRenderFirstFramePartialDownloadError(t *testing.T) {
	defer resetWEBPs()

	r := bytes.NewReader(animatedWEBP[:4728])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err == nil {
		t.Fatalf("expected error rendering first frame, got nil")
	}
}

func TestRenderFirstFrameMinimumDownload(t *testing.T) {
	defer resetWEBPs()

	// test just enough data (first frame in vp8x)

	r := bytes.NewReader(animatedWEBP[:5234])
	w := bytes.NewBuffer([]byte{})

	if err := RenderFirstFrame(r, w); err != nil {
		t.Fatalf("failed to render first frame: %v", err)
	}

	goldentest.Equals(t, "animated_golden.webp", w.Bytes())

	i, err := gowebp.Decode(w)
	if err != nil {
		t.Fatalf("first frame buffer invalid: %v", err)
	}
	if i == nil {
		t.Fatalf("decode is nil")
	}
	if bounds := i.Bounds(); bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 400 || bounds.Max.Y != 400 {
		t.Fatalf("expected bounds (0,0)-(400,400), got %s", bounds)
	}
}
