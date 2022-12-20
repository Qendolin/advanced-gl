#version 450 core
//meta:name color_frag

layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

uniform vec3 u_color;

void main() {
  out_color = vec4(u_color, 1.);
}