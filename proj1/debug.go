package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/go-gl/gl/v4.5-core/gl"
)

type TextureDump struct {
	Name          string
	Width, Height uint32
	Data          []float32
}

func DumpFramebuffers() error {
	fbos := map[string]UnboundFramebuffer{
		"bloom":    s.bloomBuffer,
		"geometry": s.gBuffer,
		"post":     s.postBuffer,
		"ssao":     s.ssaoBuffer,
	}
	if _, err := os.Stat("./dump/"); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir("./dump/", 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for name, fbo := range fbos {
		buffers := DumpFramebufferColorAttachments(fbo)
		if ok, buf := DumpFramebufferDepthAttachment(fbo); ok {
			buffers = append(buffers, buf)
		}
		if ok, buf := DumpFramebufferStencilAttachment(fbo); ok {
			buffers = append(buffers, buf)
		}
		for _, tex := range buffers {
			filename := fmt.Sprintf("./dump/%v_%v.f32", name, tex.Name)
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			binary.Write(file, binary.LittleEndian, tex.Width)
			binary.Write(file, binary.LittleEndian, tex.Height)
			binary.Write(file, binary.LittleEndian, tex.Data)
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func DumpFramebufferColorAttachments(fbo UnboundFramebuffer) []*TextureDump {
	fbo.Bind(gl.READ_FRAMEBUFFER)
	var maxAttachments int32
	gl.GetIntegerv(gl.MAX_COLOR_ATTACHMENTS, &maxAttachments)
	bufs := []*TextureDump{}
	for i := 0; i < int(maxAttachments); i++ {
		attachment := uint32(gl.COLOR_ATTACHMENT0 + i)
		var attachmentType, attachmentId int32
		gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), attachment, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_TYPE, &attachmentType)
		if attachmentType != gl.TEXTURE {
			continue
		}

		gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), attachment, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_NAME, &attachmentId)
		if attachmentId <= 0 {
			continue
		}
		var width, height int32
		gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_WIDTH, &width)
		gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_HEIGHT, &height)
		buf := make([]float32, 4*width*height)
		gl.NamedFramebufferReadBuffer(fbo.Id(), attachment)
		// gl.GetTextureImage(uint32(attachmentId), 0, gl.RGBA, gl.FLOAT, int32(len(buf)*4), Pointer(buf))
		gl.ReadnPixels(0, 0, width, height, gl.RGBA, gl.FLOAT, int32(len(buf)*4), Pointer(buf))

		bufs = append(bufs, &TextureDump{
			Width:  uint32(width),
			Height: uint32(height),
			Data:   buf,
			Name:   fmt.Sprintf("color-attachment-%d", i),
		})
	}
	return bufs
}

func DumpFramebufferDepthAttachment(fbo UnboundFramebuffer) (bool, *TextureDump) {
	var attachmentType, attachmentId int32
	gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), gl.DEPTH_ATTACHMENT, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_TYPE, &attachmentType)
	if attachmentType != gl.TEXTURE {
		return false, nil
	}

	gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), gl.DEPTH_ATTACHMENT, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_NAME, &attachmentId)
	if attachmentId <= 0 {
		return false, nil
	}
	var width, height int32
	gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_WIDTH, &width)
	gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_HEIGHT, &height)
	depth := make([]float32, width*height)
	// gl.GetTextureImage(uint32(attachmentId), 0, gl.DEPTH_COMPONENT, gl.FLOAT, int32(len(buf)*4), Pointer(buf))
	gl.ReadnPixels(0, 0, width, height, gl.DEPTH_COMPONENT, gl.FLOAT, int32(len(depth)*4), Pointer(depth))
	buf := make([]float32, width*height*4)
	for j, v := range depth {
		buf[j*4+0] = v
		buf[j*4+1] = v
		buf[j*4+2] = v
		buf[j*4+3] = v
	}
	return true, &TextureDump{
		Width:  uint32(width),
		Height: uint32(height),
		Data:   buf,
		Name:   "depth-attachment",
	}
}

func DumpFramebufferStencilAttachment(fbo UnboundFramebuffer) (bool, *TextureDump) {
	var attachmentType, attachmentId int32
	gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), gl.STENCIL_ATTACHMENT, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_TYPE, &attachmentType)
	if attachmentType != gl.TEXTURE {
		return false, nil
	}

	gl.GetNamedFramebufferAttachmentParameteriv(fbo.Id(), gl.STENCIL_ATTACHMENT, gl.FRAMEBUFFER_ATTACHMENT_OBJECT_NAME, &attachmentId)
	if attachmentId <= 0 {
		return false, nil
	}
	var width, height int32
	gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_WIDTH, &width)
	gl.GetTextureLevelParameteriv(uint32(attachmentId), 0, gl.TEXTURE_HEIGHT, &height)
	stencil := make([]float32, width*height)
	// gl.GetTextureImage(uint32(attachmentId), 0, gl.STENCIL_INDEX, gl.FLOAT, int32(len(buf)*4), Pointer(buf))
	gl.ReadnPixels(0, 0, width, height, gl.STENCIL_INDEX, gl.FLOAT, int32(len(stencil)*4), Pointer(stencil))
	buf := make([]float32, width*height*4)
	for j, v := range stencil {
		buf[j*4+0] = v
		buf[j*4+1] = v
		buf[j*4+2] = v
		buf[j*4+3] = v
	}
	return true, &TextureDump{
		Width:  uint32(width),
		Height: uint32(height),
		Data:   buf,
		Name:   "stencil-attachment",
	}
}
