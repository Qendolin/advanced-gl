#version 330 core
layout (location = 0) in vec3 in_pos;

out vec3 out_dir;

uniform mat4 u_projection_mat;
uniform mat4 u_view_mat;

void main()
{
    out_dir = in_pos;
    gl_Position = u_projection_mat * u_view_mat * vec4(in_pos, 1.0);
}