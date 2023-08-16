package main

import (
	"strings"

	"github.com/go-gl/gl/v4.5-core/gl"
)

type GlCapability uint32

const (
	DepthTest         GlCapability = gl.DEPTH_TEST
	Blend             GlCapability = gl.BLEND
	StencilTest       GlCapability = gl.STENCIL_TEST
	ScissorTest       GlCapability = gl.SCISSOR_TEST
	CullFace          GlCapability = gl.CULL_FACE
	DepthClamp        GlCapability = gl.DEPTH_CLAMP
	PolygonOffsetFill GlCapability = gl.POLYGON_OFFSET_FILL
)

type GlBlendFactor uint32

const (
	BlendZero                  GlBlendFactor = gl.ZERO
	BlendOne                   GlBlendFactor = gl.ONE
	BlendSrcColor              GlBlendFactor = gl.SRC_COLOR
	BlendOneMinusSrcColor      GlBlendFactor = gl.ONE_MINUS_SRC_COLOR
	BlendDstColor              GlBlendFactor = gl.DST_COLOR
	BlendOneMinusDstColor      GlBlendFactor = gl.ONE_MINUS_DST_COLOR
	BlendSrcAlpha              GlBlendFactor = gl.SRC_ALPHA
	BlendOneMinusSrcAlpha      GlBlendFactor = gl.ONE_MINUS_SRC_ALPHA
	BlendDstAlpha              GlBlendFactor = gl.DST_ALPHA
	BlendOneMinusDstAlpha      GlBlendFactor = gl.ONE_MINUS_DST_ALPHA
	BlendConstantColor         GlBlendFactor = gl.CONSTANT_COLOR
	BlendContstantAlpha        GlBlendFactor = gl.CONSTANT_ALPHA
	BlendOneMinusConstantAlpha GlBlendFactor = gl.ONE_MINUS_CONSTANT_ALPHA
	BlendSrcAlphaSaturate      GlBlendFactor = gl.SRC_ALPHA_SATURATE
	BlendSrcOneColor           GlBlendFactor = gl.SRC1_COLOR
	BlendOneMinusSrcOneColor   GlBlendFactor = gl.ONE_MINUS_SRC1_COLOR
	BlendSrcOneAlpha           GlBlendFactor = gl.SRC1_ALPHA
	BlendOneMinusSrcOneALpha   GlBlendFactor = gl.ONE_MINUS_SRC1_ALPHA
)

type GlBlendEquation uint32

const (
	BlendFuncAdd            GlBlendEquation = gl.FUNC_ADD
	BlendFuncSubtract       GlBlendEquation = gl.FUNC_SUBTRACT
	BlendFuncReverseSubtrct GlBlendEquation = gl.FUNC_REVERSE_SUBTRACT
	BlendMin                GlBlendEquation = gl.MIN
	BlendMax                GlBlendEquation = gl.MAX
)

type GlStencilFunc uint32

const (
	StencilFuncNever    GlStencilFunc = gl.NEVER
	StencilFuncLess     GlStencilFunc = gl.LESS
	StencilFuncLEqual   GlStencilFunc = gl.LEQUAL
	StencilFuncGreater  GlStencilFunc = gl.GREATER
	StencilFuncGEqual   GlStencilFunc = gl.GEQUAL
	StencilFuncEqual    GlStencilFunc = gl.EQUAL
	StencilFuncNotEqual GlStencilFunc = gl.NOTEQUAL
	StencilFuncAlways   GlStencilFunc = gl.ALWAYS
)

type GlStencilOp uint32

const (
	StencilOpKeep          GlStencilOp = gl.KEEP
	StencilOpZero          GlStencilOp = gl.ZERO
	StencilOpReplace       GlStencilOp = gl.REPLACE
	StencilOpIncrement     GlStencilOp = gl.INCR
	StencilOpIncrementWrap GlStencilOp = gl.INCR_WRAP
	StencilOpDecrement     GlStencilOp = gl.DECR
	StencilOpDecrementWrap GlStencilOp = gl.DECR_WRAP
	StencilOpInvert        GlStencilOp = gl.INVERT
)

