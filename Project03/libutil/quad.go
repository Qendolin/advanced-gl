package libutil

import (
	"advanced-gl/Project03/libgl"

	"github.com/go-gl/gl/v4.5-core/gl"
)

var sharedQuad libgl.UnboundVertexArray

func DrawQuad() {
	if sharedQuad == nil {
		vbo := libgl.NewBuffer()
		vbo.Allocate([]float32{-1, -1, 1, -1, -1, 1, 1, 1}, 0)

		sharedQuad = libgl.NewVertexArray()
		sharedQuad.Layout(0, 0, 2, gl.FLOAT, false, 0)
		sharedQuad.BindBuffer(0, vbo, 0, 2*4)
	}

	sharedQuad.Bind()
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
}
