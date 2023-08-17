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
	"io"
	"unsafe"
)

type Configuration struct {
	HdrToLdrGamma, HdrToLdrScale float32
	LdrToHdrGamma, LdrToHdrScale float32
	Unpremultiply                bool
	FlipVertically               bool
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

func (conf *Configuration) Load(r io.Reader) (*image.RGBA, error) {
	conf.apply()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return conf.LoadBytes(b)
}

func (conf *Configuration) LoadBytes(b []byte) (*image.RGBA, error) {
	conf.apply()
	var x, y C.int
	mem := (*C.uchar)(unsafe.Pointer(&b[0]))
	data := C.stbi_load_from_memory(mem, C.int(len(b)), &x, &y, nil, 4)
	if data == nil {
		msg := C.GoString(C.stbi_failure_reason())
		return nil, errors.New(msg)
	}
	defer C.stbi_image_free(unsafe.Pointer(data))

	return &image.RGBA{
		Pix:    C.GoBytes(unsafe.Pointer(data), y*x*4),
		Stride: 4,
		Rect:   image.Rect(0, 0, int(x), int(y)),
	}, nil
}

// Load wraps stbi_load to decode an image into an RGBA pixel struct.
func Load(r io.Reader) (*image.RGBA, error) {
	return Default.Load(r)
}

func LoadBytes(b []byte) (*image.RGBA, error) {
	return Default.LoadBytes(b)
}