type GlDepthFunc uint32

const (
	DepthFuncNever    GlDepthFunc = gl.NEVER
	DepthFuncLess     GlDepthFunc = gl.LESS
	DepthFuncLEqual   GlDepthFunc = gl.LEQUAL
	DepthFuncGreater  GlDepthFunc = gl.GREATER
	DepthFuncGEqual   GlDepthFunc = gl.GEQUAL
	DepthFuncNotEqual GlDepthFunc = gl.NOTEQUAL
	DepthFuncEqual    GlDepthFunc = gl.EQUAL
	DepthFuncAlways   GlDepthFunc = gl.ALWAYS
)

type GlStateManager struct {
	Caps                                                            map[GlCapability]bool
	TextureUnits, SamplerUnits                                      []uint32
	DrawFramebuffer, ReadFramebuffer, Renderbuffer                  uint32
	ArrayBuffer, DrawIndirectBuffer, ElementArrayBuffer             uint32
	TextureBuffer, UniformBuffer, TransformFeedbackBuffer           uint32
	CopyReadBuffer, CopyWriteBuffer, ShaderStorageBuffer            uint32
	ProgramPipeline, VertexArray                                    uint32
	ActiveTextureUnit                                               int
	ViewportRect, ScissorRect                                       [4]int
	BlendRGBFactorSrc, BlendAlphaFactorSrc                          GlBlendFactor
	BlendRGBFactorDst, BlendAlphaFactorDst                          GlBlendFactor
	BlendEquationRGB, BlendEquationAlpha                            GlBlendEquation
	StencilFrontFuncFn, StencilBackFuncFn                           GlStencilFunc
	StencilFrontFuncMask, StencilBackFuncMask                       uint32
	StencilFrontFuncRef, StencilBackFuncRef                         int32
	StencilFrontOpSFail, StencilFrontOpDPFail, StencilFrontOpDPPass GlStencilOp
	StencilBackOpSFail, StencilBackOpDPFail, StencilBackOpDPPass    GlStencilOp
	StencilFrontMask, StencilBackMask                               uint32
	DepthFuncFn                                                     GlDepthFunc
	DepthWriteMask                                                  bool
	CullFaceMask                                                    uint32
	ClearColorRGBA                                                  [4]float32
	PolygonOffsets                                                  [3]float32
	PolygonModeFront, PolygonModeBack                               uint32
}

var GlEnv *GlEnvironment

type GlEnvironment struct {
	Vendor                     string
	UseIntelTextureBindingFix  bool
	IntelTextureBindingTargets map[uint32]uint32
	Features                   GlFeatures
}

type GlFeatures struct {
	MaxTextureMaxAnisotropy float32
}

const (
	VendorIntel   = "intel"
	VendorNvidia  = "ati"
	VendorAmd     = "ati"
	VendorUnknown = "unknown"
)

func GetGlEnv() *GlEnvironment {
	vendor := string(gl.GoStr(gl.GetString(gl.VENDOR)))
	vendor = strings.ToLower(strings.TrimSuffix(vendor, "\x00"))
	if strings.Contains(vendor, "intel") {
		vendor = VendorIntel
	} else if strings.Contains(vendor, "nvidia ") {
		vendor = VendorNvidia
	} else if strings.Contains(vendor, "ati ") {
		vendor = VendorAmd
	} else {
		vendor = VendorUnknown
	}

	features := GlFeatures{}

	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &features.MaxTextureMaxAnisotropy)

	return &GlEnvironment{
		Vendor:                     vendor,
		UseIntelTextureBindingFix:  vendor == VendorIntel,
		IntelTextureBindingTargets: map[uint32]uint32{},
		Features:                   features,
	}
}

var GlState *GlStateManager

func NewGlStateManager() *GlStateManager {
	return &GlStateManager{
		Caps:         map[GlCapability]bool{},
		TextureUnits: make([]uint32, 32),
		SamplerUnits: make([]uint32, 32),
	}
}

func (s *GlStateManager) Enable(cap GlCapability) {
	if s.Caps[cap] {
		return
	}
	gl.Enable(uint32(cap))
	s.Caps[cap] = true
}

