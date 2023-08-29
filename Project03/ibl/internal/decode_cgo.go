package ibl_internal

// #include "../c/decode.c"
import "C"
import "unsafe"

func DecodeRgbe(components int, data []byte, buf []float32) int {
	// bounds check
	_ = buf[len(data)*components/4-1]
	n := C.DecodeRgbe(C.uint64_t(components), C.uint64_t(len(data)), (*C.uchar)(unsafe.Pointer(&data[0])), (*C.float)(unsafe.Pointer(&buf[0])))

	return int(n)
}
