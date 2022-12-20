#version 450 core
//meta:name quad_vert

layout(location = 0) in vec2 in_position;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec2 out_uv;

void main() {
  gl_Position = vec4(in_position, 1., 1.);
  out_uv = in_position.xy * 0.5 + 0.5;
}