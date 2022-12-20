package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	i "github.com/inkyblackness/imgui-go/v4"
)

type uiscene struct {
	initialized   bool
	cursorVisible bool

	wireframe           bool
	enableDirectDraw    bool
	accuratePerformance bool
	bufferViewIndex     int
	bufferViews         []string

	enableShadows                                            bool
	shadowBiasDraw                                           float32
	shadowBiasSample                                         float32
	shadowOffsetFactor, shadowOffsetUnits, shadowOffsetClamp float32
	ambientLight                                             mgl32.Vec3
	ambientMinLight                                          float32
	enablePointLights                                        bool
	enableSpotLights                                         bool
	enableOrthoLights                                        bool
	enableAmbientLight                                       bool
	vizualizePointLights                                     bool
	vizualizeSpotLights                                      bool
	orthoLights                                              []SceneOrthoLight
	spotLights                                               []SceneSpotLight
	pointLights                                              []ScenePointLight

	applyColorMapping bool
	applyDithering    bool
	applyBloom        bool

	enableSsao        bool
	ssaoRadius        float32
	ssaoExponent      float32
	ssaoSamples       int32
	ssaoSamplesMin    float32
	ssaoSamplesCurve  float32
	ssaoEdgeThreshold float32

	enableBloom    bool
	bloomThreshold float32
	bloomKnee      float32
	bloomFactor    float32

	enableSky bool

	frameTimer       float32
	frameTimes       []float32
	frameTime        float32
	frameTimeIndex   int
	averageTimer     float32
	averageCounter   int
	averageTimes     []float32
	averageTimeIndex int
	averageTime      float32
	minTimer         float32
	minTimes         []float32
	minTime          float32
	maxTimer         float32
	maxTimes         []float32
	maxTime          float32

	texturePreviewIndex int
	shaderEditIndex     int
	shaderEditStage     int
	shaderEditStageBit  int
	shaderEditCode      string
	shaderEditError     error
}

type ScenePointLight struct {
	Position    mgl32.Vec3
	Color       mgl32.Vec3
	Brightness  float32
	Attenuation float32
	Radius      float32
}

type SceneOrthoLight struct {
	Azimuth     float32
	Elevation   float32
	Color       mgl32.Vec3
	Brightness  float32
	ShadowIndex int
}

type SceneSpotLight struct {
	Position    mgl32.Vec3
	Azimuth     float32
	Elevation   float32
	InnerAngle  float32
	OuterAngle  float32
	Color       mgl32.Vec3
	Radius      float32
	Attenuation float32
	Brightness  float32
	ShadowIndex int
}

var uiDefaults = uiscene{
	enableDirectDraw:    true,
	accuratePerformance: true,

	bufferViews: []string{"None", "Final", "Position", "Normal", "Albedo", "Depth", "AO", "Bloom", "Stencil"},

	applyDithering: true,
	applyBloom:     true,

	enableSsao:        true,
	ssaoRadius:        1.0,
	ssaoExponent:      3.0,
	ssaoSamples:       24,
	ssaoSamplesMin:    0.05,
	ssaoSamplesCurve:  1.5,
	ssaoEdgeThreshold: 0.05,

	enableBloom:    true,
	bloomThreshold: 2.0,
	bloomKnee:      0.5,
	bloomFactor:    0.3,

	enableSky: true,

	frameTimes:   make([]float32, 128),
	averageTimes: make([]float32, 32),
	minTimes:     make([]float32, 32),
	maxTimes:     make([]float32, 32),

	enableShadows:  true,
	shadowBiasDraw: 0.035,
	// shadowBiasSample:   0.0013,
	shadowBiasSample:   0.000,
	shadowOffsetFactor: 1.4, shadowOffsetUnits: 4.5, shadowOffsetClamp: 0.05,
	ambientLight:       mgl32.Vec3{0.063, 0.061, 0.189},
	ambientMinLight:    0.2,
	enablePointLights:  true,
	enableSpotLights:   true,
	enableOrthoLights:  true,
	enableAmbientLight: true,

	orthoLights: []SceneOrthoLight{},
	spotLights:  []SceneSpotLight{},
	pointLights: []ScenePointLight{},

	texturePreviewIndex: 0,
}

