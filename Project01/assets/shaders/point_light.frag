#version 450 core
//meta:name point_light_frag

#define CONSTANT_ATTENUATION 0.0

layout(location = 0) flat in vec3 in_light_position;
layout(location = 1) flat in vec3 in_color;
layout(location = 2) flat in float in_attenuation;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D g_position;
layout(binding = 1) uniform sampler2D g_normal;
layout(binding = 2) uniform sampler2D g_albedo;
layout(binding = 3) uniform sampler2D g_depth;
layout(binding = 4) uniform sampler2D u_ao;

vec3 calcDiffuseLight(vec3 color, vec3 dir, vec3 n, float dist, float a) {
  vec3 diffuse = max(dot(n, dir), 0.0) * color;
  float intensity = 1.0 / (CONSTANT_ATTENUATION + a * dist * dist);
  return diffuse * intensity;
}

vec3 calcLight(vec3 fragPos, vec3 fragNormal) {
  vec3 lightDir = normalize(in_light_position - fragPos);
  float lightDistance = length(in_light_position - fragPos);

  vec3 lighting = calcDiffuseLight(in_color, lightDir, fragNormal, lightDistance, in_attenuation);
  return lighting;
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