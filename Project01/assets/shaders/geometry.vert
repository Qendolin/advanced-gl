#version 450 core
//meta:name geometry_vert

layout(location = 0) in vec3 in_position;
layout(location = 1) in vec3 in_normal;
layout(location = 2) in vec2 in_uv;
layout(location = 3) in mat4 in_model_mat;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec3 out_view_normal;
layout(location = 1) out vec3 out_position;
layout(location = 2) out vec2 out_uv;

uniform mat4 u_view_projection_mat;
uniform mat4 u_view_mat;

void main() {
  gl_Position = u_view_projection_mat * in_model_mat * vec4(in_position, 1.);

  mat3 normalMatrix = transpose(inverse(mat3(in_model_mat)));
  out_view_normal = (transpose(inverse(u_view_mat)) * vec4(normalMatrix * in_normal, 1.)).xyz;
  out_position = (u_view_mat * in_model_mat * vec4(in_position, 1.)).xyz;
  out_uv = in_uv;
}