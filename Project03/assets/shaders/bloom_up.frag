#version 450 core


layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_result;
layout(binding = 1) uniform sampler2D u_color;
uniform vec2 u_factor;

// 9-tap bilinear upsampler (tent filter)
// . . . . . . .
// . A . B . C .
// . . . . . . .
// . D . E . F .
// . . . . . . .
// . G . H . I .
// . . . . . . .
vec4 sampleTent(vec2 uv, vec2 texelSize) {
	vec4 a = texture(u_color, uv + vec2(-1.0, -1.0) * texelSize);
	vec4 b = texture(u_color, uv + vec2( 0.0, -1.0) * texelSize);
	vec4 c = texture(u_color, uv + vec2( 1.0, -1.0) * texelSize);
	vec4 d = texture(u_color, uv + vec2(-1.0,  0.0) * texelSize);
	vec4 e = texture(u_color, uv);
	vec4 f = texture(u_color, uv + vec2( 1.0,  0.0) * texelSize);
	vec4 g = texture(u_color, uv + vec2(-1.0,  1.0) * texelSize);
	vec4 h = texture(u_color, uv + vec2( 0.0,  1.0) * texelSize);
	vec4 i = texture(u_color, uv + vec2(-1.0,  1.0) * texelSize);

    vec4 result = 1*a + 2*b + 1*c;
    result +=     2*d + 4*e + 2*f;
    result +=     1*g + 2*h + 1*i;

    return result * (1.0 / 16.0);
}

void main() {
  vec4 bloom = sampleTent(in_uv, vec2(1) / textureSize(u_color, 0));
  out_color = u_factor.x * bloom + u_factor.y * textureLod(u_result, in_uv, 0);
}