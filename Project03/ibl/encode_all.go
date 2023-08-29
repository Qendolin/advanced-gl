//go:build !amd64

package ibl

import ibl_internal "advanced-gl/Project03/ibl/internal"

func encodeRgbeChunk(components int, data []float32, buf []byte) {
	ibl_internal.EncodeRgbe(components, data, buf)
}
