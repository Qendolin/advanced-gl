package ibl_test

import (
	"advanced-gl/Project03/ibl"
	ibl_internal "advanced-gl/Project03/ibl/internal"
	"advanced-gl/Project03/stbi"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/chewxy/math32"
)

func TestDecodeIblEnv(t *testing.T) {
	hdri, err := ibl.DecodeIblEnv(bytes.NewBuffer(testdata.iblLevelFast))
	if err != nil {
		t.Error(err)
		return
	}

	expected := []float32{0.2168, 0.1641, 0.1250, 0.2168, 0.1895, 0.1680}

	for i := 0; i < 6; i++ {
		is := hdri.Face(0, i)[len(hdri.Face(0, i))-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestDecodeRgbeChunk(t *testing.T) {
	hdrData := randomFloats(300, 0, 100)
	rgbeBuf := new(bytes.Buffer)
	ibl.EncodeRgbe(rgbeBuf, hdrData, false)
	rgbe := rgbeBuf.Bytes()
	check, err := ibl.DecodeRgbe(bytes.NewBuffer(rgbe), false)
	if err != nil {
		t.Error(err)
	}

	result := make([]float32, len(hdrData))
	ibl.DecodeRgbeChunk(3, rgbe, result)

	for i := 0; i < len(check); i++ {
		if result[i] != check[i] || math.Abs(float64(result[i]-hdrData[i])) > 0.5 {
			t.Errorf("Decoded float %d should be %.4f but was %.4f\n", i, hdrData[i], result[i])
		}
	}
}

func TestDecodeRgbe(t *testing.T) {
	hdrData := randomFloats(30000, 0, 100)
	rgbeBuf := new(bytes.Buffer)
	ibl.EncodeRgbe(rgbeBuf, hdrData, false)
	rgbe := rgbeBuf.Bytes()
	check, err := ibl.DecodeRgbe(bytes.NewBuffer(rgbe), false)
	if err != nil {
		t.Error(err)
	}

	result, err := ibl.DecodeRgbe(bytes.NewBuffer(rgbe), false)
	if err != nil {
		t.Error(err)
	}

	if len(check) != len(result) {
		t.Fatalf("Decoded length should be %d but was %d\n", len(check), len(result))
	}

	for i := 0; i < len(check); i++ {
		if result[i] != check[i] || math.Abs(float64(result[i]-hdrData[i])) > 0.5 {
			t.Fatalf("Decoded float %d should be %.4f but was %.4f\n", i, hdrData[i], result[i])
		}
	}
}

func TestDecodeRgbeChunkAlpha(t *testing.T) {
	hdrData := randomFloats(400, 0, 100)
	rgbeBuf := new(bytes.Buffer)
	ibl.EncodeRgbe(rgbeBuf, hdrData, true)
	rgbe := rgbeBuf.Bytes()
	check, err := ibl.DecodeRgbe(bytes.NewBuffer(rgbe), true)
	if err != nil {
		t.Error(err)
	}

	result := make([]float32, len(hdrData))
	ibl.DecodeRgbeChunk(4, rgbe, result)

	for i := 0; i < len(check); i++ {
		if result[i] != check[i] {
			t.Errorf("Decoded float %d should be %.4f but was %.4f\n", i, hdrData[i], result[i])
		}
	}
}

func BenchmarkDecodeRgbeGoPure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decodeRgbeGo(bytes.NewBuffer(testdata.rgbeEnc), false)
	}
}

func BenchmarkDecodeRgbeGoAsm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ibl.DecodeRgbe(bytes.NewBuffer(testdata.rgbeEnc), false)
	}
}

func BenchmarkDecodeRgbeChunkGoAsm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// important to alloc here for a fair benchmark
		buf := make([]float32, len(testdata.hdr.Pix))
		ibl.DecodeRgbeChunk(3, testdata.rgbeEnc, buf)
	}
}

func BenchmarkDecodeRgbeChunkCGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := make([]float32, len(testdata.hdr.Pix))
		ibl_internal.DecodeRgbe(3, testdata.rgbeEnc, buf)
	}
}

func BenchmarkDecodeStbiNoCopy(b *testing.B) {
	stbi.Default.CopyData = false
	for i := 0; i < b.N; i++ {
		img, err := stbi.LoadHdrBytes(testdata.hdrEncRaw)
		defer close(img, b)
		if err != nil {
			b.Error(err)
		}
	}
	b.StopTimer()
}

func BenchmarkDecodeStbiCopy(b *testing.B) {
	stbi.Default.CopyData = true
	for i := 0; i < b.N; i++ {
		img, err := stbi.LoadHdrBytes(testdata.hdrEncRaw)
		defer close(img, b)
		if err != nil {
			b.Error(err)
		}
	}
	b.StopTimer()
}

func BenchmarkDecodeOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ibl.DecodeIblEnv(bytes.NewBuffer(testdata.iblLevelNone))
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeCompressFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ibl.DecodeIblEnv(bytes.NewBuffer(testdata.iblLevelFast))
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDecodeCompressLevel1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ibl.DecodeIblEnv(bytes.NewBuffer(testdata.iblLevel1))
		if err != nil {
			b.Error(err)
		}
	}
}

func decodeRgbeGo(r io.Reader, hasAlpha bool) ([]float32, error) {
	// 16 kb
	buf := make([]byte, 16384)

	components := 4
	if !hasAlpha {
		components = 3
	}
	result := make([]float32, 0, 16384/4*components)

	for {
		n, err := io.ReadFull(r, buf)

		if err != nil && !(errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)) {
			// UnexpectedEOF can happen when buf is bigger than all the data
			return nil, err
		}

		if n%4 != 0 {
			return nil, fmt.Errorf("buffer limit not multiple of 4")
		}

		for i := 0; i < n; i += 4 {
			var (
				r = buf[i+0]
				g = buf[i+1]
				b = buf[i+2]
				e = buf[i+3]
			)

			if e == 0 {
				if hasAlpha {
					result = append(result, 0.0, 0.0, 0.0, 1.0)
				} else {
					result = append(result, 0.0, 0.0, 0.0)
				}
				continue
			}

			f := math32.Ldexp(1.0, int(e)-(128+8))
			if hasAlpha {
				result = append(result, float32(r)*f, float32(g)*f, float32(b)*f, 1.0)
			} else {
				result = append(result, float32(r)*f, float32(g)*f, float32(b)*f)
			}
		}

		if err != nil {
			break
		}
	}

	return result[:len(result):len(result)], nil
}
