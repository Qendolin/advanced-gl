package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const MagicNumberGEO = 0xc9dae18c

type AssetIndex struct {
	Materials []string `json:"materials"`
	Meshes    []string `json:"meshes"`
	Models    []string `json:"models"`
	Shaders   []string `json:"shaders"`
}

type ModelDesc struct {
	Mesh     string `json:"mesh"`
	Material string `json:"material"`
}

type MaterialDesc struct {
	Albedo string `json:"albedo"`
	Normal string `json:"normal"`
	ORM    string `json:"orm"`
}

type LoadCallback func(ctx *glfw.Window)

type LoadManger interface {
	OnLoad(cb LoadCallback)
	Reload(ctx *glfw.Window)
}

type SimpleLoadManager struct {
	cbs []LoadCallback
}

func (lm *SimpleLoadManager) OnLoad(cb LoadCallback) {
	lm.cbs = append(lm.cbs, cb)
}

func (lm *SimpleLoadManager) Reload(ctx *glfw.Window) {
	for _, cb := range lm.cbs {
		cb(ctx)
	}
}

type DirPack struct {
	MeshIndex     map[string]string
	ModelIndex    map[string]string
	MaterialIndex map[string]string
	ShaderIndex   map[string]string
}

func (pack *DirPack) AddIndexFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("could not add index file %q: %w", name, err)
	}

	return pack.AddIndex(f, path.Dir(name))
}

func (pack *DirPack) AddIndex(r io.Reader, root string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	index := AssetIndex{}
	err = json.Unmarshal(data, &index)
	if err != nil {
		return err
	}

	if pack.MeshIndex == nil {
		pack.MeshIndex = map[string]string{}
	}

	if pack.ModelIndex == nil {
		pack.ModelIndex = map[string]string{}
	}

	if pack.MaterialIndex == nil {
		pack.MaterialIndex = map[string]string{}
	}

	if pack.ShaderIndex == nil {
		pack.ShaderIndex = map[string]string{}
	}

	root = path.Clean(root)
	err = pack.addAllMatches(root, index.Materials, pack.MaterialIndex)
	if err != nil {
		return err
	}
	err = pack.addAllMatches(root, index.Meshes, pack.MeshIndex)
	if err != nil {
		return err
	}
	err = pack.addAllMatches(root, index.Models, pack.ModelIndex)
	if err != nil {
		return err
	}
	err = pack.addAllMatches(root, index.Shaders, pack.ShaderIndex)
	if err != nil {
		return err
	}

	return nil
}

func (pack *DirPack) addAllMatches(root string, patterns []string, index map[string]string) error {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(path.Join(root, pattern))
		if err != nil {
			return err
		}
		for _, match := range matches {
			match = filepath.ToSlash(match)
			name := strings.TrimSuffix(path.Base(match), path.Ext(match))
			index[name] = match
		}
	}
	return nil
}

type Model struct {
	Mesh     *Mesh
	Material *Material
}

func (pack *DirPack) LoadModel(name string) (*Model, error) {
	filename, ok := pack.ModelIndex[name]
	if !ok {
		return nil, fmt.Errorf("model %q is not registered in this pack", name)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open model file %q: %w", filename, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read model file %q: %w", filename, err)
	}

	modelDesc := &ModelDesc{}
	err = json.Unmarshal(data, modelDesc)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal model file %q: %w", filename, err)
	}

	material, err := pack.LoadMaterial(modelDesc.Material)
	if err != nil {
		return nil, fmt.Errorf("could not load material %q for model %q: %w", modelDesc.Material, filename, err)
	}
	mesh, err := pack.LoadMesh(modelDesc.Mesh)
	if err != nil {
		return nil, fmt.Errorf("could not load mesh %q for model %q: %w", modelDesc.Material, filename, err)
	}

	return &Model{
		Mesh:     mesh,
		Material: material,
	}, nil
}

