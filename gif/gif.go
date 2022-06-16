// Package gif provides animated GIF image format support for deanimator.
package gif

import (
	"image/png"
	"io"

	"github.com/slackhq/deanimator"
	"github.com/slackhq/deanimator/gif/parser"
	"github.com/slackhq/deanimator/window"
)

//DecodeFunc lets you override the gif parser decode func. This implementation is our own
//since the default in the std lib parses all frames even if only returning the first one.
//once the std library is updated, this could be changed to be the default. See:
//https://github.com/golang/go/pull/46813
var DecodeFunc = parser.Decode

func RenderFirstFrame(r io.Reader, w io.Writer) error {
	i, err := DecodeFunc(r)
	if err != nil {
		return err
	}
	return png.Encode(w, i)
}

func IsAnimated(r io.Reader) (bool, error) {
	// TODO: read and check header to confirm a valid gif?

	const windowSize = 3
	count := 0

	wr := window.NewReader(r, windowSize)
	data := make([]byte, windowSize)
	for {
		_, err := wr.ReadWindow(data)
		if err == io.EOF {
			return false, nil
		} else if err != nil {
			return false, err
		}

		if data[0] == 0 && string(data[1]) == "!" && data[2] == 249 {
			count += 1
		}

		if count > 1 {
			return true, nil
		}
	}
}

func init() {
	deanimator.RegisterFormat("gif", "GIF8?a", IsAnimated, RenderFirstFrame)
}
