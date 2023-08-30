package ibl

import (
	"advanced-gl/Project03/libgl"
	"advanced-gl/Project03/libutil"
	"advanced-gl/Project03/stbi"

	_ "embed"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

//go:embed convert.vert
var vertShaderSrc string

//go:embed convert.frag
var fragShaderSrc string

type glConverter struct {
	hdrSampler     libgl.UnboundSampler
	cubemapSampler libgl.UnboundSampler
	captureFbo     libgl.UnboundFramebuffer
	shader         libgl.UnboundShaderPipeline
	cubeVao        libgl.UnboundVertexArray
	cubeVbo        libgl.UnboundBuffer
}

// size should generally be 1/4 of the equirectangular width
func NewGlConverter() (conv Converter, err error) {
	cleanup := []libutil.Deleter{}
	defer func() {
		if err != nil {
			for _, v := range cleanup {
				v.Delete()
			}
		}
	}()

	hdrSampler := libgl.NewSampler()
	hdrSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, 0)
	hdrSampler.FilterMode(gl.LINEAR, gl.LINEAR)
	cleanup = append(cleanup, hdrSampler)

	dummyCubemap := libgl.NewTexture(gl.TEXTURE_CUBE_MAP)
	dummyCubemap.Allocate(1, gl.RGB16F, 2, 2, 0)
	cleanup = append(cleanup, dummyCubemap)

	cubemapSampler := libgl.NewSampler()
	cubemapSampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE)
	cubemapSampler.FilterMode(gl.LINEAR, gl.LINEAR)
	cleanup = append(cleanup, cubemapSampler)

	captureFbo := libgl.NewFramebuffer()
	captureFbo.AttachTextureLayer(0, dummyCubemap, 0)
	captureFbo.BindTargets(0)
	cleanup = append(cleanup, captureFbo)
	if err := captureFbo.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		return nil, err
	}

	shader := libgl.NewPipeline()
	cleanup = append(cleanup, shader)
	vsh := libgl.NewShader(vertShaderSrc, gl.VERTEX_SHADER)
	cleanup = append(cleanup, vsh)
	if err := vsh.Compile(); err != nil {
		return nil, err
	}
	shader.Attach(vsh, gl.VERTEX_SHADER_BIT)
	fsh := libgl.NewShader(fragShaderSrc, gl.FRAGMENT_SHADER)
	cleanup = append(cleanup, fsh)
	if err := fsh.Compile(); err != nil {
		return nil, err
	}
	shader.Attach(fsh, gl.FRAGMENT_SHADER_BIT)

	cubeVao := libgl.NewVertexArray()
	cleanup = append(cleanup, cubeVao)
	cubeVbo := libgl.NewBuffer()
	cleanup = append(cleanup, cubeVbo)
	cubeVbo.Allocate(NewUnitCube(), 0)
	cubeVao.Layout(0, 0, 3, gl.FLOAT, false, 0)
	cubeVao.BindBuffer(0, cubeVbo, 0, 3*4)

	dummyCubemap.Delete()

	return &glConverter{
		hdrSampler:     hdrSampler,
		cubemapSampler: cubemapSampler,
		captureFbo:     captureFbo,
		shader:         shader,
		cubeVao:        cubeVao,
		cubeVbo:        cubeVbo,
	}, nil
}

// Converts an equirectangular hdr image to six hdr cubemap faces
func (conv *glConverter) Convert(image *stbi.RgbaHdr, size int) (*IblEnv, error) {
	hdrTexture := libgl.NewTexture(gl.TEXTURE_2D)
	hdrTexture.Allocate(1, gl.RGB16F, image.Rect.Dx(), image.Rect.Dy(), 0)
	hdrTexture.Load(0, image.Rect.Dx(), image.Rect.Dy(), 0, gl.RGBA, image.Pix)
	defer hdrTexture.Delete()

	cubemap := libgl.NewTexture(gl.TEXTURE_CUBE_MAP)
	cubemap.Allocate(1, gl.RGB16F, size, size, 0)
	defer cubemap.Delete()

	captureProjection := mgl32.Perspective(mgl32.DegToRad(90.0), 1.0, 0.1, 10.0)
	captureViews := []mgl32.Mat4{
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{1.0, 0.0, 0.0}, mgl32.Vec3{0.0, -1.0, 0.0}),
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{-1.0, 0.0, 0.0}, mgl32.Vec3{0.0, -1.0, 0.0}),
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, 1.0, 0.0}, mgl32.Vec3{0.0, 0.0, 1.0}),
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, -1.0, 0.0}, mgl32.Vec3{0.0, 0.0, -1.0}),
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, 0.0, 1.0}, mgl32.Vec3{0.0, -1.0, 0.0}),
		mgl32.LookAtV(mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, 0.0, -1.0}, mgl32.Vec3{0.0, -1.0, 0.0}),
	}

	// convert HDR equirectangular environment map to cubemap equivalent
	conv.shader.Bind()
	conv.shader.Get(gl.FRAGMENT_SHADER).SetUniform("u_equirectangular_texture", 0)
	conv.shader.Get(gl.VERTEX_SHADER).SetUniform("u_projection_mat", captureProjection)
	hdrTexture.Bind(0)
	conv.hdrSampler.Bind(0)
	conv.cubeVao.Bind()

	// don't forget to configure the viewport to the capture dimensions.
	libgl.GlState.Viewport(0, 0, size, size)
	conv.captureFbo.Bind(gl.DRAW_FRAMEBUFFER)
	for i := 0; i < 6; i++ {
		conv.shader.Get(gl.VERTEX_SHADER).SetUniform("u_view_mat", captureViews[i])

		conv.captureFbo.AttachTextureLayer(0, cubemap, i)

		gl.DrawArrays(gl.TRIANGLES, 0, 6*6)
	}

	faceLen := size * size * 3
	result := make([]float32, 6*faceLen)

	if libgl.GlEnv.UseIntelCubemaDsaFix {
		for i := 0; i < 6; i++ {
			face := uint32(gl.TEXTURE_CUBE_MAP_POSITIVE_X + i)
			cubemap.As(face).Bind(0)
			gl.GetTexImage(face, 0, gl.RGB, gl.FLOAT, libutil.Pointer(&result[i*faceLen]))
		}
	} else {
		gl.GetTextureImage(cubemap.Id(), 0, gl.RGB, gl.FLOAT, int32(len(result)*4), libutil.Pointer(result))
	}

	return NewIblEnv(result, size, 1), nil
}

func (conv *glConverter) Release() {
	conv.captureFbo.Delete()
	conv.cubeVao.Delete()
	conv.cubeVbo.Delete()
	conv.cubemapSampler.Delete()
	conv.hdrSampler.Delete()
	conv.shader.Get(gl.VERTEX_SHADER).Delete()
	conv.shader.Get(gl.FRAGMENT_SHADER).Delete()
	conv.shader.Delete()
}
