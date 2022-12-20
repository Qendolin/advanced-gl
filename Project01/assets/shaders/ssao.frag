#version 450 core
//meta:name ssao_frag

#define SAMPLE_COUNT 64

layout(location = 0) in vec2 in_uv;

layout(location = 0) out float out_color;

layout(binding = 0) uniform sampler2D g_position;
layout(binding = 1) uniform sampler2D g_normal;
layout(binding = 2) uniform sampler2D u_noise;

uniform vec3 u_samples[SAMPLE_COUNT];
uniform mat4 u_projection_mat;

uniform float u_radius;
uniform float u_exponent;

vec2 signNotZero(vec2 v) {
    return fma(step(vec2(0.0), v), vec2(2.0), vec2(-1.0));
}

// Unpacking from octahedron normals, input is the output from pack_normal_octahedron
vec3 unpackNormal(vec2 n) {
  vec3 v = vec3(n.xy, 1.0 - abs(n.x) - abs(n.y));
  if (v.z < 0) v.xy = (1.0 - abs(v.yx)) * signNotZero(v.xy);
  return normalize(v);
}

void main() {
	ivec2 co = ivec2(gl_FragCoord.xy);
	float noiseSize = textureSize(u_noise, 0).x;

	vec3 fragPos = texelFetch(g_position, co, 0).xyz;
	vec3 normal = unpackNormal(texelFetch(g_normal, co, 0).xy);
	vec3 randomVec = texture(u_noise, co / noiseSize).xyz;

	vec3 tangent = normalize(randomVec - normal * dot(randomVec, normal));
	vec3 bitangent = cross(normal, tangent);
	mat3 tbn = mat3(tangent, bitangent, normal);

	float viewDepth = fragPos.z;

	float occlusion = 0.0;
	for(int i = 0; i < SAMPLE_COUNT; i++) {
		// get sample position
		vec3 samplePos = tbn * u_samples[i]; // from tangent to view-space
		samplePos = fragPos + samplePos * u_radius;
		
		vec4 offset = vec4(samplePos, 1.0);
		offset      = u_projection_mat * offset;    // from view to clip-space
		offset.xyz /= offset.w;               // perspective divide
		offset.xyz  = offset.xyz * 0.5 + 0.5; // transform to range 0.0 - 1.0

		vec3 worldPos = texture(g_position, offset.xy).xyz;
		// Sample is in 'void'
		if(worldPos == vec3(0)) continue;

		float sampleDepth = worldPos.z;
		float rangeCheck = smoothstep(0.0, 1.0, u_radius / abs(viewDepth - sampleDepth));
		occlusion += (sampleDepth >= samplePos.z ? 1.0 : 0.0) * rangeCheck;
	}

	occlusion = 1.0 - (occlusion / SAMPLE_COUNT);
	out_color = pow(occlusion, u_exponent);
}