package main

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/inkyblackness/imgui-go/v4"
)

type ImGui struct {
	IO        imgui.IO
	FrameTime float32
	vao       uint32
	vbo       uint32
	vboSize   int
	ebo       uint32
	eboSize   int
	shader    UnboundShaderPipeline
}

var Gui *ImGui

func NewImGui(shader UnboundShaderPipeline) *ImGui {
	imgui.CreateContext(nil)

	io := imgui.CurrentIO()
	win := glfw.GetCurrentContext()
	dispWidth, dispHeight := win.GetSize()
	io.SetDisplaySize(imgui.Vec2{X: float32(dispWidth), Y: float32(dispHeight)})
	imgui.StyleColorsDark()

	var vao uint32
	gl.CreateVertexArrays(1, &vao)

	_, vertexOffsetPos, vertexOffsetUv, vertexOffsetCol := imgui.VertexBufferLayout()
	gl.EnableVertexArrayAttrib(vao, 0)
	gl.VertexArrayAttribFormat(vao, 0, 2, gl.FLOAT, false, uint32(vertexOffsetPos))
	gl.VertexArrayAttribBinding(vao, 0, 0)
	gl.EnableVertexArrayAttrib(vao, 1)
	gl.VertexArrayAttribFormat(vao, 1, 2, gl.FLOAT, false, uint32(vertexOffsetUv))
	gl.VertexArrayAttribBinding(vao, 1, 0)
	gl.EnableVertexArrayAttrib(vao, 2)
	gl.VertexArrayAttribFormat(vao, 2, 4, gl.UNSIGNED_BYTE, true, uint32(vertexOffsetCol))
	gl.VertexArrayAttribBinding(vao, 2, 0)

	image := io.Fonts().TextureDataRGBA32()
	var atlas uint32
	gl.CreateTextures(gl.TEXTURE_2D, 1, &atlas)
	gl.TextureStorage2D(atlas, 1, gl.RGBA32F, int32(image.Width), int32(image.Height))
	gl.TextureSubImage2D(atlas, 0, 0, 0, int32(image.Width), int32(image.Height), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer((*byte)(image.Pixels)))
	gl.GenerateTextureMipmap(atlas)
	io.Fonts().SetTextureID(imgui.TextureID(atlas))

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
		vbo:       0,
		vboSize:   0,
		ebo:       0,
		eboSize:   0,
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

	GlState.BindVertexArray(gui.vao)
	gui.shader.Bind()
	gui.shader.Get(gl.VERTEX_SHADER).SetUniform("u_proj_mat", ortho)

	GlState.SetEnabled(Blend, ScissorTest)
	GlState.BlendEquation(BlendFuncAdd)
	GlState.BlendFunc(BlendSrcAlpha, BlendOneMinusSrcAlpha)
	GlState.ActiveTextue(0)
	GlState.BindSampler(0, 0)

	imgui.Render()
	drawData := imgui.RenderedDrawData()
	drawData.ScaleClipRects(imgui.Vec2{
		X: float32(fbWidth) / float32(dispWidth),
		Y: float32(fbHeight) / float32(dispHeight),
	})

	for _, list := range drawData.CommandLists() {
		vertexBuffer, vertexBufferSize := list.VertexBuffer()
		if vertexBufferSize > gui.vboSize {
			vertexSize, _, _, _ := imgui.VertexBufferLayout()
			gui.vboSize = vertexBufferSize
			gl.CreateBuffers(1, &gui.vbo)
			gl.NamedBufferStorage(gui.vbo, gui.vboSize, nil, gl.DYNAMIC_STORAGE_BIT)
			gl.VertexArrayVertexBuffer(gui.vao, 0, gui.vbo, 0, int32(vertexSize))
		}
		if vertexBufferSize > 0 {
			gl.NamedBufferSubData(gui.vbo, 0, vertexBufferSize, vertexBuffer)
		}

		indexBuffer, indexBufferSize := list.IndexBuffer()
		if indexBufferSize > gui.eboSize {
			gui.eboSize = indexBufferSize
			gl.CreateBuffers(1, &gui.ebo)
			gl.NamedBufferStorage(gui.ebo, gui.eboSize, nil, gl.DYNAMIC_STORAGE_BIT)
			gl.VertexArrayElementBuffer(gui.vao, gui.ebo)
		}
		if indexBufferSize > 0 {
			gl.NamedBufferSubData(gui.ebo, 0, indexBufferSize, indexBuffer)
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
