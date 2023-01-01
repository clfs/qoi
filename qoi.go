// Package qoi implements a QOI image decoder and encoder.
//
// The QOI specification is at https://qoiformat.org/qoi-specification.pdf.
package qoi

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// headerLen is the length of a QOI header.
const headerLen = 14

// magic is the magic signature for the QOI format.
const magic = "qoif"

// endMarker is the QOI EOF marker.
const endMarker = "\x00\x00\x00\x00\x00\x00\x00\x01"

// Start-of-chunk tag.
const (
	tagRGB   = 0b1111_1110
	tagRGBA  = 0b1111_1111
	tagIndex = 0b00
	tagDiff  = 0b01
	tagLuma  = 0b10
	tagRun   = 0b11
)

// A FormatError reports that the input is not a valid QOI image.
type FormatError string

func (e FormatError) Error() string {
	return "qoi: invalid format: " + string(e)
}

func hash(r, g, b, a uint8) uint8 {
	return (r*3 + g*5 + b*7 + a*11) % 64
}

// Decoding stage.
const (
	dsHeader = iota
	dsChunks
	dsEndMarker
)

type decoder struct {
	r     io.Reader
	img   image.Image
	stage int
}

func (d *decoder) checkHeader() error {
	return nil
}

func (d *decoder) parseChunk() error {
	return nil
}

// DecodeConfig returns the color model and dimensions of a QOI image without
// decoding the entire image. The color model is always color.NRGBAModel,
// regardless of the QOI header's "channels" value.
func DecodeConfig(r io.Reader) (image.Config, error) {
	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return image.Config{}, err
	}

	if !bytes.Equal(header[0:4], []byte(magic)) {
		return image.Config{}, FormatError("bad magic")
	}

	// TODO: Dimension overflow checks.

	config := image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(binary.BigEndian.Uint32(header[4:8])),
		Height:     int(binary.BigEndian.Uint32(header[8:12])),
	}

	return config, nil
}

// Decode reads a QOI image from r and returns it as an image.Image. The type of
// Image returned depends on the QOI contents.
func Decode(r io.Reader) (image.Image, error) {
	d := &decoder{r: r}

	if err := d.checkHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	for d.stage != dsEndMarker {
		if err := d.parseChunk(); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
	}

	return d.img, nil
}

// Encode writes the Image m to w in QOI format. Any Image may be encoded, but
// images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	return nil
}
