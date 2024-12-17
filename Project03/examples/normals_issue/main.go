package main

import (
	"advanced-gl/Project03/libutil"
	"log"
	"runtime"
	"unsafe"

	_ "embed"

	. "advanced-gl/Project03/libgl"
	. "advanced-gl/Project03/libscn"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	im "github.com/inkyblackness/imgui-go/v4"
)

func main() {
	runtime.LockOSThread()
	err := glfw.Init()
	check(err)

	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	glfw.WindowHint(glfw.Maximized, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	ctx, err := glfw.CreateWindow(1600, 900, "Normals Test", nil, nil)
	check(err)
	ctx.MakeContextCurrent()

	err = gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		addr := glfw.GetProcAddress(name)
		if addr == nil {
			return unsafe.Pointer(libutil.InvalidAddress)
		}
		return addr
	})
	check(err)

	gl.Enable(gl.TEXTURE_CUBE_MAP_SEAMLESS)
	gl.Enable(gl.DEBUG_OUTPUT)
	gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)

	gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, userParam unsafe.Pointer) {
		if gltype == gl.DEBUG_TYPE_PUSH_GROUP || gltype == gl.DEBUG_TYPE_POP_GROUP {
			// glDebugMessageControl is ignored in nsight, so double check to prevent log spam
			return
		}
		log.Printf("GL: %v\n", message)
	}, nil)

	State = NewGlStateManager()
	GlEnv = GetGlEnv()
	Input = NewInputManager(ctx)

	var (
		pack  *DirPack
		batch *RenderBatch
	)

	lm := &SimpleLoadManager{}

	lm.OnLoad(func(ctx *glfw.Window) {
		pack = &DirPack{}
		pack.AddIndexFile("assets/index.json")
		mesh, err := pack.LoadMesh("plane")
		check(err)
		material, err := pack.LoadMaterial("square_floor")
		check(err)

		batch = NewRenderBatch()
		batch.Upload(mesh)
		batch.AddMaterial(material)

		for x := -2; x <= 2; x++ {
			for z := -2; z <= 2; z++ {
				batch.Add(mesh.Name, material.Name, InstanceAttributes{
					ModelMatrix: mgl32.Translate3D(float32(x*2), 0, float32(z*2)),
				})
			}
		}
	})

	viewportDims := [4]int32{}
	gl.GetIntegerv(gl.VIEWPORT, &viewportDims[0])
	viewportWidth := int(viewportDims[2])
	viewportHeight := int(viewportDims[3])

	var (
		pbrShader UnboundShaderPipeline
		gui       *ImGui
	)
	lm.OnLoad(func(ctx *glfw.Window) {
		pbrShader, err = pack.LoadShaderPipeline("pbr_normals_issue")
		check(err)

		imguiShader, err := pack.LoadShaderPipeline("imgui")
		check(err)
		gui = NewImGui(imguiShader)
	})

	cam := &Camera{
		Position:          mgl32.Vec3{0.0, 1.0, -1.0},
		Orientation:       mgl32.Vec3{0.0, 180.0, 0.0},
		VerticalFov:       70,
		ViewportDimension: mgl32.Vec2{float32(viewportWidth), float32(viewportHeight)},
		ClippingPlanes:    mgl32.Vec2{0.01, 100},
	}
	cam.UpdateProjectionMatrix()

	texAlbedoSampler := NewSampler()
	texAlbedoSampler.SetDebugLabel("albedo")
	texAlbedoSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texAlbedoSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)
	texAlbedoSampler.AnisotropicFilter(8.0)

	texNormalSampler := NewSampler()
	texNormalSampler.SetDebugLabel("normal")
	texNormalSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texNormalSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)
	texNormalSampler.AnisotropicFilter(8.0)

	texOrmSampler := NewSampler()
	texOrmSampler.SetDebugLabel("orm")
	texOrmSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	texOrmSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)
	texOrmSampler.LodBias(-3.0)

	lightPositions := []mgl32.Vec3{
		{-4.5, 1.0, -4.5},
		{4.5, 1.0, -4.5},
		{-4.5, 1.0, 4.5},
		{4.5, 1.0, 4.5},
	}

	lightColors := []mgl32.Vec3{
		mgl32.Vec3{1.0, 1.0, 1.0}.Mul(100),
		mgl32.Vec3{1.0, 1.0, 1.0}.Mul(100),
		mgl32.Vec3{1.0, 1.0, 1.0}.Mul(100),
		mgl32.Vec3{1.0, 1.0, 1.0}.Mul(100),
	}

	lm.Reload(ctx)

	reload := false
	showIssueArea := false
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
			cam.Orientation[0] -= rotation[1] * 0.35
			cam.Orientation[1] -= rotation[0] * 0.35
		}
		cam.UpdateViewMatrix()

		State.SetEnabled(DepthTest, CullFace)
		State.DepthFunc(DepthFuncLess)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		materials := batch.GenerateDrawCommands()
		batch.CommandBuffer.Bind(gl.DRAW_INDIRECT_BUFFER)
		batch.VertexArray.Bind()
		pbrShader.Bind()
		pbrShader.VertexStage().SetUniform("u_view_projection_mat", cam.ProjectionMatrix.Mul4(cam.ViewMatrix))
		pbrShader.FragmentStage().SetUniform("u_camera_position", cam.Position)

		for i := 0; i < 4; i++ {
			pbrShader.FragmentStage().SetUniformIndexed("u_light_positions", i, lightPositions[i])
			pbrShader.FragmentStage().SetUniformIndexed("u_light_colors", i, lightColors[i])
		}

		texAlbedoSampler.Bind(0)
		texNormalSampler.Bind(1)
		texOrmSampler.Bind(2)

		for _, mat := range materials {
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

		if im.Checkbox("Show Issue Area", &showIssueArea) {
			if showIssueArea {
				pbrShader.FragmentStage().SetUniform("u_show_issue_area", float32(1.0))
			} else {
				pbrShader.FragmentStage().SetUniform("u_show_issue_area", float32(0.0))
			}
		}

		im.End()
		gui.Draw()

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
