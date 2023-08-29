//go:build !amd64

package ibl

import ibl_internal "advanced-gl/Project03/ibl/internal"

func decodeRgbeChunk(components int, data []byte, buf []float32) (n int) {
	return ibl_internal.DecodeRgbe(components, data, buf)
}
