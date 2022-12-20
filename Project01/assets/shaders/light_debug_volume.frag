#version 450 core
//meta:name light_debug_volume_frag

layout(location = 0) flat in vec3 in_world_position;
layout(location = 1) flat in vec3 in_color;
layout(location = 2) flat in float in_attenuation;

layout(location = 0) out vec4 out_color;

void main() {
  out_color.rgb = normalize(in_color.rgb);
  out_color.a = 1;
}