func (s *GlStateManager) SetEnabled(caps ...GlCapability) {
	diff := map[GlCapability]bool{}
	for c, v := range s.Caps {
		if v {
			diff[c] = false
		}
	}
	for _, c := range caps {
		if !diff[c] || !s.Caps[c] {
			diff[c] = true
		}
	}
	for c, v := range diff {
		if v {
			s.Enable(c)
		} else {
			s.Disable(c)
		}
	}
}

func (s *GlStateManager) Disable(cap GlCapability) {
	if !s.Caps[cap] {
		return
	}
	gl.Disable(uint32(cap))
	s.Caps[cap] = false
}

func (s *GlStateManager) CullFront() {
	if s.CullFaceMask == gl.FRONT {
		return
	}
	gl.CullFace(gl.FRONT)
	s.CullFaceMask = gl.FRONT
}

func (s *GlStateManager) CullBack() {
	if s.CullFaceMask == gl.BACK {
		return
	}
	gl.CullFace(gl.BACK)
	s.CullFaceMask = gl.BACK
}

func (s *GlStateManager) BlendFunc(sfactor, dfactor GlBlendFactor) {
	if s.BlendAlphaFactorSrc == sfactor && s.BlendRGBFactorSrc == sfactor && s.BlendAlphaFactorDst == dfactor && s.BlendRGBFactorDst == dfactor {
		return
	}
	gl.BlendFunc(uint32(sfactor), uint32(dfactor))
	s.BlendAlphaFactorSrc = sfactor
	s.BlendRGBFactorSrc = sfactor
	s.BlendAlphaFactorDst = dfactor
	s.BlendRGBFactorDst = dfactor
}

func (s *GlStateManager) BlendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha GlBlendFactor) {
	if s.BlendAlphaFactorSrc == srcAlpha && s.BlendRGBFactorSrc == srcRGB && s.BlendAlphaFactorDst == dstAlpha && s.BlendRGBFactorDst == dstRGB {
		return
	}
	gl.BlendFuncSeparate(uint32(srcRGB), uint32(dstRGB), uint32(srcAlpha), uint32(dstAlpha))
	s.BlendAlphaFactorSrc = srcAlpha
	s.BlendRGBFactorSrc = srcRGB
	s.BlendAlphaFactorDst = dstAlpha
	s.BlendRGBFactorDst = dstRGB
}

func (s *GlStateManager) BlendEquation(mode GlBlendEquation) {
	if s.BlendEquationAlpha == mode && s.BlendEquationRGB == mode {
		return
	}
	gl.BlendEquation(uint32(mode))
	s.BlendEquationAlpha = mode
	s.BlendEquationRGB = mode
}

func (s *GlStateManager) BlendEquationSeparate(modeRGB, modeAlpha GlBlendEquation) {
	if s.BlendEquationAlpha == modeAlpha && s.BlendEquationRGB == modeRGB {
		return
	}
	gl.BlendEquationSeparate(uint32(modeRGB), uint32(modeAlpha))
	s.BlendEquationAlpha = modeAlpha
	s.BlendEquationRGB = modeRGB
}

func (s *GlStateManager) StencilFunc(fn GlStencilFunc, ref int32, mask uint32) {
	if s.StencilFrontFuncFn == fn && s.StencilFrontFuncRef == ref && s.StencilFrontFuncMask == mask && s.StencilBackFuncFn == fn && s.StencilBackFuncRef == ref && s.StencilBackFuncMask == mask {
		return
	}
	gl.StencilFunc(uint32(fn), ref, mask)
	s.StencilFrontFuncFn = fn
	s.StencilFrontFuncRef = ref
	s.StencilFrontFuncMask = mask
	s.StencilBackFuncFn = fn
	s.StencilBackFuncRef = ref
	s.StencilBackFuncMask = mask
}

