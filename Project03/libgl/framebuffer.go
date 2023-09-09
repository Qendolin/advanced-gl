package libgl

import (
	"fmt"

	"github.com/go-gl/gl/v4.5-core/gl"
)

const MaxAttachments = 8

type framebuffer struct {
	glId          uint32
	textures      []UnboundTexture
	renderbuffers []UnboundRenderbuffer
}

type UnboundFramebuffer interface {
	LabeledGlObject
	Id() uint32
	// target must be GL_DRAW_FRAMEBUFFER, GL_READ_FRAMEBUFFER or GL_FRAMEBUFFER
	Bind(target uint32) BoundFramebuffer
	// target must be GL_DRAW_FRAMEBUFFER, GL_READ_FRAMEBUFFER or GL_FRAMEBUFFER
	Check(target uint32) error
	GetTexture(index int) UnboundTexture
	GetRenderbuffer(index int) UnboundRenderbuffer
	AttachTexture(index int, texture UnboundTexture)
	AttachTextureLevel(index int, texture UnboundTexture, level int)
	AttachTextureLayer(index int, texture UnboundTexture, layer int)
	AttachTextureLayerLevel(index int, texture UnboundTexture, layer, level int)
	AttachRenderbuffer(index int, renderbuffer UnboundRenderbuffer)
	BindTargets(attachments ...int)
	Delete()
}

type BoundFramebuffer interface {
	UnboundFramebuffer
}

// FIXME: https://forums.developer.nvidia.com/t/framebuffer-incomplete-when-attaching-color-buffers-of-different-sizes-with-dsa/211550
func NewFramebuffer() UnboundFramebuffer {
	var id uint32
	gl.CreateFramebuffers(1, &id)

	return &framebuffer{
		glId:          id,
		textures:      make([]UnboundTexture, MaxAttachments+2),
		renderbuffers: make([]UnboundRenderbuffer, MaxAttachments+2),
	}
}

func (fb *framebuffer) Id() uint32 {
	return fb.glId
}

func (fb *framebuffer) SetDebugLabel(label string) {
	setObjectLabel(gl.FRAMEBUFFER, fb.glId, label)
}

func (fb *framebuffer) BindTargets(indices ...int) {
	attachments := make([]uint32, len(indices))
	for i, v := range indices {
		if v <= MaxAttachments {
			attachments[i] = uint32(gl.COLOR_ATTACHMENT0 + v)
		} else {
			attachments[i] = uint32(v)
		}
	}
	n := len(indices)
	gl.NamedFramebufferDrawBuffers(fb.glId, int32(n), &attachments[0])
}

func (fb *framebuffer) Check(target uint32) error {
	status := gl.CheckNamedFramebufferStatus(fb.glId, target)
	switch status {
	case gl.FRAMEBUFFER_COMPLETE:
		return nil
	case gl.FRAMEBUFFER_INCOMPLETE_ATTACHMENT:
		return fmt.Errorf("an attachment is framebuffer incomplete (GL_FRAMEBUFFER_INCOMPLETE_ATTACHMENT)")
	case gl.FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT:
		return fmt.Errorf("the framebuffer has no attachments (GL_FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT)")
	case gl.FRAMEBUFFER_INCOMPLETE_DRAW_BUFFER:
		return fmt.Errorf("the object type of a draw attachment is none (GL_FRAMEBUFFER_INCOMPLETE_DRAW_BUFFER)")
	case gl.FRAMEBUFFER_INCOMPLETE_READ_BUFFER:
		return fmt.Errorf("the object type of the read attachment is none (GL_FRAMEBUFFER_INCOMPLETE_READ_BUFFER)")
	case gl.FRAMEBUFFER_UNSUPPORTED:
		return fmt.Errorf("the combination of internal formats of the attachments is not supported (GL_FRAMEBUFFER_UNSUPPORTED)")
	case gl.FRAMEBUFFER_INCOMPLETE_MULTISAMPLE:
		return fmt.Errorf("the attachments have different sampling (GL_FRAMEBUFFER_INCOMPLETE_MULTISAMPLE)")
	case gl.FRAMEBUFFER_INCOMPLETE_LAYER_TARGETS:
		return fmt.Errorf("FRAMEBUFFER_INCOMPLETE_LAYER_TARGETS")
	}
	return fmt.Errorf("unknown framebuffer status: %X", status)
}

func (fb *framebuffer) Bind(target uint32) BoundFramebuffer {
	State.BindFramebuffer(target, fb.glId)
	return BoundFramebuffer(fb)
}

