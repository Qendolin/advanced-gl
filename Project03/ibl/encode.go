package ibl

import (
	"advanced-gl/Project03/libio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pierrec/lz4/v4"
)

type EncodeContext struct {
	Compression IblEnvCompression
	Writer      io.Writer
}

type EncodeOption func(ctx *EncodeContext) error

func OptCompress(level int) EncodeOption {
	levels := []lz4.CompressionLevel{lz4.Fast, lz4.Level1, lz4.Level2, lz4.Level3, lz4.Level4, lz4.Level5, lz4.Level6, lz4.Level7, lz4.Level8, lz4.Level9}
	if level < 0 {
		return nil
	}

	if level >= len(levels) {
		level = len(levels) - 1
	}

	return func(ctx *EncodeContext) error {
		if ctx.Compression != IblEnvCompressionNone {
			return fmt.Errorf("compression already configured")
		}
		lzw := lz4.NewWriter(ctx.Writer)
		lzw.Apply(lz4.CompressionLevelOption(levels[level]))
		if level == 0 {
			ctx.Compression = IblEnvCompressionLZ4Fast
		} else {
			ctx.Compression = IblEnvCompressionLZ4
		}
		ctx.Writer = lzw
		return nil
	}
}

func EncodeIblEnv(w io.Writer, env *IblEnv, options ...EncodeOption) (err error) {
	var bw *libio.BinaryWriter
	var ok bool

	if bw, ok = w.(*libio.BinaryWriter); !ok {
		bw = &libio.BinaryWriter{
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

	ctx := EncodeContext{
		Writer: bw.Dst,
	}

	for _, opt := range options {
		if opt != nil {
			err = opt(&ctx)
			if err != nil {
				return err
			}
		}
	}

	header := IblEnvHeader{
		Check:       MagicNumberIBLENV,
		Version:     IblEnvVersion1_001_000,
		Compression: ctx.Compression,
		Size:        uint32(env.Size),
	}
	if !bw.WriteRef(&header) {
		return fmt.Errorf("could not write ibl env header: %w", bw.Err)
	}

	if err := EncodeRgbe(ctx.Writer, env.Concat(), false); err != nil {
		return fmt.Errorf("could not write ibl env encoded pixels: %w", err)
	}

	if closer, ok := (ctx.Writer).(io.WriteCloser); ok {
		err = closer.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func EncodeRgbe(w io.Writer, data []float32, hasAlpha bool) error {
	// 16 kib
	components := 4
	rsize := 16384
	if !hasAlpha {
		components = 3
		// 12 kib
		rsize = 12288
	}
	buf := make([]byte, 16384)

	if len(data)%components != 0 {
		return fmt.Errorf("source not a multiple of %d bytes", components)
	}

	len := len(data)
	for i := 0; i < len; i += rsize {
		j := i + rsize
		if j > len {
			j = len
		}
		chunk := data[i:j]
		n := encodeRgbeChunk(components, chunk, buf)

		_, err := w.Write(buf[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func EncodeRgbeBytes(data []float32, hasAlpha bool) ([]byte, error) {
	components := 4
	if !hasAlpha {
		components = 3
	}

	if len(data)%components != 0 {
		return nil, fmt.Errorf("source not a multiple of %d bytes", components)
	}

	result := make([]byte, len(data)*4/components)
	n := encodeRgbeChunk(components, data, result)

	result = result[:n]

	return result, nil
}
