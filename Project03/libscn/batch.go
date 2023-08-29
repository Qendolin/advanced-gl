package libscn

import (
	"advanced-gl/Project03/libgl"
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type RenderBatch struct {
	VertexArray        libgl.UnboundVertexArray
	AttributesBuffer   libgl.UnboundBuffer
	VertexBuffer       libgl.UnboundBuffer
	ElementBuffer      libgl.UnboundBuffer
	CommandBuffer      libgl.UnboundBuffer
	TotatCommandRange  [2]int
	Materials          []MaterialSlice
	MaterialIndex      map[string]int
	meshLocations      []MeshLocation
	meshIndex          map[string]int
	attributesPosition int
	vertexPosition     int
	elementPosition    int
	commands           []DrawElementsIndirectCommand
}

type MeshLocation struct {
	BaseVertex int32
	BaseIndex  uint32
	Indices    uint32
}

type MaterialSlice struct {
	Material      *Material
	ElementOffset int
	ElementCount  int
	instances     []MeshInstance
}

type InstanceAttributes struct {
	ModelMatrix mgl32.Mat4
}

type MeshInstance struct {
	MeshIndex      int
	AttributeIndex int
}

type DrawElementsIndirectCommand struct {
	Count         uint32
	InstanceCount uint32
	FirstIndex    uint32 // The offset for the first index of the mesh in the ebo
	BaseVertex    int32  // The offset for the first vertex of the mesh in the vbo
	BaseInstance  uint32
}

func NewRenderBatch() *RenderBatch {
	vertices := libgl.NewBuffer()
	vertices.AllocateEmpty(8*1_000_000*VertexSize, gl.DYNAMIC_STORAGE_BIT)

	elements := libgl.NewBuffer()
	elements.AllocateEmpty(8*1_000_000*ElementIndexSize, gl.DYNAMIC_STORAGE_BIT)

	attributes := libgl.NewBuffer()
	attributes.AllocateEmpty(InstanceAttributesSize*1024, gl.DYNAMIC_STORAGE_BIT)

	vao := libgl.NewVertexArray()
	vao.Layout(0, 0, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Position)))
	vao.Layout(0, 1, 2, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Uv)))
	vao.Layout(0, 2, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Normal)))
	vao.Layout(0, 3, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Bitangent)))
	vao.Layout(0, 4, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Tangent)))

	vao.Layout(1, 5, 4, gl.FLOAT, false, 0*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 6, 4, gl.FLOAT, false, 1*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 7, 4, gl.FLOAT, false, 2*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 8, 4, gl.FLOAT, false, 3*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.AttribDivisor(1, 1)

	vao.BindBuffer(0, vertices, 0, VertexSize)
	vao.BindBuffer(1, attributes, 0, InstanceAttributesSize)
	vao.BindElementBuffer(elements)

	commands := libgl.NewBuffer()
	commands.AllocateEmpty(DrawCommandSize*4096, gl.DYNAMIC_STORAGE_BIT)

	return &RenderBatch{
		VertexBuffer:     vertices,
		ElementBuffer:    elements,
		AttributesBuffer: attributes,
		VertexArray:      vao,
		Materials:        []MaterialSlice{},
		MaterialIndex:    map[string]int{},
		meshLocations:    []MeshLocation{},
		meshIndex:        map[string]int{},
		CommandBuffer:    commands,
	}
}

func (batch *RenderBatch) Upload(mesh *Mesh) {
	verticesSize := len(mesh.Vertices) * VertexSize
	if batch.VertexBuffer.Grow(batch.vertexPosition + verticesSize) {
		batch.VertexArray.BindBuffer(0, batch.VertexBuffer, 0, verticesSize)
	}
	batch.VertexBuffer.Write(batch.vertexPosition, mesh.Vertices)
	location := MeshLocation{
		BaseVertex: int32(batch.vertexPosition / VertexSize),
	}
	batch.vertexPosition += verticesSize

	indicesSize := len(mesh.Indices) * ElementIndexSize
	if batch.ElementBuffer.Grow(batch.elementPosition + indicesSize) {
		batch.VertexArray.BindElementBuffer(batch.ElementBuffer)
	}
	batch.ElementBuffer.Write(batch.elementPosition, mesh.Indices)
	location.BaseIndex = uint32(batch.elementPosition)
	location.Indices = uint32(len(mesh.Indices))
	batch.elementPosition += indicesSize

	batch.meshLocations = append(batch.meshLocations, location)
	batch.meshIndex[mesh.Name] = len(batch.meshLocations) - 1
}

func (batch *RenderBatch) AddMaterial(material *Material) {
	slice := MaterialSlice{
		Material:      material,
		ElementOffset: 0,
		ElementCount:  0,
		instances:     []MeshInstance{},
	}
	batch.Materials = append(batch.Materials, slice)
	batch.MaterialIndex[material.Name] = len(batch.Materials) - 1
}

func (batch *RenderBatch) Add(mesh, material string, attributes InstanceAttributes) {
	if _, ok := batch.meshIndex[mesh]; !ok {
		log.Panicf("Mesh %q is not contained in this batch", mesh)
	}
	if _, ok := batch.MaterialIndex[material]; !ok {
		log.Printf("Material %q is not contained in this batch\n", material)
	}
	vbo := batch.AttributesBuffer
	if vbo.Grow(batch.attributesPosition + InstanceAttributesSize) {
		batch.VertexArray.BindBuffer(1, vbo, 0, InstanceAttributesSize)
	}
	vbo.Write(batch.attributesPosition, []InstanceAttributes{attributes})
	mBatch := batch.ByMaterial(material)
	mBatch.instances = append(mBatch.instances, MeshInstance{
		MeshIndex:      batch.meshIndex[mesh],
		AttributeIndex: batch.attributesPosition / InstanceAttributesSize,
	})

	batch.attributesPosition += InstanceAttributesSize
}

func (batch *RenderBatch) ByMaterial(material string) *MaterialSlice {
	return &batch.Materials[batch.MaterialIndex[material]]
}

func (batch *RenderBatch) GenerateDrawCommands() {
	if batch.commands == nil {
		batch.commands = []DrawElementsIndirectCommand{}
	}
	batch.commands = batch.commands[:0]

	for i := range batch.Materials {
		slice := &batch.Materials[i]
		slice.ElementOffset = len(batch.commands) * DrawCommandSize
		slice.ElementCount = len(slice.instances)
		for _, instance := range slice.instances {
			loc := batch.meshLocations[instance.MeshIndex]
			cmd := DrawElementsIndirectCommand{
				Count:         loc.Indices,
				InstanceCount: 1,
				FirstIndex:    loc.BaseIndex,
				BaseVertex:    int32(loc.BaseVertex),
				BaseInstance:  uint32(instance.AttributeIndex),
			}
			batch.commands = append(batch.commands, cmd)
		}
	}

	batch.CommandBuffer.Grow(len(batch.commands) * DrawCommandSize)
	batch.CommandBuffer.Write(0, batch.commands)
	batch.TotatCommandRange = [2]int{0, len(batch.commands)}
}
