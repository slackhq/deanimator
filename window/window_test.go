package window

import (
	"bytes"
	"io"
	"testing"
)

func TestWindowedReader(t *testing.T) {
	wr := NewReader(bytes.NewReader([]byte("0123456789ABCDE")), 3)

	data := make([]byte, 2)
	if read, err := wr.ReadWindow(data); err == nil || read != 0 {
		t.Fatalf("expected error, slice wrong size")
	}

	data = make([]byte, 3)
	read, err := wr.ReadWindow(data)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if read != 3 {
		t.Fatalf("expected to read 3, got %d", read)
	}
	if string(data[:read]) != "012" {
		t.Fatalf("expected data to be \"012\", got %q", string(data))
	}

	skipped, err := wr.Skip(6)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if skipped != 6 {
		t.Fatalf("expected to skip 6, got %d", skipped)
	}

	read, err = wr.ReadWindow(data)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if read != 3 {
		t.Fatalf("expected to read 3, got %d", read)
	}
	if string(data[:read]) != "789" {
		t.Fatalf("expected data to be \"789\", got %q", string(data))
	}

	skipped, err = wr.Skip(5)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if skipped != 5 {
		t.Fatalf("expected to skip 5, got %d", skipped)
	}

	read, err = wr.ReadWindow(data)
	if err != io.EOF {
		t.Fatalf("expected io.EOF")
	}
	if read != 0 {
		t.Fatalf("expected read to be 0, got: %d", read)
	}

	skipped, err = wr.Skip(1)
	if err != io.EOF {
		t.Fatalf("expected io.EOF")
	}
	if skipped != 0 {
		t.Fatalf("expected skipped to be 0, got: %d", skipped)
	}
}

func TestWindowedReader_partialFirstWindow(t *testing.T) {
	// data smaller than the previous bytes in window
	t.Run("< n-1", func(t *testing.T) {
		wr := NewReader(bytes.NewReader([]byte("0")), 3)

		data := make([]byte, 3)
		read, err := wr.ReadWindow(data)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err)
		}
		if read != 1 {
			t.Fatalf("expected to read 3, got %d", read)
		}
		if string(data[:read]) != "0" {
			t.Fatalf("expected data to be \"0\", got %q", string(data))
		}

		read, err = wr.ReadWindow(data)
		if err != io.EOF {
			t.Fatalf("expected io.EOF")
		}
		if read != 0 {
			t.Fatalf("expected read to be 0, got: %d", read)
		}
	})

	// data the same size as the previous bytes in window
	t.Run("= n-1", func(t *testing.T) {
		wr := NewReader(bytes.NewReader([]byte("01")), 3)

		data := make([]byte, 3)
		read, err := wr.ReadWindow(data)
		if err != nil {
			t.Fatalf("expected no error, got: %s", err)
		}
		if read != 2 {
			t.Fatalf("expected to read 3, got %d", read)
		}
		if string(data[:read]) != "01" {
			t.Fatalf("expected data to be \"01\", got %q", string(data))
		}

		read, err = wr.ReadWindow(data)
		if err != io.EOF {
			t.Fatalf("expected io.EOF")
		}
		if read != 0 {
			t.Fatalf("expected read to be 0, got: %d", read)
		}
	})
}

func TestWindowedReader_partialSkip(t *testing.T) {
	wr := NewReader(bytes.NewReader([]byte("01")), 3)

	skipped, err := wr.Skip(3)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
	if skipped != 2 {
		t.Fatalf("expected skipped to be 2, got: %d", skipped)
	}

	skipped, err = wr.Skip(1)
	if err != io.EOF {
		t.Fatalf("expected io.EOF")
	}
	if skipped != 0 {
		t.Fatalf("expected skipped to be 0, got: %d", skipped)
	}
}

func TestWindowedReader_emptyReader(t *testing.T) {
	wr := NewReader(bytes.NewReader([]byte{}), 3)

	data := make([]byte, 3)
	read, err := wr.ReadWindow(data)
	if err != io.EOF {
		t.Fatalf("expected io.EOF")
	}
	if read != 0 {
		t.Fatalf("expected read to be 0, got: %d", read)
	}

	skipped, err := wr.Skip(1)
	if err != io.EOF {
		t.Fatalf("expected io.EOF")
	}
	if skipped != 0 {
		t.Fatalf("expected skipped to be 0, got: %d", skipped)
	}
}
