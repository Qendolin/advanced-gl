package main

import (
	"log"
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// TODO: Direct draw commands for debugging
// e.g. lines
type DirectBuffer struct {
	vao    UnboundVertexArray
	vbo    UnboundBuffer
	shader UnboundShaderPipeline
	data   []mgl32.Vec3
	color  mgl32.Vec3
	stroke float32
}

func CreateDirectBuffer() *DirectBuffer {
	vao := NewVertexArray()
	vao.Layout(0, 0, 3, gl.FLOAT, false, 0)
	vao.Layout(0, 1, 3, gl.FLOAT, false, 3*4)
	vbo := NewBuffer()
	vbo.AllocateEmpty(1e6, gl.DYNAMIC_STORAGE_BIT)
	vao.BindBuffer(0, vbo, 0, 2*3*4)
	vertSh := NewShader(Res_DirectVshSrc, gl.VERTEX_SHADER)
	if err := vertSh.Compile(); err != nil {
		log.Panic(err)
	}
	fragSh := NewShader(Res_DirectFshSrc, gl.FRAGMENT_SHADER)
	if err := fragSh.Compile(); err != nil {
		log.Panic(err)
	}
	shader := NewPipeline()
	shader.Attach(vertSh, gl.VERTEX_SHADER_BIT)
	shader.Attach(fragSh, gl.FRAGMENT_SHADER_BIT)
	return &DirectBuffer{
		vao:    vao,
		vbo:    vbo,
		shader: shader,
		data:   []mgl32.Vec3{},
		color:  mgl32.Vec3{1, 1, 1},
		stroke: 0.05,
	}
}

func (db *DirectBuffer) Stroke(width float32) {
	db.stroke = width
}

func (db *DirectBuffer) Color(r, g, b float32) {
	db.color[0] = r
	db.color[1] = g
	db.color[2] = b
}

func (db *DirectBuffer) Color3(c mgl32.Vec3) {
	db.Color(c[0], c[1], c[2])
}

func (db *DirectBuffer) Light3(c mgl32.Vec3) {
	max := float32(math.Max(math.Max(float64(c[0]), float64(c[1])), float64(c[2])))
	db.Color(c[0]/max, c[1]/max, c[2]/max)
}

func (db *DirectBuffer) Vert(pos mgl32.Vec3) {
	db.data = append(db.data, pos, db.color)
}

// A--B
// | /
// C
func (db *DirectBuffer) Tri(a, b, c mgl32.Vec3) {
	db.Vert(a)
	db.Vert(c)
	db.Vert(b)
}

// A--B
// |  |
// C--D
func (db *DirectBuffer) Quad(a, b, c, d mgl32.Vec3) {
	db.Tri(a, b, c)
	db.Tri(d, c, b)
}

// A--B
func (db *DirectBuffer) Line(a, b mgl32.Vec3) {
	v := b.Sub(a)
	normal := Perpendicular(v).Normalize().Mul(db.stroke)
	bitangent := normal.Cross(v).Normalize().Mul(db.stroke)
	a0 := mgl32.Vec3{a[0] + normal[0]/2, a[1] + normal[1]/2, a[2] + normal[2]/2}
	a1 := mgl32.Vec3{a[0] - normal[0]/2, a[1] - normal[1]/2, a[2] - normal[2]/2}
	b0 := mgl32.Vec3{b[0] + normal[0]/2, b[1] + normal[1]/2, b[2] + normal[2]/2}
	b1 := mgl32.Vec3{b[0] - normal[0]/2, b[1] - normal[1]/2, b[2] - normal[2]/2}
	db.Quad(a0, b0, a1, b1)
	a0 = mgl32.Vec3{a[0] + bitangent[0]/2, a[1] + bitangent[1]/2, a[2] + bitangent[2]/2}
	a1 = mgl32.Vec3{a[0] - bitangent[0]/2, a[1] - bitangent[1]/2, a[2] - bitangent[2]/2}
	b0 = mgl32.Vec3{b[0] + bitangent[0]/2, b[1] + bitangent[1]/2, b[2] + bitangent[2]/2}
	b1 = mgl32.Vec3{b[0] - bitangent[0]/2, b[1] - bitangent[1]/2, b[2] - bitangent[2]/2}
	db.Quad(a0, b0, a1, b1)
}

// center, normal, radius
func (db *DirectBuffer) circleSides(r float32) int {
	return 24 + (int)(0.6*r)
}

// center, normal, radius
func (db *DirectBuffer) CircleLine(c, n mgl32.Vec3, r float32) {
	db.RegularPolyLine(c, n, r, db.circleSides(r))
}

// center, normal, radius
func (db *DirectBuffer) Circle(c, n mgl32.Vec3, r float32) {
	db.RegularPoly(c, n, r, db.circleSides(r))
}

// center, normal, radius, sides
func (db *DirectBuffer) RegularPolyLine(c, n mgl32.Vec3, r float32, s int) {
	step := float32(2*math.Pi) / float32(s)
	r0, r1 := r-db.stroke/2, r+db.stroke/2
	rot := mgl32.HomogRotate3D(step, n.Normalize()).Mat3()
	v0 := Perpendicular(n).Normalize()
	for i := 0; i < s; i++ {
		v1 := rot.Mul3x1(v0)
		db.Quad(c.Add(v0.Mul(r0)), c.Add(v0.Mul(r1)), c.Add(v1.Mul(r0)), c.Add(v1.Mul(r1)))
		v0 = v1
	}
}

// center, normal, radius, sides
func (db *DirectBuffer) RegularPoly(c, n mgl32.Vec3, r float32, s int) {
	step := float32(2*math.Pi) / float32(s)
	rot := mgl32.HomogRotate3D(step, n.Normalize()).Mat3()
	v0 := Perpendicular(n).Normalize()
	for i := 0; i < s; i++ {
		v1 := rot.Mul3x1(v0)
		db.Tri(c, c.Add(v0.Mul(r)), c.Add(v1.Mul(r)))
		v0 = v1
	}
}

// center, dir, angle
func (db *DirectBuffer) ConeLine(c, d mgl32.Vec3, a float32) {
	r := d.Len() * float32(math.Cos(float64(a)))
	db.RegularPyramidLine(c, d, a, db.circleSides(r))
}

// center, dir, angle
func (db *DirectBuffer) RegularPyramidLine(c, d mgl32.Vec3, a float32, s int) {
	r := d.Len() * float32(math.Sin(float64(a)))
	step := float32(2*math.Pi) / float32(s)
	rot := mgl32.HomogRotate3D(step, d.Normalize()).Mat3()
	v := d.Add(Perpendicular(d).Normalize().Mul(r))
	for i := 0; i < s; i++ {
		db.Line(c, c.Add(v))
		v = rot.Mul3x1(v)
	}
}

func (db *DirectBuffer) Draw(viewProj mgl32.Mat4) {
	bufferSize := len(db.data) * 2 * 3 * 4
	if db.vbo.Grow(bufferSize) {
		db.vao.BindBuffer(0, db.vbo, 0, 2*3*4)
	}
	db.vbo.Write(0, db.data)

	db.vao.Bind()
	db.shader.Bind()
	db.shader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", viewProj)
	GlState.SetEnabled(DepthTest, PolygonOffsetFill)
	GlState.DepthFunc(DepthFuncLess)
	GlState.DepthMask(true)
	GlState.PolygonOffset(-1, -1)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(db.data)/2))

	db.data = db.data[0:0]
}

func (db *DirectBuffer) Clear() {
	db.data = db.data[0:0]
}
