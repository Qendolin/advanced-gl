#version 450 core
//meta:name point_light_frag

#define ENABLE_SHADOWS
#define MAX_SHADOW_MAPS 8
#define SHADOW_BIAS 0.005

layout(location = 0) flat in vec3 in_light_position;
layout(location = 1) flat in vec3 in_color;
layout(location = 2) flat in float in_attenuation;
layout(location = 3) flat in vec3 in_direction;
layout(location = 4) flat in vec2 in_angles;
layout(location = 5) flat in int in_shadow_index;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D g_position;
layout(binding = 1) uniform sampler2D g_normal;
layout(binding = 2) uniform sampler2D g_albedo;
layout(binding = 3) uniform sampler2D g_depth;
layout(binding = 4) uniform sampler2D u_ao;
layout(binding = 5) uniform sampler2DShadow u_shadow_maps[MAX_SHADOW_MAPS];

uniform float u_shadow_bias;
uniform mat4 u_shadow_transforms[MAX_SHADOW_MAPS];

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

vec3 calcDiffuseLight(vec3 color, float cosTheta, float dist, float a) {
  vec3 diffuse = cosTheta * color;
  float intensity = 1.0 / (1.0 + a * dist * dist);
  return diffuse * intensity;
}

float calcConeFalloff(vec3 coneDir, vec3 lightDir, vec2 coneAngles) {
  float theta = dot(-coneDir, lightDir);
  return smoothstep(0.0, 1.0, (theta - coneAngles.x) * coneAngles.y);
}

vec3 calcLight(vec3 fragPos, vec3 fragNormal) {
  vec3 lightDir = normalize(in_light_position - fragPos);
  float lightDistance = length(in_light_position - fragPos);
  float cosTheta = max(dot(lightDir, fragNormal), 0.0);

  vec3 coneDir = in_direction;
  float coneFalloff = calcConeFalloff(coneDir, lightDir, in_angles);
  float shadow = calcShadow(fragPos, cosTheta);
  if(coneFalloff == 0.0 || shadow == 0.0) return vec3(0);

  vec3 lighting = calcDiffuseLight(in_color, cosTheta, lightDistance, in_attenuation);
  return lighting * coneFalloff * shadow;
}

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
  vec3 albedo = texelFetch(g_albedo, co, 0).rgb;
  vec3 pos = texelFetch(g_position, co, 0).xyz;
  vec3 normal = unpackNormal(texelFetch(g_normal, co, 0).xy);

  vec3 lightColor = calcLight(pos, normal);

  float ao = texelFetch(u_ao, co, 0).r;

  out_color.rgb = lightColor * albedo * ao;
  out_color.a = 1;
}