package main

import (
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

//	For my sanity:
//
//	src=1 * (w x h)
//
//		Up		Down
//	0=	1		1/2
//	1=	1/2		1/4
//	2=	1/4		1/8
//	3=	1/8		1/16
//	4=	1/16	1/32
//	5=	1/32	1/64
//
//	Down:
//
//	down(src) -> Down[0]
//	down(Down[1]) -> Down[2]
//	down(Down[2]) -> Down[3]
//	down(Down[3]) -> Down[4]
//	down(Down[4]) -> Down[5]
//
//	Up:
//
//	Down[4] + up(Down[5]) -> Up[5]
//	Down[3] + up(Up[5]) -> Up[4]
//	Down[2] + up(Up[4]) -> Up[3]
//	Down[1] + up(Up[3]) -> Up[2]
//	Down[0] + up(Up[2]) -> Up[1]
//	up(Up[1]) -> Up[0]
func DrawBloom() {
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Bloom\x00"))
	defer gl.PopDebugGroup()

	if !ui.enableBloom {
		s.bloomBuffer.AttachTextureLevel(0, s.bloomDown, 0)
		s.bloomBuffer.Bind(gl.DRAW_FRAMEBUFFER)
		GlState.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		return
	}

	s.quad.Bind()
	s.bloomDownShader.Bind()
	s.bloomBuffer.Bind(gl.DRAW_FRAMEBUFFER)
	s.bloomSampler.Bind(0)
	s.bloomSampler.Bind(1)

	fragSh := s.bloomDownShader.Get(gl.FRAGMENT_SHADER)
	tex := s.bloomDown

	vw, vh := ViewportWidth, ViewportHeight

	GlState.SetEnabled()
	s.postBuffer.GetTexture(0).Bind(0)
	s.bloomBuffer.AttachTextureLevel(0, tex, 0)
	knee := ui.bloomThreshold*ui.bloomKnee + 1e-5
	fragSh.SetUniform("u_threshold", mgl32.Vec4{ui.bloomThreshold, ui.bloomThreshold - knee, knee * 2, 0.25 / knee})
	GlState.Viewport(0, 0, vw/2, vh/2)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	fragSh.SetUniform("u_threshold", mgl32.Vec4{})

	for i := 0; i < len(s.bloomDownViews)-1; i++ {
		s.bloomDownViews[i].Bind(0)
		s.bloomBuffer.AttachTextureLevel(0, tex, i+1)
		GlState.Viewport(0, 0, vw/(4<<i), vh/(4<<i))
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	}

	s.bloomUpShader.Bind()
	upView := s.bloomDownViews[len(s.bloomDownViews)-1]
	for i := len(s.bloomUpViews) - 1; i >= 0; i-- {
		if i == 0 {
			GlState.BindTextureUnit(0, 0)
		} else {
			s.bloomDownViews[i-1].Bind(0)
		}
		upView.Bind(1)
		s.bloomBuffer.AttachTextureLevel(0, s.bloomUp, i)
		GlState.Viewport(0, 0, vw/(1<<i), vh/(1<<i))
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
		upView = s.bloomUpViews[i].Bind(1)
	}
}