func (fb *framebuffer) AttachTexture(index int, texture UnboundTexture) {
	fb.AttachTextureLevel(index, texture, 0)
}

func (fb *framebuffer) AttachTextureLevel(index int, texture UnboundTexture, level int) {
	fb.textures[fb.mapAttachmentIndex(index)] = texture
	if index <= MaxAttachments {
		index += gl.COLOR_ATTACHMENT0
	}
	gl.NamedFramebufferTexture(fb.glId, uint32(index), texture.Id(), int32(level))
}

func (fb *framebuffer) AttachTextureLayer(index int, texture UnboundTexture, layer int) {
	fb.AttachTextureLayerLevel(index, texture, layer, 0)
}

func (fb *framebuffer) AttachTextureLayerLevel(index int, texture UnboundTexture, layer, level int) {
	fb.textures[fb.mapAttachmentIndex(index)] = texture
	if index <= MaxAttachments {
		index += gl.COLOR_ATTACHMENT0
	}
	// https://community.intel.com/t5/Graphics/glNamedFramebufferTextureLayer-rejects-cubemaps-of-any-kind/td-p/1167643
	if texture.Type() == gl.TEXTURE_CUBE_MAP && GlEnv.UseIntelCubemaDsaFix {
		prevDraw := State.DrawFramebuffer
		prevRead := State.DrawFramebuffer
		fb.Bind(gl.FRAMEBUFFER)
		gl.BindTexture(gl.TEXTURE_CUBE_MAP, texture.Id())
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, uint32(index), uint32(gl.TEXTURE_CUBE_MAP_POSITIVE_X+layer), texture.Id(), int32(level))
		State.BindDrawFramebuffer(prevDraw)
		State.BindReadFramebuffer(prevRead)
	} else {
		gl.NamedFramebufferTextureLayer(fb.glId, uint32(index), texture.Id(), int32(level), int32(layer))
	}
}

func (fb *framebuffer) AttachRenderbuffer(index int, renderbuffer UnboundRenderbuffer) {
	fb.renderbuffers[fb.mapAttachmentIndex(index)] = renderbuffer
	if index <= MaxAttachments {
		index += gl.COLOR_ATTACHMENT0
	}
	gl.NamedFramebufferRenderbuffer(fb.glId, uint32(index), gl.RENDERBUFFER, renderbuffer.Id())
}

func (fb *framebuffer) mapAttachmentIndex(index int) int {
	if index == gl.DEPTH_ATTACHMENT || index == gl.DEPTH_STENCIL_ATTACHMENT {
		return 0
	} else if index == gl.STENCIL_ATTACHMENT {
		return 1
	}
	return index + 2
}

func (fb *framebuffer) GetTexture(index int) UnboundTexture {
	return fb.textures[fb.mapAttachmentIndex(index)]
}

func (fb *framebuffer) GetRenderbuffer(index int) UnboundRenderbuffer {
	return fb.renderbuffers[fb.mapAttachmentIndex(index)]
}

func (fb *framebuffer) Delete() {
	gl.DeleteFramebuffers(1, &fb.glId)
	fb.glId = 0
}

type renderbuffer struct {
	glId uint32
}

type UnboundRenderbuffer interface {
	LabeledGlObject
	Id() uint32
	Bind() BoundRenderbuffer
	Allocate(internalFormat uint32, width, height int)
	AllocateMS(internalFormat uint32, width, height, samples int)
}

type BoundRenderbuffer interface {
	UnboundRenderbuffer
}

func NewRenderbuffer() UnboundRenderbuffer {
	var id uint32
	gl.CreateRenderbuffers(1, &id)
	return &renderbuffer{
		glId: id,
	}
}

func (rb *renderbuffer) Id() uint32 {
	return rb.glId
}

func (rb *renderbuffer) SetDebugLabel(label string) {
	setObjectLabel(gl.RENDERBUFFER, rb.glId, label)
}

func (rb *renderbuffer) Bind() BoundRenderbuffer {
	State.BindRenderbuffer(rb.glId)
	return BoundRenderbuffer(rb)
}

func (rb *renderbuffer) Allocate(internalFormat uint32, width, height int) {
	gl.NamedRenderbufferStorage(rb.glId, internalFormat, int32(width), int32(height))
}

func (rb *renderbuffer) AllocateMS(internalFormat uint32, width, height, samples int) {
	gl.NamedRenderbufferStorageMultisample(rb.glId, int32(samples), internalFormat, int32(width), int32(height))
}

func (rb *renderbuffer) Delete() {
	gl.DeleteRenderbuffers(1, &rb.glId)
	rb.glId = 0
}