func (s *GlStateManager) StencilFuncFront(fn GlStencilFunc, ref int32, mask uint32) {
	if s.StencilFrontFuncFn == fn && s.StencilFrontFuncRef == ref && s.StencilFrontFuncMask == mask {
		return
	}
	gl.StencilFuncSeparate(gl.FRONT, uint32(fn), ref, mask)
	s.StencilFrontFuncFn = fn
	s.StencilFrontFuncRef = ref
	s.StencilFrontFuncMask = mask
}

func (s *GlStateManager) StencilFuncBack(fn GlStencilFunc, ref int32, mask uint32) {
	if s.StencilBackFuncFn == fn && s.StencilBackFuncRef == ref && s.StencilBackFuncMask == mask {
		return
	}
	gl.StencilFuncSeparate(gl.BACK, uint32(fn), ref, mask)
	s.StencilBackFuncFn = fn
	s.StencilBackFuncRef = ref
	s.StencilBackFuncMask = mask
}

func (s *GlStateManager) StencilOp(sfail, dpfail, dppass GlStencilOp) {
	if s.StencilFrontOpSFail == sfail && s.StencilFrontOpDPFail == dpfail && s.StencilFrontOpDPPass == dppass && s.StencilBackOpSFail == sfail && s.StencilBackOpDPFail == dpfail && s.StencilBackOpDPPass == dppass {
		return
	}
	gl.StencilOp(uint32(sfail), uint32(dpfail), uint32(dppass))
	s.StencilFrontOpSFail = sfail
	s.StencilFrontOpDPFail = dpfail
	s.StencilFrontOpDPPass = dppass
	s.StencilBackOpSFail = sfail
	s.StencilBackOpDPFail = dpfail
	s.StencilBackOpDPPass = dppass
}

func (s *GlStateManager) StencilOpFront(sfail, dpfail, dppass GlStencilOp) {
	if s.StencilFrontOpSFail == sfail && s.StencilFrontOpDPFail == dpfail && s.StencilFrontOpDPPass == dppass {
		return
	}
	gl.StencilOpSeparate(gl.FRONT, uint32(sfail), uint32(dpfail), uint32(dppass))
	s.StencilFrontOpSFail = sfail
	s.StencilFrontOpDPFail = dpfail
	s.StencilFrontOpDPPass = dppass
}

func (s *GlStateManager) StencilOpBack(sfail, dpfail, dppass GlStencilOp) {
	if s.StencilBackOpSFail == sfail && s.StencilBackOpDPFail == dpfail && s.StencilBackOpDPPass == dppass {
		return
	}
	gl.StencilOpSeparate(gl.BACK, uint32(sfail), uint32(dpfail), uint32(dppass))
	s.StencilBackOpSFail = sfail
	s.StencilBackOpDPFail = dpfail
	s.StencilBackOpDPPass = dppass
}

func (s *GlStateManager) StencilMask(mask uint32) {
	if s.StencilFrontMask == mask && s.StencilBackMask == mask {
		return
	}
	gl.StencilMask(mask)
	s.StencilFrontMask = mask
	s.StencilBackMask = mask
}

func (s *GlStateManager) StencilMaskFront(mask uint32) {
	if s.StencilFrontMask == mask {
		return
	}
	gl.StencilMaskSeparate(gl.FRONT, mask)
	s.StencilFrontMask = mask
}

func (s *GlStateManager) StencilMaskBack(mask uint32) {
	if s.StencilBackMask == mask {
		return
	}
	gl.StencilMaskSeparate(gl.BACK, mask)
	s.StencilBackMask = mask
}

func (s *GlStateManager) DepthFunc(fn GlDepthFunc) {
	if s.DepthFuncFn == fn {
		return
	}
	gl.DepthFunc(uint32(fn))
	s.DepthFuncFn = fn
}

func (s *GlStateManager) DepthMask(flag bool) {
	if s.DepthWriteMask == flag {
		return
	}
	gl.DepthMask(flag)
	s.DepthWriteMask = flag
}

func (s *GlStateManager) PolygonMode(face, mode uint32) {
	if face == gl.FRONT_AND_BACK && (s.PolygonModeFront != mode || s.PolygonModeBack != mode) {
		gl.PolygonMode(face, mode)
		s.PolygonModeBack = mode
		s.PolygonModeFront = mode
	} else if face == gl.FRONT && s.PolygonModeFront != mode {
		gl.PolygonMode(face, mode)
		s.PolygonModeFront = mode
	} else if face == gl.BACK && s.PolygonModeBack != mode {
		gl.PolygonMode(face, mode)
		s.PolygonModeBack = mode
	}
}

