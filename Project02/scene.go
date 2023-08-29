package main

import (
	"log"
	"math"
	"sync"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"golang.org/x/exp/slices"
)

type Camera struct {
	Pos mgl32.Vec3
	// Radians
	Pitch, Yaw, Roll float32
	// Vertical FOV in Radians
	Fov       float32
	Near, Far float32
	Speed     float32
}

func (cam *Camera) ViewMatrix() mgl32.Mat4 {
	rotate := cam.Quaternion().Mat4()
	translate := mgl32.Translate3D(-cam.Pos[0], -cam.Pos[1], -cam.Pos[2])
	return rotate.Mul4(translate)
}

func (cam *Camera) ProjectionMatrix(aspect float32) mgl32.Mat4 {
	return mgl32.Perspective(cam.Fov, aspect, cam.Near, cam.Far)
}

func (cam *Camera) Quaternion() mgl32.Quat {
	return mgl32.AnglesToQuat(cam.Pitch, cam.Yaw, cam.Roll, mgl32.XYZ)
}

func (cam *Camera) Rotate(dPitch, dYaw, dRoll float32) {
	cam.Pitch += dPitch
	if cam.Pitch > math.Pi/2 {
		cam.Pitch = math.Pi
	} else if cam.Pitch < -math.Pi/2 {
		cam.Pitch = -math.Pi / 2
	}
	cam.Yaw += dYaw
	cam.Roll += dRoll
}

func (cam *Camera) Move(dX, dY, dZ float32) {
	cam.Pos.Add(mgl32.Vec3{dX, dY, dZ}.Mul(cam.Speed))
}

type InitFunc func(self *Node) interface{}
type UpdateFunc func(delta float32)
type DrawFunc func()

type Node struct {
	Name      string
	Parent    *Node
	Init      InitFunc
	Update    UpdateFunc
	Draw      DrawFunc
	Children  []*Node
	Interface interface{}
}

func (n *Node) Remove(child *Node) {
	index := slices.Index(n.Children, child)
	if index == -1 {
		return
	}
	n.Children = slices.Delete(n.Children, index, index+1)
	child.Parent = nil
}

func (n *Node) Add(child *Node) {
	if n == child {
		log.Printf("Error: added node to itself")
		return
	}

	if n.IsChildOf(child) {
		log.Printf("Error: added node to child of itself")
		return
	}

	if child.Parent != nil {
		child.Parent.Remove(child)
	}
	child.Parent = n
	n.Children = append(n.Children, child)
}

func (n *Node) IsChildOf(anchestor *Node) bool {
	return anchestor.IsAnchestorOf(n)
}

func (n *Node) IsAnchestorOf(child *Node) bool {
	for child.Parent != nil {
		if child.Parent == n {
			return true
		}
		child = child.Parent
	}
	return false
}

type Positioned interface {
	Position() mgl32.Vec3
}

type Rotated interface {
	Rotation() mgl32.Quat
}

type Scaled interface {
	Scale() float32
}

type Transformed interface {
	Positioned
	Rotated
	Scaled
	TransformMatrix() mgl32.Mat4
}

type TransformComp struct {
	position mgl32.Vec3
	rotation mgl32.Quat
	scale    float32
}

func (c *TransformComp) Position() mgl32.Vec3 {
	return c.position
}

func (c *TransformComp) SetPosition(position mgl32.Vec3) {
	c.position = position
}

func (c *TransformComp) Rotation() mgl32.Quat {
	return c.rotation
}

func (c *TransformComp) SetRotation(rotation mgl32.Quat) {
	c.rotation = rotation
}

func (c *TransformComp) Scale() float32 {
	return c.scale
}

func (c *TransformComp) SetScale(scale float32) {
	c.scale = scale
}

func (c *TransformComp) TransformMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(c.position[0], c.position[1], c.position[2]).
		Mul4(c.Rotation().Mat4().
			Mul4(mgl32.Scale3D(c.scale, c.scale, c.scale)))
}

type PositionFunc func() mgl32.Vec3

func (pos PositionFunc) Position() mgl32.Vec3 {
	return pos()
}

type globals struct {
	Input         InputManager
	GlContext     GlContext
	RenderManager RenderManager
	AssetManager  AssetManager
	Root          *Node
}

