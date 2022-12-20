#version 450 core
//meta:name point_light_vert

layout(location = 0) in vec3 in_position;
layout(location = 1) in vec3 in_light_position;
layout(location = 2) in vec3 in_color;
layout(location = 3) in float in_attenuation;
layout(location = 4) in float in_radius;

layout(location = 0) flat out vec3 out_light_position;
layout(location = 1) flat out vec3 out_color;
layout(location = 2) flat out float out_attenuation;

out gl_PerVertex {
  vec4 gl_Position;
};

uniform mat4 u_view_projection_mat = mat4(1.0);
uniform mat4 u_view_mat = mat4(1.0);

void main() {
  gl_Position = u_view_projection_mat * vec4(in_position * in_radius + in_light_position, 1.);
  out_light_position = (u_view_mat * vec4(in_light_position, 1.)).xyz;
  out_color = in_color;
  out_attenuation = in_attenuation;
}