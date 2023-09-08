package ibl_test

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libgl"
	"advanced-gl/Project03/libio"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand"
	"runtime"
	"unsafe"

	"advanced-gl/Project03/stbi"
	"os"
	"testing"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/pierrec/lz4/v4"
)

var onMain chan func()
var onMainDone chan struct{}

var context *glfw.Window

type bufferWriter []byte

func (bw bufferWriter) Write(b []byte) (int, error) {
	return copy(bw, b), nil
}

type mockWriter struct{}

func (w *mockWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

var testdata struct {
	rgbeEnc                         []byte
	byteBuffer                      []byte
	writer                          bufferWriter
	hdr                             *stbi.RgbaHdr
	iblEnv                          *ibl.IblEnv
	hdrEncRaw                       []byte
	iblLevelNone                    []byte
	iblLevelFast                    []byte
	iblLevel1                       []byte
	iblStudioSmall                  *ibl.IblEnv
	iblStudioSmallSpecularReference *ibl.IblEnv
}

func TestMain(m *testing.M) {
	runtime.LockOSThread()
	var err error

	check(glfw.Init())
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	ctx, err := glfw.CreateWindow(640, 480, "Testing Window", nil, nil)
	check(err)
	ctx.MakeContextCurrent()
	context = ctx

	check(gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		addr := glfw.GetProcAddress(name)
		if addr == nil {
			return unsafe.Pointer(uintptr(0xffff_ffff_ffff_ffff))
		}
		return addr
	}))

	libgl.GlEnv = libgl.GetGlEnv()
	libgl.GlState = libgl.NewGlStateManager()
	libgl.GlState.Enable(gl.DEBUG_OUTPUT)
	libgl.GlState.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)

	gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
		fmt.Printf("GL: %v\n", message)
	}, nil)

	hdrFile, err := os.Open("./testdata/cubemap_testing.hdr.lz4")
	check(err)
	lzr := lz4.NewReader(hdrFile)
	hdrEncRaw, err := io.ReadAll(lzr)
	check(err)

	stbi.Default.CopyData = true
	stbi.Default.FlipVertically = true
	hdr, err := stbi.LoadHdrBytes(hdrEncRaw)
	check(err)

	testdata.hdr = hdr
	testdata.byteBuffer = make([]byte, len(hdr.Pix))
	testdata.writer = make([]byte, len(hdr.Pix)/3*4)

	buf := new(bytes.Buffer)
	err = ibl.EncodeRgbe(buf, hdr.Pix, true)
	check(err)
	testdata.rgbeEnc = buf.Bytes()

	iblFile, err := os.Open("./testdata/cubemap_testing_full_uncompressed.iblenv.lz4")
	lzr = lz4.NewReader(iblFile)
	check(err)
	testdata.iblLevelNone, err = io.ReadAll(lzr)
	check(err)

	testdata.iblEnv, err = ibl.DecodeOldIblEnv(bytes.NewBuffer(testdata.iblLevelNone))
	check(err)

	iblFile, err = os.Open("./testdata/cubemap_testing_full_fast_compressed.iblenv")
	check(err)
	testdata.iblLevelFast, err = io.ReadAll(iblFile)
	check(err)

	iblFile, err = os.Open("./testdata/cubemap_testing_full_low_compressed.iblenv")
	check(err)
	testdata.iblLevel1, err = io.ReadAll(iblFile)
	check(err)

	iblFile, err = os.Open("./testdata/studio_small.iblenv")
	check(err)
	iblStudioSmallData, err := io.ReadAll(iblFile)
	check(err)
	testdata.iblStudioSmall, err = ibl.DecodeOldIblEnv(bytes.NewBuffer(iblStudioSmallData))
	check(err)

	iblFile, err = os.Open("./testdata/studio_small_specular_reference.iblenv")
	check(err)
	iblStudioSmallSpecularReferenceData, err := io.ReadAll(iblFile)
	check(err)
	testdata.iblStudioSmallSpecularReference, err = ibl.DecodeOldIblEnv(bytes.NewBuffer(iblStudioSmallSpecularReferenceData))
	check(err)

	onMain = make(chan func())
	onMainDone = make(chan struct{})

	go func() {
		os.Exit(m.Run())
	}()

	for fn := range onMain {
		fn()
		onMainDone <- struct{}{}
	}
}

func compressHdrFile() {
	src, err := os.Open("./testdata/cubemap_testing_full_uncompressed.iblenv")
	check(err)
	dst, err := os.OpenFile("./testdata/cubemap_testing_full_uncompressed.iblenv.lz4", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	check(err)

	lzw := lz4.NewWriter(dst)
	lzw.Apply(lz4.CompressionLevelOption(lz4.Level9))

	_, err = io.Copy(lzw, src)
	check(err)

	lzw.Flush()
	src.Close()
	dst.Close()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type Failer interface {
	Error(...any)
}

func close(closer io.Closer, fail Failer) {
	if err := closer.Close(); err != nil {
		fail.Error(err)
	}
}

func randomFloats(count int, min, max float32) []float32 {
	rng := rand.New(rand.NewSource(0))
	ret := make([]float32, count)
	for i := range ret {
		ret[i] = rng.Float32()*(max-min) + min
	}
	return ret
}

func saveResultIbl(name string, hdri *ibl.IblEnv) {
	file, err := os.OpenFile(fmt.Sprintf("testout/%s.iblenv", name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	ibl.EncodeIblEnv(file, hdri, ibl.OptCompress(1))

	for i := 0; i < hdri.Levels; i++ {
		file, err = os.OpenFile(fmt.Sprintf("testout/%s_%d.png", name, i), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if err != nil {
			return
		}
		defer file.Close()

		size := hdri.Size(i)
		rgba := image.NewRGBA(image.Rect(0, 0, size, size*6))
		hdr := hdri.Level(i)
		for i := 0; i < size*size*6; i++ {
			rgba.Pix[i*4+0] = uint8(math32.Min(hdr[i*3+0], 1.0) * 0xff)
			rgba.Pix[i*4+1] = uint8(math32.Min(hdr[i*3+1], 1.0) * 0xff)
			rgba.Pix[i*4+2] = uint8(math32.Min(hdr[i*3+2], 1.0) * 0xff)
			rgba.Pix[i*4+3] = 0xff
		}

		png.Encode(file, rgba)
	}

}

func saveResultFloatImage(name string, img *libio.FloatImage, gamma, scale float32) {
	file, err := os.OpenFile(fmt.Sprintf("testout/%s.png", name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()
	img.Tonemap(gamma, scale)
	png.Encode(file, img.ToIntImage().ToRGBA())
}
