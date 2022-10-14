package main

import (
	"bytes"
	"image"
	"log"
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Context = *glfw.Window

type scene struct {
	scene            *Scene
	batch            *RenderBatch
	quad             UnboundVertexArray
	lightSpheres     UnboundVertexArray
	lightQuads       UnboundVertexArray
	lightCones       UnboundVertexArray
	pointLightBuffer UnboundBuffer
	orthoLightBuffer UnboundBuffer
	spotLightBuffer  UnboundBuffer
	pointLights      []PointLightBlock
	orthoLights      []OrthoLightBlock
	spotLights       []SpotLightBlock
	shadowCasters    []*ShadowCaster

	geometryShader              UnboundShaderPipeline
	pointLightShader            UnboundShaderPipeline
	pointLightDebugVolumeShader UnboundShaderPipeline
	orthoLightShader            UnboundShaderPipeline
	spotLightShader             UnboundShaderPipeline
	ambientShader               UnboundShaderPipeline
	postShader                  UnboundShaderPipeline
	directShader                UnboundShaderPipeline
	ssaoShader                  UnboundShaderPipeline
	ssaoBlurShader              UnboundShaderPipeline
	debugShader                 UnboundShaderPipeline
	bloomDownShader             UnboundShaderPipeline
	bloomUpShader               UnboundShaderPipeline
	shadowShader                UnboundShaderPipeline
	skyShader                   UnboundShaderPipeline

	projMat   mgl32.Mat4
	viewMat   mgl32.Mat4
	camPos    mgl32.Vec3
	camOrient mgl32.Vec3

	gBuffer     UnboundFramebuffer
	postBuffer  UnboundFramebuffer
	ssaoBuffer  UnboundFramebuffer
	bloomBuffer UnboundFramebuffer

	ditherPattern  UnboundTexture
	ssaoNoise      UnboundTexture
	colorLut       UnboundTexture
	bloomDown      UnboundTexture
	bloomUp        UnboundTexture
	bloomDownViews []UnboundTexture
	bloomUpViews   []UnboundTexture
	stencilView    UnboundTexture
	bufferSampler  UnboundSampler
	textureSampler UnboundSampler
	patternSampler UnboundSampler
	lutSampler     UnboundSampler
	bloomSampler   UnboundSampler
	shadowSampler  UnboundSampler

	direct *DirectBuffer
}

var s scene

type PointLightBlock struct {
	Position    mgl32.Vec3
	Color       mgl32.Vec3
	Attenuation float32
	Radius      float32
}

type OrthoLightBlock struct {
	Direction   mgl32.Vec3
	Color       mgl32.Vec3
	ShadowIndex int32
}

type SpotLightBlock struct {
	Position    mgl32.Vec3
	Color       mgl32.Vec3
	Attenuation float32
	Radius      float32
	Direction   mgl32.Vec3
	Angles      mgl32.Vec2
	ShadowIndex int32
}

func Setup(ctx Context) {
	// #region models

	main := make(chan func())
	var sc *Scene
	var batch *RenderBatch
	go func() {
		log.Println("Lodaing ...")
		log.Println(" - Geometry")
		meshes, err := LoadMeshes(bytes.NewReader(Res_SceneGeometry))
		if err != nil {
			log.Fatal(err)
		}

		batchBuilder := RenderBatchBuilder{
			textures: map[string]*image.RGBA{},
		}
		for _, m := range meshes {
			batchBuilder.AddMesh(m)
		}

		log.Println(" - Scene")
		// TODO: LoadScene async
		main <- func() {
			sc, err = LoadScene(bytes.NewReader(Res_Scene))
			if err != nil {
				log.Fatal(err)
			}
		}
		<-main

		log.Println(" - Materials")
		wg := sync.WaitGroup{}
		poolSize := 16
		wg.Add(poolSize)
		for i := 0; i < poolSize; i++ {
			start := i * len(sc.Materials) / poolSize
			end := (i + 1) * len(sc.Materials) / poolSize
			go func(mats []SceneMaterial) {
				for _, m := range mats {
					err := batchBuilder.AddMaterial(m, "./assets/textures/sponza/")
					if err != nil {
						log.Fatal(err)
					}
				}
				wg.Done()
			}(sc.Materials[start:end])
		}
		wg.Wait()

		log.Println(" - Upload")
		main <- func() {
			batch = batchBuilder.Upload()
		}
		<-main

		log.Println(" - Instancing")
		main <- func() {
			for _, so := range sc.Objects {
				batch.Add(so.Mesh, so.Material, InstanceAttributes{
					ModelMatrix: so.Transform(),
				})
			}
			ui.orthoLights = sc.OrthoLights
			ui.spotLights = sc.SpotLights
			ui.pointLights = sc.PointLights
		}
		<-main
		close(main)
	}()

loading:
	for {
		select {
		case <-time.After(16 * time.Millisecond):
			ctx.SwapBuffers()
		case fn, ok := <-main:
			if !ok {
				break loading
			}
			fn()
			main <- nil
		}
	}

	//#endregion

	// #region lights
	quad := NewVertexArray()
	quad.Layout(0, 0, 2, gl.FLOAT, false, 0)
	vbo := NewBuffer()
	vbo.Allocate([]mgl32.Vec2{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}}, 0)
	quad.BindBuffer(0, vbo, 0, 2*4)

	lightSpheres := NewVertexArray()
	lightSpheres.Layout(0, 0, 3, gl.FLOAT, false, 0)
	lightSpheres.Layout(1, 1, 3, gl.FLOAT, false, int(unsafe.Offsetof(PointLightBlock{}.Position)))
	lightSpheres.Layout(1, 2, 3, gl.FLOAT, false, int(unsafe.Offsetof(PointLightBlock{}.Color)))
	lightSpheres.Layout(1, 3, 1, gl.FLOAT, false, int(unsafe.Offsetof(PointLightBlock{}.Attenuation)))
	lightSpheres.Layout(1, 4, 1, gl.FLOAT, false, int(unsafe.Offsetof(PointLightBlock{}.Radius)))
	vbo = NewBuffer()
	lightSphereVerts := generateLightSphere()
	vbo.Allocate(lightSphereVerts, 0)
	lightSpheres.BindBuffer(0, vbo, 0, 3*4)

	pointLightBuffer := NewBuffer()
	pointLights := make([]PointLightBlock, len(ui.pointLights), 1024)
	pointLightBuffer.Allocate(pointLights, gl.DYNAMIC_STORAGE_BIT)
	lightSpheres.BindBuffer(1, pointLightBuffer, 0, int(unsafe.Sizeof(PointLightBlock{})))
	lightSpheres.AttribDivisor(1, 1)

	lightQuads := NewVertexArray()
	lightQuads.Layout(0, 0, 2, gl.FLOAT, false, 0)
	lightQuads.Layout(1, 1, 3, gl.FLOAT, false, int(unsafe.Offsetof(OrthoLightBlock{}.Direction)))
	lightQuads.Layout(1, 2, 3, gl.FLOAT, false, int(unsafe.Offsetof(OrthoLightBlock{}.Color)))
	lightQuads.LayoutI(1, 3, 1, gl.INT, int(unsafe.Offsetof(OrthoLightBlock{}.ShadowIndex)))
	vbo = NewBuffer()
	vbo.Allocate([]mgl32.Vec2{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}}, 0)
	lightQuads.BindBuffer(0, vbo, 0, 2*4)

	orthoLightBuffer := NewBuffer()
	orthoLights := make([]OrthoLightBlock, len(ui.orthoLights))
	orthoLightBuffer.Allocate(orthoLights, gl.DYNAMIC_STORAGE_BIT)
	lightQuads.BindBuffer(1, orthoLightBuffer, 0, int(unsafe.Sizeof(OrthoLightBlock{})))
	lightQuads.AttribDivisor(1, 1)

	lightCones := NewVertexArray()
	lightCones.Layout(0, 0, 3, gl.FLOAT, false, 0)
	lightCones.Layout(1, 1, 3, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Position)))
	lightCones.Layout(1, 2, 3, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Color)))
	lightCones.Layout(1, 3, 1, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Attenuation)))
	lightCones.Layout(1, 4, 1, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Radius)))
	lightCones.Layout(1, 5, 3, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Direction)))
	lightCones.Layout(1, 6, 2, gl.FLOAT, false, int(unsafe.Offsetof(SpotLightBlock{}.Angles)))
	lightCones.LayoutI(1, 7, 1, gl.INT, int(unsafe.Offsetof(SpotLightBlock{}.ShadowIndex)))
	vbo = NewBuffer()
	// TODO: Generate better fitting geometry
	lightConeVerts := generateLightSphere()
	vbo.Allocate(lightConeVerts, 0)
	lightCones.BindBuffer(0, vbo, 0, 3*4)

	spotLightBuffer := NewBuffer()
	spotLights := make([]SpotLightBlock, len(ui.spotLights), 1024)
	spotLightBuffer.Allocate(spotLights, gl.DYNAMIC_STORAGE_BIT)
	lightCones.BindBuffer(1, spotLightBuffer, 0, int(unsafe.Sizeof(SpotLightBlock{})))
	lightCones.AttribDivisor(1, 1)

	shadowCasters := sc.Shadows

	// #endregion

	log.Println(" - Shaders")
	// #region shaders

	vertSh := NewShader(Res_GeometryVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh := NewShader(Res_GeometryFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	geomPipeline := NewPipeline()
	geomPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	geomPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_ShadowVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	shadowPipeline := NewPipeline()
	shadowPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)

	vertSh = NewShader(Res_PointLightVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_PointLightFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	pointLightPipeline := NewPipeline()
	pointLightPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	pointLightPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	fragSh = NewShader(Res_LightDebugVolumeFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	lightDebugVolumePipeline := NewPipeline()
	lightDebugVolumePipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	lightDebugVolumePipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_OrthoLightVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_OrthoLightFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	orthoLightPipeline := NewPipeline()
	orthoLightPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	orthoLightPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_SpotLightVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_SpotLightFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	spotLightPipeline := NewPipeline()
	spotLightPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	spotLightPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_QuadVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_PostProcessFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	postPipeline := NewPipeline()
	postPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	postPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	directPipeline := NewPipeline()
	vertSh = NewShader(Res_TransformVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_NormalFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	directPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	directPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_QuadVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_SsaoFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	ssaoPipeline := NewPipeline()
	ssaoPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	ssaoPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)
	ssaoNoise := GenerateSsaoPattern(4)

	fragSh = NewShader(Res_SsaoBlurFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	ssaoBlurPipeline := NewPipeline()
	ssaoBlurPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	ssaoBlurPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	fragSh = NewShader(Res_DebugFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	debugPipeline := NewPipeline()
	debugPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	debugPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	bloomDownPipeline := NewPipeline()
	fragSh = NewShader(Res_BloomDownFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	bloomDownPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	bloomDownPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	bloomUpPipeline := NewPipeline()
	fragSh = NewShader(Res_BloomUpFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	bloomUpPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	bloomUpPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	fragSh = NewShader(Res_AmbientFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	ambientPipeline := NewPipeline()
	ambientPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	ambientPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	vertSh = NewShader(Res_SkyVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh = NewShader(Res_SkyFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	skyPipeline := NewPipeline()
	skyPipeline.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	skyPipeline.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)

	// #endregion

	log.Println(" - Framebuffers")
	// #region textures / fbos

	textureSampler := NewSampler()
	textureSampler.FilterMode(gl.LINEAR_MIPMAP_LINEAR, gl.LINEAR)
	textureSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)

	textureImg := LoadImage(bytes.NewReader(Res_UvTexture))
	texture := NewTexture(gl.TEXTURE_2D)
	texture.Allocate(0, gl.SRGB8, textureImg.Bounds().Dx(), textureImg.Bounds().Dy(), 0)
	texture.Load(0, textureImg.Bounds().Dx(), textureImg.Bounds().Dy(), 0, gl.RGBA, textureImg.Pix)
	texture.GenerateMipmap()

	patternSampler := NewSampler()
	patternSampler.FilterMode(gl.LINEAR, gl.LINEAR)
	patternSampler.WrapMode(gl.REPEAT, gl.REPEAT, 0)

	ditherPatternImg := LoadImage(bytes.NewReader(Res_BayerTexture))
	ditherPattern := NewTexture(gl.TEXTURE_2D)
	ditherPattern.Allocate(1, gl.R8, ditherPatternImg.Bounds().Dx(), ditherPatternImg.Bounds().Dy(), 0)
	ditherPattern.Load(0, ditherPatternImg.Bounds().Dx(), ditherPatternImg.Bounds().Dy(), 0, gl.RGBA, ditherPatternImg.Pix)

	gBufferSampler := NewSampler()
	gBufferSampler.FilterMode(gl.NEAREST, gl.NEAREST)
	gBufferSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, 0)

	gBuffer := NewFramebuffer()

	gPosition := NewTexture(gl.TEXTURE_2D)
	gPosition.Allocate(1, gl.RGB16F, ViewportWidth, ViewportHeight, 0)
	gBuffer.AttachTexture(0, gPosition)

	gNormal := NewTexture(gl.TEXTURE_2D)
	gNormal.Allocate(1, gl.RG16_SNORM, ViewportWidth, ViewportHeight, 0)
	gBuffer.AttachTexture(1, gNormal)

	gAlbedo := NewTexture(gl.TEXTURE_2D)
	gAlbedo.Allocate(1, gl.RGB8, ViewportWidth, ViewportHeight, 0)
	gBuffer.AttachTexture(2, gAlbedo)

	gDepth := NewTexture(gl.TEXTURE_2D)
	gDepth.Allocate(1, gl.DEPTH24_STENCIL8, ViewportWidth, ViewportHeight, 0)
	gBuffer.AttachTexture(gl.DEPTH_STENCIL_ATTACHMENT, gDepth)

	stencilView := gDepth.CreateView(gl.TEXTURE_2D, gl.DEPTH24_STENCIL8, 0, 0, 0, 0)
	stencilView.DepthStencilTextureMode(gl.STENCIL_INDEX)

	gBuffer.BindTargets(0, 1, 2)
	if err := gBuffer.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		log.Panic(err)
	}

	postBuffer := NewFramebuffer()
	pColor := NewTexture(gl.TEXTURE_2D)
	pColor.Allocate(1, gl.RGB16F, ViewportWidth, ViewportHeight, 0)
	postBuffer.AttachTexture(0, pColor)
	postBuffer.AttachTexture(gl.DEPTH_STENCIL_ATTACHMENT, gDepth)

	postBuffer.BindTargets(0)
	if err := postBuffer.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		log.Panic(err)
	}

	ssaoBuffer := NewFramebuffer()
	ssaoRaw := NewTexture(gl.TEXTURE_2D)
	ssaoRaw.Allocate(1, gl.R8, ViewportWidth, ViewportHeight, 0)
	ssaoBuffer.AttachTexture(0, ssaoRaw)
	ssaoFinal := NewTexture(gl.TEXTURE_2D)
	ssaoFinal.Allocate(1, gl.R8, ViewportWidth, ViewportHeight, 0)
	ssaoBuffer.AttachTexture(1, ssaoFinal)
	ssaoBuffer.AttachTexture(gl.DEPTH_STENCIL_ATTACHMENT, gDepth)

	ssaoBuffer.BindTargets(0, 1)
	if err := ssaoBuffer.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		log.Panic(err)
	}

	bloomUp := NewTexture(gl.TEXTURE_2D)
	bloomUp.Allocate(7, gl.R11F_G11F_B10F, ViewportWidth, ViewportHeight, 0)
	bloomUpViews := make([]UnboundTexture, 7)
	for i := 0; i < 7; i++ {
		bloomUpViews[i] = bloomUp.CreateView(gl.TEXTURE_2D, gl.R11F_G11F_B10F, i, i, 0, 0)
	}
	bloomDown := NewTexture(gl.TEXTURE_2D)
	bloomDown.Allocate(7, gl.R11F_G11F_B10F, ViewportWidth/2, ViewportHeight/2, 0)
	bloomDownViews := make([]UnboundTexture, 7)
	for i := 0; i < 7; i++ {
		bloomDownViews[i] = bloomDown.CreateView(gl.TEXTURE_2D, gl.R11F_G11F_B10F, i, i, 0, 0)
	}

	bloomSampler := NewSampler()
	bloomSampler.FilterMode(gl.LINEAR_MIPMAP_NEAREST, gl.LINEAR)
	bloomSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, 0)

	bloomBuffer := NewFramebuffer()
	bloomBuffer.BindTargets(0)

	colorLutImg := LoadImage(bytes.NewReader(Res_TestLutTexture))
	colorLut := NewTexture(gl.TEXTURE_3D)
	colorLut.Allocate(1, gl.RGB8, 32, 32, 32)
	colorLut.Load(0, 32, 32, 32, gl.RGBA, colorLutImg.Pix)

	lutSampler := NewSampler()
	lutSampler.FilterMode(gl.LINEAR, gl.LINEAR)
	lutSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)

	shadowSampler := NewSampler()
	shadowSampler.FilterMode(gl.LINEAR, gl.LINEAR)
	shadowSampler.WrapMode(gl.CLAMP_TO_BORDER, gl.CLAMP_TO_BORDER, 0)
	shadowSampler.BorderColor(mgl32.Vec4{1, 1, 1, 1})
	shadowSampler.CompareMode(gl.COMPARE_REF_TO_TEXTURE, 0)

	// #endregion

	projMat := mgl32.Perspective(70, float32(ViewportWidth)/float32(ViewportHeight), 0.1, 1000)
	viewMat := mgl32.Ident4()

	log.Println("Done!")

	s = scene{
		scene:            sc,
		batch:            batch,
		quad:             quad,
		lightSpheres:     lightSpheres,
		lightCones:       lightCones,
		lightQuads:       lightQuads,
		pointLights:      pointLights,
		pointLightBuffer: pointLightBuffer,
		orthoLights:      orthoLights,
		orthoLightBuffer: orthoLightBuffer,
		spotLights:       spotLights,
		spotLightBuffer:  spotLightBuffer,
		shadowCasters:    shadowCasters,

		geometryShader:              geomPipeline,
		pointLightShader:            pointLightPipeline,
		pointLightDebugVolumeShader: lightDebugVolumePipeline,
		orthoLightShader:            orthoLightPipeline,
		spotLightShader:             spotLightPipeline,
		ambientShader:               ambientPipeline,
		postShader:                  postPipeline,
		directShader:                directPipeline,
		ssaoShader:                  ssaoPipeline,
		ssaoBlurShader:              ssaoBlurPipeline,
		debugShader:                 debugPipeline,
		bloomDownShader:             bloomDownPipeline,
		bloomUpShader:               bloomUpPipeline,
		shadowShader:                shadowPipeline,
		skyShader:                   skyPipeline,

		projMat:   projMat,
		viewMat:   viewMat,
		camPos:    mgl32.Vec3{-15, 4, 0},
		camOrient: mgl32.Vec3{33 * Deg2Rad, 90 * Deg2Rad, 0},

		gBuffer:     gBuffer,
		postBuffer:  postBuffer,
		ssaoBuffer:  ssaoBuffer,
		bloomBuffer: bloomBuffer,

		ditherPattern:  ditherPattern,
		ssaoNoise:      ssaoNoise,
		colorLut:       colorLut,
		bloomDown:      bloomDown,
		bloomUp:        bloomUp,
		bloomUpViews:   bloomUpViews,
		bloomDownViews: bloomDownViews,
		stencilView:    stencilView,
		textureSampler: textureSampler,
		patternSampler: patternSampler,
		bufferSampler:  gBufferSampler,
		lutSampler:     lutSampler,
		bloomSampler:   bloomSampler,
		shadowSampler:  shadowSampler,

		direct: CreateDirectBuffer(),
	}
}

func Draw(ctx Context) {
	speed := float32(5 * Input.TimeDelta())

	if !ui.cursorVisible {
		var camInput mgl32.Vec3
		if Input.IsKeyDown(glfw.KeyLeftShift) {
			speed *= 2
		}
		if Input.IsKeyDown(glfw.KeyW) {
			camInput = camInput.Add(mgl32.Vec3{0, 0, -1}.Mul(speed))
		}
		if Input.IsKeyDown(glfw.KeyA) {
			camInput = camInput.Add(mgl32.Vec3{-1, 0, 0}.Mul(speed))
		}
		if Input.IsKeyDown(glfw.KeyS) {
			camInput = camInput.Add(mgl32.Vec3{0, 0, 1}.Mul(speed))
		}
		if Input.IsKeyDown(glfw.KeyD) {
			camInput = camInput.Add(mgl32.Vec3{1, 0, 0}.Mul(speed))
		}

		s.camOrient = s.camOrient.Add(mgl32.Vec3{Input.CursorDelta()[1], Input.CursorDelta()[0]}.Mul(0.003))
		if s.camOrient[0] > math.Pi/2 {
			s.camOrient = mgl32.Vec3{math.Pi / 2, s.camOrient[1]}
		} else if s.camOrient[0] < -math.Pi/2 {
			s.camOrient = mgl32.Vec3{-math.Pi / 2, s.camOrient[1]}
		}

		camQuat := mgl32.AnglesToQuat(s.camOrient[0], s.camOrient[1], s.camOrient[2], mgl32.XYZ)
		camMove := camQuat.Inverse().Mat4().Mul4x1(mgl32.Vec4{camInput[0], camInput[1], camInput[2], 1})
		s.camPos = s.camPos.Add(mgl32.Vec3{camMove[0], camMove[1], camMove[2]})
		s.viewMat = camQuat.Mat4().Mul4(mgl32.Translate3D(-s.camPos[0], -s.camPos[1], -s.camPos[2]))
	}

	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Geometry\x00"))
	s.geometryShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", s.projMat.Mul4(s.viewMat))
	s.geometryShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", s.viewMat)

	s.batch.VertexArray.Bind()
	s.geometryShader.Bind()
	s.gBuffer.Bind(gl.DRAW_FRAMEBUFFER)
	GlState.ClearColor(0, 0, 0, 0)
	GlState.SetEnabled(DepthTest, StencilTest)
	if ui.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
		GlState.Enable(CullFace)
		GlState.CullBack()
	}
	GlState.DepthFunc(DepthFuncLess)
	GlState.DepthMask(true)
	GlState.StencilFuncBack(StencilFuncAlways, 0xff, 0xff)
	GlState.StencilFuncFront(StencilFuncAlways, 0x80, 0xff)
	GlState.StencilMask(0xff)
	GlState.StencilOp(StencilOpKeep, StencilOpKeep, StencilOpReplace)
	s.batch.GenerateDrawCommands()
	s.batch.CommandBuffer.Bind(gl.DRAW_INDIRECT_BUFFER)
	GlState.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
	for _, sm := range s.scene.Materials {
		for i, tex := range s.batch.MaterialTextures[sm.Name] {
			tex.Bind(i)
			s.textureSampler.Bind(i)
		}
		commads := s.batch.MaterialCommandRanges[sm.Name]
		gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(commads[0]), int32(commads[1]), 0)
	}
	if ui.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	gl.PopDebugGroup()

	// Shadow Maps
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Shadow Maps\x00"))
	for _, l := range s.orthoLights {
		if l.ShadowIndex == 0 {
			continue
		}
		sc := s.shadowCasters[l.ShadowIndex-1]
		if ui.enableShadows {
			sc.LookAt(mgl32.Vec3{0, 0, 0}, l.Direction, 20)
			// TODO: ortho shadow distance
			// sc.LookAt(s.camPos, s.orthoLights[0].Direction, 10)
			sc.Draw()
		} else {
			sc.Clear()
		}
	}
	for _, l := range s.spotLights {
		if l.ShadowIndex == 0 {
			continue
		}
		sc := s.shadowCasters[l.ShadowIndex-1]
		if ui.enableShadows {
			sc.LookFrom(l.Position, l.Direction)
			sc.Draw()
		} else {
			sc.Clear()
		}
	}
	gl.PopDebugGroup()

	// SSAO
	DrawSSAO()

	// Light
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Light\x00"))
	s.postBuffer.Bind(gl.DRAW_FRAMEBUFFER)
	GlState.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	s.pointLightShader.Bind()
	s.pointLightShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", s.projMat.Mul4(s.viewMat))
	s.pointLightShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", s.viewMat)
	s.bufferSampler.Bind(0)
	s.gBuffer.GetTexture(0).Bind(0)
	s.bufferSampler.Bind(1)
	s.gBuffer.GetTexture(1).Bind(1)
	s.bufferSampler.Bind(2)
	s.gBuffer.GetTexture(2).Bind(2)
	s.bufferSampler.Bind(3)
	s.gBuffer.GetTexture(gl.DEPTH_ATTACHMENT).Bind(3)
	s.bufferSampler.Bind(4)
	s.ssaoBuffer.GetTexture(1).Bind(4)

	GlState.SetEnabled(DepthTest, DepthClamp, CullFace, Blend)
	GlState.DepthMask(false)
	GlState.DepthFunc(DepthFuncGEqual)
	GlState.CullFront()
	GlState.BlendFunc(BlendOne, BlendOne)

	// Don't need stenicl test, depth test does the same
	// Enable(STENCIL_TEST)
	// StencilFunc(NOTEQUAL, 0x0, 0xff)

	// StencilMask(0xff)
	// Clear(STENCIL_BUFFER_BIT)

	s.lightSpheres.Bind()

	// Goals:
	// 1. Cull volume faces
	// 2. Cull the correct faces
	// 3. Prevent overdraw

	// Wrong implementation:
	// 	All		9.5	ms
	// 	1   	11.5ms
	// 	4   	9.5	ms
	// 	8   	8.5	ms
	// 	16   	8.75ms
	// 	128   	10	ms
	// 	512   	10.5ms
	// 	1024   	9.5 ms

	//  No Test 8.2 ms
	//  Is faster then the wrong implementation
	//  -> Stencil Test doesn't make sense when most lights are visible at once

	// FIXME:
	// Rendering multiple at once does not work
	// since the stencil OR-ed together
	// Meaning if a single light failes the depth test the stencil is set for all lights
	// The solution would be to only set the stencil only if all lights fail the depth test

	// Possible Solution 1:
	// 1. Pass:
	//   Increment the stencil for each fail
	// 2. Pass
	//   Only draw where the stencil value != batch size
	// The batch size is thus limited to 255
	//
	// Does not work, probably because the stencil value does not
	// rach batch size because not all volumes overlap
	// Thus a detached light does not discard any fragments?

	// Possible Solution 2:
	// 1. Pass:
	//   Set the stencil for each pass
	// 2. Pass
	//   Only draw where the stencil value != 0
	//
	// Does not work when no front face is visible

	/*lightBatchSize := 1
	for i := 0; i < 1848; i += lightBatchSize {
		Clear(STENCIL_BUFFER_BIT)

		DepthFunc(LEQUAL)
		CullFace(BACK)
		StencilFunc(ALWAYS, 0x0, 0xff)
		StencilOp(KEEP, INCR, KEEP)

		start := i
		count := math.Min(float64(lightBatchSize), float64(1848-start))

		ColorMask(false, false, false, false)
		DrawArraysInstancedBaseInstance(TRIANGLE_STRIP, 0, 64, int32(count), uint32(start))

		DepthFunc(GEQUAL)
		CullFace(FRONT)
		StencilFunc(EQUAL, 0x0, 0xff)
		StencilOp(ZERO, ZERO, ZERO)

		ColorMask(true, true, true, true)
		DrawArraysInstancedBaseInstance(TRIANGLE_STRIP, 0, 64, int32(count), uint32(start))
	}*/
	// TODO: Use variables
	if ui.enablePointLights {
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 64, int32(len(s.pointLights)))
	}

	if ui.enableSpotLights {
		s.spotLightShader.Bind()
		s.lightCones.Bind()
		s.spotLightShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", s.projMat.Mul4(s.viewMat))
		s.spotLightShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", s.viewMat)

		fragSh := s.spotLightShader.Get(gl.FRAGMENT_SHADER)
		fragSh.SetUniform("u_shadow_bias", ui.shadowBiasSample)
		for i := 0; i < 8; i++ {
			var sc *ShadowCaster
			if i < len(s.shadowCasters) {
				sc = s.shadowCasters[i]
			} else {
				GlState.BindSampler(i+8, 0)
				GlState.BindTextureUnit(i+8, 0)
				continue
			}
			s.shadowSampler.Bind(i + 8)
			fragSh.SetUniformIndexed("u_shadow_maps", i, i+8)
			sc.ShadowMap.GetTexture(gl.DEPTH_ATTACHMENT).Bind(i + 8)
			fragSh.SetUniformIndexed("u_shadow_transforms", i, sc.Transform.Mul4(s.viewMat.Inv()))
		}
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 64, int32(len(s.spotLights)))
	}

	GlState.Disable(CullFace)

	if ui.enableOrthoLights {
		// FIXME: Increadibly slow on intel
		s.lightQuads.Bind()
		s.orthoLightShader.Bind()
		fragSh := s.orthoLightShader.Get(gl.FRAGMENT_SHADER)
		fragSh.SetUniform("u_shadow_bias", ui.shadowBiasSample)
		for i := 0; i < 8; i++ {
			var sc *ShadowCaster
			if i < len(s.shadowCasters) {
				sc = s.shadowCasters[i]
			} else {
				GlState.BindSampler(i+8, 0)
				GlState.BindTextureUnit(i+8, 0)
				continue
			}
			s.shadowSampler.Bind(i + 8)
			fragSh.SetUniformIndexed("u_shadow_maps", i, i+8)
			sc.ShadowMap.GetTexture(gl.DEPTH_ATTACHMENT).Bind(i + 8)
			fragSh.SetUniformIndexed("u_shadow_transforms", i, sc.Transform.Mul4(s.viewMat.Inv()))
		}
		s.orthoLightShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", s.viewMat)
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, int32(len(s.orthoLights)))
	}

	s.quad.Bind()
	s.ambientShader.Bind()
	s.ambientShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_ambient_light", ui.ambientLight)
	s.ambientShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_min_light", ui.ambientMinLight)
	s.gBuffer.GetTexture(2).Bind(0)
	s.ssaoBuffer.GetTexture(1).Bind(1)
	if ui.enableAmbientLight {
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}
	gl.PopDebugGroup()

	// Sky
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Sky\x00"))
	if ui.enableSky {
		GlState.SetEnabled(StencilTest)
		GlState.StencilMask(0)
		GlState.StencilFunc(StencilFuncEqual, 0, 0xff)
		s.quad.Bind()
		s.skyShader.Bind()
		s.skyShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", mgl32.AnglesToQuat(s.camOrient[0], s.camOrient[1], s.camOrient[2], mgl32.XYZ).Mat4())
		s.skyShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_view_mat", mgl32.AnglesToQuat(s.camOrient[0], s.camOrient[1], s.camOrient[2], mgl32.XYZ).Mat4())
		s.skyShader.Get(gl.VERTEX_SHADER).SetUniform("u_projection_mat", s.projMat)
		s.skyShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_projection_mat", s.projMat)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}
	gl.PopDebugGroup()

	// Bloom
	DrawBloom()

	// Post Processing
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw PostFx\x00"))
	GlState.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	s.postShader.Bind()
	s.postShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_bloom_fac", ui.bloomFactor)
	s.bufferSampler.Bind(0)
	s.postBuffer.GetTexture(0).Bind(0)
	s.patternSampler.Bind(1)
	s.ditherPattern.Bind(1)
	s.lutSampler.Bind(2)
	s.colorLut.Bind(2)
	s.bufferSampler.Bind(3)
	s.bloomBuffer.GetTexture(0).Bind(3)
	s.quad.Bind()
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.PopDebugGroup()

	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Debug\x00"))
	if ui.bufferViewIndex != 0 {
		s.debugShader.Bind()
		s.bufferSampler.Bind(0)
		s.postBuffer.GetTexture(0).Bind(0)
		s.bufferSampler.Bind(1)
		s.gBuffer.GetTexture(0).Bind(1)
		s.bufferSampler.Bind(2)
		s.gBuffer.GetTexture(1).Bind(2)
		s.bufferSampler.Bind(3)
		s.gBuffer.GetTexture(2).Bind(3)
		s.bufferSampler.Bind(4)
		s.gBuffer.GetTexture(gl.DEPTH_ATTACHMENT).Bind(4)
		s.bufferSampler.Bind(5)
		s.ssaoBuffer.GetTexture(1).Bind(5)
		s.bufferSampler.Bind(6)
		s.bloomBuffer.GetTexture(0).Bind(6)
		s.bufferSampler.Bind(7)
		s.stencilView.Bind(7)
		s.debugShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_sampler", ui.bufferViewIndex-1)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}

	if ui.vizualizePointLights {
		for _, l := range s.pointLights {
			s.direct.Light3(l.Color)
			s.direct.CircleLine(l.Position, mgl32.Vec3{1, 0, 0}, l.Radius)
			s.direct.CircleLine(l.Position, mgl32.Vec3{0, 1, 0}, l.Radius)
			s.direct.CircleLine(l.Position, mgl32.Vec3{0, 0, 1}, l.Radius)
			s.direct.Circle(l.Position, s.camPos.Sub(l.Position), 0.5)
		}
	}

	if ui.vizualizeSpotLights {
		for _, l := range s.spotLights {
			far := l.Position.Add(l.Direction.Normalize().Mul(l.Radius))
			gamma := float32(math.Acos(float64(l.Angles[0])))
			s.direct.Light3(l.Color)
			s.direct.RegularPyramidLine(l.Position, l.Direction.Normalize().Mul(l.Radius), gamma, 8)
			s.direct.CircleLine(far, l.Direction, float32(math.Sin(float64(gamma)))*l.Radius)
			s.direct.Circle(l.Position, s.camPos.Sub(l.Position), 0.5)
		}
	}

	if ui.enableDirectDraw {
		gl.BlitNamedFramebuffer(s.gBuffer.Id(), 0, 0, 0, int32(ViewportWidth), int32(ViewportHeight), 0, 0, int32(ViewportWidth), int32(ViewportHeight), gl.DEPTH_BUFFER_BIT, gl.NEAREST)
		s.direct.Color(0.9, 0, 0)
		s.direct.Line(mgl32.Vec3{}, mgl32.Vec3{50, 0, 0})
		s.direct.Color(0, 0.9, 0)
		s.direct.Line(mgl32.Vec3{}, mgl32.Vec3{0, 50, 0})
		s.direct.Color(0, 0, 0.9)
		s.direct.Line(mgl32.Vec3{}, mgl32.Vec3{0, 0, 50})
		s.direct.Draw(s.projMat.Mul4(s.viewMat))
	} else {
		s.direct.Clear()
	}
	gl.PopDebugGroup()
}
