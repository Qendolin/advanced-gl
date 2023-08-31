package libio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/chewxy/math32"
	"github.com/pierrec/lz4/v4"
)

func EncodeFloatImage(w io.Writer, img *FloatImage, compression FloatImageCompression) (err error) {
	var bw *BinaryWriter
	var ok bool

	if bw, ok = w.(*BinaryWriter); !ok {
		bw = &BinaryWriter{
			Dst:   w,
			Order: binary.LittleEndian,
		}

		defer func() {
			if bw.Err != nil {
				if err == nil {
					err = bw.Err
				} else {
					err = fmt.Errorf("%v: %w", err, bw.Err)
				}
			}
		}()
	}

	header := FloatImageHeader{
		Check:       MagicNumberF32,
		Version:     F32Version1_001_000,
		Width:       uint32(img.Width),
		Height:      uint32(img.Height),
		Channels:    uint8(img.Channels),
		Compression: compression,
	}

	if !bw.WriteRef(header) {
		return fmt.Errorf("could not write f32 header: %w", bw.Err)
	}

	var data []byte

	switch compression {
	case FloatImageCompressionNone:
		buf := bytes.NewBuffer(make([]byte, img.Bytes()))
		err = binary.Write(buf, bw.Order, img.Pix)
	case FloatImageCompressionFixedPoint16Lz4:
		data, err = compressFixedPoint16(img.Channels, img.Count(), img.Pix)
		if err != nil {
			break
		}
		buf := bytes.NewBuffer(nil)
		lzw := lz4.NewWriter(buf)
		lzw.Apply(lz4.CompressionLevelOption(lz4.Fast))
		_, err = lzw.Write(data)
		if err != nil {
			break
		}
		err = lzw.Flush()
		data = buf.Bytes()
	}

	if err != nil {
		return fmt.Errorf("could not compress f32 pixels: %w", err)
	}

	if !bw.WriteBytes(data) {
		return fmt.Errorf("could not write f32 encoded pixels: %w", bw.Err)
	}

	return nil
}

func compressFixedPoint16(channels int, count int, pix []float32) ([]byte, error) {
	rangeBytes := 4 * 2 * channels
	dataBytes := count * channels * 2
	buf := bytes.NewBuffer(make([]byte, 0, rangeBytes+dataBytes))
	bw := &BinaryWriter{Order: binary.LittleEndian, Dst: buf}
	for ch := 0; ch < channels; ch++ {
		compressChannelFixedPoint16(channels, count, pix, bw, ch)
		if bw.Err != nil {
			return nil, bw.Err
		}
	}
	return buf.Bytes(), nil
}

func compressChannelFixedPoint16(channels int, count int, pix []float32, bw *BinaryWriter, ch int) {
	var min, max float32 = math32.Inf(1), math32.Inf(-1)

	for i := 0; i < count; i++ {
		v := pix[i*channels+ch]
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	bw.WriteUInt32(math32.Float32bits(min))
	bw.WriteUInt32(math32.Float32bits(max))

	r := max - min
	for i := 0; i < count; i++ {
		flt := pix[i*channels+ch]
		fix := uint16(((flt - min) / r) * 0xffff)
		bw.WriteUInt16(fix)
	}
}
