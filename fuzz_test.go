package qoi

import (
	"bytes"
	"image"
	"testing"
)

func FuzzDecode(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil {
			return
		}

		if cfg.Width*cfg.Height > 1e6 {
			return
		}

		img, typ, err := image.Decode(bytes.NewReader(b))
		if err != nil || typ != "png" {
			return
		}

		opts := []struct {
			ch Channels
			co ColorSpace
		}{
			{RGB, SRGB},
			{RGB, Linear},
			{RGBA, SRGB},
			{RGBA, Linear},
		}

		for _, opt := range opts {
			var w bytes.Buffer

			e := &Encoder{
				Channels:   opt.ch,
				ColorSpace: opt.co,
			}

			err = e.Encode(&w, img)
			if err != nil {
				t.Fatalf("failed to encode valid image: %s", err)
			}

			img1, err := Decode(&w)
			if err != nil {
				t.Fatalf("failed to decode roundtripped image: %s", err)
			}

			got := img1.Bounds()
			want := img.Bounds()
			if !got.Eq(want) {
				t.Fatalf("roundtripped image bounds have changed, got: %s, want: %s", got, want)
			}
		}
	})
}