type ShaderPipelineDesc struct {
	Vertex   string `json:"vertex"`
	Fragment string `json:"fragment"`
}

func (pack *DirPack) LoadShaderPipeline(name string) (UnboundShaderPipeline, error) {
	filename, ok := pack.ShaderIndex[name]
	if !ok {
		return nil, fmt.Errorf("shader pipeline %q is not registered in this pack", name)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open shader pipeline file %q: %w", filename, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read shader pipeline file %q: %w", filename, err)
	}

	shaderDesc := &ShaderPipelineDesc{}
	err = json.Unmarshal(data, shaderDesc)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal shader pipeline file %q: %w", filename, err)
	}

	root := path.Dir(filename)

	vertexSh, err := pack.LoadShader(path.Join(root, shaderDesc.Vertex), gl.VERTEX_SHADER)
	if err != nil {
		return nil, fmt.Errorf("could not load vertex shader %q for shader pipeline %q: %w", shaderDesc.Vertex, filename, err)
	}
	if err = vertexSh.Compile(); err != nil {
		return nil, fmt.Errorf("could not compile vertex shader %q for shader pipeline %q: %w", shaderDesc.Vertex, filename, err)
	}
	fragmentSh, err := pack.LoadShader(path.Join(root, shaderDesc.Fragment), gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, fmt.Errorf("could not load fragment shader %q for shader pipeline %q: %w", shaderDesc.Fragment, filename, err)
	}
	if err = fragmentSh.Compile(); err != nil {
		return nil, fmt.Errorf("could not compile fragment shader %q for shader pipeline %q: %w", shaderDesc.Fragment, filename, err)
	}

	pipeline := NewPipeline()
	pipeline.Attach(vertexSh, gl.VERTEX_SHADER_BIT)
	pipeline.Attach(fragmentSh, gl.FRAGMENT_SHADER_BIT)

	return pipeline, nil
}

func (pack *DirPack) LoadShader(filename string, stage int) (ShaderProgram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open shader file %q: %w", filename, err)
	}

	src, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read shader file %q: %w", filename, err)
	}

	return NewShader(string(src), stage), nil
}

type Material struct {
	Name   string
	Albedo UnboundTexture
	Normal UnboundTexture
	ORM    UnboundTexture
}

func (pack *DirPack) LoadMaterial(name string) (*Material, error) {
	filename, ok := pack.MaterialIndex[name]
	if !ok {
		return nil, fmt.Errorf("material %q is not registered in this pack", name)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open material file %q: %w", filename, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read material file %q: %w", filename, err)
	}

	materialDesc := &MaterialDesc{}
	err = json.Unmarshal(data, materialDesc)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal material file %q: %w", filename, err)
	}

	root := path.Dir(filename)

	albedo, err := pack.LoadTextureImage(path.Join(root, materialDesc.Albedo))
	if err != nil {
		return nil, fmt.Errorf("could not load 'albedo' texture image %q for material %q: %w", materialDesc.Albedo, filename, err)
	}

	normal, err := pack.LoadTextureImage(path.Join(root, materialDesc.Normal))
	if err != nil {
		return nil, fmt.Errorf("could not load 'normal' texture image %q for material %q: %w", materialDesc.Normal, filename, err)
	}

	orm, err := pack.LoadTextureImage(path.Join(root, materialDesc.ORM))
	if err != nil {
		return nil, fmt.Errorf("could not load 'orm' texture image %q for material %q: %w", materialDesc.ORM, filename, err)
	}

	albeoTexture := NewTexture(gl.TEXTURE_2D)
	albeoTexture.Allocate(0, gl.RGBA8, albedo.Rect.Dx(), albedo.Rect.Dy(), 0)
	albeoTexture.Load(0, albedo.Rect.Dx(), albedo.Rect.Dy(), 0, gl.RGBA, albedo.Pix)
	albeoTexture.GenerateMipmap()

	normalTexture := NewTexture(gl.TEXTURE_2D)
	normalTexture.Allocate(0, gl.RGB8, normal.Rect.Dx(), normal.Rect.Dy(), 0)
	normalTexture.Load(0, normal.Rect.Dx(), normal.Rect.Dy(), 0, gl.RGBA, normal.Pix)
	normalTexture.GenerateMipmap()

	ormTexture := NewTexture(gl.TEXTURE_2D)
	ormTexture.Allocate(0, gl.RGB8, orm.Rect.Dx(), orm.Rect.Dy(), 0)
	ormTexture.Load(0, orm.Rect.Dx(), orm.Rect.Dy(), 0, gl.RGBA, orm.Pix)
	ormTexture.GenerateMipmap()

	return &Material{
		Name:   name,
		Albedo: albeoTexture,
		Normal: normalTexture,
		ORM:    ormTexture,
	}, nil
}