var Globals globals

type AssetManager interface {
	ById(id string) *AssetRef[any]
	MeshById(id string) MeshAssetRef
}

type AssetStatus int

const (
	AssetLoading = AssetStatus(iota)
	AssetReady
	AssetFailed
)

type AssetRef[T any] struct {
	Id     string
	Status AssetStatus
	Data   T
}

type MeshAssetRef *AssetRef[MeshAsset]

type MeshAsset struct {
	LocationIndex int
}

type MeshInstance struct {
	Mesh           MeshAssetRef
	AttributeIndex int
}

type assetManager struct {
	queue     []*AssetRef[any]
	assets    map[string]*AssetRef[any]
	signal    sync.WaitGroup
	semaphore *Semaphore
}

type Semaphore struct {
	lock    sync.Mutex
	count   uint64
	signal  chan struct{}
	waiting bool
	gate    sync.Mutex
}

func (sem *Semaphore) Post() {
	sem.lock.Lock()
	sem.count += 1

	if sem.signal == nil {
		sem.makeSignal()
	}

	if sem.waiting {
		sem.signal <- struct{}{}
	}

	sem.lock.Unlock()
}

func (sem *Semaphore) Wait() {
	sem.gate.Lock()
	sem.lock.Lock()

	if sem.count > 0 {
		sem.count--
		sem.lock.Unlock()
		sem.gate.Unlock()
		return
	}

	sem.waiting = true
	if sem.signal == nil {
		sem.makeSignal()
	}
	sem.lock.Unlock()

	for {
		<-sem.signal

		sem.lock.Lock()

		if sem.count > 0 {
			sem.count--
			sem.waiting = false
			sem.lock.Unlock()
			break
		}

		sem.lock.Unlock()
	}

	sem.gate.Unlock()
}

func (sem *Semaphore) TryWait() bool {
	if !sem.gate.TryLock() {
		return false
	}
	if !sem.lock.TryLock() {
		return false
	}

	if sem.count > 0 {
		sem.count--
		sem.lock.Unlock()
		sem.gate.Unlock()
		return true
	}

	sem.lock.Unlock()
	sem.gate.Unlock()

	return false
}

func (sem *Semaphore) makeSignal() {
	// need a buffered channel to avoid deadlocks
	sem.signal = make(chan struct{}, 1)
}

func (man *assetManager) Enqueue(asset *AssetRef[any]) {
	man.assets[asset.Id] = asset
	man.queue = append(man.queue, asset)
	man.semaphore.Post()
}

func (man *assetManager) Run() {
	for {
		man.semaphore.Wait()
		for _, asset := range man.queue {
			_ = asset
			// TODO: load asset
		}
	}
}

func (man *assetManager) ById(id string) *AssetRef[any] {
	asset, ok := man.assets[id]
	if ok {
		return asset
	}

	asset = &AssetRef[any]{
		Id: id,
	}

	man.Enqueue(asset)
	return asset
}

func (man *assetManager) MeshById(id string) MeshAssetRef {
	asset, ok := man.assets[id]
	if ok {
		if meshAsset, ok := any(asset).(MeshAssetRef); ok {
			return meshAsset
		}
		return nil
	}

	meshAsset := &AssetRef[MeshAsset]{
		Id: id,
	}

	man.assets[id] = asset
	man.queue = append(man.queue, asset)
	return meshAsset
}

type MeshLocation struct {
	BaseVertex int32
	BaseIndex  uint32
	Indices    uint32
}

type RenderManager interface {
	Draw(asset *MeshInstance)
	Submit()
}

type DrawElementsIndirectCommand struct {
	Count         uint32
	InstanceCount uint32
	FirstIndex    uint32 // The offset for the first index of the mesh in the ebo
	BaseVertex    int32  // The offset for the first vertex of the mesh in the vbo
	BaseInstance  uint32
}

type renderManager struct {
	commandBuffer UnboundBuffer
	commands      []DrawElementsIndirectCommand
	meshLocations []MeshLocation
}

