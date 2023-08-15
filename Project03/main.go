package main

import (
	_ "embed"
	"flag"
	"log"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	im "github.com/inkyblackness/imgui-go/v4"
)

//go:embed assets/shaders/imgui.vert
var Res_ImguiVshSrc string

//go:embed assets/shaders/imgui.frag
var Res_ImguiFshSrc string

//go:embed assets/shaders/pbr.vert
var Res_PbrVshSrc string

//go:embed assets/shaders/pbr.frag
var Res_PbrFshSrc string

//go:embed assets/shaders/direct.vert
var Res_DirectVshSrc string

//go:embed assets/shaders/direct.frag
var Res_DirectFshSrc string

var Arguments struct {
	EnableCompatibilityProfile bool
}

func main() {
	flag.BoolVar(&Arguments.EnableCompatibilityProfile, "enable-compatibility-profile", Arguments.EnableCompatibilityProfile, "")
	flag.Parse()

	runtime.LockOSThread()
	err := glfw.Init()
	check(err)

	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	if Arguments.EnableCompatibilityProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
	} else {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	ctx, err := glfw.CreateWindow(1600, 900, "Cubemap Test", nil, nil)
	check(err)
	ctx.MakeContextCurrent()

	gl.Init()
	err = gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		addr := glfw.GetProcAddress(name)
		if addr == nil {
			return unsafe.Pointer(uintptr(0xffff_ffff_ffff_ffff))
		}
		return addr
	})
	check(err)

	GlState = NewGlStateManager()
	GlEnv = GetGlEnv()
	Input = NewInputManager(ctx)

	batch := NewRenderBatch()

	pack := &DirPack{}
	pack.AddIndexFile("assets/index.json")
	pavementModel, err := pack.LoadModel("white_paint_floor")
	check(err)

	batch.Upload(pavementModel.Mesh)
	batch.AddMaterial(pavementModel.Material)
	batch.Add(pavementModel.Mesh.Name, pavementModel.Material.Name, InstanceAttributes{
		ModelMatrix: mgl32.Ident4(),
	})

	imguiShader, err := pack.LoadShaderPipeline("imgui")
	check(err)
	gui := NewImGui(imguiShader)

	pbrShader, err := pack.LoadShaderPipeline("pbr")
	check(err)

	viewportDims := [4]int32{}
	gl.GetIntegerv(gl.VIEWPORT, &viewportDims[0])
	viewportWidth := int(viewportDims[2])
	viewportHeight := int(viewportDims[3])

	cam := &Camera{
		Position:          mgl32.Vec3{0, 1, 2},
		Orientation:       mgl32.Vec3{30, 0, 0},
		VerticalFov:       70,
		ViewportDimension: mgl32.Vec2{float32(viewportWidth), float32(viewportHeight)},
		ClippingPlanes:    mgl32.Vec2{0.1, 1000},
	}
	cam.UpdateProjectionMatrix()

	textureSampler := NewSampler()
	textureSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	textureSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)

	lightPositions := []mgl32.Vec3{
		{-0.9, 0.5, -0.9},
		{0.9, 0.5, -0.9},
		{-0.9, 0.5, 0.9},
		{0.9, 0.5, 0.9},
	}

	lightColors := []mgl32.Vec3{
		{3.75, 7.0, 1.75},
		{7.0, 2.9, 3.0},
		{7.0, 4.35, 2.05},
		{4.25, 1.75, 7.0},
	}

	wireframe := false

	directShader, err := pack.LoadShaderPipeline("direct")
	check(err)
	dd := NewDirectDrawBuffer(directShader)

	// FIXME: Normal maps don't seem to reflect light from the correct direction

	for !ctx.ShouldClose() {
		glfw.PollEvents()
		Input.Update(ctx)

		movement := Input.GetMovement(glfw.KeyW, glfw.KeyS, glfw.KeyA, glfw.KeyD, glfw.KeySpace, glfw.KeyLeftControl)
		if movement.LenSqr() != 0 {
			movement = movement.Normalize().Mul(Input.TimeDelta())
			cam.Fly(movement)
		}
		if Input.IsMouseDown(glfw.MouseButtonRight) {
			rotation := Input.CursorDelta()
			cam.Orientation[0] += rotation[1] * 0.35
			cam.Orientation[1] += rotation[0] * 0.35
		}
		cam.UpdateViewMatrix()

		GlState.SetEnabled(DepthTest, CullFace)
		GlState.DepthFunc(DepthFuncLess)
		GlState.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		if wireframe {
			GlState.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
		}

		batch.GenerateDrawCommands()
		batch.CommandBuffer.Bind(gl.DRAW_INDIRECT_BUFFER)
		batch.VertexArray.Bind()
		pbrShader.Bind()
		pbrShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", cam.ProjectionMatrix.Mul4(cam.ViewMatrix))
		pbrShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_camera_position", cam.Position)

		for i := 0; i < 4; i++ {
			pbrShader.Get(gl.FRAGMENT_SHADER).SetUniformIndexed("u_light_positions", i, lightPositions[i])
			pbrShader.Get(gl.FRAGMENT_SHADER).SetUniformIndexed("u_light_colors", i, lightColors[i])
		}

		for _, mat := range batch.Materials {
			textureSampler.Bind(0)
			mat.Material.Albedo.Bind(0)
			textureSampler.Bind(1)
			mat.Material.Normal.Bind(1)
			textureSampler.Bind(2)
			mat.Material.ORM.Bind(2)

			gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(mat.ElementOffset), int32(mat.ElementCount), 0)
		}

		im.NewFrame()
		im.Begin("main_window")

		if im.Checkbox("Wireframe", &wireframe) {
			GlState.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
		}

		im.PushID("camera")
		if im.CollapsingHeader("Camera") {
			im.DragFloat3("Pos", (*[3]float32)(&cam.Position))
			im.DragFloat3("Dir", (*[3]float32)(&cam.Orientation))

		}
		im.PopID()

		im.PushID("point_lights")
		if im.CollapsingHeader("Lights") {
			for i := 0; i < 4; i++ {
				if im.TreeNodef("Light %d", i+1) {
					dd.Shaded()
					dd.Light3(lightColors[i])
					dd.UvSphere(lightPositions[i], 0.05)
					dd.Unshaded()

					im.SliderFloat3("Pos", (*[3]float32)(&lightPositions[i]), -5, 5)
					im.ColorEdit3V("Col", (*[3]float32)(&lightColors[i]), im.ColorEditFlagsFloat|im.ColorEditFlagsHSV|im.ColorEditFlagsHDR)

					im.TreePop()
				}
			}
		}
		im.PopID()

		im.End()

		dd.Draw(cam.ProjectionMatrix.Mul4(cam.ViewMatrix), cam.Position)

		if wireframe {
			GlState.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
		}

		gui.Draw()

		ctx.SwapBuffers()
	}
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}
