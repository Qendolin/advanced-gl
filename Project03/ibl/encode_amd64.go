//go:build amd64

package ibl

import (
	"fmt"
	"unsafe"
)

//go:noescape
func _EncodeRgbe(components uint64, len uint64, data unsafe.Pointer, buf unsafe.Pointer) (n uint64)

func encodeRgbeChunk(components int, data []float32, buf []byte) int {
	// bounds check
	required := len(data) * 4 / components
	if len(buf) < required {
		panic(fmt.Errorf("buffer too small, only %d of %d", len(buf), required))
	}
	n := _EncodeRgbe(uint64(components), uint64(len(data)), unsafe.Pointer(&data[0]), unsafe.Pointer(&buf[0]))

	return int(n)
}
