package qoi_test

import (
	"fmt"
	"image"
	"log"
	"os"

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

	imgs := make([]image.Image, 10)

	// 10 identical, empty images.
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
