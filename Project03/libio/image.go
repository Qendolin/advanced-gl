package libio

import (
	"unsafe"

	goimg "image"

	"github.com/chewxy/math32"
)

const MagicNumberF32 = 0x6d16837d

type FloatImageVersion uint32

const (
	F32Version1_001_000 = FloatImageVersion(1_001_000)
)

type FloatImageCompression uint32

const (
	FloatImageCompressionNone = FloatImageCompression(iota)
	FloatImageCompressionFixedPoint16Lz4
)

type image struct {
	Channels      int
	Width, Height int
}

// Calculates the tuple index into the images data.
//
// Note that the origin (0,0) is in the bottom left, as opposed to Go's top left origin
func (img *image) Index(x, y int) int {
	return x*img.Channels + y*img.Channels*img.Width
}

func (img *image) Count() int {
	return img.Width * img.Height
}

type IntImage struct {
	image
	Pix []uint8
}

func NewIntImage(pix []uint8, channels int, width, height int) *IntImage {
	return &IntImage{
		Pix: pix,
		image: image{
			Channels: channels,
			Width:    width,
			Height:   height,
		},
	}
}

func (img *IntImage) Pointer() unsafe.Pointer {
	return unsafe.Pointer(&img.Pix[0])
}

func (img *IntImage) Bytes() int {
	return img.Width * img.Height * img.Channels
}

func (img *IntImage) ToChannels(nr int, defaults ...uint8) *IntImage {
	dst := toChannels(img.Channels, nr, img.Count(), img.Pix, defaults...)

	return NewIntImage(dst, nr, img.Width, img.Height)
}

func toChannels[P ~[]E, E any](srcCh, dstCh int, count int, pix P, defaults ...E) P {
	if srcCh == dstCh {
		return pix
	}

	if len(defaults) < dstCh {
		missing := len(defaults) - dstCh
		defaults = append(defaults, make([]E, missing)...)
	}

	dst := make([]E, count*dstCh)

	if dstCh > srcCh {
		for i := 0; i < count; i++ {
			for c := 0; c < srcCh; c++ {
				dst[i*dstCh+c] = pix[i*srcCh+c]
			}
			for c := srcCh; c < dstCh; c++ {
				dst[i*dstCh+c] = defaults[c]
			}
		}
	}

	if dstCh < srcCh {
		for i := 0; i < count; i++ {
			for c := 0; c < dstCh; c++ {
				dst[i*dstCh+c] = pix[i*srcCh+c]
			}
		}
	}

	return dst
}

func (img *IntImage) ToRGBA() *goimg.RGBA {
	rgba := goimg.NewRGBA(goimg.Rect(0, 0, img.Width, img.Height))

	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			i := (x + y*img.Width) * img.Channels
			// flipped vertically
			j := (x + (img.Height-y-1)*img.Width) * 4
			for c := 0; c < img.Channels; c++ {
				rgba.Pix[j+c] = img.Pix[i+c]
			}
			for c := img.Channels; c < 3; c++ {
				rgba.Pix[j+c] = 0
			}
			if img.Channels < 4 {
				rgba.Pix[j+3] = 0xff
			}
		}
	}

	return rgba
}

type FloatImageHeader struct {
	Check         uint32
	Version       FloatImageVersion
	Width, Height uint32
	Channels      uint8
	Compression   FloatImageCompression
	Unused        [14]uint8
}

type FloatImage struct {
	image
	Pix []float32
}

func NewFloatImage(pix []float32, channels int, width, height int) *FloatImage {
	return &FloatImage{
		Pix: pix,
		image: image{
			Channels: channels,
			Width:    width,
			Height:   height,
		},
	}
}

func (img *FloatImage) Pointer() unsafe.Pointer {
	return unsafe.Pointer(&img.Pix[0])
}

func (img *FloatImage) Bytes() int {
	return img.Width * img.Height * img.Channels * 4
}

func (img *FloatImage) ToChannels(nr int, defaults ...float32) *FloatImage {
	dst := toChannels(img.Channels, nr, img.Count(), img.Pix, defaults...)

	return NewFloatImage(dst, nr, img.Width, img.Height)
}

func (img *FloatImage) ToIntImage(gamma, scale float32) *IntImage {
	pix := make([]uint8, len(img.Pix))

	for i := 0; i < len(img.Pix); i++ {
		pix[i] = uint8(tonemap(img.Pix[i], 1.0/gamma, scale) * 0xff)
	}

	return NewIntImage(pix, img.Channels, img.Width, img.Height)
}

func tonemap(value, gamma, scale float32) float32 {
	value = math32.Pow(value, gamma) * scale
	return math32.Min(math32.Max(0.0, value), 1.0)
}