func (man *renderManager) Draw(instance *MeshInstance) {
	loc := man.meshLocations[instance.Mesh.Data.LocationIndex]
	cmd := DrawElementsIndirectCommand{
		Count:         loc.Indices,
		InstanceCount: 1,
		FirstIndex:    loc.BaseIndex,
		BaseVertex:    int32(loc.BaseVertex),
		BaseInstance:  uint32(instance.AttributeIndex),
	}
	man.commands = append(man.commands, cmd)
}

func (man *renderManager) Submit() {
	if len(man.commands) == 0 {
		return
	}
	man.commandBuffer.Write(0, man.commands)
	gl.MultiDrawElementsIndirect(gl.TRIANGLES, gl.UNSIGNED_INT, gl.PtrOffset(0), int32(len(man.commands)), 0)
	man.commands = man.commands[:0]
}

func Setup(glctx GlContext) {

	Globals.GlContext = glctx
	Globals.Input = NewInputManager()
	Globals.Input.Init(glctx)
	Globals.AssetManager = &assetManager{
		assets:    make(map[string]*AssetRef[any]),
		semaphore: &Semaphore{},
	}
	commands := NewBuffer()
	commands.AllocateEmpty(int(unsafe.Sizeof(DrawElementsIndirectCommand{}))*4096, gl.DYNAMIC_STORAGE_BIT)
	Globals.RenderManager = &renderManager{commandBuffer: commands}

	Globals.Root = &Node{
		Name: "Root",
		Init: func(self *Node) interface{} {
			self.Update = func(delta float32) {
				Globals.Input.Update(Globals.GlContext)
			}

			return nil
		},
	}

	camera := Node{
		Name: "Camera",
		Init: func(self *Node) interface{} {
			cam := Camera{Fov: 90 * Deg2Rad, Speed: 1, Near: 0.1, Far: 100, Pos: mgl32.Vec3{0, 1.7, 0}}

			self.Update = func(delta float32) {
				var moveInput mgl32.Vec3
				speedMod := float32(1)
				if Globals.Input.IsKeyDown(glfw.KeyLeftShift) {
					speedMod *= 2
				}
				if Globals.Input.IsKeyDown(glfw.KeyW) {
					moveInput = moveInput.Add(mgl32.Vec3{0, 0, -1})
				}
				if Globals.Input.IsKeyDown(glfw.KeyA) {
					moveInput = moveInput.Add(mgl32.Vec3{-1, 0, 0})
				}
				if Globals.Input.IsKeyDown(glfw.KeyS) {
					moveInput = moveInput.Add(mgl32.Vec3{0, 0, 1})
				}
				if Globals.Input.IsKeyDown(glfw.KeyD) {
					moveInput = moveInput.Add(mgl32.Vec3{1, 0, 0})
				}
				moveInput.Mul(speedMod * delta)
				cam.Move(moveInput[0], moveInput[1], moveInput[2])
			}

			return struct {
				PositionFunc
			}{
				PositionFunc: func() mgl32.Vec3 { return cam.Pos },
			}
		},
	}

	Globals.Root.Add(&camera)
	Globals.Root.Add(&Node{
		Name: "Floor",
		Init: InitMeshInstance(Globals.AssetManager.MeshById("floor")),
	})

	InitializeNodesRecursive(Globals.Root)
}

func InitMeshInstance(asset MeshAssetRef) InitFunc {
	return func(self *Node) interface{} {
		transform := &TransformComp{}

		instance := &MeshInstance{
			Mesh:           asset,
			AttributeIndex: 0,
		}

		self.Draw = func() {
			if asset.Status == AssetReady {
				Globals.RenderManager.Draw(instance)
			}
		}

		return struct {
			*TransformComp
		}{
			TransformComp: transform,
		}
	}
}

func InitializeNodesRecursive(n *Node) {
	if n.Init != nil {
		n.Interface = n.Init(n)
	}
	for _, child := range n.Children {
		InitializeNodesRecursive(child)
	}
}

func UpdateNodesRecursive(n *Node, delta float32) {
	if n.Update != nil {
		n.Update(delta)
	}
	for _, child := range n.Children {
		UpdateNodesRecursive(child, delta)
	}
}

func DrawNodesRecursive(n *Node) {
	if n.Draw != nil {
		n.Draw()
	}
	for _, child := range n.Children {
		DrawNodesRecursive(child)
	}
}
