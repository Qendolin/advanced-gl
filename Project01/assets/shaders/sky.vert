#version 450 core
//meta:name sky_vert

layout(location = 0) in vec2 in_position;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec3 out_dir;

uniform mat4 u_projection_mat;
uniform mat4 u_view_mat;

void main() {
  gl_Position = vec4(in_position, 1., 1.);
  vec4 dir =  vec4(vec3(in_position, 0), 1);
  out_dir = dir.xyz;

  // vec4 dir = inverse(u_view_mat) * inverse(u_projection_mat) * vec4(in_position, 0.0, 1.0);
  // dir.xyz /= dir.w;
  // out_dir = dir.xyz;
}