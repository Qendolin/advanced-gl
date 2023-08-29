package ibl_internal

// #include "../c/encode.c"
import "C"
import "unsafe"

func EncodeRgbe(components int, data []float32, buf []byte) int {
	// bounds check
	_ = buf[len(data)*components/4-1]
	n := C.EncodeRgbe(C.uint64_t(components), C.uint64_t(len(data)), (*C.float)(unsafe.Pointer(&data[0])), (*C.uchar)(unsafe.Pointer(&buf[0])))

	return int(n)
}