var ui *uiscene

func init() {
	resetUi()
}

func resetUi() {
	if ui == nil {
		ui = &uiDefaults
	}
	old := *ui
	defaults := uiDefaults
	ui = &defaults
	ui.cursorVisible = old.cursorVisible
	ui.frameTimeIndex = old.frameTimeIndex
	ui.averageTimeIndex = old.averageTimeIndex
	ui.orthoLights = append([]SceneOrthoLight{}, ui.orthoLights...)
	ui.spotLights = append([]SceneSpotLight{}, ui.spotLights...)
	ui.pointLights = append([]ScenePointLight{}, ui.pointLights...)
}

func DrawUi() {

	SliderDir := func(azimuth, elevation *float32) bool {
		i.BeginTable("dir_input", 2)
		i.TableNextColumn()
		dirty := false
		if i.SliderFloatV("Az", azimuth, 0, 360, "%.1f °", i.SliderFlagsNone) {
			dirty = true
		}
		i.TableNextColumn()
		if i.SliderFloatV("El", elevation, -90, 90, "%.1f °", i.SliderFlagsNone) {
			dirty = true
		}
		i.EndTable()
		return dirty
	}

	DirToVec := func(azimuth, elevation float32) mgl32.Vec3 {
		az := (azimuth - 180) * Deg2Rad
		el := elevation * Deg2Rad
		return mgl32.Vec3{
			float32(math.Sin(float64(az)) * math.Cos(float64(el))),
			float32(math.Sin(float64(el))),
			float32(math.Cos(float64(az)) * math.Cos(float64(el))),
		}.Normalize()
	}

	i.NewFrame()
	if Input.IsKeyTap(glfw.KeyLeftAlt) {
		ui.cursorVisible = !ui.cursorVisible
		if ui.cursorVisible {
			glfw.GetCurrentContext().SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			i.CurrentIO().SetConfigFlags(i.ConfigFlagsNone)
		} else {
			glfw.GetCurrentContext().SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			i.CurrentIO().SetConfigFlags(i.ConfigFlagsNoMouse)

		}
	}
	i.Begin("main_window")
	i.Text("Press Left Alt to release the cursor.")
	i.Button("Reset Values")
	if i.IsItemClicked() {
		resetUi()
	}
	i.Checkbox("Wireframe", &ui.wireframe)
	i.SameLine()
	i.Checkbox("Direct Draw", &ui.enableDirectDraw)
	i.SameLine()
	i.Checkbox("Accurate Performance", &ui.accuratePerformance)
	if i.Button("Dump Buffers (Ctrl+F11)") {
		DumpFramebuffers()
	}
	if Input.IsKeyDown(glfw.KeyLeftControl) && Input.IsKeyTap(glfw.KeyF11) {
		DumpFramebuffers()
	}
	if i.BeginCombo("Buffer View", ui.bufferViews[ui.bufferViewIndex]) {
		for j, v := range ui.bufferViews {
			if i.Selectable(v) {
				ui.bufferViewIndex = j
			}
		}
		i.EndCombo()
	}

	if i.CollapsingHeaderV("SSAO", i.TreeNodeFlagsDefaultOpen) || !ui.initialized {
		i.PushID("ssao")
		i.Indent()
		samplesDirty := !ui.initialized
		shaderDirty := !ui.initialized

		i.Checkbox("Enable", &ui.enableSsao)
		i.SliderFloat("Radius", &ui.ssaoRadius, 0.1, 5)
		i.SliderFloat("Exponent", &ui.ssaoExponent, 0.5, 5)
		i.SliderInt("Samples", &ui.ssaoSamples, 8, 64)
		if i.IsItemEdited() {
			shaderDirty = true
		}
		i.SliderFloat("Curve Exponent", &ui.ssaoSamplesCurve, 0.5, 4)
		if i.IsItemEdited() {
			samplesDirty = true
		}
		i.SliderFloat("Curve Min", &ui.ssaoSamplesMin, 0.01, 1)
		if i.IsItemEdited() {
			samplesDirty = true
		}
		i.SliderFloat("Blur Edge", &ui.ssaoEdgeThreshold, 0.0, 0.1)

		if shaderDirty {
			if err := s.ssaoShader.Get(gl.FRAGMENT_SHADER).CompileWith(map[string]string{
				"SAMPLE_COUNT": fmt.Sprintf("%d", ui.ssaoSamples),
			}); err != nil {
				log.Panic(err)
			}
			s.ssaoShader.ReAttach(gl.FRAGMENT_SHADER_BIT)
			samplesDirty = true
		}
		if samplesDirty {
			fsh := s.ssaoShader.Get(gl.FRAGMENT_SHADER)
			for j, v := range GenerateSsaoSamples(int(ui.ssaoSamples), ui.ssaoSamplesMin, ui.ssaoSamplesCurve) {
				fsh.SetUniformIndexed("u_samples", j, v)
			}
		}
		i.Unindent()
		i.PopID()
	}
	if i.CollapsingHeaderV("Bloom", i.TreeNodeFlagsDefaultOpen) || !ui.initialized {
		i.PushID("bloom")
		i.Indent()
		i.Checkbox("Enable", &ui.enableBloom)
		i.SliderFloat("Threshold", &ui.bloomThreshold, 0.5, 10.0)
		i.SliderFloat("Knee", &ui.bloomKnee, 0.0, 1.0)
		i.SliderFloat("Factor", &ui.bloomFactor, 0.0, 1.0)
		i.Unindent()
		i.PopID()
	}
	if i.CollapsingHeaderV("Sky", i.TreeNodeFlagsDefaultOpen) || !ui.initialized {
		i.PushID("sky")
		i.Indent()
		i.Checkbox("Enable", &ui.enableSky)
		i.Unindent()
		i.PopID()
	}
	if i.CollapsingHeaderV("PostFx", i.TreeNodeFlagsDefaultOpen) || !ui.initialized {
		i.PushID("postfx")
		i.Indent()
		dirty := !ui.initialized
		if i.Checkbox("Color Mapping", &ui.applyColorMapping) {
			dirty = true
		}
		if i.Checkbox("Dithering", &ui.applyDithering) {
			dirty = true
		}
		if i.Checkbox("Bloom", &ui.applyBloom) {
			dirty = true
		}

		if dirty {
			if err := s.postShader.Get(gl.FRAGMENT_SHADER).CompileWith(map[string]string{
				"ENABLE_COLOR_MAPPING": fmt.Sprintf("%v", ui.applyColorMapping),
				"ENABLE_DITHERING":     fmt.Sprintf("%v", ui.applyDithering),
				"ENABLE_BLOOM":         fmt.Sprintf("%v", ui.applyBloom),
			}); err != nil {
				log.Panic(err)
			}
			s.postShader.ReAttach(gl.FRAGMENT_SHADER_BIT)
		}
		i.Unindent()
		i.PopID()
	}

	if i.CollapsingHeaderV("Light", i.TreeNodeFlagsDefaultOpen) || !ui.initialized {
		i.PushID("lights")
		i.Indent()

		shaderDirty := !ui.initialized
		if i.Checkbox("Shadows", &ui.enableShadows) {
			shaderDirty = true
		}
		i.SliderFloatV("Normal Bias", &ui.shadowBiasDraw, 0.0, 0.1, "%.5f", i.SliderFlagsNone)
		i.SliderFloatV("Depth Bias", &ui.shadowBiasSample, 0.0, 0.1, "%.5f", i.SliderFlagsLogarithmic)
		i.SliderFloatV("Offset Factor", &ui.shadowOffsetFactor, -10, 10, "%.2f", i.SliderFlagsNone)
		i.SliderFloatV("Offset Units", &ui.shadowOffsetUnits, -10, 10, "%.2f", i.SliderFlagsNone)
		i.SliderFloatV("Offset Clamp", &ui.shadowOffsetClamp, -10, 10, "%.2f", i.SliderFlagsNone)

		i.Separator()

		i.Checkbox("Point Lights", &ui.enablePointLights)
		i.Checkbox("Visualize##viz_point_lights", &ui.vizualizePointLights)
		i.PushID("pointlights")
		for j := range ui.pointLights {
			lp := &ui.pointLights[j]
			dirty := !ui.initialized
			if i.TreeNodef("Light #%d", j+1) {
				if i.SliderFloat("Att", &lp.Attenuation, 0, 100) {
					dirty = true
				}
				if i.DragFloat3("Position", (*[3]float32)(&lp.Position)) {
					dirty = true
				}
				if i.ColorEdit3V("Color", (*[3]float32)(&lp.Color), i.ColorEditFlagsFloat) {
					dirty = true
				}
				if i.SliderFloat("Brightness", &lp.Brightness, 0, 10) {
					dirty = true
				}
				i.TreePop()
			}
			if dirty {
				l := &s.pointLights[j]
				l.Position = lp.Position
				l.Attenuation = lp.Attenuation
				l.Color = lp.Color.Mul(lp.Brightness)
				l.Radius = LightAttenuationRadius(l.Color, l.Attenuation)
				s.pointLightBuffer.WriteIndex(j, l)
			}
		}
		if shaderDirty {
			if err := s.spotLightShader.Get(gl.FRAGMENT_SHADER).CompileWith(map[string]string{
				"ENABLE_SHADOWS": fmt.Sprintf("%v", ui.enableShadows),
			}); err != nil {
				log.Panic(err)
			}
			s.orthoLightShader.ReAttach(gl.FRAGMENT_SHADER_BIT)
			if err := s.orthoLightShader.Get(gl.FRAGMENT_SHADER).CompileWith(map[string]string{
				"ENABLE_SHADOWS": fmt.Sprintf("%v", ui.enableShadows),
			}); err != nil {
				log.Panic(err)
			}
			s.orthoLightShader.ReAttach(gl.FRAGMENT_SHADER_BIT)
		}
		i.PopID()

		i.Separator()

		i.Checkbox("Spot Lights", &ui.enableSpotLights)
		i.Checkbox("Visualize##viz_spot_lights", &ui.vizualizeSpotLights)
		i.PushID("spotlights")
		for j := range ui.spotLights {
			lp := &ui.spotLights[j]
			dirty := !ui.initialized
			if i.TreeNodef("Light #%d", j+1) {

				i.BeginTable("attributes", 2)
				i.TableNextColumn()
				if i.SliderFloatV("Inner", &lp.InnerAngle, 0, 179, "%.1f °", i.SliderFlagsNone) {
					dirty = true
				}
				i.TableNextColumn()
				if i.SliderFloatV("Outer", &lp.OuterAngle, 0, 45, "%.1f °", i.SliderFlagsNone) {
					dirty = true
				}
				i.TableNextColumn()
				if i.SliderFloat("Dist", &lp.Radius, 0, 100) {
					dirty = true
				}
				i.TableNextColumn()
				if i.SliderFloat("Att", &lp.Attenuation, 0, 100) {
					dirty = true
				}
				i.EndTable()
				if i.DragFloat3("Position", (*[3]float32)(&lp.Position)) {
					dirty = true
				}
				if SliderDir(&lp.Azimuth, &lp.Elevation) {
					dirty = true
				}
				if i.ColorEdit3V("Color", (*[3]float32)(&lp.Color), i.ColorEditFlagsFloat) {
					dirty = true
				}
				if i.SliderFloatV("Brightness", &lp.Brightness, 0, 1000, "%.1f", i.SliderFlagsNone) {
					dirty = true
				}
				i.TreePop()
			}
			if dirty {
				l := &s.spotLights[j]
				innerAngle := float64(lp.InnerAngle * Deg2Rad)
				outerAngle := innerAngle + float64(lp.OuterAngle*Deg2Rad)
				gamma := float32(math.Cos(outerAngle))
				epsilon := float32(1. / (math.Cos(innerAngle) - math.Cos(outerAngle)))
				l.Angles = mgl32.Vec2{gamma, epsilon}
				l.Color = lp.Color.Mul(lp.Brightness)
				l.Direction = DirToVec(lp.Azimuth, lp.Elevation)
				l.ShadowIndex = int32(lp.ShadowIndex)
				l.Attenuation = lp.Attenuation
				l.Position = lp.Position
				l.Radius = LightAttenuationRadius(l.Color, l.Attenuation)
				s.spotLightBuffer.WriteIndex(j, l)
			}
		}
		i.PopID()
		i.Separator()

		i.Checkbox("Ambient Light", &ui.enableAmbientLight)
		i.ColorEdit3V("Ambient Color", (*[3]float32)(&ui.ambientLight), i.ColorEditFlagsFloat)
		i.SliderFloat("Ambient Minimum", &ui.ambientMinLight, 0.0, 1.0)

		i.Separator()

		i.Checkbox("Ortho Lights", &ui.enableOrthoLights)
		i.PushID("ortholights")
		for j := range ui.orthoLights {
			lp := &ui.orthoLights[j]
			dirty := !ui.initialized
			if i.TreeNodef("Light #%d", j+1) {
				if SliderDir(&lp.Azimuth, &lp.Elevation) {
					dirty = true
				}
				if i.ColorEdit3V("Color", (*[3]float32)(&lp.Color), i.ColorEditFlagsFloat) {
					dirty = true
				}
				if i.SliderFloat("Brightness", &lp.Brightness, 0, 10) {
					dirty = true
				}
				i.TreePop()
			}
			if dirty {
				l := &s.orthoLights[j]
				l.Direction = DirToVec(lp.Azimuth, lp.Elevation).Mul(-1)
				l.Color = lp.Color.Mul(lp.Brightness)
				l.ShadowIndex = int32(lp.ShadowIndex)
				s.orthoLightBuffer.WriteIndex(j, l)
			}
		}
		i.PopID()
		i.Unindent()
		i.PopID()
	}
	i.End()

	i.Begin("perf_window")
	drawList := i.WindowDrawList()
	time := float32(glfw.GetTime())
	frameTime := time - ui.frameTimer
	i.Text(fmt.Sprintf("%4.1f ms / %4d fps", ui.frameTime*1000, (int)(1/ui.frameTime)))
	ui.frameTimes[ui.frameTimeIndex] = frameTime * 1000
	ui.frameTimer = time
	ui.frameTime = frameTime
	ui.frameTimeIndex = (ui.frameTimeIndex + 1) % len(ui.frameTimes)
	point := i.CursorScreenPos().Plus(i.Vec2{X: 0, Y: 48})
	i.PlotHistogramV("", ui.frameTimes, ui.frameTimeIndex, fmt.Sprintf("Frame Time (ms) - %4.1f ms", ui.frameTime*1000), 0, 1000./30., i.Vec2{X: 256, Y: 96})
	drawList.AddLine(point, point.Plus(i.Vec2{X: 256}), i.Packed(color.RGBA{128, 128, 0, 255}))

	ui.averageCounter++
	if frameTime <= ui.minTimer {
		ui.minTimer = frameTime
	}
	if frameTime >= ui.maxTimer {
		ui.maxTimer = frameTime
	}
	if time-ui.averageTimer >= 1 {
		average := (time - ui.averageTimer) / float32(ui.averageCounter)
		ui.averageTime = average
		ui.averageTimes[ui.averageTimeIndex] = average * 1000
		ui.minTime = ui.minTimer
		ui.minTimes[ui.averageTimeIndex] = ui.minTimer * 1000
		ui.maxTime = ui.maxTimer
		ui.maxTimes[ui.averageTimeIndex] = ui.maxTimer * 1000
		ui.averageTimeIndex = (ui.averageTimeIndex + 1) % len(ui.averageTimes)
		ui.averageCounter = 0
		ui.averageTimer = time
		ui.minTimer = float32(math.Inf(1))
		ui.maxTimer = float32(math.Inf(-1))
	}
	point = i.CursorScreenPos().Plus(i.Vec2{X: 0, Y: 48})
	i.PlotLinesV("", ui.averageTimes, ui.averageTimeIndex, fmt.Sprintf("Avg. Frame Time - %4.1f ms", ui.averageTime*1000), 0, 1000./30., i.Vec2{X: 256, Y: 96})
	drawList.AddLine(point, point.Plus(i.Vec2{X: 256}), i.Packed(color.RGBA{128, 128, 0, 255}))
	point = i.CursorScreenPos().Plus(i.Vec2{X: 0, Y: 48})
	i.PlotLinesV("", ui.minTimes, ui.averageTimeIndex, fmt.Sprintf("Min. Frame Time - %4.1f ms", ui.minTime*1000), 0, 1000./30., i.Vec2{X: 256, Y: 96})
	drawList.AddLine(point, point.Plus(i.Vec2{X: 256}), i.Packed(color.RGBA{128, 128, 0, 255}))
	point = i.CursorScreenPos().Plus(i.Vec2{X: 0, Y: 48})
	i.PlotLinesV("", ui.maxTimes, ui.averageTimeIndex, fmt.Sprintf("Max. Frame Time - %4.1f ms", ui.maxTime*1000), 0, 1000./30., i.Vec2{X: 256, Y: 96})
	drawList.AddLine(point, point.Plus(i.Vec2{X: 256}), i.Packed(color.RGBA{128, 128, 0, 255}))

	i.End()

	i.Begin("texture_window")

	textureNames := []string{"None"}
	textureIds := []uint32{0}

	for j, sc := range s.shadowCasters {
		textureNames = append(textureNames, fmt.Sprintf("Shadow Map %d", j))
		textureIds = append(textureIds, sc.ShadowMap.GetTexture(gl.DEPTH_ATTACHMENT).Id())
	}

	if i.BeginCombo("Texture", textureNames[ui.texturePreviewIndex]) {
		for j, name := range textureNames {
			if i.Selectable(fmt.Sprintf("%v (#%d)", name, textureIds[j])) {
				ui.texturePreviewIndex = j
			}
		}
		i.EndCombo()
	}
	if ui.texturePreviewIndex > 0 {
		i.ImageV(i.TextureID(textureIds[ui.texturePreviewIndex]), i.Vec2{X: 256, Y: 256}, i.Vec2{X: 0, Y: 1}, i.Vec2{X: 1, Y: 0}, i.Vec4{X: 1, Y: 1, Z: 1, W: 1}, i.Vec4{})
	}

	i.End()

	i.Begin("shader_window")

	shaderNames := []string{"None", "SSAO Blur"}
	shaders := []UnboundShaderPipeline{nil, s.ssaoBlurShader}
	if i.BeginCombo("Shader", shaderNames[ui.shaderEditIndex]) {
		changed := false
		for j, shader := range shaders {
			name := shaderNames[j]
			if j == 0 {
				if i.Selectable(name) {
					ui.shaderEditIndex = 0
				}
				continue
			}
			fsh := shader.Get(gl.FRAGMENT_SHADER)
			vsh := shader.Get(gl.VERTEX_SHADER)
			if fsh != nil && i.Selectable(fmt.Sprintf("%v / %v", name, fsh.Name())) {
				ui.shaderEditIndex = j
				ui.shaderEditStage = gl.FRAGMENT_SHADER
				ui.shaderEditStageBit = gl.FRAGMENT_SHADER_BIT
				changed = true
			}
			if vsh != nil && i.Selectable(fmt.Sprintf("%v / %v", name, vsh.Name())) {
				ui.shaderEditIndex = j
				ui.shaderEditStage = gl.VERTEX_SHADER
				ui.shaderEditStageBit = gl.VERTEX_SHADER_BIT
				changed = true
			}
		}
		if changed {
			ui.shaderEditCode = shaders[ui.shaderEditIndex].Get(ui.shaderEditStage).Source()
		}
		i.EndCombo()
	}
	if ui.shaderEditIndex != 0 {
		pipeline := shaders[ui.shaderEditIndex]
		if i.Button("Compile") {
			program := NewShader(ui.shaderEditCode, ui.shaderEditStage)
			err := program.Compile()
			ui.shaderEditError = err
			if err == nil {
				pipeline.Attach(program, ui.shaderEditStageBit)
			}
		}
		if ui.shaderEditError != nil {
			i.SameLine()
			i.Text(ui.shaderEditError.Error())
		}
		i.PushItemWidth(i.ContentRegionAvail().X)
		i.InputTextMultilineV("Code", &ui.shaderEditCode, i.Vec2{X: 0, Y: i.ContentRegionAvail().Y}, 0, nil)
		i.PopItemWidth()
	}
	i.End()

	ui.initialized = true
}
