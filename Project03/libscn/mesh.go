package libscn

import (
	"advanced-gl/Project03/libio"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
)

const MagicNumberGEO = 0xc9dae18c

type Mesh struct {
	Name         string
	Vertices     []Vertex
	Indices      []uint32
	ShortIndices bool
}

type Vertex struct {
	Position  mgl32.Vec3
	Uv        mgl32.Vec2
	Normal    mgl32.Vec3
	Bitangent mgl32.Vec3
	Tangent   mgl32.Vec3
}

const ElementIndexSize = int(unsafe.Sizeof(uint32(0)))
const InstanceAttributesSize = int(unsafe.Sizeof(InstanceAttributes{}))
const VertexSize = int(unsafe.Sizeof(Vertex{}))
const DrawCommandSize = int(unsafe.Sizeof(DrawElementsIndirectCommand{}))

func DecodeMesh(r io.Reader) (mesh *Mesh, err error) {
	var br *libio.BinaryReader
	var ok bool

	if br, ok = r.(*libio.BinaryReader); !ok {
		br = &libio.BinaryReader{
			Src:   r,
			Order: binary.LittleEndian,
		}

		defer func() {
			if br.Err != nil {
				if err == nil {
					err = br.Err
				} else {
					err = fmt.Errorf("%v: %w", err, br.Err)
				}
			}
		}()
	}

	header := struct {
		Check       uint32
		NameLength  uint32
		VertexCount uint32
		IndexCount  uint32
	}{}
	if !br.ReadRef(&header) {
		return nil, fmt.Errorf("expected mesh header; byte 0x%08x", br.LastIndex)
	}

	if header.Check != MagicNumberGEO {
		return nil, fmt.Errorf("mesh header is corrupt; byte 0x%08x", br.LastIndex)
	}

	name := make([]byte, header.NameLength)
	if !br.ReadRef(&name) {
		return nil, fmt.Errorf("expected %d bytes for object name; byte 0x%08x", header.NameLength, br.LastIndex)
	}

	type fileVertex struct {
		Position mgl32.Vec3
		Normal   mgl32.Vec3
		Uv       mgl32.Vec2
	}

	fileVertices := make([]fileVertex, header.VertexCount)
	if !br.ReadRef(&fileVertices) {
		return nil, fmt.Errorf("expected %d mesh vertices; name %q, byte 0x%08x", header.VertexCount, name, br.LastIndex)
	}
	indexShorts := header.IndexCount
	shortIndices := header.IndexCount < 0xffff
	if !shortIndices {
		indexShorts *= 2
	}
	indices := make([]uint16, indexShorts)
	if !br.ReadRef(&indices) {
		return nil, fmt.Errorf("expected %d mesh indices; name %q, byte 0x%08x", header.IndexCount, name, br.LastIndex)
	}
	if shortIndices && header.IndexCount%2 == 1 {
		if !br.ReadBytes(2) {
			return nil, fmt.Errorf("expected index padding; name %q, byte 0x%08x", name, br.LastIndex)
		}
	}

	var intIndices []uint32
	if shortIndices {
		intIndices = make([]uint32, header.IndexCount)
		for i, v := range indices {
			intIndices[i] = uint32(v)
		}
	} else {
		first := (*uint32)(unsafe.Pointer(&indices[0]))
		intIndices = unsafe.Slice(first, header.IndexCount)
	}

	vertices := make([]Vertex, len(fileVertices))

	for i := 0; i < len(intIndices); i += 3 {
		i0, i1, i2 := intIndices[i+0], intIndices[i+1], intIndices[i+2]
		v0, v1, v2 := fileVertices[i0], fileVertices[i1], fileVertices[i2]

		dPos01, dPos02 := v1.Position.Sub(v0.Position), v2.Position.Sub(v0.Position)
		dUV01, dUV02 := v1.Uv.Sub(v0.Uv), v2.Uv.Sub(v0.Uv)

		det := dUV01[0]*dUV02[1] - dUV02[0]*dUV01[1]
		f := 1 / det

		tan := mgl32.Vec3{
			f * (dUV02[1]*dPos01[0] - dUV01[1]*dPos02[0]),
			f * (dUV02[1]*dPos01[1] - dUV01[1]*dPos02[1]),
			f * (dUV02[1]*dPos01[2] - dUV01[1]*dPos02[2]),
		}

		bitan := mgl32.Vec3{
			f * (-dUV02[0]*dPos01[0] + dUV01[0]*dPos02[0]),
			f * (-dUV02[0]*dPos01[1] + dUV01[0]*dPos02[1]),
			f * (-dUV02[0]*dPos01[2] + dUV01[0]*dPos02[2]),
		}

		vertices[i0].Position = v0.Position
		vertices[i0].Uv = v0.Uv
		vertices[i0].Normal = v0.Normal
		// 'average' the tanget vectors
		vertices[i0].Bitangent = vertices[i0].Bitangent.Add(bitan)
		vertices[i0].Tangent = vertices[i0].Tangent.Add(tan)

		vertices[i1].Position = v1.Position
		vertices[i1].Uv = v1.Uv
		vertices[i1].Normal = v1.Normal
		// 'average' the tanget vectors
		vertices[i1].Bitangent = vertices[i1].Bitangent.Add(bitan)
		vertices[i1].Tangent = vertices[i1].Tangent.Add(tan)

		vertices[i2].Position = v2.Position
		vertices[i2].Uv = v2.Uv
		vertices[i2].Normal = v2.Normal
		// 'average' the tanget vectors
		vertices[i2].Bitangent = vertices[i2].Bitangent.Add(bitan)
		vertices[i2].Tangent = vertices[i2].Tangent.Add(tan)
	}

	return &Mesh{
		Name:     string(name),
		Vertices: vertices,
		Indices:  intIndices,
	}, nil
}
