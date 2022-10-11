package main

import (
	"math"
	"math/rand"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

func GenerateSsaoSamples(samples int, min, curve float32) []mgl32.Vec3 {
	rng := rand.New(rand.NewSource(0))
	// Avoid shallow angles to prevent issues caused by imprecision (e.g. acne)
	bias := float32(7.5 * math.Pi / 180.)
	kernel := make([]mgl32.Vec3, samples)
	for i := 0; i < samples; i++ {
		theta := rng.Float64() * math.Pi * 2
		phi := float64(lerp(0, math.Pi/2-bias, rng.Float32()))

		sample := mgl32.Vec3{
			float32(math.Sin(phi) * math.Cos(theta)),
			float32(math.Sin(phi) * math.Sin(theta)),
			float32(math.Cos(phi)),
		}
		sample = sample.Normalize()

		scale := float32(i) / float32(samples)
		scale = lerp(min, 1.0, float32(math.Pow(float64(scale), float64(curve))))
		sample = sample.Mul(scale)
		kernel[i] = sample
	}
	return kernel
}

func GenerateSsaoNoise(size int) []mgl32.Vec3 {
	length := size * size
	noise := make([]mgl32.Vec3, length)
	for i := 0; i < length; i++ {
		vec := mgl32.Vec3{
			rand.Float32()*2 - 1,
			rand.Float32()*2 - 1,
			rand.Float32(),
		}
		noise[i] = vec.Normalize()
	}
	return noise
}

func GenerateSsaoPattern(size int) UnboundTexture {
	kernel := GenerateSsaoNoise(size)
	texture := NewTexture(gl.TEXTURE_2D)
	texture.Allocate(1, gl.RGBA16F, size, size, 0)
	texture.Load(0, size, size, 0, gl.RGB, kernel)
	return texture
}

func DrawSSAO() {
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw SSAO\x00"))
	defer gl.PopDebugGroup()
	if ui.enableSsao {
		s.quad.Bind()
		s.ssaoShader.Bind()
		s.ssaoShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_projection_mat", s.projMat)
		s.ssaoShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_view_mat", s.viewMat)
		s.ssaoShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_radius", ui.ssaoRadius)
		s.ssaoShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_exponent", ui.ssaoExponent)
		s.bufferSampler.Bind(0)
		s.gBuffer.GetTexture(0).Bind(0)
		s.bufferSampler.Bind(1)
		s.gBuffer.GetTexture(1).Bind(1)
		s.ssaoNoise.Bind(2)
		s.patternSampler.Bind(2)
		s.ssaoBuffer.Bind(gl.DRAW_FRAMEBUFFER)
		s.ssaoBuffer.BindTargets(0, 1)
		GlState.ClearColor(1, 1, 1, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		// FIXME: Stencil Test causes issues, but also fixes some
		// Flip normals on back faces?
		GlState.SetEnabled(StencilTest, DepthTest)
		GlState.DepthFunc(DepthFuncGreater)
		GlState.DepthMask(false)
		GlState.StencilMask(0)
		GlState.StencilFunc(StencilFuncNotEqual, 0xff, 0xff)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

		s.ssaoBlurShader.Bind()
		s.ssaoBlurShader.Get(gl.FRAGMENT_SHADER).SetUniform("u_edge_threshold", ui.ssaoEdgeThreshold)
		s.ssaoBuffer.BindTargets(1)
		s.ssaoBuffer.GetTexture(1).Bind(1)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	} else {
		s.ssaoBuffer.Bind(gl.DRAW_FRAMEBUFFER)
		s.ssaoBuffer.BindTargets(0, 1)
		GlState.ClearColor(1, 1, 1, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}
}
