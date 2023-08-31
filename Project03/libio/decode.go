package libio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/chewxy/math32"
	"github.com/pierrec/lz4/v4"
)

func DecodeFloatImage(r io.Reader) (img *FloatImage, err error) {
	var br *BinaryReader
	var ok bool

	if br, ok = r.(*BinaryReader); !ok {
		br = &BinaryReader{
			Src:   r,
			Order: binary.LittleEndian,
		}

		defer func() {
			if br.Err != nil {
				if err == nil {
					err = br.Err
				} else {
					err = fmt.Errorf("%v: %w", err, br.Err)
				}
			}
		}()
	}

	header := FloatImageHeader{}
	if !br.ReadRef(&header) {
		return nil, fmt.Errorf("expected f32 header; byte 0x%08x", br.LastIndex)
	}

	if header.Check != MagicNumberF32 {
		return nil, fmt.Errorf("f32 header is corrupt; byte 0x%08x", br.LastIndex)
	}

	if header.Version != F32Version1_001_000 {
		return nil, fmt.Errorf("f32 version %d unsupported; byte 0x%08x", header.Version, br.LastIndex)
	}

	var data []float32

	switch header.Compression {
	case FloatImageCompressionNone:
		data = make([]float32, header.Width*header.Height*uint32(header.Channels))
		br.ReadRef(data)
		err = br.Err
	case FloatImageCompressionFixedPoint16Lz4:
		rangeBytes := 4 * 2 * int(header.Channels)
		dataBytes := int(header.Width*header.Height) * int(header.Channels) * 2
		buf := make([]byte, rangeBytes+dataBytes)
		lzr := lz4.NewReader(br.Src)
		_, err = io.ReadFull(lzr, buf)
		if err != nil {
			break
		}
		data, err = decompressFixedPoint16(int(header.Channels), int(header.Width*header.Height), buf)
	}

	if err != nil {
		return nil, fmt.Errorf("could not decompress f32 pixels: %w", err)
	}

	return NewFloatImage(data, int(header.Channels), int(header.Width), int(header.Height)), nil
}

func decompressFixedPoint16(channels, count int, data []byte) ([]float32, error) {
	result := make([]float32, count*channels)
	br := &BinaryReader{
		Src:   bytes.NewBuffer(data),
		Order: binary.LittleEndian,
	}
	for ch := 0; ch < channels; ch++ {
		decompressChannelFixedPoint16(channels, count, result, br, ch)
		if br.Err != nil {
			return nil, br.Err
		}
	}
	return result, nil
}

func decompressChannelFixedPoint16(channels, count int, pix []float32, br *BinaryReader, ch int) {
	var imin, imax int
	br.ReadUInt32(&imin)
	br.ReadUInt32(&imax)

	min := math32.Float32frombits(uint32(imin))
	max := math32.Float32frombits(uint32(imax))

	data := make([]uint16, count)
	br.ReadRef(data)

	r := max - min
	for i := 0; i < count; i++ {
		fix := data[i]
		flt := (float32(fix)/0xffff)*r + min
		pix[i*channels+ch] = flt
	}
}
