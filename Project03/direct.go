package main

import (
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type DirectBuffer struct {
	vao       UnboundVertexArray
	vbo       UnboundBuffer
	shader    UnboundShaderPipeline
	data      []float32
	color     mgl32.Vec3
	stroke    float32
	shaded    bool
	autoShade bool
	normal    mgl32.Vec3
}

func NewDirectDrawBuffer(shader UnboundShaderPipeline) *DirectBuffer {
	vao := NewVertexArray()
	vao.Layout(0, 0, 3, gl.FLOAT, false, 0)
	vao.Layout(0, 1, 3, gl.FLOAT, false, 3*4)
	vao.Layout(0, 2, 3, gl.FLOAT, false, 6*4)
	vbo := NewBuffer()
	vbo.AllocateEmpty(1e6, gl.DYNAMIC_STORAGE_BIT)
	// 3 floats position + 3 floats color + 3 float normal
	vao.BindBuffer(0, vbo, 0, (3+3+3)*4)

	return &DirectBuffer{
		vao:    vao,
		vbo:    vbo,
		shader: shader,
		data:   []float32{},
		color:  mgl32.Vec3{1, 1, 1},
		stroke: 0.05,
	}
}

func (db *DirectBuffer) Stroke(width float32) {
	db.stroke = width
}

func (db *DirectBuffer) Shaded() {
	db.shaded = true
}

func (db *DirectBuffer) Unshaded() {
	db.shaded = false
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
	var normal mgl32.Vec3
	if db.shaded {
		normal = db.normal
	}
	db.data = append(db.data, pos[0], pos[1], pos[2], db.color[0], db.color[1], db.color[2], normal[0], normal[1], normal[2])
}

// A--B
// | /
// C
func (db *DirectBuffer) Tri(a, b, c mgl32.Vec3) {
	if db.shaded && db.autoShade {
		ab := b.Sub(a)
		ac := c.Sub(a)
		db.normal = ab.Cross(ac).Normalize()
	}
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
	db.normal = n.Normalize()
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
	db.normal = n.Normalize()
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

// center, dir, angle, sides
func (db *DirectBuffer) RegularPyramidLine(c, d mgl32.Vec3, a float32, s int) {
	r := d.Len() * float32(math.Sin(float64(a)))
	step := float32(2*math.Pi) / float32(s)
	rot := mgl32.HomogRotate3D(step, d.Normalize()).Mat3()
	n := Perpendicular(d).Normalize()
	for i := 0; i < s; i++ {
		db.normal = n
		v := d.Add(n.Mul(r))
		db.Line(c, c.Add(v))
		n = rot.Mul3x1(n)
	}
}

// center, radius
func (db *DirectBuffer) UvSphere(c mgl32.Vec3, r float32) {
	db.autoShade = true
	rings, segments := db.circleSides(r)/2, db.circleSides(r)

	dTheta := math.Pi / float64(rings)
	dPhi := -math.Pi / float64(segments)

	prevRing := make([]mgl32.Vec3, segments)
	currRing := make([]mgl32.Vec3, segments)

	for ring := 0; ring < rings+1; ring++ {
		theta := float64(ring) * dTheta
		for segment := 0; segment < segments; segment++ {
			phi := 2 * float64(segment) * dPhi

			x := r * float32(math.Sin(theta)*math.Cos(phi))
			y := r * float32(math.Cos(theta))
			z := r * float32(math.Sin(theta)*math.Sin(phi))
			v := c.Add(mgl32.Vec3{float32(x), float32(y), float32(z)})

			currRing[segment] = v
			if segment > 0 {
				if ring > 0 {
					db.Quad(currRing[segment-1], currRing[segment], prevRing[segment-1], prevRing[segment])
				}
			}
		}
		if ring > 0 {
			db.Quad(currRing[segments-1], currRing[0], prevRing[segments-1], prevRing[0])
		}
		currRing, prevRing = prevRing, currRing
	}
	db.autoShade = false
}

func (db *DirectBuffer) Draw(viewProj mgl32.Mat4, camPos mgl32.Vec3) {
	if len(db.data) == 0 {
		return
	}

	bufferSize := len(db.data) * 4
	if db.vbo.Grow(bufferSize) {
		db.vao.BindBuffer(0, db.vbo, 0, (3+3+3)*4)
	}
	db.vbo.Write(0, db.data)

	db.vao.Bind()
	db.shader.Bind()
	db.shader.Get(gl.VERTEX_SHADER).SetUniform("u_view_projection_mat", viewProj)
	db.shader.Get(gl.FRAGMENT_SHADER).SetUniform("u_camera_position", camPos)
	GlState.SetEnabled(DepthTest, PolygonOffsetFill)
	GlState.DepthFunc(DepthFuncLess)
	GlState.DepthMask(true)
	GlState.PolygonOffset(-1, -1)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(db.data)/9))

	db.Clear()
}

func (db *DirectBuffer) Clear() {
	db.data = db.data[0:0]
}
