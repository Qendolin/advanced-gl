#version 450 core
//meta:name ortho_light_frag

#define ENABLE_SHADOWS
#define MAX_SHADOW_MAPS 8
#define SHADOW_BIAS 0.005

layout(location = 0) out vec4 out_color;

layout(location = 0) flat in vec3 in_direction;
layout(location = 1) flat in vec3 in_color;
layout(location = 2) flat in int in_shadow_index;

layout(binding = 0) uniform sampler2D g_position;
layout(binding = 1) uniform sampler2D g_normal;
layout(binding = 2) uniform sampler2D g_albedo;
layout(binding = 3) uniform sampler2D g_depth;
layout(binding = 4) uniform sampler2D u_ao;
layout(binding = 5) uniform sampler2DShadow u_shadow_maps[MAX_SHADOW_MAPS];

uniform float u_shadow_bias;
uniform mat4 u_shadow_transforms[MAX_SHADOW_MAPS];

vec2 signNotZero(vec2 v) {
    return fma(step(vec2(0.0), v), vec2(2.0), vec2(-1.0));
}

// Unpacking from octahedron normals, input is the output from pack_normal_octahedron
vec3 unpackNormal(vec2 n) {
  vec3 v = vec3(n.xy, 1.0 - abs(n.x) - abs(n.y));
  if (v.z < 0) v.xy = (1.0 - abs(v.yx)) * signNotZero(v.xy);
  return normalize(v);
}

float calcShadow(vec3 pos, float cosTheta) {
  if(in_shadow_index == 0) return 1.0;
  mat4 transform = u_shadow_transforms[in_shadow_index-1];
  vec4 projectedPos = transform * vec4(pos, 1.0);
  projectedPos.xyz /= projectedPos.w;
  projectedPos = projectedPos * 0.5 + 0.5;

  float bias = u_shadow_bias*tan(acos(cosTheta));
  bias = clamp(bias, 0.0, 0.01);

  // GPU Gems 1 / Chapter 11.4
  vec2 texelSize = 1.0 / textureSize(u_shadow_maps[in_shadow_index-1], 0);
  vec2 offset = vec2(fract(gl_FragCoord.x * 0.5) > 0.25, fract(gl_FragCoord.y * 0.5) > 0.25); // mod
  offset.y += offset.x; // y ^= x in floating point
  if (offset.y > 1.1) offset.y = 0;
  float shadow = 0.0;
  shadow += texture(u_shadow_maps[in_shadow_index-1], vec3(projectedPos.xy + (offset + vec2(-1.5, 0.5)) * texelSize, projectedPos.z - bias));
  shadow += texture(u_shadow_maps[in_shadow_index-1], vec3(projectedPos.xy + (offset + vec2(0.5, 0.5)) * texelSize, projectedPos.z - bias));
  shadow += texture(u_shadow_maps[in_shadow_index-1], vec3(projectedPos.xy + (offset + vec2(-1.5, -1.5)) * texelSize, projectedPos.z - bias));
  shadow += texture(u_shadow_maps[in_shadow_index-1], vec3(projectedPos.xy + (offset + vec2(0.5, -1.5)) * texelSize, projectedPos.z - bias));

  return shadow * 0.25;
}

void main() {
  ivec2 co = ivec2(gl_FragCoord.xy);
  vec3 pos = texelFetch(g_position, co, 0).xyz;
  vec3 albedo = texelFetch(g_albedo, co, 0).rgb;
  vec3 normal = unpackNormal(texelFetch(g_normal, co, 0).xy);
  float ao = texelFetch(u_ao, co, 0).r;

  float cosTheta = max(dot(normal, -in_direction), 0.0);
  vec3 diffuse = cosTheta * in_color;
  float shadow = calcShadow(pos, cosTheta);

  out_color.rgb = diffuse * albedo * ao * shadow;
  out_color.a = 1;
}