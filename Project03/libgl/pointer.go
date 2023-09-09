package libgl

import (
	"fmt"
	"reflect"
	"unsafe"
)

func Pointer(data any) unsafe.Pointer {
	if data == nil {
		return unsafe.Pointer(nil)
	}
	var addr unsafe.Pointer
	v := reflect.ValueOf(data)
	switch v.Type().Kind() {
	case reflect.Ptr:
		e := v.Elem()
		addr = unsafe.Pointer(e.UnsafeAddr())
	case reflect.Uintptr:
		addr = unsafe.Pointer(data.(uintptr))
	case reflect.Slice:
		addr = unsafe.Pointer(v.Index(0).UnsafeAddr())
	default:
		panic(fmt.Errorf("unsupported type %s; must be a slice, uintptr or pointer to a value", v.Type()))
	}
	return addr
}
