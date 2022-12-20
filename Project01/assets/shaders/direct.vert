#version 430
//meta:name direct_vert

layout(location = 0) in vec3 in_pos;
layout(location = 1) in vec3 in_color;

layout(location = 0) out vec3 out_color;

uniform mat4 u_view_projection_mat;

out gl_PerVertex {
  vec4 gl_Position;
};

void main() {
	gl_Position = u_view_projection_mat * vec4(in_pos, 1);
	out_color = in_color;
}