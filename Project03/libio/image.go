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

func (img *IntImage) ToChannels(count int, defaults ...uint8) *IntImage {
	dst := toChannels(img.Channels, count, img.Count(), img.Pix, defaults...)

	return NewIntImage(dst, count, img.Width, img.Height)
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

func (img *IntImage) Shuffle(order []int, defaults ...uint8) *IntImage {
	if len(order) > 4 {
		order = order[:4]
	}
	dst := shuffle(order, img.Channels, img.Count(), img.Pix, defaults...)

	return NewIntImage(dst, len(order), img.Width, img.Height)
}

func shuffle[P ~[]E, E any](order []int, srcCh int, count int, pix P, defaults ...E) P {
	dstCh := len(order)
	dst := make([]E, count*dstCh)

	if len(defaults) < 8 {
		missing := 8 - len(defaults)
		defaults = append(defaults, make([]E, missing)...)
	}

	for dch, sch := range order {
		if sch >= srcCh {
			for i := 0; i < count; i++ {
				dst[i*dstCh+dch] = defaults[sch]
			}
		} else {
			for i := 0; i < count; i++ {
				dst[i*dstCh+dch] = pix[i*srcCh+sch]
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

func (img *FloatImage) ToChannels(count int, defaults ...float32) *FloatImage {
	dst := toChannels(img.Channels, count, img.Count(), img.Pix, defaults...)

	return NewFloatImage(dst, count, img.Width, img.Height)
}

func (img *FloatImage) Shuffle(order []int, defaults ...float32) *FloatImage {
	if len(order) > 4 {
		order = order[:4]
	}
	dst := shuffle(order, img.Channels, img.Count(), img.Pix, defaults...)

	return NewFloatImage(dst, len(order), img.Width, img.Height)
}

func (img *FloatImage) Copy() *FloatImage {
	pix := make([]float32, len(img.Pix))
	copy(pix, img.Pix)
	return NewFloatImage(pix, img.Channels, img.Width, img.Height)
}

func (img *FloatImage) ToIntImage() *IntImage {
	pix := make([]uint8, len(img.Pix))

	for i := 0; i < len(img.Pix); i++ {
		pix[i] = uint8(math32.Min(math32.Max(0.0, img.Pix[i]), 1.0) * 0xff)
	}

	return NewIntImage(pix, img.Channels, img.Width, img.Height)
}

func (img *FloatImage) Tonemap(gamma, scale float32) {
	for i := 0; i < len(img.Pix); i++ {
		img.Pix[i] = tonemap(img.Pix[i], 1.0/gamma, scale)
	}
}

// Normalizes all pixel values to be from 0 to 1
func (img *FloatImage) Normalize() {
	var min, max float32 = math32.Inf(1), math32.Inf(-1)
	for i := 0; i < len(img.Pix); i++ {
		if img.Pix[i] > max {
			max = img.Pix[i]
		}
		if img.Pix[i] < min {
			min = img.Pix[i]
		}
	}
	diff := max - min
	for i := 0; i < len(img.Pix); i++ {
		img.Pix[i] = (img.Pix[i] - min) / diff
	}
}

func tonemap(value, gamma, scale float32) float32 {
	value = math32.Pow(value, gamma) * scale
	return math32.Min(math32.Max(0.0, value), 1.0)
}
