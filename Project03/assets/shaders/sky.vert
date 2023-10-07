#version 450 core


layout(location = 0) in vec3 in_position;

out gl_PerVertex {
    vec4 gl_Position;
};

layout(location = 0) out vec3 out_direction;

uniform mat4 u_view_mat;
uniform mat4 u_projection_mat;
uniform mat4 u_environment_transform;
uniform vec3 u_environment_origin;

void main()
{
    out_direction = (u_environment_transform * vec4(0.5 * in_position, 1.0)).xyz - u_environment_origin;
    gl_Position = u_projection_mat * u_view_mat * u_environment_transform * vec4((0.5 * in_position), 1.0);
    // gl_Position = gl_Position.xyww;
}