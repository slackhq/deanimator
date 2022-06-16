// Package deanimator provides a common interface for detecting animation and deanimation of
// image data streams. Multiple image formats are supported and additional formats can be registered
// by consumers.
package deanimator

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("deanimator: unknown format")

// A format holds an image format's name, magic header and how to decode it.
type format struct {
	name, magic      string
	isAnimated       func(io.Reader) (bool, error)
	renderFirstFrame func(io.Reader, io.Writer) error
}

// Formats is the list of registered formats.
var (
	formatsMu     sync.Mutex
	atomicFormats atomic.Value
)

// RegisterFormat allows consumers to create their own deanimation format support,the format
// registration/handling code for deanimator closely follows that of the image package
// in the std library. To create your own format handling, you can read more about its
// magic string matching to determing the best way to structure your registration.
func RegisterFormat(name, magic string, isAnimated func(io.Reader) (bool, error), renderFirstFrame func(io.Reader, io.Writer) error) {
	formatsMu.Lock()
	formats, _ := atomicFormats.Load().([]format)
	atomicFormats.Store(append(formats, format{name, magic, isAnimated, renderFirstFrame}))
	formatsMu.Unlock()
}

type reader interface {
	io.Reader
	Peek(int) ([]byte, error)
}

// asReader converts an io.Reader to a reader.
func asReader(r io.Reader) reader {
	if rr, ok := r.(reader); ok {
		return rr
	}
	return bufio.NewReader(r)
}

// Match reports whether magic matches b. Magic may contain "?" wildcards.
func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// Sniff determines the format of r's data.
func sniff(r reader) format {
	formats, _ := atomicFormats.Load().([]format)
	for _, f := range formats {
		b, err := r.Peek(len(f.magic))
		if err == nil && match(f.magic, b) {
			return f
		}
	}
	return format{}
}

// IsAnimated returns whether the image data in the reader is a known format and is recognized
// as having multiple animation frames. If a format is not matched, ErrFormat is returned, otherwise
// a flag indicating whether it is animated and the format name are returned. This function only
// consumes as much of the reader as is necessary to determine if something is animated.
func IsAnimated(r io.Reader) (bool, string, error) {
	rr := asReader(r)
	f := sniff(rr)
	if f.isAnimated == nil {
		return false, "", ErrFormat
	}
	b, err := f.isAnimated(rr)
	return b, f.name, err
}

// RenderFirstFrame renders the first frame of an animated image to the provided writer. It will read
// as much of the reader as is necessary to do so. It will also return the matching format. If
// no format matched, it will return ErrFormat.
//
// Some implementations of RenderFirstFrame may change the encoding format of the first frame (for
// example from GIF to PNG).
func RenderFirstFrame(r io.Reader, w io.Writer) (string, error) {
	rr := asReader(r)
	f := sniff(rr)
	if f.renderFirstFrame == nil {
		return "", ErrFormat
	}
	err := f.renderFirstFrame(rr, w)
	return f.name, err
}
