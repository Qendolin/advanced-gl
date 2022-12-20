package main

import (
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

type ImGui struct {
	IO        imgui.IO
	FrameTime float32
	vao       UnboundVertexArray
	vbo       UnboundBuffer
	ebo       UnboundBuffer
	shader    UnboundShaderPipeline
}

var Gui *ImGui

func NewImGui() *ImGui {
	imgui.CreateContext(nil)

	io := imgui.CurrentIO()
	io.SetConfigFlags(imgui.ConfigFlagsNoMouse)
	win := glfw.GetCurrentContext()
	dispWidth, dispHeight := win.GetSize()
	io.SetDisplaySize(imgui.Vec2{X: float32(dispWidth), Y: float32(dispHeight)})
	imgui.StyleColorsDark()

	vao := NewVertexArray()

	vertexSize, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	vao.Layout(0, 0, 2, gl.FLOAT, false, vertexOffsetPos)
	vao.Layout(0, 1, 2, gl.FLOAT, false, vertexOffsetUv)
	vao.Layout(0, 2, 4, gl.UNSIGNED_BYTE, true, vertexOffsetCol)

	vbo := NewBuffer()
	vbo.AllocateEmpty(1024*8, gl.DYNAMIC_STORAGE_BIT)
	vao.BindBuffer(0, vbo, 0, vertexSize)

	ebo := NewBuffer()
	ebo.AllocateEmpty(1024*8, gl.DYNAMIC_STORAGE_BIT)
	vao.BindElementBuffer(ebo)

	shader := NewPipeline()
	vertSh := NewShader(Res_ImguiVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	shader.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	fragSh := NewShader(Res_ImguiFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	shader.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	image := io.Fonts().TextureDataAlpha8()
	atlas := NewTexture(gl.TEXTURE_2D)
	atlas.Allocate(0, gl.R8, image.Width, image.Height, 0)
	atlas.Load(0, image.Width, image.Height, 0, gl.RED, (*byte)(image.Pixels))
	atlas.GenerateMipmap()
	io.Fonts().SetTextureID(imgui.TextureID(atlas.Id()))

	win.SetCursorPosCallback(func(w *glfw.Window, mx, my float64) {
		io.SetMousePosition(imgui.Vec2{X: float32(mx), Y: float32(my)})
	})
	win.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		io.SetMouseButtonDown(int(button), action == glfw.Press)
	})
	win.SetScrollCallback(func(w *glfw.Window, x, y float64) {
		io.AddMouseWheelDelta(float32(x), float32(y))
	})
	win.SetCharCallback(func(w *glfw.Window, char rune) {
		io.AddInputCharacters(string(char))
	})
	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press {
			io.KeyPress(int(key))
		}
		if action == glfw.Release {
			io.KeyRelease(int(key))
		}

		// Modifiers are not reliable across systems
		io.KeyCtrl(int(glfw.KeyLeftControl), int(glfw.KeyRightControl))
		io.KeyShift(int(glfw.KeyLeftShift), int(glfw.KeyRightShift))
		io.KeyAlt(int(glfw.KeyLeftAlt), int(glfw.KeyRightAlt))
		io.KeySuper(int(glfw.KeyLeftSuper), int(glfw.KeyRightSuper))
	})

	io.KeyMap(imgui.KeyTab, int(glfw.KeyTab))
	io.KeyMap(imgui.KeyLeftArrow, int(glfw.KeyLeft))
	io.KeyMap(imgui.KeyRightArrow, int(glfw.KeyRight))
	io.KeyMap(imgui.KeyUpArrow, int(glfw.KeyUp))
	io.KeyMap(imgui.KeyDownArrow, int(glfw.KeyDown))
	io.KeyMap(imgui.KeyPageUp, int(glfw.KeyPageUp))
	io.KeyMap(imgui.KeyPageDown, int(glfw.KeyPageDown))
	io.KeyMap(imgui.KeyHome, int(glfw.KeyHome))
	io.KeyMap(imgui.KeyEnd, int(glfw.KeyEnd))
	io.KeyMap(imgui.KeyInsert, int(glfw.KeyInsert))
	io.KeyMap(imgui.KeyDelete, int(glfw.KeyDelete))
	io.KeyMap(imgui.KeyBackspace, int(glfw.KeyBackspace))
	io.KeyMap(imgui.KeySpace, int(glfw.KeySpace))
	io.KeyMap(imgui.KeyEnter, int(glfw.KeyEnter))
	io.KeyMap(imgui.KeyEscape, int(glfw.KeyEscape))
	io.KeyMap(imgui.KeyA, int(glfw.KeyA))
	io.KeyMap(imgui.KeyC, int(glfw.KeyC))
	io.KeyMap(imgui.KeyV, int(glfw.KeyV))
	io.KeyMap(imgui.KeyX, int(glfw.KeyX))
	io.KeyMap(imgui.KeyY, int(glfw.KeyY))
	io.KeyMap(imgui.KeyZ, int(glfw.KeyZ))

	return &ImGui{
		IO:        io,
		FrameTime: float32(glfw.GetTime()),
		vao:       vao,
		vbo:       vbo,
		ebo:       ebo,
		shader:    shader,
	}
}