func (s *GlStateManager) PolygonOffset(factor, units float32) {
	if s.PolygonOffsets[0] == factor && s.PolygonOffsets[1] == units && s.PolygonOffsets[2] == 0 {
		return
	}
	gl.PolygonOffset(factor, units)
	s.PolygonOffsets = [3]float32{factor, units, 0}
}

func (s *GlStateManager) PolygonOffsetClamp(factor, units, clamp float32) {
	if s.PolygonOffsets[0] == factor && s.PolygonOffsets[1] == units && s.PolygonOffsets[2] == clamp {
		return
	}
	gl.PolygonOffsetClamp(factor, units, clamp)
	s.PolygonOffsets = [3]float32{factor, units, clamp}
}

func (s *GlStateManager) BindTextureUnit(unit int, texture uint32) {
	if s.TextureUnits[unit] == texture {
		return
	}
	if GlEnv.UseIntelTextureBindingFix {
		s.ActiveTextue(unit)
		if texture == 0 {
			// Not sure if this is without issues
			s.TextureUnits[unit] = texture
			return
		}
		gl.BindTexture(GlEnv.IntelTextureBindingTargets[texture], texture)
		s.TextureUnits[unit] = texture
		return
	}
	gl.BindTextureUnit(uint32(unit), texture)
	s.TextureUnits[unit] = texture
}

func (s *GlStateManager) BindTexture(target uint32, texture uint32) {
	if s.TextureUnits[s.ActiveTextureUnit] == texture {
		return
	}
	gl.BindTexture(target, texture)
	s.TextureUnits[s.ActiveTextureUnit] = texture
}

func (s *GlStateManager) ActiveTextue(unit int) {
	if s.ActiveTextureUnit == unit {
		return
	}
	gl.ActiveTexture(gl.TEXTURE0 + uint32(unit))
	s.ActiveTextureUnit = unit
}

func (s *GlStateManager) BindSampler(unit int, sampler uint32) {
	if s.SamplerUnits[unit] == sampler {
		return
	}
	gl.BindSampler(uint32(unit), sampler)
	s.SamplerUnits[unit] = sampler
}

func (s *GlStateManager) BindBuffer(target uint32, buffer uint32) {
	switch target {
	case gl.ARRAY_BUFFER:
		s.BindArrayBuffer(buffer)
	case gl.ELEMENT_ARRAY_BUFFER:
		s.BindElementArrayBuffer(buffer)
	case gl.DRAW_INDIRECT_BUFFER:
		s.BindDrawIndirectBuffer(buffer)
	case gl.COPY_READ_BUFFER:
		s.BindCopyReadBuffer(buffer)
	case gl.COPY_WRITE_BUFFER:
		s.BindCopyWriteBuffer(buffer)
	case gl.UNIFORM_BUFFER:
		s.BindUniformBuffer(buffer)
	case gl.SHADER_STORAGE_BUFFER:
		s.BindShaderStorageBuffer(buffer)
	case gl.TEXTURE_BUFFER:
		s.BindTextureBuffer(buffer)
	case gl.TRANSFORM_FEEDBACK_BUFFER:
		s.BindTransformFeedbackBuffer(buffer)
	default:
		gl.BindBuffer(target, buffer)
	}
}

func (s *GlStateManager) BindArrayBuffer(buffer uint32) {
	if s.ArrayBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer)
	s.ArrayBuffer = buffer
}

func (s *GlStateManager) BindElementArrayBuffer(buffer uint32) {
	if s.ElementArrayBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, buffer)
	s.ElementArrayBuffer = buffer
}

func (s *GlStateManager) BindDrawIndirectBuffer(buffer uint32) {
	if s.DrawIndirectBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, buffer)
	s.DrawIndirectBuffer = buffer
}

