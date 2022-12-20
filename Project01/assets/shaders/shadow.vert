#version 450 core
//meta:name shadow_vert

layout(location = 0) in vec3 in_position;
layout(location = 1) in vec3 in_normal;
layout(location = 3) in mat4 in_model_mat;

out gl_PerVertex {
  vec4 gl_Position;
};

uniform float u_bias;
uniform mat4 u_view_projection_mat;

void main() {
  gl_Position = u_view_projection_mat * in_model_mat * vec4(in_position - in_normal * u_bias, 1.);
}