#version 450 core
//meta:name albedo_frag

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D g_albedo;
layout(binding = 1) uniform sampler2D u_ao;

uniform vec3 u_ambient_light;
uniform float u_min_light;

void main() {
  ivec2 co = ivec2(gl_FragCoord.xy);
  vec3 albedo = texelFetch(g_albedo, co, 0).rgb;
  float ao = texelFetch(u_ao, co, 0).r;

  out_color.rgb = u_ambient_light * albedo * mix(u_min_light, 1, clamp(ao, 0, 1));
  out_color.a = 1;
}