package main

import (
	_ "embed"
)

//go:embed assets/shaders/geometry.vert
var Res_GeometryVshSrc string

//go:embed assets/shaders/shadow.vert
var Res_ShadowVshSrc string

//go:embed assets/shaders/quad.vert
var Res_QuadVshSrc string

//go:embed assets/shaders/transform.vert
var Res_TransformVshSrc string

//go:embed assets/shaders/point_light.vert
var Res_PointLightVshSrc string

//go:embed assets/shaders/ortho_light.vert
var Res_OrthoLightVshSrc string

//go:embed assets/shaders/spot_light.vert
var Res_SpotLightVshSrc string

//go:embed assets/shaders/imgui.vert
var Res_ImguiVshSrc string

//go:embed assets/shaders/direct.vert
var Res_DirectVshSrc string

//go:embed assets/shaders/sky.vert
var Res_SkyVshSrc string

//go:embed assets/shaders/geometry.frag
var Res_GeometryFshSrc string

//go:embed assets/shaders/point_light.frag
var Res_PointLightFshSrc string

//go:embed assets/shaders/ortho_light.frag
var Res_OrthoLightFshSrc string

//go:embed assets/shaders/spot_light.frag
var Res_SpotLightFshSrc string

//go:embed assets/shaders/light_debug_volume.frag
var Res_LightDebugVolumeFshSrc string

//go:embed assets/shaders/ambient.frag
var Res_AmbientFshSrc string

//go:embed assets/shaders/postprocess.frag
var Res_PostProcessFshSrc string

//go:embed assets/shaders/color.frag
var Res_ColorFshSrc string

//go:embed assets/shaders/normal.frag
var Res_NormalFshSrc string

//go:embed assets/shaders/ssao.frag
var Res_SsaoFshSrc string

//go:embed assets/shaders/ssao_blur.frag
var Res_SsaoBlurFshSrc string

//go:embed assets/shaders/debug.frag
var Res_DebugFshSrc string

//go:embed assets/shaders/imgui.frag
var Res_ImguiFshSrc string

//go:embed assets/shaders/bloom_down.frag
var Res_BloomDownFshSrc string

//go:embed assets/shaders/bloom_up.frag
var Res_BloomUpFshSrc string

//go:embed assets/shaders/direct.frag
var Res_DirectFshSrc string

//go:embed assets/shaders/sky.frag
var Res_SkyFshSrc string

//go:embed assets/models/sponza.geo.lz4
var Res_SceneGeometry []byte

//go:embed assets/models/sponza.scn.lz4
var Res_Scene []byte

//go:embed assets/textures/uv.png
var Res_UvTexture []byte

//go:embed assets/textures/bayer16.png
var Res_BayerTexture []byte

//go:embed assets/textures/test_lut.png
var Res_TestLutTexture []byte
