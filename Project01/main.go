package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const WindowWidth, WindowHeight = 1920, 1080

var ViewportWidth, ViewportHeight int

type context = *glfw.Window

type arguments struct {
	DisableShaderCache         bool
	EnableCompatibilityProfile bool
}

var Arguments = arguments{}

func main() {
	flag.BoolVar(&Arguments.DisableShaderCache, "disable-shader-cache", Arguments.DisableShaderCache, "")
	flag.BoolVar(&Arguments.EnableCompatibilityProfile, "enable-compatibility-profile", Arguments.EnableCompatibilityProfile, "")
	flag.Parse()

	ctx, err := initGLFW()
	if err != nil {
		log.Panic(err)
	}
	defer glfw.Terminate()
	err = initGL(ctx)
	if err != nil {
		log.Panic(err)
	}

	ctx.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	GlEnv = GetGlEnv()
	GlState = NewStateManager()
	Input.Init(ctx)
	Gui = NewImGui()
	Setup(ctx)
	ctx.Show()
	for !ctx.ShouldClose() {
		glfw.PollEvents()
		Input.Update(ctx)
		Draw(ctx)
		if ui.accuratePerformance {
			// Finish to get accurate time
			gl.Finish()
		}
		DrawUi()
		Gui.Draw()
		ctx.SwapBuffers()
	}
}

func initGLFW() (context, error) {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return nil, err
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	if Arguments.EnableCompatibilityProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
	} else {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False)

	ctx, err := glfw.CreateWindow(WindowWidth, WindowHeight, "Deferred GL - Project 1", nil, nil)
	if err != nil {
		return nil, err
	}
	ctx.MakeContextCurrent()
	_, top, _, _ := ctx.GetFrameSize()
	ctx.SetPos(0, top)
	glfw.SwapInterval(0)

	return ctx, nil
}

func initGL(ctx context) error {
	init := gl.InitWithProcAddrFunc
	vendorSuffixes := []string{"3DFX", "PGI", "SGIX", "SGIS", "SGI", "IBM", "HP", "NV", "NVX", "INGR", "ARB", "EXT", "AMD", "ATI", "MESA", "KHR", "INTEL", "GREMEDY", "APPLE", "OES", "SUN", "SUNX"}
	err := init(func(name string) unsafe.Pointer {
		addr := glfw.GetProcAddress(name)
		if addr == nil {
			vendorSuffix := false
			for _, suffix := range vendorSuffixes {
				if strings.HasSuffix(name, suffix) {
					vendorSuffix = true
				}
			}
			if !vendorSuffix {
				log.Printf("Proc missing: %v\n", name)
			}
			return unsafe.Pointer(uintptr(0xffff_ffff_ffff_ffff))
		}
		return addr
	})
	if err != nil {
		return err
	}
	gl.Enable(gl.DEBUG_OUTPUT)
	gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)
	groupStack := []string{"top"}
	gl.DebugMessageCallback(
		func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
			if gltype == gl.DEBUG_TYPE_PUSH_GROUP {
				groupStack = append(groupStack, message)
				return
			} else if gltype == gl.DEBUG_TYPE_POP_GROUP {
				groupStack = groupStack[:len(groupStack)-1]
				return
			}
			var (
				severityStr string
				typeStr     string
				sourceStr   string
			)
			switch severity {
			case gl.DEBUG_SEVERITY_HIGH:
				severityStr = "CRITICAL_ERROR"
			case gl.DEBUG_SEVERITY_MEDIUM:
				severityStr = "ERROR"
			case gl.DEBUG_SEVERITY_LOW:
				severityStr = "WARNING"
			case gl.DEBUG_SEVERITY_NOTIFICATION:
				severityStr = "INFO"
			}
			switch gltype {
			case gl.DEBUG_TYPE_ERROR:
				typeStr = "ERROR"
			case gl.DEBUG_TYPE_DEPRECATED_BEHAVIOR:
				typeStr = "DEPRECATED_BEHAVIOR"
			case gl.DEBUG_TYPE_UNDEFINED_BEHAVIOR:
				typeStr = "UNDEFINED_BEHAVIOR"
			case gl.DEBUG_TYPE_PERFORMANCE:
				typeStr = "PERFORMANCE"
			case gl.DEBUG_TYPE_PORTABILITY:
				typeStr = "PORTABILITY"
			case gl.DEBUG_TYPE_OTHER:
				typeStr = "OTHER"
			case gl.DEBUG_TYPE_MARKER:
				typeStr = "MARKER"
			case gl.DEBUG_TYPE_PUSH_GROUP:
				typeStr = "PUSH_GROUP"
			case gl.DEBUG_TYPE_POP_GROUP:
				typeStr = "POP_GROUP"
			}
			switch source {
			case gl.DEBUG_SOURCE_API:
				sourceStr = "GRAPHICS_LIBRARY"
			case gl.DEBUG_SOURCE_SHADER_COMPILER:
				sourceStr = "SHADER_COMPILER"
			case gl.DEBUG_SOURCE_WINDOW_SYSTEM:
				sourceStr = "WINDOW_SYSTEM"
			case gl.DEBUG_SOURCE_THIRD_PARTY:
				sourceStr = "THIRD_PARTY"
			case gl.DEBUG_SOURCE_APPLICATION:
				sourceStr = "APPLICATION"
			case gl.DEBUG_SOURCE_OTHER:
				sourceStr = "OTHER"
			}
			err := fmt.Sprintf("[%v] %v #%v from %v: %v", severityStr, typeStr, id, sourceStr, message)
			if severity == gl.DEBUG_SEVERITY_HIGH {
				stack := strings.Join(groupStack, " > ")
				log.Panicf("%v\ndebug stack: %v", err, stack)
			}
			log.Println(err)
		}, nil)
	disabledMessages := []uint32{131185}
	gl.DebugMessageControl(gl.DEBUG_SOURCE_API, gl.DEBUG_TYPE_OTHER, gl.DONT_CARE, int32(len(disabledMessages)), &disabledMessages[0], false)
	disabledMessages = []uint32{131222}
	gl.DebugMessageControl(gl.DEBUG_SOURCE_API, gl.DEBUG_TYPE_UNDEFINED_BEHAVIOR, gl.DONT_CARE, int32(len(disabledMessages)), &disabledMessages[0], false)

	dims := [4]int32{}
	gl.GetIntegerv(gl.VIEWPORT, &dims[0])
	ViewportWidth = int(dims[2])
	ViewportHeight = int(dims[3])

	return nil
}
