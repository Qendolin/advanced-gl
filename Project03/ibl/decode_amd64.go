//go:build amd64

package ibl

import (
	"fmt"
	"unsafe"
)

//go:noescape
func _DecodeRgbe(components uint64, len uint64, data unsafe.Pointer, buf unsafe.Pointer) (n uint64)

func decodeRgbeChunk(components int, data []byte, buf []float32) int {
	// bounds check
	required := len(data) * components / 4
	if len(buf) < required {
		panic(fmt.Errorf("buffer too small, only %d of %d", len(buf), required))
	}
	n := _DecodeRgbe(uint64(components), uint64(len(data)), unsafe.Pointer(&data[0]), unsafe.Pointer(&buf[0]))

	return int(n)
}
