package stbi

// #cgo LDFLAGS: -lm
// #define STB_IMAGE_IMPLEMENTATION
// #define STBI_FAILURE_USERMSG
// #define STBI_ONLY_JPEG
// #define STBI_ONLY_PNG
// #define STBI_ONLY_HDR
// #define STBI_ONLY_TGA
// #include "stb_image.h"
import "C"
import (
	"errors"
	"image"
	"image/color"
	"io"
	"unsafe"
)

type Configuration struct {
	HdrToLdrGamma, HdrToLdrScale float32
	LdrToHdrGamma, LdrToHdrScale float32
	Unpremultiply                bool
	FlipVertically               bool
	CopyData                     bool
}

var Default Configuration = Configuration{
	HdrToLdrGamma:  2.2,
	HdrToLdrScale:  1.0,
	LdrToHdrGamma:  2.2,
	LdrToHdrScale:  1.0,
	Unpremultiply:  false,
	FlipVertically: false,
}

var active Configuration = Default

type RgbaLdr struct {
	// Pix holds the image's pixels, in R, G, B, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*4].
	Pix []uint8
	// Stride is the Pix stride (in floats) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect      image.Rectangle
	needsFree bool
}

func (p *RgbaLdr) ColorModel() color.Model { return color.RGBAModel }

func (p *RgbaLdr) Bounds() image.Rectangle { return p.Rect }

func (p *RgbaLdr) At(x, y int) (color [4]uint8) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color
	}
	i := p.PixOffset(x, y)
	color[0] = p.Pix[i+0]
	color[1] = p.Pix[i+1]
	color[2] = p.Pix[i+2]
	color[3] = p.Pix[i+3]

	return color
}

func (p *RgbaLdr) Close() error {
	if p.needsFree {
		p.needsFree = false
		C.stbi_image_free(unsafe.Pointer(&p.Pix[0]))
	}
	return nil
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RgbaLdr) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*4
}

type RgbaHdr struct {
	// Pix holds the image's pixels, in R, G, B, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*4].
	Pix []float32
	// Stride is the Pix stride (in floats) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect      image.Rectangle
	needsFree bool
}

func (p *RgbaHdr) ColorModel() color.Model { return color.RGBAModel }

func (p *RgbaHdr) Bounds() image.Rectangle { return p.Rect }

func (p *RgbaHdr) At(x, y int) (color [4]float32) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color
	}
	i := p.PixOffset(x, y)
	color[0] = p.Pix[i+0]
	color[1] = p.Pix[i+1]
	color[2] = p.Pix[i+2]
	color[3] = p.Pix[i+3]

	return color
}

func (p *RgbaHdr) Close() error {
	if p.needsFree {
		p.needsFree = false
		C.stbi_image_free(unsafe.Pointer(&p.Pix[0]))
	}
	return nil
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *RgbaHdr) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*4
}

func (conf *Configuration) apply() {
	if active.HdrToLdrGamma != conf.HdrToLdrGamma {
		C.stbi_hdr_to_ldr_gamma((C.float)(conf.HdrToLdrGamma))
		active.HdrToLdrGamma = conf.HdrToLdrGamma
	}
	if active.HdrToLdrScale != conf.HdrToLdrScale {
		C.stbi_hdr_to_ldr_scale((C.float)(conf.HdrToLdrScale))
		active.HdrToLdrScale = conf.HdrToLdrScale
	}
	if active.LdrToHdrGamma != conf.LdrToHdrGamma {
		C.stbi_ldr_to_hdr_gamma((C.float)(conf.LdrToHdrGamma))
		active.LdrToHdrGamma = conf.LdrToHdrGamma
	}
	if active.LdrToHdrScale != conf.LdrToHdrScale {
		C.stbi_ldr_to_hdr_scale((C.float)(conf.LdrToHdrScale))
		active.LdrToHdrScale = conf.LdrToHdrScale
	}
	if active.Unpremultiply != conf.Unpremultiply {
		var flag C.int
		if conf.Unpremultiply {
			flag = 1
		}
		C.stbi_set_unpremultiply_on_load(flag)
		active.Unpremultiply = conf.Unpremultiply
	}
	if active.FlipVertically != conf.FlipVertically {
		var flag C.int
		if conf.FlipVertically {
			flag = 1
		}
		C.stbi_set_flip_vertically_on_load(flag)
		active.FlipVertically = conf.FlipVertically
	}
}

func (conf *Configuration) Load(r io.Reader) (*RgbaLdr, error) {
	conf.apply()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return conf.LoadBytes(b)
}

func (conf *Configuration) LoadBytes(b []byte) (*RgbaLdr, error) {
	conf.apply()
	var x, y C.int
	mem := (*C.uchar)(unsafe.Pointer(&b[0]))
	data := C.stbi_load_from_memory(mem, C.int(len(b)), &x, &y, nil, 4)
	if data == nil {
		msg := C.GoString(C.stbi_failure_reason())
		return nil, errors.New(msg)
	}

	var pix []uint8
	if conf.CopyData {
		defer C.stbi_image_free(unsafe.Pointer(data))
		pix = C.GoBytes(unsafe.Pointer(data), y*x*4)
	} else {
		pix = unsafe.Slice((*uint8)(data), y*x*4)
	}

	return &RgbaLdr{
		Pix:       pix,
		Stride:    int(x) * 4,
		Rect:      image.Rect(0, 0, int(x), int(y)),
		needsFree: !conf.CopyData,
	}, nil
}

func (conf *Configuration) LoadHdr(r io.Reader) (*RgbaHdr, error) {
	conf.apply()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return conf.LoadHdrBytes(b)
}

func (conf *Configuration) LoadHdrBytes(b []byte) (*RgbaHdr, error) {
	conf.apply()
	var x, y C.int
	mem := (*C.uchar)(unsafe.Pointer(&b[0]))
	data := C.stbi_loadf_from_memory(mem, C.int(len(b)), &x, &y, nil, 4)
	if data == nil {
		msg := C.GoString(C.stbi_failure_reason())
		return nil, errors.New(msg)
	}

	var pix []float32
	if conf.CopyData {
		defer C.stbi_image_free(unsafe.Pointer(data))
		bytes := C.GoBytes(unsafe.Pointer(data), y*x*4*4)
		pix = unsafe.Slice((*float32)(unsafe.Pointer(&bytes[0])), y*x*4)
	} else {
		pix = unsafe.Slice((*float32)(data), y*x*4)
	}

	return &RgbaHdr{
		Pix:       pix,
		Stride:    int(x) * 4,
		Rect:      image.Rect(0, 0, int(x), int(y)),
		needsFree: !conf.CopyData,
	}, nil
}

func Load(r io.Reader) (*RgbaLdr, error) {
	return Default.Load(r)
}

func LoadBytes(b []byte) (*RgbaLdr, error) {
	return Default.LoadBytes(b)
}

func LoadHdr(r io.Reader) (*RgbaHdr, error) {
	return Default.LoadHdr(r)
}

func LoadHdrBytes(b []byte) (*RgbaHdr, error) {
	return Default.LoadHdrBytes(b)
}
