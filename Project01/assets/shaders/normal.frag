#version 450 core
//meta:name normal_frag

layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

const float nearZ = 0.1;
const float farZ = 1000;

void main() {
  float ndc_depth = gl_FragCoord.z / gl_FragCoord.w;
  float depth = (((farZ-nearZ) * ndc_depth) + nearZ + farZ) / 2.0;
  vec3 dx = dFdx(vec3(gl_FragCoord.x, gl_FragCoord.y, depth));
  vec3 dy = dFdy(vec3(gl_FragCoord.x, gl_FragCoord.y, depth));

  vec3 n = cross(normalize(dx), normalize(dy));
  n = normalize(vec3(n.xy, 0));
  n.z = -(n.x + n.y)/2.;

  out_color = vec4(n, 1.);
  if(!gl_FrontFacing) {
    out_color *= 0.5;
  }
}