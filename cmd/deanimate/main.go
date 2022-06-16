package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	// register formats
	_ "github.com/slackhq/deanimator/gif"
	_ "github.com/slackhq/deanimator/png"
	_ "github.com/slackhq/deanimator/webp"

	"github.com/slackhq/deanimator"
)

func main() {
	err := run(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		log.Fatal("expected in and out filenames as arguments")
	}
	src, _ := filepath.Abs(args[0])
	dst, _ := filepath.Abs(args[1])

	data, err := ioutil.ReadFile(src)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %w", src, err)
	}
	r := bytes.NewReader(data)
	animated, format, err := deanimator.IsAnimated(r)
	if err != nil {
		return fmt.Errorf("unable to determine animation state for %q: %w", src, err)
	}
	log.Printf("detected as %q", format)
	if !animated {
		log.Printf("%q is not animated", src)
	}

	r = bytes.NewReader(data)
	w := bytes.NewBuffer([]byte{})
	_, err = deanimator.RenderFirstFrame(r, w)
	if err != nil {
		return fmt.Errorf("unable to render first frame from %q: %w", src, err)
	}
	err = ioutil.WriteFile(dst, w.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("unable to write file %q: %w", dst, err)
	}

	log.Printf("deanimated version written to %q", dst)

	return nil
}
