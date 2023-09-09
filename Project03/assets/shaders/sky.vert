#version 450 core


layout(location = 0) in vec3 in_position;

out gl_PerVertex {
  vec4 gl_Position;
};

layout(location = 0) out vec3 out_direction;

uniform mat4 u_view_mat;
uniform mat4 u_projection_mat;

void main()
{
    out_direction = in_position;
	mat4 rot_only_view = mat4(mat3(u_view_mat));
    gl_Position = u_projection_mat * rot_only_view * vec4(in_position, 1.0);
	gl_Position = gl_Position.xyww;
}