package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"sync"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/pierrec/lz4/v4"
)

const MagicNumberLZ4 = 0x184d2204
const MagicNumberGEO = 0xc9dae18c

var lz4header = []byte{0x04, 0x22, 0x4d, 0x18}

type Mesh struct {
	Name         string
	Vertices     []Vertex
	Indices      []byte
	ShortIndices bool
}

type Vertex struct {
	Position mgl32.Vec3
	Normal   mgl32.Vec3
	Uv       mgl32.Vec2
}

type BinaryReader struct {
	Order     binary.ByteOrder
	Src       io.Reader
	Index     int
	LastIndex int
	Err       error
	buf       []byte
}

type Scene struct {
	PointLights []ScenePointLight
	OrthoLights []SceneOrthoLight
	SpotLights  []SceneSpotLight
	Objects     []SceneObject
	Materials   []SceneMaterial
	Shadows     []*ShadowCaster
}

type SceneObject struct {
	Name     string
	Mesh     string
	Material string
	Position mgl32.Vec3
	Rotation mgl32.Quat
	Scale    mgl32.Vec3
}

type SceneMaterial struct {
	Name          string
	AlbedoTexture string
	NormalTexture string
	Transparent   bool
}

func (o SceneObject) Transform() mgl32.Mat4 {
	translate := mgl32.Translate3D(o.Position[0], o.Position[1], o.Position[2])
	rotate := o.Rotation.Mat4()
	scale := mgl32.Scale3D(o.Scale[0], o.Scale[1], o.Scale[2])
	return translate.Mul4(rotate).Mul4(scale)
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

func LoadMeshes(r io.Reader) (models []Mesh, err error) {
	br := &BinaryReader{
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

	var magic int
	if !br.ReadUInt32(&magic) {
		return nil, fmt.Errorf("expected one uint32 for the file identifier; byte 0x%08x", br.LastIndex)
	}

	if magic == MagicNumberLZ4 {
		lz4decoder := lz4.NewReader(io.MultiReader(bytes.NewReader(lz4header), br.Src))
		return LoadMeshes(lz4decoder)
	} else if magic != MagicNumberGEO {
		return nil, fmt.Errorf("expected magic number 0x%08x (lz4) or 0x%08x (geo) but was 0x%08x; byte 0x%08x", MagicNumberLZ4, MagicNumberGEO, magic, br.LastIndex)
	}

	var count int
	if !br.ReadUInt32(&count) {
		return nil, fmt.Errorf("expected one uint32 for object count; byte 0x%08x", br.LastIndex)
	}

	offsets := make([]uint32, count)
	if !br.ReadRef(offsets) {
		return nil, fmt.Errorf("expected %d uint32 for offsets; byte 0x%08x", count, br.LastIndex)
	}
	models = make([]Mesh, count)
	for i := 0; i < count; i++ {
		mesh, err := LoadMesh(br)

		if err != nil {
			return nil, fmt.Errorf("error loading mesh at index %d/%d: %w", i+1, count, err)
		}

		models[i] = *mesh
	}

	return models, nil
}

func LoadMesh(r io.Reader) (mesh *Mesh, err error) {
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
		return nil, fmt.Errorf("expected %d byte for object name; byte 0x%08x", header.NameLength, br.LastIndex)
	}

	vertices := make([]Vertex, header.VertexCount)
	if !br.ReadRef(&vertices) {
		return nil, fmt.Errorf("expected %d mesh vertices; name %q, byte 0x%08x", header.VertexCount, name, br.LastIndex)
	}
	indexBytes := header.IndexCount * 2
	shortIndices := header.IndexCount < 0xffff
	if !shortIndices {
		indexBytes *= 2
	}
	indices := make([]byte, indexBytes)
	if !br.ReadRef(&indices) {
		return nil, fmt.Errorf("expected %d mesh indices; name %q, byte 0x%08x", header.IndexCount, name, br.LastIndex)
	}
	if shortIndices && header.IndexCount%2 == 1 {
		if !br.ReadBytes(2) {
			return nil, fmt.Errorf("expected index padding; name %q, byte 0x%08x", name, br.LastIndex)
		}
	}

	return &Mesh{
		Name:         string(name),
		Vertices:     vertices,
		Indices:      indices,
		ShortIndices: header.IndexCount < 0xffff,
	}, nil
}

func LoadScene(r io.Reader) (*Scene, error) {
	header := make([]byte, 4)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(header, lz4header) {
		lz4decoder := lz4.NewReader(io.MultiReader(bytes.NewReader(lz4header), r))
		return LoadScene(lz4decoder)
	}

	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(header), r))
	if err != nil {
		return nil, err
	}
	sceneModel := struct {
		Objects []struct {
			Name     string    `json:"name"`
			Mesh     string    `json:"mesh"`
			Material string    `json:"material"`
			Position []float32 `json:"position"`
			Rotation []float32 `json:"rotation"`
			Scale    []float32 `json:"scale"`
		} `json:"objects"`
		Lights []struct {
			Name        string    `json:"name"`
			Type        string    `json:"type"`
			Position    []float32 `json:"position"`
			Rotation    []float32 `json:"rotation"`
			Color       []float32 `json:"color"`
			Radius      float32   `json:"radius"`
			Attenuation float32   `json:"attenuation"`
			Power       float32   `json:"power"`
			InnerAngle  float32   `json:"innerAngle"`
			OuterAngle  float32   `json:"outerAngle"`
			Shadow      bool      `json:"shadow"`
		} `json:"lights"`
		Materials []struct {
			Name          string `json:"name"`
			AlbedoTexture string `json:"albedoTexture"`
			NormalTexture string `json:"normalTexture"`
			Transparent   bool   `json:"transparent"`
		} `json:"materials"`
	}{}
	err = json.Unmarshal(data, &sceneModel)
	if err != nil {
		return nil, err
	}

	scene := Scene{
		PointLights: []ScenePointLight{},
		OrthoLights: []SceneOrthoLight{},
		SpotLights:  []SceneSpotLight{},
		Objects:     []SceneObject{},
		Materials:   []SceneMaterial{},
		Shadows:     []*ShadowCaster{},
	}
	unitSphere := float32(4 * math.Pi)
	for _, l := range sceneModel.Lights {
		position := mgl32.Vec3{l.Position[0], l.Position[1], l.Position[2]}
		color := mgl32.Vec3{l.Color[0], l.Color[1], l.Color[2]}
		radius := l.Radius
		if len(l.Rotation) == 0 {
			l.Rotation = []float32{1, 0, 0, 0}
		}
		quat := mgl32.Quat{W: l.Rotation[0], V: mgl32.Vec3{l.Rotation[1], l.Rotation[2], l.Rotation[3]}}
		direction := quat.Rotate(mgl32.Vec3{0, -1, 0})
		azimuth := float32(math.Atan2(float64(direction[2]), float64(direction[0]))) * Rad2Deg
		elevation := float32(math.Asin(float64(direction[1]))) * Rad2Deg
		shadowIndex := len(scene.Shadows) + 1
		if l.Type == "POINT" || !l.Shadow {
			shadowIndex = 0
		}
		if l.Type == "POINT" {
			scene.PointLights = append(scene.PointLights, ScenePointLight{
				Position:    position,
				Color:       color,
				Brightness:  l.Power / unitSphere,
				Radius:      radius,
				Attenuation: 1,
			})
		} else if l.Type == "SPOT" {
			if shadowIndex != 0 {
				// TODO: Shadow Clip Planes
				scene.Shadows = append(scene.Shadows, CreateShadowCasterPerspective(512, l.OuterAngle*Deg2Rad, 50))
			}
			scene.SpotLights = append(scene.SpotLights, SceneSpotLight{
				Position:    position,
				Color:       color,
				Brightness:  l.Power / unitSphere,
				Azimuth:     azimuth,
				Elevation:   elevation,
				Radius:      radius,
				Attenuation: 1,
				InnerAngle:  l.InnerAngle / 2 * Rad2Deg,
				OuterAngle:  (l.OuterAngle - l.InnerAngle) / 2 * Rad2Deg,
				ShadowIndex: shadowIndex,
			})
		} else if l.Type == "SUN" {
			if shadowIndex != 0 {
				// TODO: Shadow Size & Clip Planes
				scene.Shadows = append(scene.Shadows, CreateShadowCasterOrtho(1024, 50, 0.5, 50))
			}
			scene.OrthoLights = append(scene.OrthoLights, SceneOrthoLight{
				Azimuth:     azimuth,
				Elevation:   -elevation,
				Color:       color,
				Brightness:  l.Power,
				ShadowIndex: shadowIndex,
			})
		}
	}

	for _, m := range sceneModel.Materials {
		scene.Materials = append(scene.Materials, SceneMaterial{
			Name:          m.Name,
			AlbedoTexture: m.AlbedoTexture,
			NormalTexture: m.NormalTexture,
			Transparent:   m.Transparent,
		})
	}

	for _, o := range sceneModel.Objects {
		scene.Objects = append(scene.Objects, SceneObject{
			Name:     o.Name,
			Mesh:     o.Mesh,
			Material: o.Material,
			Position: mgl32.Vec3{o.Position[0], o.Position[1], o.Position[2]},
			Rotation: mgl32.Quat{W: o.Rotation[0], V: mgl32.Vec3{o.Rotation[1], o.Rotation[2], o.Rotation[3]}},
			Scale:    mgl32.Vec3{o.Scale[0], o.Scale[1], o.Scale[2]},
		})
	}

	return &scene, nil
}

