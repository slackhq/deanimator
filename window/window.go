// Package window provides a windowed reader implementation used by some of the image format packages.
package window

import (
	"errors"
	"io"
)

type WindowedReader struct {
	r          io.Reader
	windowSize int

	previous []byte
}

func NewReader(r io.Reader, size int) *WindowedReader {
	return &WindowedReader{
		r:          r,
		windowSize: size,
	}
}

func (w *WindowedReader) Skip(size int) (int, error) {
	// TODO: optimize this by just ignoring anything that wouldn't be in the last window?
	discard := make([]byte, w.windowSize)
	for i := 0; i < size; i++ {
		_, err := w.ReadWindow(discard)
		if err == io.EOF {
			if i == 0 {
				// if its a re-read and an EOF, just return EOF
				return 0, err
			}
			// if its the first encounter of the EOF, just return successful skip count
			return i + 1, nil
		} else if err != nil {
			return i, err
		}
	}
	return size, nil
}

// ReadWindow returns the current window of bytes. Each successive call to ReadWindow
// only advances the reader a single byte.
func (w *WindowedReader) ReadWindow(window []byte) (int, error) {
	if len(window) != w.windowSize {
		return 0, errors.New("window is incorrect length")
	}

	firstRead := w.previous == nil

	if firstRead {
		previous := make([]byte, w.windowSize-1)
		read, err := w.r.Read(previous)
		if err != nil {
			return 0, err
		} else if read < len(previous) {
			copy(window, previous[:read])
			return read, nil
		}
		w.previous = previous
	}

	next := make([]byte, 1)
	_, err := w.r.Read(next)
	if err == io.EOF && firstRead {
		// if the full length is the same size as the previous window, just return that
		copy(window, w.previous)
		return len(w.previous), nil
	} else if err != nil {
		return 0, err
	}

	// copy to output window
	copy(window, w.previous)
	window[w.windowSize-1] = next[0]

	// update previous read
	w.previous = append(w.previous[1:], next[0])

	return w.windowSize, nil
}