func (gui *ImGui) Draw() {
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw ImGui\x00"))
	defer gl.PopDebugGroup()

	io := imgui.CurrentIO()
	win := glfw.GetCurrentContext()

	dispWidth, dispHeight := win.GetSize()
	fbWidth, fbHeight := win.GetFramebufferSize()
	GlState.Viewport(0, 0, fbWidth, fbHeight)
	io.SetDisplaySize(imgui.Vec2{X: float32(dispWidth), Y: float32(dispHeight)})
	ortho := mgl32.Ortho2D(0, float32(dispWidth), float32(dispHeight), 0)

	time := float32(glfw.GetTime())
	io.SetDeltaTime(time - gui.FrameTime)
	gui.FrameTime = time

	gui.vao.Bind()
	gui.shader.Bind()
	gui.shader.Get(gl.VERTEX_SHADER).SetUniform("u_proj_mat", ortho)

	GlState.SetEnabled(Blend, ScissorTest)
	GlState.BlendEquation(BlendFuncAdd)
	GlState.BlendFunc(BlendSrcAlpha, BlendOneMinusSrcAlpha)
	GlState.ActiveTextue(0)

	imgui.Render()
	drawData := imgui.RenderedDrawData()
	drawData.ScaleClipRects(imgui.Vec2{
		X: float32(fbWidth) / float32(dispWidth),
		Y: float32(fbHeight) / float32(dispHeight),
	})

	for _, list := range drawData.CommandLists() {
		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		if gui.vbo.Grow(vertexBufferSize) {
			gui.vao.ReBindBuffer(0, gui.vbo)
		}
		if vertexBufferSize > 0 {
			gui.vbo.Write(0, unsafe.Slice((*byte)(vertexBuffer), vertexBufferSize))
		}

		indexBuffer, indexBufferSize := list.IndexBuffer()
		if gui.ebo.Grow(indexBufferSize) {
			gui.vao.BindElementBuffer(gui.ebo)
		}
		if vertexBufferSize > 0 {
			gui.ebo.Write(0, unsafe.Slice((*byte)(indexBuffer), indexBufferSize))
		}

		var indexType uint32
		indexSize := imgui.IndexBufferLayout()
		switch indexSize {
		case 1:
			indexType = gl.UNSIGNED_BYTE
		case 2:
			indexType = gl.UNSIGNED_SHORT
		case 4:
			indexType = gl.UNSIGNED_INT
		}

		for _, cmd := range list.Commands() {
			if cmd.HasUserCallback() {
				cmd.CallUserCallback(list)
			} else {
				GlState.BindTexture(gl.TEXTURE_2D, uint32(cmd.TextureID()))
				clipRect := cmd.ClipRect()
				x, y := int(clipRect.X), int(fbHeight)-int(clipRect.W)
				if y <= 0 {
					y = 0
				}
				GlState.Scissor(x, y, int(clipRect.Z-clipRect.X), int(clipRect.W-clipRect.Y))
				gl.DrawElementsBaseVertexWithOffset(gl.TRIANGLES, int32(cmd.ElementCount()), indexType, uintptr(cmd.IndexOffset()*indexSize), int32(cmd.VertexOffset()))
			}
		}
	}
}