func FlipV(img *image.RGBA) {
	upperHalf := make([]byte, img.Stride*(img.Rect.Dy()/2))
	copy(upperHalf, img.Pix)
	end := len(img.Pix)
	for y := 0; y < img.Rect.Dy()/2; y++ {
		top := img.Pix[y*img.Stride : (y+1)*img.Stride]
		bottom := img.Pix[end-(y+1)*img.Stride : end-y*img.Stride]
		copy(top, bottom)

		top = upperHalf[y*img.Stride : (y+1)*img.Stride]
		copy(bottom, top)
	}
}

func LoadTextureImage(p string) (*image.RGBA, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	if fi, err := file.Stat(); err == nil && fi.IsDir() {
		return nil, os.ErrNotExist
	} else if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(raw.Bounds())
	draw.Draw(rgba, rgba.Bounds(), raw, image.Point{}, draw.Over)
	FlipV(rgba)
	return rgba, nil
}

type RenderBatchBuilder struct {
	ShortIndices  bool
	vertexSize    int
	indexSize     int
	vertexOffsets []int
	indexOffsets  []int
	meshes        []Mesh
	materials     []SceneMaterial
	textures      map[string]*image.RGBA
	mutex         sync.Mutex
}

func (builder *RenderBatchBuilder) AddMesh(mesh Mesh) {
	for _, m := range builder.meshes {
		if m.Name == mesh.Name {
			log.Printf("Mesh %q is already in batch", mesh.Name)
			return
		}
	}

	if builder.ShortIndices && !mesh.ShortIndices {
		log.Panicf("Mesh %q does not use short indices", mesh.Name)
	} else if !builder.ShortIndices && mesh.ShortIndices {
		intIndices := make([]byte, len(mesh.Indices)*2)
		for i := 0; i < len(mesh.Indices); i += 2 {
			intIndices[i*2] = mesh.Indices[i]
			intIndices[i*2+1] = mesh.Indices[i+1]
		}
		mesh.Indices = intIndices
	}

	builder.meshes = append(builder.meshes, mesh)

	vertexSize := len(mesh.Vertices) * int(unsafe.Sizeof(Vertex{}))
	builder.vertexOffsets = append(builder.vertexOffsets, builder.vertexSize+vertexSize)
	builder.vertexSize += vertexSize

	indexSize := len(mesh.Indices)
	builder.indexOffsets = append(builder.indexOffsets, builder.indexSize+indexSize)
	builder.indexSize += indexSize
}

