package main

import (
	"log"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type ShadowCaster struct {
	ShadowMap  UnboundFramebuffer
	Transform  mgl32.Mat4
	projMat    mgl32.Mat4
	resolution int
}

func CreateShadowCasterOrtho(resolution int, size, nearz, farz float32) *ShadowCaster {
	fbo := NewFramebuffer()
	depthTex := NewTexture(gl.TEXTURE_2D)
	depthTex.Allocate(0, gl.DEPTH_COMPONENT16, resolution, resolution, 0)
	fbo.AttachTexture(gl.DEPTH_ATTACHMENT, depthTex)
	fbo.Bind(gl.DRAW_FRAMEBUFFER)
	if err := fbo.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		log.Panic(err)
	}

	projMat := mgl32.Ortho(-size/2, size/2, -size/2, size/2, nearz, farz)

	return &ShadowCaster{
		ShadowMap:  fbo,
		projMat:    projMat,
		resolution: resolution,
	}
}

func CreateShadowCasterPerspective(resolution int, fov, farz float32) *ShadowCaster {
	fbo := NewFramebuffer()
	depthTex := NewTexture(gl.TEXTURE_2D)
	depthTex.Allocate(0, gl.DEPTH_COMPONENT16, resolution, resolution, 0)
	fbo.AttachTexture(gl.DEPTH_ATTACHMENT, depthTex)
	fbo.Bind(gl.DRAW_FRAMEBUFFER)
	if err := fbo.Check(gl.DRAW_FRAMEBUFFER); err != nil {
		log.Panic(err)
	}

	projMat := mgl32.Perspective(fov, 1, 0.5, farz)

	return &ShadowCaster{
		ShadowMap:  fbo,
		projMat:    projMat,
		resolution: resolution,
	}
}

func (caster *ShadowCaster) LookAt(target, dir mgl32.Vec3, distance float32) {
	viewMat := mgl32.LookAtV(target.Add(dir.Normalize().Mul(-distance)), target, mgl32.Vec3{0, 1, 0})
	caster.Transform = caster.projMat.Mul4(viewMat)
}

func (caster *ShadowCaster) LookFrom(pos, dir mgl32.Vec3) {
	viewMat := mgl32.LookAtV(pos, pos.Add(dir), mgl32.Vec3{0, 1, 0})
	caster.Transform = caster.projMat.Mul4(viewMat)
}

func (caster *ShadowCaster) Draw() {
	caster.ShadowMap.Bind(gl.DRAW_FRAMEBUFFER)
	GlState.Viewport(0, 0, caster.resolution, caster.resolution)
	GlState.SetEnabled(DepthTest, PolygonOffsetFill)
	GlState.DepthFunc(DepthFuncLess)
	GlState.DepthMask(true)
	GlState.PolygonOffsetClamp(ui.shadowOffsetFactor, ui.shadowOffsetUnits, ui.shadowOffsetClamp)
	s.shadowShader.Bind()
	s.shadowShader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", caster.Transform)
	s.shadowShader.Get(gl.VERTEX_SHADER).SetUniform("u_bias", ui.shadowBiasDraw)
	s.batch.VertexArray.Bind()
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(s.batch.TotatCommandRange[0]), int32(s.batch.TotatCommandRange[1]), 0)
	GlState.Viewport(0, 0, ViewportWidth, ViewportHeight)
}

func (caster *ShadowCaster) Clear() {
	caster.ShadowMap.Bind(gl.DRAW_FRAMEBUFFER)
	GlState.Viewport(0, 0, caster.resolution, caster.resolution)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	GlState.Viewport(0, 0, ViewportWidth, ViewportHeight)
}
