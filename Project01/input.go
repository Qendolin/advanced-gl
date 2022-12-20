package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type InputManager interface {
	CursorDelta() mgl32.Vec2
	TimeDelta() float32
	IsKeyDown(key glfw.Key) bool
	IsKeyTap(key glfw.Key) bool
	Update(context Context)
	Init(ctx Context)
}

type input struct {
	curr inputState
	prev inputState
}

type inputState struct {
	time      float32
	cursorPos mgl32.Vec2
	keys      []bool
}

var Input InputManager = &input{
	curr: inputState{
		keys: make([]bool, glfw.KeyLast+1),
	},
	prev: inputState{
		keys: make([]bool, glfw.KeyLast+1),
	},
}

func (i *input) Init(ctx Context) {
	i.Update(ctx)
	i.prev.cursorPos = i.curr.cursorPos
	// Make sure dTime != 0 to avoid possible errors
	i.prev.time = i.curr.time - 1./60.
	copy(i.prev.keys[:], i.curr.keys[:])
}

func (i *input) CursorDelta() mgl32.Vec2 {
	return i.curr.cursorPos.Sub(i.prev.cursorPos)
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

func (i *input) Update(ctx Context) {
	keys := i.prev.keys
	i.prev = i.curr
	cursorX, cursorY := ctx.GetCursorPos()

	for key := 32; key <= int(glfw.KeyLast); key++ {
		keys[key] = ctx.GetKey(glfw.Key(key)) != glfw.Release
	}

	i.curr = inputState{
		time:      float32(glfw.GetTime()),
		cursorPos: mgl32.Vec2{float32(cursorX), float32(cursorY)},
		keys:      keys,
	}
}
