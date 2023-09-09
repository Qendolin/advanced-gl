#version 450 core


layout(location = 0) in vec3 in_position;
layout(location = 1) in vec2 in_uv;
layout(location = 2) in vec3 in_normal;
layout(location = 3) in vec3 in_bitangent;
layout(location = 4) in vec3 in_tangent;
layout(location = 5) in mat4 in_model_mat;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec3 out_world_position;
layout(location = 1) out vec2 out_uv;
layout(location = 2) out mat3 out_tbn;

uniform mat4 u_view_projection_mat;

void main() {
  vec4 worldPosition = in_model_mat * vec4(in_position, 1.);
  gl_Position = u_view_projection_mat * worldPosition;

  out_world_position = worldPosition.xyz;
  out_uv = in_uv;

  // FIXME: I'm not sure if I should use the inverse transpose or the regular model matrix.
  // LearnOpenGL uses the regular but normals usually require the inverse transpose
  // mat3 normalMatrix = transpose(inverse(mat3(in_model_mat)));
  mat3 normalMatrix = mat3(in_model_mat);

  vec3 T = normalize(normalMatrix * in_tangent);
  vec3 B = normalize(normalMatrix * in_bitangent);
  vec3 N = normalize(normalMatrix * in_normal);
  out_tbn = mat3(T, B, N);
}