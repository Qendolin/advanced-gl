#version 450 core
//meta:name ortho_light_vert

layout(location = 0) in vec2 in_position;
layout(location = 1) in vec3 in_direction;
layout(location = 2) in vec3 in_color;
layout(location = 3) in int in_shadow_index;

layout(location = 0) flat out vec3 out_direction;
layout(location = 1) flat out vec3 out_color;
layout(location = 2) flat out int out_shadow_index;

out gl_PerVertex {
  vec4 gl_Position;
};

uniform mat4 u_view_mat = mat4(1.0);

void main() {
  gl_Position = vec4(in_position, 1., 1.);
  out_direction = mat3(u_view_mat) * normalize(in_direction);
  out_color = in_color;
  out_shadow_index = in_shadow_index;
}