func (builder *RenderBatchBuilder) AddMaterial(material SceneMaterial, root string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("could not load material %q: %w", material.Name, err)
		}
	}()

	for _, m := range builder.materials {
		if m.Name == material.Name {
			return fmt.Errorf("material %q is already in batch", material.Name)
		}
	}

	fileExt := decideTextureFileExtension(path.Join(root, material.AlbedoTexture))

	albedo, err := LoadTextureImage(path.Join(root, material.AlbedoTexture+fileExt))
	if errors.Is(err, os.ErrNotExist) {
		albedo = image.NewRGBA(image.Rect(0, 0, 1, 1))
	} else if err != nil {
		return err
	}
	normal, err := LoadTextureImage(path.Join(root, material.NormalTexture+fileExt))
	if errors.Is(err, os.ErrNotExist) {
		normal = image.NewRGBA(image.Rect(0, 0, 1, 1))
		normal.Pix[0] = 255 / 2
		normal.Pix[1] = 255 / 2
		normal.Pix[2] = 255
	} else if err != nil {
		return err
	}

	builder.mutex.Lock()
	builder.materials = append(builder.materials, material)
	builder.textures[material.AlbedoTexture] = albedo
	builder.textures[material.NormalTexture] = normal
	builder.mutex.Unlock()

	return nil
}

func decideTextureFileExtension(base string) string {
	if fi, err := os.Stat(base + ".png"); err == nil && !fi.IsDir() {
		return ".png"
	}
	if fi, err := os.Stat(base + ".jpg"); err == nil && !fi.IsDir() {
		return ".jpg"
	}
	return ""
}

