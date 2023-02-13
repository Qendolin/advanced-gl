#version 450 core
//meta:name geometry_frag

// #define TRANSPARENT_PASS

#ifndef TRANSPARENT_PASS

layout(early_fragment_tests) in;

#endif

layout(location = 0) in vec3 in_view_normal;
layout(location = 1) in vec3 in_position;
layout(location = 2) in vec2 in_uv;

layout(location = 0) out vec3 g_position;
layout(location = 1) out vec2 g_normal;
layout(location = 2) out vec3 g_albedo;

layout(location = 0) uniform sampler2D u_texture;
layout(location = 1) uniform sampler2D u_normal;

// Octahedral Normal Packing
// Credit: https://discourse.panda3d.org/t/glsl-octahedral-normal-packing/15233
// For each component of v, returns -1 if the component is < 0, else 1
vec2 signNotZero(vec2 v) {
    return fma(step(vec2(0.0), v), vec2(2.0), vec2(-1.0));
}

// Packs a 3-component normal to 2 channels using octahedron normals
vec2 packNormal(vec3 n) {
  n.xy /= dot(abs(n), vec3(1));
  return mix(n.xy, (1.0 - abs(n.yx)) * signNotZero(n.xy), step(n.z, 0.0));
}

void main() {
  vec4 albedo = texture(u_texture, in_uv);

#ifdef TRANSPARENT_PASS

  if(albedo.a <= 0.5) discard;

#endif

  g_position = in_position;
  g_normal.xy = packNormal(normalize(in_view_normal));
  g_albedo = albedo.rgb;
}