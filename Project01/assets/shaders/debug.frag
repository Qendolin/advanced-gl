#version 450 core
//meta:name debug_frag

layout(location = 0) in vec2 in_uv;
layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_final;
layout(binding = 1) uniform sampler2D g_position;
layout(binding = 2) uniform sampler2D g_normal;
layout(binding = 3) uniform sampler2D g_albedo;
layout(binding = 4) uniform sampler2D g_depth;
layout(binding = 5) uniform sampler2D u_ao;
layout(binding = 6) uniform sampler2D u_bloom;
layout(binding = 7) uniform usampler2D g_stencil;

uniform int u_sampler;

vec2 signNotZero(vec2 v) {
    return fma(step(vec2(0.0), v), vec2(2.0), vec2(-1.0));
}

// Unpacking from octahedron normals, input is the output from pack_normal_octahedron
vec3 unpackNormal(vec2 n) {
  vec3 v = vec3(n.xy, 1.0 - abs(n.x) - abs(n.y));
  if (v.z < 0) v.xy = (1.0 - abs(v.yx)) * signNotZero(v.xy);
  return normalize(v);
}

float linearize_depth(float d, float n, float f)
{
	float ndc = d * 2.0 - 1.0; 
    return (2.0 * n * f) / (f + n - ndc * (f - n));
}

void main() {
	vec3 result = vec3(0.5);
	if (u_sampler == 0) {
		result = texture(u_final, in_uv).rgb;
	} else if (u_sampler == 1) {
		vec3 pos = texture(g_position, in_uv).xyz;
		result = abs(pos) * vec3(0.1, 0.1, 0);
	} else if (u_sampler == 2) {
		result = unpackNormal(texture(g_normal, in_uv).xy) / 2. + 0.5;
	} else if (u_sampler == 3) {
		result = texture(g_albedo, in_uv).rgb;
	} else if (u_sampler == 4) {
		result = vec3(linearize_depth(texture(g_depth, in_uv).x, 0.1, 1000.) / 1000.);
	} else if (u_sampler == 5) {
		result = texture(u_ao, in_uv).rrr;
	} else if (u_sampler == 6) {
		result = texture(u_bloom, in_uv).rgb;
	} else if (u_sampler == 7) {
		result = vec3(texture(g_stencil, in_uv).r) / 255. - (1 - texture(u_ao, in_uv).r) / 10.;
	}
	result = pow(result, vec3(1./2.4));
	out_color = vec4(result, 1.0);
}