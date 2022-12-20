#version 450 core
//meta:name bloom_down

layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_color;
uniform vec4 u_threshold;

// Better, temporally stable box filtering
// [Jimenez14] http://goo.gl/eomGso
// . . . . . . .
// . A . B . C .
// . . D . E . .
// . F . G . H .
// . . I . J . .
// . K . L . M .
// . . . . . . .
vec4 sampleBox13Tap(vec2 uv, vec2 texelSize) {
	vec4 a = texture(u_color, uv + vec2(-1.0, -1.0) * texelSize);
	vec4 b = texture(u_color, uv + vec2( 0.0, -1.0) * texelSize);
	vec4 c = texture(u_color, uv + vec2( 1.0, -1.0) * texelSize);
	vec4 d = texture(u_color, uv + vec2(-0.5, -0.5) * texelSize);
	vec4 e = texture(u_color, uv + vec2( 0.5, -0.5) * texelSize);
	vec4 f = texture(u_color, uv + vec2(-1.0,  0.0) * texelSize);
	vec4 g = texture(u_color, uv);
	vec4 h = texture(u_color, uv + vec2( 1.0,  0.0) * texelSize);
	vec4 i = texture(u_color, uv + vec2(-0.5,  0.5) * texelSize);
	vec4 j = texture(u_color, uv + vec2( 0.5,  0.5) * texelSize);
	vec4 k = texture(u_color, uv + vec2(-1.0,  1.0) * texelSize);
	vec4 l = texture(u_color, uv + vec2( 0.0,  1.0) * texelSize);
	vec4 m = texture(u_color, uv + vec2( 1.0,  1.0) * texelSize);

    vec4 result = (d + e + i + j) * 0.25 * 0.5;
    result += (a + b + g + f) * 0.25 * 0.125;
    result += (b + c + h + g) * 0.25 * 0.125;
    result += (f + g + l + k) * 0.25 * 0.125;
    result += (g + h + m + l) * 0.25 * 0.125;

    return result;
}

vec4 quadraticThreshold(vec4 color, float threshold, vec3 curve) {
	// Pixel brightness
    float br = max(color.r, max(color.g, color.b));

    // Under-threshold part: quadratic curve
    float rq = clamp(br - curve.x, 0.0, curve.y);
    rq = curve.z * rq * rq;

    // Combine and apply the brightness response curve.
    color *= max(rq, br - threshold) / max(br, 1.0e-4);

    return color;
}

void main() {
  out_color = sampleBox13Tap(in_uv, vec2(1) / textureSize(u_color, 0));
  if(u_threshold != vec4(0)) {
  	out_color = quadraticThreshold(out_color, u_threshold.x, u_threshold.yzw);
  }
}