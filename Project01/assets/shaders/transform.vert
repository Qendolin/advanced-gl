#version 450 core
//meta:name transform_vert

layout(location = 0) in vec3 in_position;

out gl_PerVertex {
  vec4 gl_Position;
};

uniform mat4 u_transform_mat = mat4(1.0);

void main() {
  gl_Position = u_transform_mat * vec4(in_position, 1.);
}