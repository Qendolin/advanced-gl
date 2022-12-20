#version 450 core
//meta:name sky_frag

layout(location = 0) in vec3 in_dir;

layout(location = 0) out vec3 out_color;


uniform mat4 u_projection_mat;
uniform mat4 u_view_mat;

// TODO: Finish
void main() {
  // vec4 dir = inverse(u_view_mat) * inverse(u_projection_mat) * vec4(in_dir, 1.0); // OK 1
  // vec4 dir = inverse(u_view_mat) * vec4(in_dir, 1.0);
  vec4 dir = inverse(u_projection_mat) * vec4(in_dir, 1.0); // OK 2
  // dir.xyz /= dir.w;

  // vec4 dir = vec4(in_dir, 0.);

  // vec4 up = vec4(0, 1, 0, 1); // OK 1
  vec4 up = u_view_mat * vec4(0, 1, 0, 1); // OK 2
  // up.xyz /= up.w;
  
  
  vec3 gamma = vec3(2.4);
  vec3 col0 = pow(vec3(40, 40, 43) / 255, gamma);
  vec3 col1 = pow(vec3(72, 72, 85) / 255, gamma);
  vec3 col2 = pow(vec3(232, 248, 255) / 255, gamma);
  vec3 col3 = pow(vec3(199, 231, 245) / 255, gamma);
  vec3 col4 = pow(vec3(	56, 163, 209) / 255, gamma);

  float theta = dot(normalize(up.xyz), normalize(dir.xyz));
  vec3 color = mix(col0, col1, smoothstep(-1.0, -0.04, theta));
  color = mix(color, col2, smoothstep(-0.04, 0.0, theta));
  color = mix(color, col3, smoothstep(0.0, 0.12, theta));
  color = mix(color, col4, smoothstep(0.12, 1.0, theta));
  out_color = pow(color + 1, vec3(2)) - 1;
}