func (builder *RenderBatchBuilder) Upload() *RenderBatch {
	if builder.vertexSize == 0 {
		log.Panic("builder has no data to upload")
	}
	meshLocations := make([]MeshLocation, len(builder.meshes))
	meshIndex := make(map[string]int, len(builder.meshes))

	vertices := NewBuffer()
	vertices.AllocateEmpty(builder.vertexSize, gl.DYNAMIC_STORAGE_BIT)
	offset := 0
	for i, m := range builder.meshes {
		meshIndex[m.Name] = i
		meshLocations[i] = MeshLocation{
			BaseVertex: int32(offset / int(unsafe.Sizeof(Vertex{}))),
		}
		vertices.Write(offset, m.Vertices)
		offset = builder.vertexOffsets[i]
	}

	var bytesPerIndex uint32
	if builder.ShortIndices {
		bytesPerIndex = 2
	} else {
		bytesPerIndex = 4
	}

	elements := NewBuffer()
	elements.AllocateEmpty(builder.indexSize, gl.DYNAMIC_STORAGE_BIT)
	offset = 0
	for i, m := range builder.meshes {
		loc := &meshLocations[i]
		loc.BaseIndex = uint32(offset) / bytesPerIndex
		loc.Indices = uint32(len(m.Indices) / int(bytesPerIndex))

		elements.Write(offset, m.Indices)
		offset = builder.indexOffsets[i]
	}

	attributes := NewBuffer()
	attributes.AllocateEmpty(int(unsafe.Sizeof(InstanceAttributes{}))*1024, gl.DYNAMIC_STORAGE_BIT)

	vao := NewVertexArray()
	vao.Layout(0, 0, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Position)))
	vao.Layout(0, 1, 3, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Normal)))
	vao.Layout(0, 2, 2, gl.FLOAT, false, int(unsafe.Offsetof(Vertex{}.Uv)))

	vao.Layout(1, 3, 4, gl.FLOAT, false, 0*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 4, 4, gl.FLOAT, false, 1*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 5, 4, gl.FLOAT, false, 2*int(unsafe.Sizeof(mgl32.Vec4{})))
	vao.Layout(1, 6, 4, gl.FLOAT, false, 3*int(unsafe.Sizeof(mgl32.Vec4{})))
	// TODO: move to vao interface
	gl.VertexArrayBindingDivisor(vao.Id(), 1, 1)

	vao.BindBuffer(0, vertices, 0, int(unsafe.Sizeof(Vertex{})))
	vao.BindBuffer(1, attributes, 0, int(unsafe.Sizeof(InstanceAttributes{})))
	vao.BindElementBuffer(elements)

	commands := NewBuffer()
	commands.AllocateEmpty(int(unsafe.Sizeof(DrawElementsIndirectCommand{}))*4096, gl.DYNAMIC_STORAGE_BIT)

	builder.meshes = nil

	materialsBatches := make([]MaterialBatch, len(builder.materials))
	materialIndex := make(map[string]int, len(builder.materials))

	for i, mat := range builder.materials {
		batch := &materialsBatches[i]
		batch.Name = mat.Name
		batch.Transparent = mat.Transparent
		batch.instances = []MeshInstance{}
		batch.CommandRange = [2]int{}

		albedo := NewTexture(gl.TEXTURE_2D)
		albedoImg := builder.textures[mat.AlbedoTexture]
		albedo.Allocate(0, gl.SRGB8_ALPHA8, albedoImg.Rect.Dx(), albedoImg.Rect.Dy(), 0)
		albedo.Load(0, albedoImg.Rect.Dx(), albedoImg.Rect.Dy(), 0, gl.RGBA, albedoImg.Pix)
		albedo.GenerateMipmap()
		normal := NewTexture(gl.TEXTURE_2D)
		normalImg := builder.textures[mat.NormalTexture]
		normal.Allocate(1, gl.RGB8, normalImg.Rect.Dx(), normalImg.Rect.Dy(), 0)
		normal.Load(0, normalImg.Rect.Dx(), normalImg.Rect.Dy(), 0, gl.RGBA, normalImg.Pix)
		batch.Textures = []UnboundTexture{albedo, normal}
	}

	// Sort transparent materials to the end
	sort.Slice(materialsBatches, func(i, j int) bool {
		return !materialsBatches[i].Transparent && materialsBatches[j].Transparent
	})

	for i, mBatch := range materialsBatches {
		materialIndex[mBatch.Name] = i
	}

	builder.materials = nil
	builder.textures = nil

	batch := &RenderBatch{
		MeshBuffer:       vertices,
		ElementBuffer:    elements,
		AttributesBuffer: attributes,
		VertexArray:      vao,
		MaterialBatches:  materialsBatches,
		MaterialIndex:    materialIndex,
		meshLocations:    meshLocations,
		meshIndex:        meshIndex,
		CommandBuffer:    commands,
	}

	return batch
}

