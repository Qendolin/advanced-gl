package effects

import (
	"advanced-gl/Project03/libgl"
	"advanced-gl/Project03/libutil"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type BloomEffect struct {
	Threshold     float32
	Knee          float32
	Factors       []float32
	levels        int
	width, height int
	upShader      libgl.UnboundShaderPipeline
	upTexture     libgl.UnboundTexture
	upViews       []libgl.UnboundTexture
	downShader    libgl.UnboundShaderPipeline
	downTexture   libgl.UnboundTexture
	downViews     []libgl.UnboundTexture
	sampler       libgl.UnboundSampler
	framebuffer   libgl.UnboundFramebuffer
}

func NewBloomEffect(levels int, up, down libgl.UnboundShaderPipeline) *BloomEffect {

	sampler := libgl.NewSampler()
	sampler.FilterMode(gl.LINEAR_MIPMAP_NEAREST, gl.LINEAR)
	sampler.WrapMode(gl.CLAMP_TO_EDGE, gl.CLAMP_TO_EDGE, 0)

	fbo := libgl.NewFramebuffer()
	fbo.BindTargets(0)

	factors := make([]float32, levels)
	for i := range factors {
		factors[i] = 1.0
	}

	return &BloomEffect{
		levels:      levels,
		Factors:     factors,
		upViews:     make([]libgl.UnboundTexture, levels),
		downViews:   make([]libgl.UnboundTexture, levels),
		sampler:     sampler,
		framebuffer: fbo,
		upShader:    up,
		downShader:  down,
		Threshold:   2.0,
		Knee:        0.5,
	}
}

func (effect *BloomEffect) Relase() {
	effect.upShader.Delete()
	effect.downShader.Delete()
	effect.framebuffer.Delete()
	if effect.upTexture != nil {
		for _, v := range effect.upViews {
			v.Delete()
		}
		effect.upTexture.Delete()
	}
	if effect.downTexture != nil {
		for _, v := range effect.downViews {
			v.Delete()
		}
		effect.downTexture.Delete()
	}
	effect.sampler.Delete()
}

func (effect *BloomEffect) Resize(width, height int) {
	if width == effect.width && height == effect.height {
		return
	}
	effect.width = width
	effect.height = height

	if effect.upTexture != nil {
		for _, v := range effect.upViews {
			v.Delete()
		}
		effect.upTexture.Delete()
	}
	if effect.downTexture != nil {
		for _, v := range effect.downViews {
			v.Delete()
		}
		effect.downTexture.Delete()
	}

	effect.upTexture = libgl.NewTexture(gl.TEXTURE_2D)
	effect.upTexture.Allocate(effect.levels, gl.R11F_G11F_B10F, width, height, 0)
	for i := 0; i < effect.levels; i++ {
		effect.upViews[i] = effect.upTexture.CreateView(gl.TEXTURE_2D, gl.R11F_G11F_B10F, i, i, 0, 0)
	}
	effect.downTexture = libgl.NewTexture(gl.TEXTURE_2D)
	effect.downTexture.Allocate(effect.levels, gl.R11F_G11F_B10F, width/2, height/2, 0)
	for i := 0; i < effect.levels; i++ {
		effect.downViews[i] = effect.downTexture.CreateView(gl.TEXTURE_2D, gl.R11F_G11F_B10F, i, i, 0, 0)
	}
}

func (effect *BloomEffect) Render(hdrColorTexture libgl.UnboundTexture) libgl.UnboundTexture {
	gl.PushDebugGroup(gl.DEBUG_SOURCE_APPLICATION, 999, -1, gl.Str("Draw Bloom\x00"))
	defer gl.PopDebugGroup()

	vw, vh := effect.width, effect.height

	effect.downShader.Bind()
	effect.sampler.Bind(0)
	effect.sampler.Bind(1)
	effect.framebuffer.Bind(gl.DRAW_FRAMEBUFFER)
	effect.framebuffer.AttachTextureLevel(0, effect.downTexture, 0)

	libgl.State.SetEnabled()

	hdrColorTexture.Bind(0)
	knee := effect.Threshold*effect.Knee + 1e-5
	effect.downShader.FragmentStage().SetUniform("u_threshold", mgl32.Vec4{effect.Threshold, effect.Threshold - knee, knee * 2, 0.25 / knee})
	libgl.State.Viewport(0, 0, vw/2, vh/2)
	libutil.DrawQuad()
	effect.downShader.FragmentStage().SetUniform("u_threshold", mgl32.Vec4{})

	for i := 0; i < len(effect.downViews)-1; i++ {
		effect.downViews[i].Bind(0)
		effect.framebuffer.AttachTextureLevel(0, effect.downTexture, i+1)
		libgl.State.Viewport(0, 0, vw/(4<<i), vh/(4<<i))
		libutil.DrawQuad()
	}

	effect.upShader.Bind()
	// start with last down scale level
	upView := effect.downViews[len(effect.downViews)-1]
	for i := len(effect.upViews) - 1; i >= 0; i-- {
		if i == 0 {
			libgl.State.BindTextureUnit(0, 0)
			effect.upShader.FragmentStage().SetUniform("u_factor", mgl32.Vec2{1, 1})
		} else {
			effect.downViews[i-1].Bind(0)
			if i == len(effect.upViews)-1 {
				effect.upShader.FragmentStage().SetUniform("u_factor", mgl32.Vec2{effect.Factors[i], effect.Factors[i-1]})
			} else {
				effect.upShader.FragmentStage().SetUniform("u_factor", mgl32.Vec2{1, effect.Factors[i-1]})
			}
		}
		upView.Bind(1)
		effect.framebuffer.AttachTextureLevel(0, effect.upTexture, i)
		libgl.State.Viewport(0, 0, vw/(1<<i), vh/(1<<i))
		libutil.DrawQuad()
		upView = effect.upViews[i].Bind(1)
	}

	return effect.framebuffer.GetTexture(0)
}
