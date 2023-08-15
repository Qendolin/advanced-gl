package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type InputManager interface {
	CursorDelta() mgl32.Vec2
	TimeDelta() float32
	IsKeyDown(key glfw.Key) bool
	IsMouseDown(button glfw.MouseButton) bool
	IsKeyTap(key glfw.Key) bool
	IsMouseTap(button glfw.MouseButton) bool
	Update(context *glfw.Window)
	GetMovement(forward, backward, left, right, up, down glfw.Key) mgl32.Vec3
}

type input struct {
	curr inputState
	prev inputState
}

type inputState struct {
	time         float32
	cursorPos    mgl32.Vec2
	keys         []bool
	mousebuttons []bool
}

var Input InputManager

func NewInputManager(ctx *glfw.Window) *input {
	i := &input{
		curr: inputState{
			keys:         make([]bool, glfw.KeyLast+1),
			mousebuttons: make([]bool, glfw.MouseButtonLast+1),
		},
		prev: inputState{
			keys:         make([]bool, glfw.KeyLast+1),
			mousebuttons: make([]bool, glfw.MouseButtonLast+1),
		},
	}

	i.Update(ctx)
	i.prev.cursorPos = i.curr.cursorPos
	// Make sure dTime != 0 to avoid possible errors
	i.prev.time = i.curr.time - 1./60.
	copy(i.prev.keys[:], i.curr.keys[:])
	copy(i.prev.mousebuttons[:], i.curr.mousebuttons[:])

	return i
}

func (i *input) CursorDelta() mgl32.Vec2 {
	return i.curr.cursorPos.Sub(i.prev.cursorPos)
}

func (i *input) CursorPos() mgl32.Vec2 {
	return i.curr.cursorPos
}

func (i *input) TimeDelta() float32 {
	return i.curr.time - i.prev.time
}

func (i *input) IsKeyDown(key glfw.Key) bool {
	return i.curr.keys[key]
}

func (i *input) IsKeyTap(key glfw.Key) bool {
	return i.curr.keys[key] && !i.prev.keys[key]
}

func (i *input) IsMouseDown(button glfw.MouseButton) bool {
	return i.curr.mousebuttons[button]
}

func (i *input) IsMouseTap(button glfw.MouseButton) bool {
	return i.curr.mousebuttons[button] && !i.prev.mousebuttons[button]
}

func (i *input) GetMovement(forward, backward, left, right, up, down glfw.Key) mgl32.Vec3 {
	var x, y, z float32
	if forward != 0 && i.IsKeyDown(forward) {
		z -= 1
	}
	if backward != 0 && i.IsKeyDown(backward) {
		z += 1
	}
	if left != 0 && i.IsKeyDown(left) {
		x -= 1
	}
	if right != 0 && i.IsKeyDown(right) {
		x += 1
	}
	if up != 0 && i.IsKeyDown(up) {
		y += 1
	}
	if down != 0 && i.IsKeyDown(down) {
		y -= 1
	}
	return mgl32.Vec3{x, y, z}
}

func (i *input) Update(ctx *glfw.Window) {
	keys := i.prev.keys
	mousebuttons := i.prev.mousebuttons
	i.prev = i.curr
	cursorX, cursorY := ctx.GetCursorPos()

	for key := 32; key <= int(glfw.KeyLast); key++ {
		keys[key] = ctx.GetKey(glfw.Key(key)) != glfw.Release
	}

	for button := 0; button <= int(glfw.MouseButtonLast); button++ {
		mousebuttons[button] = ctx.GetMouseButton(glfw.MouseButton(button)) != glfw.Release
	}

	i.curr = inputState{
		time:         float32(glfw.GetTime()),
		cursorPos:    mgl32.Vec2{float32(cursorX), float32(cursorY)},
		keys:         keys,
		mousebuttons: mousebuttons,
	}
}
