package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	im "github.com/inkyblackness/imgui-go/v4"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"advanced-gl/Project03/effects"
	"advanced-gl/Project03/ibl"
	. "advanced-gl/Project03/libgl"
	. "advanced-gl/Project03/libscn"
	"advanced-gl/Project03/libutil"
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
			return unsafe.Pointer(libutil.InvalidAddress)
		}
		return addr
	})
	check(err)

	glRenderer := string(gl.GoStr(gl.GetString(gl.RENDERER)))
	log.Printf("Using GPU: %s\n", glRenderer)

	State = NewGlStateManager()
	GlEnv = GetGlEnv()
	Input = NewInputManager(ctx)

	wireframe := false
	lodBias := []float32{0.0, 0.0, -3.0}
	reload := false
	speed := float32(1.0)
	abmientFactor := float32(1.0)
	exposure := float32(1.0)
	bloomFactor := float32(0.3)
	var (
		selectedMaterial  string = "dirty_mirror" // array_test, dirty_mirror, square_floor
		selectedMesh      string = "plane"        // array_spheres_uv, plane
		selectedHdriName  string
		selectedHdriLevel int32
		selectedHdri      *ibl.IblEnv
	)

	// FIXME: I can't use GlState here because it would be disabled by SetEnabled
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
		mesh, err := pack.LoadMesh(selectedMesh)
		check(err)
		material, err := pack.LoadMaterial(selectedMaterial)
		check(err)

		batch = NewRenderBatch()
		batch.Upload(mesh)
		batch.AddMaterial(material)

		// for x := -2; x <= 2; x++ {
		// 	for z := -2; z <= 2; z++ {
		// 		batch.Add(mesh.Name, material.Name, InstanceAttributes{
		// 			ModelMatrix: mgl32.Translate3D(float32(x*2), 0, float32(z*2)),
		// 		})
		// 	}
		// }

		batch.Add(mesh.Name, material.Name, InstanceAttributes{
			ModelMatrix: mgl32.Scale3D(1.0, 1.0, 1.0),
		})
	})

	viewportDims := [4]int32{}
	gl.GetIntegerv(gl.VIEWPORT, &viewportDims[0])
	viewportWidth := int(viewportDims[2])
	viewportHeight := int(viewportDims[3])

	hdrFbo := NewFramebuffer()
	hdrFbo.SetDebugLabel("hdr_fbo")
	hdrColorAttachment := NewTexture(gl.TEXTURE_2D)
	hdrColorAttachment.SetDebugLabel("hdr_fbo")
	hdrColorAttachment.Allocate(1, gl.R11F_G11F_B10F, viewportWidth, viewportHeight, 0)
	hdrFbo.AttachTexture(0, hdrColorAttachment)
	hdrFbo.BindTargets(0)

	hdrDepthAttachment := NewTexture(gl.TEXTURE_2D)
	hdrDepthAttachment.SetDebugLabel("hdr_fbo")
	hdrDepthAttachment.Allocate(1, gl.DEPTH24_STENCIL8, viewportWidth, viewportHeight, 0)
	hdrFbo.AttachTexture(gl.DEPTH_ATTACHMENT, hdrDepthAttachment)
	check(hdrFbo.Check(gl.DRAW_FRAMEBUFFER))

	var (
		pbrShader          UnboundShaderPipeline
		skyShader          UnboundShaderPipeline
		postShader         UnboundShaderPipeline
		dd                 *libutil.DirectBuffer
		gui                *ImGui
		envCubemap         UnboundTexture
		iblDiffuseCubemap  UnboundTexture
		iblSpecularCubemap UnboundTexture
		iblBdrfLut         UnboundTexture
		bloom              *effects.BloomEffect
	)

	lm.OnLoad(func(ctx *glfw.Window) {
		pbrShader, err = pack.LoadShaderPipeline("pbr")
		check(err)

		directShader, err := pack.LoadShaderPipeline("direct")
		check(err)
		dd = libutil.NewDirectDrawBuffer(directShader)

		imguiShader, err := pack.LoadShaderPipeline("imgui")
		check(err)
		gui = NewImGui(imguiShader)

		skyShader, err = pack.LoadShaderPipeline("sky")
		check(err)

		bloomUpShader, err := pack.LoadShaderPipeline("bloom_up")
		check(err)
		bloomDownShader, err := pack.LoadShaderPipeline("bloom_down")
		check(err)
		bloom = effects.NewBloomEffect(8, bloomUpShader, bloomDownShader)
		for i := range bloom.Factors {
			if i == 0 {
				bloom.Factors[i] = 1
			} else {
				bloom.Factors[i] = bloom.Factors[i-1] * 0.75
			}
		}

		postShader, err = pack.LoadShaderPipeline("post")
		check(err)
	})

	skyBoxVbo := NewBuffer()
	skyBoxVbo.SetDebugLabel("sky_box")
	skyBoxVbo.Allocate(ibl.NewUnitCube(), 0)
	skyBox := NewVertexArray()
	skyBox.SetDebugLabel("sky_box")
	skyBox.Layout(0, 0, 3, gl.FLOAT, false, 0)
	skyBox.BindBuffer(0, skyBoxVbo, 0, 3*4)

	lm.OnLoad(func(ctx *glfw.Window) {
		hdri, err := pack.LoadHdri("studio_small_02_4k")
		check(err)
		hdriIrradiance, err := pack.LoadHdri("studio_small_02_4k_diffuse")
		check(err)
		hdriReflection, err := pack.LoadHdri("studio_small_02_4k_specular")
		check(err)

		envCubemap = NewTexture(gl.TEXTURE_CUBE_MAP)
		envCubemap.SetDebugLabel("environment")
		envCubemap.Allocate(1, gl.RGB16F, hdri.BaseSize, hdri.BaseSize, 0)
		envCubemap.Load(0, hdri.BaseSize, hdri.BaseSize, 6, gl.RGB, hdri.All())

		iblDiffuseCubemap = NewTexture(gl.TEXTURE_CUBE_MAP)
		iblDiffuseCubemap.SetDebugLabel("ibl_diffuse")
		iblDiffuseCubemap.Allocate(1, gl.RGB16F, hdriIrradiance.BaseSize, hdriIrradiance.BaseSize, 0)
		iblDiffuseCubemap.Load(0, hdriIrradiance.BaseSize, hdriIrradiance.BaseSize, 6, gl.RGB, hdriIrradiance.All())

		iblSpecularCubemap = NewTexture(gl.TEXTURE_CUBE_MAP)
		iblSpecularCubemap.SetDebugLabel("ibl_specular")
		iblSpecularCubemap.Allocate(hdriReflection.Levels, gl.RGB16F, hdriReflection.BaseSize, hdriReflection.BaseSize, 0)
		for i := 0; i < hdriReflection.Levels; i++ {
			iblSpecularCubemap.Load(i, hdriReflection.Size(i), hdriReflection.Size(i), 6, gl.RGB, hdriReflection.Level(i))
		}

		lut, err := pack.LoadTextureFloat("ibl_brdf_lut")
		check(err)
		iblBdrfLut = NewTexture(gl.TEXTURE_2D)
		iblBdrfLut.SetDebugLabel("ibl_lut")
		iblBdrfLut.Allocate(1, gl.RG32F, lut.Width, lut.Height, 0)
		iblBdrfLut.Load(0, lut.Width, lut.Height, 0, gl.RG, lut.Pix)
	})

	cam := &Camera{
		Position:          mgl32.Vec3{0.0, 1.0, -1.0},
		Orientation:       mgl32.Vec3{0.0, 180.0, 0.0},
		VerticalFov:       70,
		ViewportDimension: mgl32.Vec2{float32(viewportWidth), float32(viewportHeight)},
		ClippingPlanes:    mgl32.Vec2{0.1, 1000},
	}
	cam.UpdateProjectionMatrix()

	cubemapSampler := NewSampler()
	cubemapSampler.SetDebugLabel("cubemap")
	cubemapSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	cubemapSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)

	framebufferSampler := NewSampler()
	framebufferSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	framebufferSampler.FilterMode(gl.NEAREST, gl.NEAREST)

	lutSampler := NewSampler()
	lutSampler.SetDebugLabel("lut")
	lutSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	lutSampler.FilterMode(gl.LINEAR, gl.LINEAR)

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
		mgl32.Vec3{3.75, 7.0, 1.75}.Mul(10),
		mgl32.Vec3{7.0, 2.9, 3.0}.Mul(10),
		mgl32.Vec3{7.0, 4.35, 2.05}.Mul(10),
		mgl32.Vec3{4.25, 1.75, 7.0}.Mul(10),
	}

	lm.Reload(ctx)

	for !ctx.ShouldClose() {
		glfw.PollEvents()
		Input.Update(ctx)

		hdrFbo.Bind(gl.DRAW_FRAMEBUFFER)

		movement := Input.GetMovement(glfw.KeyW, glfw.KeyS, glfw.KeyA, glfw.KeyD, glfw.KeySpace, glfw.KeyLeftControl)
		if movement.LenSqr() != 0 {
			movement = movement.Normalize().Mul(Input.TimeDelta() * speed)
			cam.Fly(movement)
		}
		if Input.IsMouseDown(glfw.MouseButtonRight) {
			rotation := Input.CursorDelta()
			cam.Orientation[0] += rotation[1] * 0.35
			cam.Orientation[1] += rotation[0] * 0.35
		}
		cam.UpdateViewMatrix()

		State.SetEnabled(DepthTest, CullFace)
		State.DepthFunc(DepthFuncLess)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		if wireframe {
			State.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
		}

		materials := batch.GenerateDrawCommands()
		batch.CommandBuffer.Bind(gl.DRAW_INDIRECT_BUFFER)
		batch.VertexArray.Bind()
		pbrShader.Bind()
		pbrShader.VertexStage().SetUniform("u_view_projection_mat", cam.ProjectionMatrix.Mul4(cam.ViewMatrix))
		pbrShader.FragmentStage().SetUniform("u_camera_position", cam.Position)
		pbrShader.FragmentStage().SetUniform("u_ambient_factor", abmientFactor)

		for i := 0; i < 4; i++ {
			pbrShader.FragmentStage().SetUniformIndexed("u_light_positions", i, lightPositions[i])
			pbrShader.FragmentStage().SetUniformIndexed("u_light_colors", i, lightColors[i])
		}

		texAlbedoSampler.Bind(0)
		texNormalSampler.Bind(1)
		texOrmSampler.Bind(2)
		cubemapSampler.Bind(3)
		cubemapSampler.Bind(4)
		lutSampler.Bind(5)

		iblDiffuseCubemap.Bind(3)
		iblSpecularCubemap.Bind(4)
		iblBdrfLut.Bind(5)
		for _, mat := range materials {
			mat.Material.Albedo.Bind(0)
			mat.Material.Normal.Bind(1)
			mat.Material.ORM.Bind(2)

			gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(mat.ElementOffset), int32(mat.ElementCount), 0)
		}

		skyShader.Bind()
		skyShader.VertexStage().SetUniform("u_view_mat", cam.ViewMatrix)
		skyShader.VertexStage().SetUniform("u_projection_mat", cam.ProjectionMatrix)
		envCubemap.Bind(0)
		cubemapSampler.Bind(0)
		skyBox.Bind()
		State.DepthFunc(gl.LEQUAL)
		gl.DrawArrays(gl.TRIANGLES, 0, 6*6)

		bloom.Resize(viewportWidth, viewportHeight)
		bloomResult := bloom.Render(hdrFbo.GetTexture(0))

		State.SetEnabled()

		State.BindFramebuffer(State.DrawFramebuffer, 0)
		postShader.Bind()
		postShader.FragmentStage().SetUniform("u_bloom_factor", bloomFactor)
		postShader.FragmentStage().SetUniform("u_exposure", exposure)
		framebufferSampler.Bind(0)
		hdrFbo.GetTexture(0).Bind(0)
		framebufferSampler.Bind(1)
		bloomResult.Bind(1)
		libutil.DrawQuad()

		im.NewFrame()
		im.Begin("main_window")

		if im.Button("Reload Assets") {
			reload = true
		}

		if im.Checkbox("Wireframe", &wireframe) {
			State.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
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
			im.DragFloat("Spd", &speed)

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

		if im.BeginCombo("Material", selectedMaterial) {
			materials := maps.Keys(pack.MaterialIndex)
			slices.Sort(materials)
			for _, name := range materials {
				if im.Selectable(name) {
					mat, err := pack.LoadMaterial(name)
					check(err)
					batch.AddMaterial(mat)
					selectedMaterial = name
				}
			}

			im.EndCombo()
		}

		if im.BeginCombo("Mesh", selectedMesh) {
			meshes := maps.Keys(pack.MeshIndex)
			slices.Sort(meshes)
			for _, name := range meshes {
				if im.Selectable(name) {
					mesh, err := pack.LoadMesh(name)
					check(err)
					batch.Upload(mesh)
					selectedMesh = name
				}
			}

			im.EndCombo()
		}

		if im.BeginCombo("Environment", selectedHdriName) {
			hdris := maps.Keys(pack.HdriIndex)
			slices.Sort(hdris)
			for _, name := range hdris {
				if im.Selectable(name) {
					hdri, err := pack.LoadHdri(name)
					check(err)
					selectedHdri = hdri
					envCubemap = NewTexture(gl.TEXTURE_CUBE_MAP)
					envCubemap.Allocate(1, gl.RGB16F, hdri.BaseSize, hdri.BaseSize, 0)
					envCubemap.Load(0, hdri.BaseSize, hdri.BaseSize, 6, gl.RGB, hdri.All())
					selectedHdriName = name
				}
			}
			im.EndCombo()
		}

		if im.SliderInt("HDRI Level", &selectedHdriLevel, 0, 4) {
			if selectedHdriLevel < int32(selectedHdri.Levels) {
				envCubemap = NewTexture(gl.TEXTURE_CUBE_MAP)
				sz := selectedHdri.Size(int(selectedHdriLevel))
				envCubemap.Allocate(1, gl.RGB16F, sz, sz, 0)
				envCubemap.Load(0, sz, sz, 6, gl.RGB, selectedHdri.Level(int(selectedHdriLevel)))
			}
		}

		im.SliderFloat("Ambient Factor", &abmientFactor, 0, 1)
		im.SliderFloat("Exposure", &exposure, 0, 1)

		if im.CollapsingHeader("Bloom") {
			im.SliderFloat("Bloom Factor", &bloomFactor, 0, 1)
			im.SliderFloat("Bloom Threshold", &bloom.Threshold, 0, 5)
			im.SliderFloat("Bloom Knee", &bloom.Knee, 0, 1)

			for i := range bloom.Factors {
				im.SliderFloat(fmt.Sprintf("Level %d", i), &bloom.Factors[i], 0, 1)
			}
		}

		im.End()

		dd.Draw(cam.ProjectionMatrix.Mul4(cam.ViewMatrix), cam.Position)

		if wireframe {
			State.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
		}

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