func (s *GlStateManager) BindCopyReadBuffer(buffer uint32) {
	if s.CopyReadBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.COPY_READ_BUFFER, buffer)
	s.CopyReadBuffer = buffer
}

func (s *GlStateManager) BindCopyWriteBuffer(buffer uint32) {
	if s.CopyWriteBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, buffer)
	s.CopyWriteBuffer = buffer
}

func (s *GlStateManager) BindShaderStorageBuffer(buffer uint32) {
	if s.ShaderStorageBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, buffer)
	s.ShaderStorageBuffer = buffer
}

func (s *GlStateManager) BindUniformBuffer(buffer uint32) {
	if s.UniformBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.UNIFORM_BUFFER, buffer)
	s.UniformBuffer = buffer
}

func (s *GlStateManager) BindTextureBuffer(buffer uint32) {
	if s.TextureBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.TEXTURE_BUFFER, buffer)
	s.TextureBuffer = buffer
}

func (s *GlStateManager) BindTransformFeedbackBuffer(buffer uint32) {
	if s.TransformFeedbackBuffer == buffer {
		return
	}
	gl.BindBuffer(gl.TRANSFORM_FEEDBACK_BUFFER, buffer)
	s.TransformFeedbackBuffer = buffer
}

func (s *GlStateManager) BindFramebuffer(target, framebuffer uint32) {
	if target == gl.DRAW_FRAMEBUFFER {
		s.BindDrawFramebuffer(framebuffer)
	} else if target == gl.READ_FRAMEBUFFER {
		s.BindReadFramebuffer(framebuffer)
	} else {
		if framebuffer == s.DrawFramebuffer && framebuffer == s.ReadFramebuffer {
			return
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, framebuffer)
		s.DrawFramebuffer = framebuffer
		s.ReadFramebuffer = framebuffer
	}
}

func (s *GlStateManager) BindDrawFramebuffer(framebuffer uint32) {
	if s.DrawFramebuffer == framebuffer {
		return
	}
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, framebuffer)
	s.DrawFramebuffer = framebuffer
}

func (s *GlStateManager) BindReadFramebuffer(framebuffer uint32) {
	if s.ReadFramebuffer == framebuffer {
		return
	}
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, framebuffer)
	s.ReadFramebuffer = framebuffer
}

func (s *GlStateManager) BindRenderbuffer(renderbuffer uint32) {
	if s.Renderbuffer == renderbuffer {
		return
	}
	gl.BindRenderbuffer(gl.RENDERBUFFER, renderbuffer)
	s.Renderbuffer = renderbuffer
}

func (s *GlStateManager) BindProgramPipeline(pipeline uint32) {
	if s.ProgramPipeline == pipeline {
		return
	}
	gl.BindProgramPipeline(pipeline)
	s.ProgramPipeline = pipeline
}

func (s *GlStateManager) BindVertexArray(array uint32) {
	if s.VertexArray == array {
		return
	}
	gl.BindVertexArray(array)
	s.VertexArray = array
}

func (s *GlStateManager) Viewport(x, y, w, h int) {
	if s.ViewportRect[0] == x && s.ViewportRect[1] == y && s.ViewportRect[2] == w && s.ViewportRect[3] == h {
		return
	}
	gl.Viewport(int32(x), int32(y), int32(w), int32(h))
	s.ViewportRect = [4]int{x, y, w, h}
}

func (s *GlStateManager) Scissor(x, y, w, h int) {
	if s.ScissorRect[0] == x && s.ScissorRect[1] == y && s.ScissorRect[2] == w && s.ScissorRect[3] == h {
		return
	}
	gl.Scissor(int32(x), int32(y), int32(w), int32(h))
	s.ScissorRect = [4]int{x, y, w, h}
}

func (s *GlStateManager) ClearColor(r, g, b, a float32) {
	if s.ClearColorRGBA[0] == r && s.ClearColorRGBA[1] == g && s.ClearColorRGBA[2] == b && s.ClearColorRGBA[3] == a {
		return
	}
	gl.ClearColor(r, g, b, a)
	s.ClearColorRGBA = [4]float32{r, g, b, a}
}
