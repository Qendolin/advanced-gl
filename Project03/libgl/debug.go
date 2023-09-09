package libgl

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

type LabeledGlObject interface {
	SetDebugLabel(string)
}

func setObjectLabel(namespace, id uint32, label string) {
	bytes := []byte(label)
	gl.ObjectLabel(namespace, id, int32(len(bytes)), (*uint8)(unsafe.Pointer(&bytes[0])))
}
