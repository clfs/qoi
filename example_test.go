package qoi_test

import (
	"encoding/base64"
	"fmt"
	"image"
	"log"
	"os"
	"strings"

	"github.com/clfs/qoi"
)

type pool struct {
	b *qoi.EncoderBuffer
}

func (p *pool) Get() *qoi.EncoderBuffer {
	return p.b
}

func (p *pool) Put(b *qoi.EncoderBuffer) {
	p.b = b
}

func ExampleEncoder_Encode() {
	const width, height = 256, 256

	// 10 identical and empty sample images.
	// In real use, you'd have visual content from one or more sources.
	imgs := make([]image.Image, 10)
	for i := 0; i < len(imgs); i++ {
		imgs[i] = image.NewNRGBA(image.Rect(0, 0, width, height))
	}

	enc := qoi.Encoder{
		Channels:   qoi.RGBA,
		BufferPool: new(pool), // Supply your own implementation.
	}

	for i := 0; i < len(imgs); i++ {
		f, err := os.Create(fmt.Sprintf("image_%d.qoi", i))
		if err != nil {
			log.Fatal(err)
		}

		if err := enc.Encode(f, imgs[i]); err != nil {
			f.Close()
			log.Fatal(err)
		}

		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

const raw = "cW9pZgAAAAEAAAABBAD+/wAAAAAAAAAAAAE="

func ExampleDecode() {
	// qoi.Decode takes an io.Reader, so build one from a base64-encoded string.
	r := base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw))

	img, err := qoi.Decode(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("dimensions: %d⨉%d", img.Bounds().Dx(), img.Bounds().Dy())

	// Importing this qoi package registers the QOI format with the standard
	// library's image package, so image.Decode works too.
	r2 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw))

	img2, format, err := image.Decode(r2)
	if err != nil || format != "qoi" {
		log.Fatal(err)
	}

	fmt.Printf("dimensions: %d⨉%d", img2.Bounds().Dx(), img2.Bounds().Dy())
}
