package main

import (
	_ "embed"
)

//go:embed assets/shaders/imgui.vert
var Res_ImguiVshSrc string

//go:embed assets/shaders/imgui.frag
var Res_ImguiFshSrc string
