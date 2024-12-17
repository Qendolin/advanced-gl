#version 450 core
//meta:name sky_vert

layout(location = 0) in vec2 in_position;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec3 out_dir;

void main() {
  gl_Position = vec4(in_position, 1., 1.);
  out_dir = vec3(in_position, 0);
}