func (pack *DirPack) LoadTextureImage(filename string) (*image.RGBA, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open texture image file %q: %w", filename, err)
	}

	return DecodeImage(file)
}

func (pack *DirPack) LoadMesh(name string) (*Mesh, error) {
	filename, ok := pack.MeshIndex[name]
	if !ok {
		return nil, fmt.Errorf("mesh %q is not registered in this pack", name)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open mesh file %q: %w", filename, err)
	}

	mesh, err := DecodeMesh(file)
	if err != nil {
		return nil, fmt.Errorf("could not decode mesh file %q: %w", filename, err)
	}

	return mesh, nil
}

type Mesh struct {
	Name         string
	Vertices     []Vertex
	Indices      []uint32
	ShortIndices bool
}

const ElementIndexSize = int(unsafe.Sizeof(uint32(0)))
const InstanceAttributesSize = int(unsafe.Sizeof(InstanceAttributes{}))
const VertexSize = int(unsafe.Sizeof(Vertex{}))
const DrawCommandSize = int(unsafe.Sizeof(DrawElementsIndirectCommand{}))

type Vertex struct {
	Position  mgl32.Vec3
	Uv        mgl32.Vec2
	Normal    mgl32.Vec3
	Bitangent mgl32.Vec3
	Tangent   mgl32.Vec3
}

type BinaryReader struct {
	Order     binary.ByteOrder
	Src       io.Reader
	Index     int
	LastIndex int
	Err       error
	buf       []byte
}

func (br *BinaryReader) ReadBytes(n int) (ok bool) {
	if br.Err != nil {
		return false
	}

	if cap(br.buf) <= n {
		br.buf = make([]byte, n)
	} else {
		br.buf = br.buf[:n]
	}

	nread, err := br.Src.Read(br.buf)
	if err != nil {
		br.Err = err
		ok = false
	}

	br.LastIndex = br.Index
	br.Index += nread

	return br.Err == nil
}

func (br *BinaryReader) Read(p []byte) (n int, err error) {
	return br.Src.Read(p)
}

func (br *BinaryReader) ReadUInt16(i *int) (ok bool) {
	if !br.ReadBytes(2) {
		return false
	}
	*i = int(br.Order.Uint16(br.buf))
	return true
}

func (br *BinaryReader) ReadUInt32(i *int) (ok bool) {
	if !br.ReadBytes(4) {
		return false
	}
	*i = int(br.Order.Uint32(br.buf))
	return true
}

func (br *BinaryReader) ReadRef(data any) (ok bool) {
	if br.Err != nil {
		return false
	}
	err := binary.Read(br.Src, br.Order, data)
	br.Err = err
	if err == nil {
		br.LastIndex = br.Index
		br.Index += binary.Size(data)
	}
	return err == nil
}

func DecodeMesh(r io.Reader) (mesh *Mesh, err error) {
	var br *BinaryReader
	var ok bool

	if br, ok = r.(*BinaryReader); !ok {
		br = &BinaryReader{
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
