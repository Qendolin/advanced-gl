package ibl_test

import (
	"advanced-gl/Project03/ibl"
	ibl_internal "advanced-gl/Project03/ibl/internal"
	"bytes"
	"io"
	"testing"

	"github.com/chewxy/math32"
	"github.com/pierrec/lz4/v4"
)

func TestEncodeRgbeChunk(t *testing.T) {
	data := randomFloats(300, 0, 100)
	buf := make([]byte, 400)
	checkBuf := bytes.NewBuffer(nil)
	ibl.EncodeRgbe(checkBuf, data, false)
	ibl.EncodeRgbeChunk(3, data, buf)

	check := checkBuf.Bytes()
	for i := 0; i < len(buf); i++ {
		if buf[i] != check[i] {
			t.Errorf("Encoded byte %d should be %02x but was %02x\n", i, check[i], buf[i])
		}
	}
}

func TestEncodeRgbe(t *testing.T) {
	data := randomFloats(30000, 0, 100)
	checkBuf := bytes.NewBuffer(nil)
	ibl.EncodeRgbe(checkBuf, data, false)
	resultBuf := bytes.NewBuffer(nil)
	ibl.EncodeRgbe(resultBuf, data, false)

	check := checkBuf.Bytes()
	result := resultBuf.Bytes()

	if len(result) != len(check) {
		t.Errorf("Decoded length should be %d but was %d\n", len(check), len(result))
	}

	for i := 0; i < len(result); i++ {
		if result[i] != check[i] {
			t.Errorf("Encoded byte %d should be %02x but was %02x\n", i, check[i], result[i])
		}
	}
}

func TestEncodeRgbeChunkAlpha(t *testing.T) {
	data := randomFloats(400, 0, 100)
	buf := make([]byte, 400)
	checkBuf := bytes.NewBuffer(nil)
	ibl.EncodeRgbe(checkBuf, data, true)
	ibl.EncodeRgbeChunk(4, data, buf)

	check := checkBuf.Bytes()
	for i := 0; i < len(buf); i++ {
		if buf[i] != check[i] {
			t.Errorf("Encoded byte %d should be %02x but was %02x\n", i, check[i], buf[i])
		}
	}
}

func BenchmarkEncodeRgbeGoPure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		encodeRgbeGo(testdata.writer, testdata.hdr.Pix, true)
	}
}

func BenchmarkEncodeRgbeGoAsm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ibl.EncodeRgbe(testdata.writer, testdata.hdr.Pix, true)
	}
}

func BenchmarkEncodeRgbeChunkGoAsm(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ibl.EncodeRgbeChunk(4, testdata.hdr.Pix, testdata.byteBuffer)
	}
}

func BenchmarkEncodeRgbeChunkCGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ibl_internal.EncodeRgbe(4, testdata.hdr.Pix, testdata.byteBuffer)
	}
}

func BenchmarkEncodeOnly(b *testing.B) {
	w := &mockWriter{}
	for i := 0; i < b.N; i++ {
		err := ibl.EncodeRgbe(w, testdata.hdr.Pix, true)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeCompressFast(b *testing.B) {
	lzw := lz4.NewWriter(&mockWriter{})
	lzw.Apply(lz4.CompressionLevelOption(lz4.Fast))
	for i := 0; i < b.N; i++ {
		err := ibl.EncodeRgbe(lzw, testdata.hdr.Pix, true)
		if err != nil {
			b.Error(err)
		}
		err = lzw.Flush()
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkEncodeCompressLevel1(b *testing.B) {
	lzw := lz4.NewWriter(&mockWriter{})
	lzw.Apply(lz4.CompressionLevelOption(lz4.Level1))
	for i := 0; i < b.N; i++ {
		err := ibl.EncodeRgbe(lzw, testdata.hdr.Pix, true)
		if err != nil {
			b.Error(err)
		}
		err = lzw.Flush()
		if err != nil {
			b.Error(err)
		}
	}
}

// See: https://www.graphics.cornell.edu/~bjw/rgbe/rgbe.c
func encodeRgbeGo(w io.Writer, data []float32, hasAlpha bool) error {
	// 16kb
	size := 16384
	buf := make([]byte, size)
	components := 3
	if hasAlpha {
		components = 4
	}
	var j int
	for i := 0; i < len(data)/components; i++ {
		var (
			r = data[i*components+0]
			g = data[i*components+1]
			b = data[i*components+2]
		)

		j = (i * 4) % size

		max := r
		if g > max {
			max = g
		}
		if b > max {
			max = b
		}

		if max < 1e-32 {
			buf[j+0] = 0
			buf[j+1] = 0
			buf[j+2] = 0
			buf[j+3] = 0
		} else {
			frac, exp := math32.Frexp(max)
			f := frac * 256.0 / max
			buf[j+0] = byte(r * f)
			buf[j+1] = byte(g * f)
			buf[j+2] = byte(b * f)
			buf[j+3] = byte(exp + 128)
		}

		if j+4 == size {
			_, err := w.Write(buf)
			if err != nil {
				return err
			}
		}
	}

	if j+4 < size {
		_, err := w.Write(buf[:j+4])
		if err != nil {
			return err
		}
	}

	return nil
}
