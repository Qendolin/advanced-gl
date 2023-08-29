package libutil

// #include <stdlib.h>
// #include "../include/renderdoc_app.h"
// void bridge_set_caputre_file_path_template(pRENDERDOC_SetCaptureFilePathTemplate fn, const char *pathtemplate) {
//   fn(pathtemplate);
// }
// const char *bridge_get_caputre_file_path_template(pRENDERDOC_GetCaptureFilePathTemplate fn) {
//   return fn();
// }
// void bridge_get_api_version(pRENDERDOC_GetAPIVersion fn, int *major, int *minor, int *patch) {
//   fn(major, minor, patch);
// }
// void bridge_set_active_window(pRENDERDOC_SetActiveWindow fn, RENDERDOC_DevicePointer device, RENDERDOC_WindowHandle wndHandle) {
//   fn(device, wndHandle);
// }
// void bridge_start_capture(pRENDERDOC_StartFrameCapture fn, RENDERDOC_DevicePointer device, RENDERDOC_WindowHandle wndHandle) {
//   fn(device, wndHandle);
// }
// uint32_t bridge_end_capture(pRENDERDOC_EndFrameCapture fn, RENDERDOC_DevicePointer device, RENDERDOC_WindowHandle wndHandle) {
//   return fn(device, wndHandle);
// }
import "C"

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type RenderDocApi struct {
	api *C.RENDERDOC_API_1_6_0
}

var renderDocApiInstance *RenderDocApi

// Has to be called before glfw.Init
func LoadRenderDocApi(libPath string) (*RenderDocApi, error) {
	if renderDocApiInstance != nil {
		return renderDocApiInstance, nil
	}

	dll, err := syscall.LoadDLL(libPath)
	if err != nil {
		return nil, err
	}
	getApi, err := dll.FindProc("RENDERDOC_GetAPI")
	if err != nil {
		return nil, err
	}

	api := new(C.RENDERDOC_API_1_6_0)
	ret, _, _ := getApi.Call(uintptr(C.eRENDERDOC_API_Version_1_6_0), uintptr(unsafe.Pointer(&api)))
	if ret == 0 {
		return nil, fmt.Errorf("the RenderDOc API version is invalid or not supported")
	}

	renderDocApiInstance = &RenderDocApi{api}
	return renderDocApiInstance, nil
}

func (rd *RenderDocApi) StartCapture(ctx *glfw.Window) {
	fn := rd.api.StartFrameCapture
	dev, win := getRenderdocContextPointers(ctx)

	C.bridge_start_capture(fn, dev, win)
}

func (rd *RenderDocApi) EndCapture(ctx *glfw.Window) (ok bool) {
	fn := rd.api.EndFrameCapture
	dev, win := getRenderdocContextPointers(ctx)

	ret := C.bridge_end_capture(fn, dev, win)

	return ret == 1
}

func (rd *RenderDocApi) SetActiveWindow(ctx *glfw.Window) {
	fn := rd.api.SetActiveWindow
	dev, win := getRenderdocContextPointers(ctx)

	C.bridge_set_active_window(fn, dev, win)
}

func (rd *RenderDocApi) GetApiVersion() (major, minor, patch int) {
	fn := rd.api.GetAPIVersion

	var cmajor, cminor, cpatch C.int

	C.bridge_get_api_version(fn, &cmajor, &cminor, &cpatch)

	return int(cmajor), int(cminor), int(cpatch)
}

func (rd *RenderDocApi) SetCaptureFilePathTemplate(template string) {
	fn := rd.api.SetCaptureFilePathTemplate

	cstr := C.CString(template + "\x00")
	// Am I supposed to free cstr?
	C.bridge_set_caputre_file_path_template(fn, cstr)
}

func (rd *RenderDocApi) GetCaptureFilePathTemplate() string {
	fn := rd.api.GetCaptureFilePathTemplate

	cstr := C.bridge_get_caputre_file_path_template(fn)
	// Am I supposed to free cstr?
	return C.GoString(cstr)
}

func getRenderdocContextPointers(ctx *glfw.Window) (C.RENDERDOC_DevicePointer, C.RENDERDOC_WindowHandle) {
	if ctx == nil {
		return nil, nil
	}
	return C.RENDERDOC_DevicePointer(unsafe.Pointer(ctx.GetWGLContext())), C.RENDERDOC_WindowHandle(unsafe.Pointer(ctx.GetWin32Window()))
}
