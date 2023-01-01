// Package qoi implements a QOI image decoder and encoder.
//
// The QOI specification is at https://qoiformat.org/qoi-specification.pdf.
package qoi

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// headerLen is the length of a QOI header.
const headerLen = 14

// magic is the QOI format magic signature.
const magic = "qoif"

// endMarker is the QOI end-of-stream marker.
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

func hash(r, g, b, a uint8) uint8 {
	return (r*3 + g*5 + b*7 + a*11) % 64
}

// A FormatError reports that the input is not a valid QOI image.
type FormatError string

func (e FormatError) Error() string {
	return "qoi: invalid format: " + string(e)
}

var chunkOrderError = FormatError("chunk out of order")

// Decoding stage.
const (
	dsStart = iota
	dsSeenHeader
	dsSeenRGB
	dsSeenRGBA
	dsSeenIndex
	dsSeenDiff
	dsSeenLuma
	dsSeenRun
	dsSeenEndMarker
)

type decoder struct {
	r             io.Reader
	img           image.Image
	width, height int
	stage         int
	tmp           [1024]byte
}

func (d *decoder) parseHeader() error {
	_, err := io.ReadFull(d.r, d.tmp[:len(magic)])
	if err != nil {
		return err
	}

	if string(d.tmp[:len(magic)]) != magic {
		return FormatError("not a QOI file")
	}

	_, err = io.ReadFull(d.r, d.tmp[:headerLen-len(magic)])
	if err != nil {
		return err
	}

	// TODO: Dimension overflow checks.

	d.width = int(binary.BigEndian.Uint32(d.tmp[0:4]))
	d.height = int(binary.BigEndian.Uint32(d.tmp[4:8]))

	return nil
}

func (d *decoder) parseChunk() error {
	if _, err := io.ReadFull(d.r, d.tmp[:1]); err != nil {
		return err
	}

	switch t := d.tmp[0]; {
	case t == tagRGB:
		if d.stage != dsStart {
			return chunkOrderError
		}
		d.stage = dsSeenRGB
	case t == tagRGBA:
		return nil
	case t>>6 == tagIndex:
		return nil
	case t>>6 == tagDiff:
		return nil
	case t>>6 == tagLuma:
		return nil
	case t>>6 == tagRun:
		return nil
	default:
		panic("uh oh")
	}

	return nil
}

// DecodeConfig returns the color model and dimensions of a QOI image without
// decoding the entire image. The color model is always color.NRGBAModel,
// regardless of the QOI header's "channels" value.
func DecodeConfig(r io.Reader) (image.Config, error) {
	d := &decoder{r: r}

	if err := d.parseHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return image.Config{}, err
	}

	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      d.width,
		Height:     d.height,
	}, nil
}

// Decode reads a QOI image from r and returns it as an image.Image. The type of
// Image returned depends on the QOI contents.
func Decode(r io.Reader) (image.Image, error) {
	d := &decoder{r: r}

	if err := d.parseHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	for d.stage != dsSeenEndMarker {
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
