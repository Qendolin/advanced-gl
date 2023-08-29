package ibl

import (
	libio "advanced-gl/Project03/libio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/pierrec/lz4/v4"
)

func DecodeIblEnv(r io.Reader) (env *IblEnv, err error) {
	var br *libio.BinaryReader
	var ok bool

	if br, ok = r.(*libio.BinaryReader); !ok {
		br = &libio.BinaryReader{
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

	header := IblEnvHeader{}
	if !br.ReadRef(&header) {
		return nil, fmt.Errorf("expected environment header; byte 0x%08x", br.LastIndex)
	}

	if header.Check != MagicNumberIBLENV {
		return nil, fmt.Errorf("environment header is corrupt; byte 0x%08x", br.LastIndex)
	}

	if header.Version != IblEnvVersion1_001_000 {
		return nil, fmt.Errorf("environment version %d unsupported; byte 0x%08x", header.Version, br.LastIndex)
	}

	pixr := br.Src
	if header.Compression == IblEnvCompressionLZ4 || header.Compression == IblEnvCompressionLZ4Fast {
		pixr = lz4.NewReader(br.Src)
	} else if header.Compression != IblEnvCompressionNone {
		return nil, fmt.Errorf("environment compression id %d unsupported; byte 0x%08x", header.Compression, br.LastIndex)
	}

	pixels := 6 * header.Size * header.Size
	data := make([]byte, pixels*4)
	_, err = io.ReadFull(pixr, data)
	if err != nil {
		return nil, fmt.Errorf("expected %d encoded pixels; %w", pixels, err)
	}

	colors, err := DecodeRgbe(bytes.NewBuffer(data), false)
	if err != nil {
		return nil, fmt.Errorf("decoding error: %w", err)
	}

	return NewIblEnv(colors, int(header.Size)), nil
}

func DecodeRgbe(r io.Reader, hasAlpha bool) ([]float32, error) {
	// 16 kib
	rbuf := make([]byte, 16384)

	components := 4
	wsize := 16384
	if !hasAlpha {
		components = 3
		wsize = 12288
	}
	result := make([]float32, wsize)

	wn := 0
	for i := 0; ; i++ {
		rn, err := io.ReadFull(r, rbuf)

		if err != nil && !(errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)) {
			// UnexpectedEOF is expected
			return nil, err
		}

		if rn == 0 {
			break
		}

		if rn%4 != 0 {
			return nil, fmt.Errorf("source not a multiple of 4 bytes")
		}

		cap := cap(result)
		mincap := i*wsize + wsize
		if cap < mincap {
			old := result
			// grow by 25%
			newsize := (cap * 5) / 4
			if newsize < cap+4*wsize {
				// grow fast at the start
				newsize = cap + 4*wsize
			}
			result = make([]float32, newsize)
			copy(result, old)
		}

		chunk := result[i*wsize : i*wsize+wsize]
		wn += decodeRgbeChunk(components, rbuf[:rn], chunk)

		if err != nil {
			break
		}
	}

	result = result[:wn]

	return result, nil
}

func DecodeRgbeBytes(data []byte, hasAlpha bool) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("source not a multiple of 4 bytes")
	}

	components := 4
	if !hasAlpha {
		components = 3
	}
	result := make([]float32, components*len(data)/4)

	n := decodeRgbeChunk(components, data, result)

	result = result[:n]

	return result, nil
}
