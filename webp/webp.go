// Package webp provides animated WEBP image format support for deanimator.
package webp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/image/riff"

	"github.com/slackhq/deanimator"
)

var (
	fccALPH = riff.FourCC{'A', 'L', 'P', 'H'}
	fccVP8  = riff.FourCC{'V', 'P', '8', ' '}
	fccVP8L = riff.FourCC{'V', 'P', '8', 'L'}
	fccVP8X = riff.FourCC{'V', 'P', '8', 'X'}
	fccWEBP = riff.FourCC{'W', 'E', 'B', 'P'}
	fccANIM = riff.FourCC{'A', 'N', 'I', 'M'}
	fccANMF = riff.FourCC{'A', 'N', 'M', 'F'}
)

var (
	errMalformedImage = errors.New("webp malformed, unable to process")
)

func IsAnimated(src io.Reader) (bool, error) {
	formType, r, err := riff.NewReader(src)
	if err != nil {
		return false, fmt.Errorf("cannot create reader: %w", errMalformedImage)
	}
	if formType != fccWEBP {
		return false, errMalformedImage
	}

	chunkID, _, chunkData, err := r.Next()
	if err != nil {
		return false, err
	}
	if chunkID != fccVP8X {
		return false, nil
	}
	extended := []byte{0}
	_, err = chunkData.Read(extended)
	if err != nil {
		return false, err
	}
	animation := extended[0]&byte(2) == byte(2)

	return animation, nil
}

/*
An extended format file consists of:

- A 'VP8X' chunk with information about features used in the file.
- An optional 'ICCP' chunk with color profile.
- An optional 'ANIM' chunk with animation control data.
- Image data.
- An optional 'EXIF' chunk with Exif metadata.
- An optional 'XMP ' chunk with XMP metadata.

For a still image, the image data consists of a single frame, which is made up of:

- An optional alpha subchunk.
- A bitstream subchunk.

+- VP8X (descriptions of features used)
+- ANIM (global animation parameters)
+- ANMF (frame1 parameters + data)
+- ANMF (frame2 parameters + data)
+- ANMF (frame3 parameters + data)
+- ANMF (frame4 parameters + data)
+- EXIF (metadata)
*/

func RenderFirstFrame(src io.Reader, dst io.Writer) error {
	formType, r, err := riff.NewReader(src)
	if err != nil {
		return errMalformedImage
	}
	if formType != fccWEBP {
		return errMalformedImage
	}
	var canvasSize []byte
	for {
		chunkID, chunkLen, chunkData, err := r.Next()
		if err != nil {
			return fmt.Errorf("unable to get next chunk: %w", err)
		}
		switch chunkID {
		case fccVP8X:
			extended := []byte{0}
			_, err = chunkData.Read(extended)
			if err != nil {
				return err
			}
			animation := extended[0]&byte(2) == byte(2)
			if !animation {
				return fmt.Errorf("not an animated image")
			}
			reserved := []byte{0, 0, 0}
			_, err = chunkData.Read(reserved)
			if err != nil {
				return fmt.Errorf("unable to read reserved: %w", err)
			}
			canvasSize = []byte{0, 0, 0, 0, 0, 0}
			_, err = chunkData.Read(canvasSize)
			if err != nil {
				return fmt.Errorf("unable to read canvas size: %w", err)
			}
		case fccANIM:
			// do nothing
		case fccANMF:
			discard := make([]byte, 16)
			_, err := chunkData.Read(discard)
			if err != nil {
				return fmt.Errorf("unable to discard ANMF data: %w", err)
			}
			bitstream, hasAlpha, err := readANMFBitstream(chunkLen-16, chunkData)
			if err != nil {
				return fmt.Errorf("unable to read ANMF bitstream: %w", err)
			}

			io.WriteString(dst, "RIFF")

			fileSize := 4 + //webp
				8 + //vp8x header
				10 + // vp8x len
				len(bitstream) //first frame data
			err = binary.Write(dst, binary.LittleEndian, uint32(fileSize))
			if err != nil {
				return fmt.Errorf("unable to write file size: %w", err)
			}

			dst.Write(fccWEBP[:])

			// VP8X chunk
			dst.Write(fccVP8X[:])
			err = binary.Write(dst, binary.LittleEndian, uint32(10))
			if err != nil {
				return fmt.Errorf("unable to write vp8x chunk size: %w", err)
			}

			extended := byte(0)
			if hasAlpha {
				extended = byte(16)
			}
			dst.Write([]byte{extended})
			dst.Write([]byte{0, 0, 0}) // reserved
			dst.Write(canvasSize)

			dst.Write(bitstream)

			return nil
		default:
			return errMalformedImage
		}
	}
}

func readANMFBitstream(anmfChunkLen uint32, anmfChunkData io.Reader) ([]byte, bool, error) {
	_, r, err := riff.NewListReader(anmfChunkLen+4, io.MultiReader(
		bytes.NewReader(fccANMF[:]),
		anmfChunkData,
	))
	if err != nil {
		return nil, false, errMalformedImage
	}
	// TODO: this should write using io.Copy or similar to the upstream writer
	// and not this intermediate buffer. Since we need to know the length though
	// that may not be possible and we have to buffer I guess? Did we get the
	// length in a previous field somewhere?
	bitstream := bytes.NewBuffer([]byte{})
	hasAlpha := false
	for {
		chunkID, chunkLen, chunkData, err := r.Next()
		if err == io.EOF {
			return bitstream.Bytes(), hasAlpha, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("unable to get next subchunk: %w", err)
		}
		switch chunkID {
		case fccALPH:
			hasAlpha = true

			fallthrough
		case fccVP8, fccVP8L:
			bitstream.Write(chunkID[:])
			err = binary.Write(bitstream, binary.LittleEndian, chunkLen)
			if err != nil {
				return nil, false, fmt.Errorf("unable to write chunk length: %w", err)
			}
			_, err = io.Copy(bitstream, chunkData)
			if err != nil {
				return nil, false, fmt.Errorf("unable to copy bistream: %w", err)
			}

			if chunkLen%2 == 1 {
				// pad if odd length
				bitstream.WriteByte(0)
			}
		default:
			return nil, false, errMalformedImage
		}
	}
}

func init() {
	deanimator.RegisterFormat("webp", "RIFF????WEBPVP8", IsAnimated, RenderFirstFrame)
}
