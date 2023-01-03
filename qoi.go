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

func init() {
	image.RegisterFormat("qoi", magic, Decode, DecodeConfig)
}

// ColorSpace is the color space of an image.
type ColorSpace int

const (
	SRGBLinearAlpha ColorSpace = 0
	AllLinear       ColorSpace = 1
)

// Channels represents the number of channels in an image.
type Channels int

const (
	RGB  Channels = 3
	RGBA Channels = 4
)

// Start-of-chunk tag constant.
const (
	opIndex = 0b0000_0000
	opDiff  = 0b0100_0000
	opLuma  = 0b1000_0000
	opRun   = 0b1100_0000
	opRGB   = 0b1111_1110
	opRGBA  = 0b1111_1111

	// Mask for two-bit tags.
	opMask2 = 0b1100_0000
)

const magic = "qoif"

const headerLen = 14

// endMarker is the QOI end-of-stream marker.
const endMarker = "\x00\x00\x00\x00\x00\x00\x00\x01"

// A FormatError reports that the input is not a valid QOI image.
type FormatError string

func (e FormatError) Error() string {
	return "qoi: invalid format: " + string(e)
}

func hash(c color.NRGBA) uint8 {
	return c.R*3 + c.G*5 + c.B*7 + c.A*11
}

type decoder struct {
	r             io.Reader
	img           *image.NRGBA
	width, height int
	tmp           [10]byte
	run           int
	index         [64]color.NRGBA
	prev          color.NRGBA
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

	d.width = int(binary.BigEndian.Uint32(d.tmp[0:4]))
	d.height = int(binary.BigEndian.Uint32(d.tmp[4:8]))

	// TODO: Dimension overflow checks.

	d.img = image.NewNRGBA(image.Rect(0, 0, d.width, d.height))

	switch c := Channels(d.tmp[8]); c {
	case RGB, RGBA:
		// ok
	default:
		return FormatError("invalid channel count")
	}

	switch c := ColorSpace(d.tmp[9]); c {
	case SRGBLinearAlpha, AllLinear:
		// ok
	default:
		return FormatError("invalid color space")
	}

	return nil
}

func (d *decoder) advance() error {
	if d.run > 0 {
		d.run--
		return nil
	}

	_, err := io.ReadFull(d.r, d.tmp[:1])
	if err != nil {
		return err
	}

	switch t := d.tmp[0]; {
	case t == opRGB:
		_, err := io.ReadFull(d.r, d.tmp[1:4])
		if err != nil {
			return err
		}
		d.prev.R = d.tmp[1]
		d.prev.G = d.tmp[2]
		d.prev.B = d.tmp[3]
	case t == opRGBA:
		_, err := io.ReadFull(d.r, d.tmp[1:5])
		if err != nil {
			return err
		}
		d.prev.R = d.tmp[1]
		d.prev.G = d.tmp[2]
		d.prev.B = d.tmp[3]
		d.prev.A = d.tmp[4]
	case t&opMask2 == opIndex:
		d.prev = d.index[t]
	case t&opMask2 == opDiff:
		d.prev.R += t>>4&0x3 - 2
		d.prev.G += t>>2&0x3 - 2
		d.prev.B += t&0x3 - 2
	case t&opMask2 == opLuma:
		_, err := io.ReadFull(d.r, d.tmp[1:2])
		if err != nil {
			return err
		}
		delta := t&^opMask2 - 32
		d.prev.R += delta - 8 + d.tmp[1]>>4&0xf
		d.prev.G += delta
		d.prev.B += delta - 8 + d.tmp[1]&0xf
	case t&opMask2 == opRun:
		d.run = int(t &^ opMask2)
	}

	d.index[hash(d.prev)%64] = d.prev

	return nil
}

// DecodeConfig returns the color model and dimensions of a QOI image without
// decoding the entire image. The color model is always color.NRGBAModel,
// regardless of QOI header metadata.
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
// Image returned is always image.NRGBA, regardless of QOI header metadata.
func Decode(r io.Reader) (image.Image, error) {
	d := &decoder{
		r:    r,
		prev: color.NRGBA{A: 255},
	}

	if err := d.parseHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	var (
		minY = d.img.Bounds().Min.Y
		maxY = d.img.Bounds().Max.Y
		minX = d.img.Bounds().Min.X
		maxX = d.img.Bounds().Max.X
	)

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			if err := d.advance(); err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return nil, err
			}
			d.img.SetNRGBA(x, y, d.prev)
		}
	}

	return d.img, nil
}

// Encode writes the Image m to w in QOI format. Any Image may be encoded, but
// images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	return nil
}

type encoder struct {
}

// Encoder configures encoding QOI images.
type Encoder struct {
	Channels   Channels
	ColorSpace ColorSpace

	// BufferPool optionally specifies a buffer pool to get temporary
	// EncoderBuffers when encoding an image.
	BufferPool EncoderBufferPool
}

func (enc *Encoder) Encode(w io.Writer, m image.Image) error {
	return nil
}

// EncoderBufferPool is an interface for getting and returning temporary
// instances of the EncoderBuffer struct. This can be used to reuse buffers when
// encoding multiple images.
type EncoderBufferPool interface {
	Get() *EncoderBuffer
	Put(*EncoderBuffer)
}

// EncoderBuffer holds the buffers used for encoding QOI images.
type EncoderBuffer encoder