type MeshLocation struct {
	BaseVertex int32
	BaseIndex  uint32
	Indices    uint32
}

type RenderBatch struct {
	VertexArray        UnboundVertexArray
	AttributesBuffer   UnboundBuffer
	MeshBuffer         UnboundBuffer
	ElementBuffer      UnboundBuffer
	CommandBuffer      UnboundBuffer
	TotatCommandRange  [2]int
	MaterialBatches    []MaterialBatch
	MaterialIndex      map[string]int
	meshLocations      []MeshLocation
	meshIndex          map[string]int
	attributesPosition int
	commands           []DrawElementsIndirectCommand
}

type MaterialBatch struct {
	Name         string
	Transparent  bool
	Textures     []UnboundTexture
	CommandRange [2]int
	instances    []MeshInstance
}

type InstanceAttributes struct {
	ModelMatrix mgl32.Mat4
}

type MeshInstance struct {
	Mesh           int
	AttributeIndex int
}

func (batch *RenderBatch) Add(mesh, material string, attributes InstanceAttributes) {
	if _, ok := batch.meshIndex[mesh]; !ok {
		log.Panicf("Mesh %q is not contained in this batch", mesh)
	}
	if _, ok := batch.MaterialIndex[material]; !ok {
		log.Printf("Material %q is not contained in this batch\n", material)
	}
	vbo := batch.AttributesBuffer
	size := int(unsafe.Sizeof(attributes))
	if vbo.Grow(batch.attributesPosition + size) {
		batch.VertexArray.BindBuffer(1, vbo, 0, int(unsafe.Sizeof(InstanceAttributes{})))
	}
	vbo.Write(batch.attributesPosition, []InstanceAttributes{attributes})
	mBatch := batch.ByName(material)
	mBatch.instances = append(mBatch.instances, MeshInstance{
		Mesh:           batch.meshIndex[mesh],
		AttributeIndex: batch.attributesPosition / size,
	})

	batch.attributesPosition += int(size)
}

func (batch *RenderBatch) ByName(material string) *MaterialBatch {
	return &batch.MaterialBatches[batch.MaterialIndex[material]]
}

type DrawElementsIndirectCommand struct {
	Count         uint32
	InstanceCount uint32
	FirstIndex    uint32 // The offset for the first index of the mesh in the ebo
	BaseVertex    int32  // The offset for the first vertex of the mesh in the vbo
	BaseInstance  uint32
}

func (batch *RenderBatch) GenerateDrawCommands() {
	if batch.commands == nil {
		batch.commands = []DrawElementsIndirectCommand{}
	}
	batch.commands = batch.commands[:0]

	for i := range batch.MaterialBatches {
		mBatch := &batch.MaterialBatches[i]
		start := len(batch.commands) * int(unsafe.Sizeof(DrawElementsIndirectCommand{}))
		mBatch.CommandRange = [2]int{start, len(mBatch.instances)}
		for _, instance := range mBatch.instances {
			loc := batch.meshLocations[instance.Mesh]
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

	size := binary.Size(batch.commands)
	if size > batch.CommandBuffer.Size() {
		batch.CommandBuffer.Grow(size)
	}
	batch.CommandBuffer.Write(0, batch.commands)
	batch.TotatCommandRange = [2]int{0, len(batch.commands)}
}

func generateLightSphere() []mgl32.Vec3 {
	// lat = theta = v = ring
	// long  = phi = u = segment

	var (
		segments = 8
		rings    = 3
		r        = 1.
	)

	verts := make([]mgl32.Vec3, 0, segments*(rings+1)*2)

	dTheta := math.Pi / float64(rings+1)
	dPhi := -math.Pi / float64(segments)

	for ring := 0; ring < rings+1; ring++ {
		theta := float64(ring) * dTheta
		for segment := 0; segment < segments; segment++ {
			phi := 2 * float64(segment) * dPhi

			x := r * math.Sin(theta) * math.Cos(phi)
			y := r * math.Cos(theta)
			z := r * math.Sin(theta) * math.Sin(phi)
			verts = append(verts, mgl32.Vec3{float32(x), float32(y), float32(z)})

			x = r * math.Sin(theta+dTheta) * math.Cos(phi)
			y = r * math.Cos(theta+dTheta)
			z = r * math.Sin(theta+dTheta) * math.Sin(phi)
			verts = append(verts, mgl32.Vec3{float32(x), float32(y), float32(z)})
		}
	}

	return verts
}
