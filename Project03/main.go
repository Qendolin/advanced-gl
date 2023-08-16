package main

import (
	_ "embed"
	"flag"
	"io"
	"log"
	"os"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	im "github.com/inkyblackness/imgui-go/v4"
)

var Arguments struct {
	EnableCompatibilityProfile bool
}

func main() {
	log.Println("Opening log file")

	logFile, err := os.OpenFile("latest.log", os.O_CREATE|os.O_TRUNC, 0666)
	check(err)
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	log.Println("Parsing arguments")

	flag.BoolVar(&Arguments.EnableCompatibilityProfile, "enable-compatibility-profile", Arguments.EnableCompatibilityProfile, "")
	flag.Parse()

	runtime.LockOSThread()
	log.Println("Initializing GLFW")

	err = glfw.Init()
	check(err)

	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	glfw.WindowHint(glfw.Maximized, glfw.True)
	if Arguments.EnableCompatibilityProfile {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
	} else {
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	}
	log.Println("Creating Window")
	ctx, err := glfw.CreateWindow(1600, 900, "Cubemap Test", nil, nil)
	check(err)
	ctx.MakeContextCurrent()

	log.Println("Initializing OpenGL")

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

	GlState.Enable(gl.DEBUG_OUTPUT)
	GlState.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)

	gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
		if gltype == gl.DEBUG_TYPE_PUSH_GROUP || gltype == gl.DEBUG_TYPE_POP_GROUP {
			// glDebugMessageControl is ignored in nsight, so double check to prevent log spam
			return
		}
		log.Printf("GL: %v\n", message)
	}, nil)

	gl.DebugMessageControl(gl.DONT_CARE, gl.DEBUG_TYPE_PUSH_GROUP, gl.DONT_CARE, 0, nil, false)
	gl.DebugMessageControl(gl.DONT_CARE, gl.DEBUG_TYPE_POP_GROUP, gl.DONT_CARE, 0, nil, false)

	var (
		pack  *DirPack
		batch *RenderBatch
	)

	lm := &SimpleLoadManager{}

	lm.OnLoad(func(ctx *glfw.Window) {
		pack = &DirPack{}
		pack.AddIndexFile("assets/index.json")
		pavementModel, err := pack.LoadModel("square_floor")
		check(err)

		batch = NewRenderBatch()
		batch.Upload(pavementModel.Mesh)
		batch.AddMaterial(pavementModel.Material)

		for x := -2; x <= 2; x++ {
			for z := -2; z <= 2; z++ {
				batch.Add(pavementModel.Mesh.Name, pavementModel.Material.Name, InstanceAttributes{
					ModelMatrix: mgl32.Translate3D(float32(x*2), 0, float32(z*2)),
				})
			}
		}
	})

	viewportDims := [4]int32{}
	gl.GetIntegerv(gl.VIEWPORT, &viewportDims[0])
	viewportWidth := int(viewportDims[2])
	viewportHeight := int(viewportDims[3])

	msFbo := NewFramebuffer()
	msColorAttachment := NewTexture(gl.TEXTURE_2D_MULTISAMPLE)
	msColorAttachment.AllocateMS(gl.RGBA8, viewportWidth, viewportHeight, 0, 2, true)
	msFbo.AttachTexture(0, msColorAttachment)
	msFbo.BindTargets(0)
	msDepthAttachment := NewRenderbuffer()
	msDepthAttachment.AllocateMS(gl.DEPTH24_STENCIL8, viewportWidth, viewportHeight, 2)
	msFbo.AttachRenderbuffer(gl.DEPTH_ATTACHMENT, msDepthAttachment)
	check(msFbo.Check(gl.DRAW_FRAMEBUFFER))

	var (
		pbrShader UnboundShaderPipeline
		dd        *DirectBuffer
		gui       *ImGui
	)

	lm.OnLoad(func(ctx *glfw.Window) {
		pbrShader, err = pack.LoadShaderPipeline("pbr")
		check(err)

		directShader, err := pack.LoadShaderPipeline("direct")
		check(err)
		dd = NewDirectDrawBuffer(directShader)

		imguiShader, err := pack.LoadShaderPipeline("imgui")
		check(err)
		gui = NewImGui(imguiShader)
	})

	cam := &Camera{
		Position:          mgl32.Vec3{0.0, 1.0, 0.0},
		Orientation:       mgl32.Vec3{90.0, 0.0, 0.0},
		VerticalFov:       70,
		ViewportDimension: mgl32.Vec2{float32(viewportWidth), float32(viewportHeight)},
		ClippingPlanes:    mgl32.Vec2{0.1, 1000},
	}
	cam.UpdateProjectionMatrix()

	texAlbedoSampler := NewSampler()
	texAlbedoSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texAlbedoSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)
	texAlbedoSampler.AnisotropicFilter(8.0)

	texNormalSampler := NewSampler()
	texNormalSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texNormalSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)
	texNormalSampler.AnisotropicFilter(8.0)

	texOrmSampler := NewSampler()
	texOrmSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texOrmSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)

	lightPositions := []mgl32.Vec3{
		{-4.5, 1.0, -4.5},
		{4.5, 1.0, -4.5},
		{-4.5, 1.0, 4.5},
		{4.5, 1.0, 4.5},
	}

	lightColors := []mgl32.Vec3{
		mgl32.Vec3{3.75, 7.0, 1.75}.Mul(10),
		mgl32.Vec3{7.0, 2.9, 3.0}.Mul(10),
		mgl32.Vec3{7.0, 4.35, 2.05}.Mul(10),
		mgl32.Vec3{4.25, 1.75, 7.0}.Mul(10),
	}

	lm.Reload(ctx)

	wireframe := false
	lodBias := make([]float32, 3)
	reload := false

	for !ctx.ShouldClose() {
		glfw.PollEvents()
		Input.Update(ctx)

		msFbo.Bind(gl.DRAW_FRAMEBUFFER)

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

		texAlbedoSampler.Bind(0)
		texNormalSampler.Bind(1)
		texOrmSampler.Bind(2)
		for _, mat := range batch.Materials {
			mat.Material.Albedo.Bind(0)
			mat.Material.Normal.Bind(1)
			mat.Material.ORM.Bind(2)

			gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(mat.ElementOffset), int32(mat.ElementCount), 0)
		}

		im.NewFrame()
		im.Begin("main_window")

		if im.Button("Reload Assets") {
			reload = true
		}

		if im.Checkbox("Wireframe", &wireframe) {
			GlState.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
		}

		im.Text("LOD Bias")
		if im.SliderFloat("Albedo", &lodBias[0], -10, 10) {
			texAlbedoSampler.LodBias(lodBias[0])
		}

		if im.SliderFloat("Normal", &lodBias[1], -10, 10) {
			texNormalSampler.LodBias(lodBias[1])
		}

		if im.SliderFloat("ORM", &lodBias[2], -10, 10) {
			texOrmSampler.LodBias(lodBias[2])
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

		GlState.SetEnabled()

		gl.BlitNamedFramebuffer(msFbo.Id(), 0, 0, 0, int32(viewportWidth), int32(viewportHeight), 0, 0, int32(viewportWidth), int32(viewportHeight), gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT, gl.NEAREST)
		ctx.SwapBuffers()

		if reload {
			reload = false
			lm.Reload(ctx)
		}
	}
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}
