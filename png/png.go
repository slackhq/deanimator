// Package png provides animated PNG image format support for deanimator.
package png

import (
	"encoding/binary"
	"errors"
	gopng "image/png"
	"io"
	"unicode"

	"github.com/slackhq/deanimator"
	"github.com/slackhq/deanimator/window"
)

const pngHeader = "\x89PNG\r\n\x1a\n"

const (
	idat = "IDAT"
	iend = "IEND"
	actl = "acTL"
	fctl = "fcTL"
)

var (
	errUnderflow = errors.New("png buffer underflow")
	iendChunk    = []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82}
)

//DecodeFunc lets you override the built-in PNG package decode if desired.
var DecodeFunc = gopng.Decode

// References:
// https://www.w3.org/TR/PNG/
// https://wiki.mozilla.org/APNG_Specification

// IsAnimated returns true if the reader is an animated PNG (APNG). A false result with no error
// indicates the buffer definitively contains a normal PNG.
func IsAnimated(r io.Reader) (bool, error) {
	wr := window.NewReader(r, 8)
	chunkHeader := make([]byte, 8)

	_, err := wr.ReadWindow(chunkHeader)
	if err != nil {
		return false, err
	}

	if string(chunkHeader) != pngHeader {
		return false, errors.New("invalid png file")
	}
	// skip the rest of the header
	_, err = wr.Skip(7)
	if err != nil {
		return false, err
	}

	for {
		_, err = wr.ReadWindow(chunkHeader)
		if err == io.EOF {
			// EOF can be expected here i think on a chunk boundary?
			return false, nil
		} else if err != nil {
			return false, err
		}

		chunkType := string(chunkHeader[4:])
		switch chunkType {
		case actl:
			return true, nil
		case idat, iend:
			// "IDAT" or "IEND" before "acTL" means this is not an animated gif.
			return false, nil
		}

		// skip the rest of the header
		_, err = wr.Skip(7)
		if err != nil {
			return false, err
		}

		// skip reads for the chunk data
		chunkLength := binary.BigEndian.Uint32(chunkHeader[:4])
		_, err = wr.Skip(int(chunkLength))
		if err != nil {
			return false, err
		}

		// skip crc
		_, err = wr.Skip(4)
		if err != nil {
			return false, err
		}
	}
}

// RenderFirstFrame extracts the first frame from an animated PNG (APNG). If the image is not
// complete, it scans the image, stripping non-public chunks while checking wether a complete
// default image is available (e.g. the start of an "fcTL" chunk after 1 or more "IDAT" chunks).
// If the complete default image can be extracted, it terminates the image with an "IEND" chunk.
func RenderFirstFrame(src io.Reader, dst io.Writer) error {
	// copy header to dst
	_, err := io.CopyN(dst, src, 8)
	if err != nil {
		return err
	}

	chunkHeader := make([]byte, 8)
	sawIDAT := false
	completeIDAT := false
	for {
		_, err = src.Read(chunkHeader)
		if err != nil {
			return err
		}

		chunkLength := binary.BigEndian.Uint32(chunkHeader[:4])
		chunkType := string(chunkHeader[4:])

		if chunkType == fctl && sawIDAT {
			completeIDAT = true
			break
		} else if chunkType == idat {
			sawIDAT = true
		}

		copyTo := io.Discard

		if unicode.IsUpper(rune(chunkType[1])) {
			// public chunk, just copy through
			_, err = dst.Write(chunkHeader)
			if err != nil {
				return err
			}
			copyTo = dst

		}

		// +4 to also copy CRC
		_, err = io.CopyN(copyTo, src, int64(chunkLength)+4)
		if err != nil {
			return err
		}
	}
	if !completeIDAT {
		return errUnderflow
	}

	_, err = dst.Write(iendChunk)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	deanimator.RegisterFormat("png", pngHeader, IsAnimated, RenderFirstFrame)
}
