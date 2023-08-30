package libscn

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libgl"
	"advanced-gl/Project03/stbi"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/pierrec/lz4/v4"
)

type AssetIndex struct {
	Materials []string `json:"materials"`
	Textures  []string `json:"textures"`
	Meshes    []string `json:"meshes"`
	Models    []string `json:"models"`
	Shaders   []string `json:"shaders"`
	Hdris     []string `json:"hdris"`
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
	TextureIndex  map[string]string
	ShaderIndex   map[string]string
	HdriIndex     map[string]string
	init          bool
}

func (pack *DirPack) AddIndexFile(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("could not add index file %q: %w", name, err)
	}
	defer file.Close()

	return pack.AddIndex(file, path.Dir(name))
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

	if !pack.init {
		pack.MeshIndex = map[string]string{}
		pack.ModelIndex = map[string]string{}
		pack.MaterialIndex = map[string]string{}
		pack.ShaderIndex = map[string]string{}
		pack.HdriIndex = map[string]string{}
		pack.TextureIndex = map[string]string{}
		pack.init = true
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
	err = pack.addAllMatches(root, index.Hdris, pack.HdriIndex)
	if err != nil {
		return err
	}
	err = pack.addAllMatches(root, index.Textures, pack.TextureIndex)
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

			name, _, _ := strings.Cut(path.Base(match), ".")
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
	defer file.Close()

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

func (pack *DirPack) LoadShaderPipeline(name string) (libgl.UnboundShaderPipeline, error) {
	filename, ok := pack.ShaderIndex[name]
	if !ok {
		return nil, fmt.Errorf("shader pipeline %q is not registered in this pack", name)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open shader pipeline file %q: %w", filename, err)
	}
	defer file.Close()

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

	pipeline := libgl.NewPipeline()
	pipeline.Attach(vertexSh, gl.VERTEX_SHADER_BIT)
	pipeline.Attach(fragmentSh, gl.FRAGMENT_SHADER_BIT)

	return pipeline, nil
}

func (pack *DirPack) LoadShader(filename string, stage int) (libgl.ShaderProgram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open shader file %q: %w", filename, err)
	}
	defer file.Close()

	src, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read shader file %q: %w", filename, err)
	}

	return libgl.NewShader(string(src), stage), nil
}

type Material struct {
	Name   string
	Albedo libgl.UnboundTexture
	Normal libgl.UnboundTexture
	ORM    libgl.UnboundTexture
}

func (mat *Material) Delete() {
	mat.Albedo.Delete()
	mat.Albedo = nil
	mat.Normal.Delete()
	mat.Normal = nil
	mat.ORM.Delete()
	mat.ORM = nil
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
	defer file.Close()

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

	albeoTexture := libgl.NewTexture(gl.TEXTURE_2D)
	albeoTexture.Allocate(0, gl.RGBA8, albedo.Rect.Dx(), albedo.Rect.Dy(), 0)
	albeoTexture.Load(0, albedo.Rect.Dx(), albedo.Rect.Dy(), 0, gl.RGBA, albedo.Pix)
	albeoTexture.GenerateMipmap()

	normalTexture := libgl.NewTexture(gl.TEXTURE_2D)
	normalTexture.Allocate(0, gl.RGB8, normal.Rect.Dx(), normal.Rect.Dy(), 0)
	normalTexture.Load(0, normal.Rect.Dx(), normal.Rect.Dy(), 0, gl.RGBA, normal.Pix)
	normalTexture.GenerateMipmap()

	ormTexture := libgl.NewTexture(gl.TEXTURE_2D)
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

func (pack *DirPack) LoadTexture(name string) (*stbi.RgbaLdr, error) {
	filename, ok := pack.TextureIndex[name]
	if !ok {
		return nil, fmt.Errorf("texture %q is not registered in this pack", name)
	}
	return pack.LoadTextureImage(filename)
}

func (pack *DirPack) LoadTextureImage(filename string) (*stbi.RgbaLdr, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open texture image file %q: %w", filename, err)
	}
	defer file.Close()
	stbi.Default.FlipVertically = true
	stbi.Default.CopyData = true
	return stbi.Load(file)
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
	defer file.Close()

	var src io.Reader = file
	if strings.HasSuffix(filename, ".lz4") {
		src = lz4.NewReader(file)
	}

	mesh, err := DecodeMesh(src)
	if err != nil {
		return nil, fmt.Errorf("could not decode mesh file %q: %w", filename, err)
	}

	return mesh, nil
}

func (pack *DirPack) LoadHdri(name string) (*ibl.IblEnv, error) {
	filename, ok := pack.HdriIndex[name]
	if !ok {
		return nil, fmt.Errorf("hdri %q is not registered in this pack", name)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open hdri file %q: %w", filename, err)
	}
	defer file.Close()

	var src io.Reader = file
	if strings.HasSuffix(filename, ".lz4") {
		src = lz4.NewReader(file)
	}

	mesh, err := ibl.DecodeIblEnv(src)
	if err != nil {
		return nil, fmt.Errorf("could not decode hdri file %q: %w", filename, err)
	}

	return mesh, nil
}
