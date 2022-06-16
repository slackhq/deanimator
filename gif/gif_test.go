package gif

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/slackhq/deanimator/goldentest"
)

func TestMain(m *testing.M) {
	goldentest.TestMain(m)
	os.Exit(m.Run())
}

func TestIsAnimated(t *testing.T) {
	for idx, tc := range []struct {
		file           string
		expectAnimated bool
	}{
		{"shaq", true},
		{"shaq-partial", true},
		{"bees", true},
		{"thumbsall", true},
		{"bubbletea", true},
		// TODO: empty file
		// TODO: add some additional underflow tests, etc.
	} {
		expectAnimated := tc.expectAnimated
		expectUnderflowError := false
		filename := filepath.Join("../testdata/", tc.file+".gif")

		t.Run(tc.file, func(t *testing.T) {
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			r := bytes.NewReader(data)
			result, err := IsAnimated(r)
			if result != expectAnimated {
				t.Errorf("%d: expected IsAnimated == %v, got %v", idx, expectAnimated, result)
			}
			if err == nil && expectUnderflowError {
				t.Errorf("%d: expected underflow error, got nil", idx)
			} else if err != nil {
				// if !expectUnderflowError {
				t.Errorf("%d: expected no error, got %v", idx, err)
				// 	} else if err != errUnderflow && err != io.EOF {
				// 		t.Errorf("%d: expected underflow error, got %v", idx, err)
				// 	}
			}
		})
	}
}

func TestRenderFirstFrame(t *testing.T) {
	for _, test := range []struct {
		file string
	}{
		{"shaq"},
		{"shaq-partial"},
		{"bees"},
		{"thumbsall"},
		{"bubbletea"},
	} {
		t.Run(test.file, func(t *testing.T) {
			filename := filepath.Join("../testdata/", test.file+".gif")
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			r := bytes.NewReader(data)
			w := bytes.NewBuffer([]byte{})

			err = RenderFirstFrame(r, w)
			if err != nil {
				t.Fatal(err)
			}

			goldentest.Equals(t, test.file+"_golden.gif", w.Bytes())
		})
	}
}
