#version 450 core
//meta:name post_process_frag

// #define ENABLE_COLOR_MAPPING
#define ENABLE_DITHERING
#define ENABLE_BLOOM

layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_color;
layout(binding = 1) uniform sampler2D u_dither_pattern;
layout(binding = 2) uniform sampler3D u_color_lut;
layout(binding = 3) uniform sampler2D u_bloom;
uniform float u_bloom_fac;
uniform vec3 u_ambient_light;

// https://github.com/dmnsgn/glsl-tone-map/blob/main/aces.glsl
vec3 aces(vec3 x) {
  const float a = 2.51;
  const float b = 0.03;
  const float c = 2.43;
  const float d = 0.59;
  const float e = 0.14;
  return clamp((x * (a * x + b)) / (x * (c * x + d) + e), 0.0, 1.0);
}

// https://github.com/dmnsgn/glsl-tone-map/blob/main/reinhard2.glsl
vec3 reinhard2(vec3 x) {
  const float L_white = 4.0;

  return (x * (1.0 + x / (L_white * L_white))) / (1.0 + x);
}

vec3 reinhard(vec3 x) {
    return x / (1.0 + x);
}

vec3 colorLookup(vec3 c) {
  // The sampling cube needs to be 0.5 px smaller then the texture
  // to ensure that sampling begins at the center of a (3d) pixel
  float size = textureSize(u_color_lut, 0).x;
  float scale = (size - 1.0) / size;
  float offset = 0.5 * 1.0 / size;
  return texture(u_color_lut, scale * c + offset).rgb;
}

void main() {
  vec2 fragCoord = in_uv * textureSize(u_color, 0);
  vec3 color = texture(u_color, in_uv).rgb;
#ifdef ENABLE_BLOOM
  color += texture(u_bloom, in_uv).rgb * u_bloom_fac;
#endif

  // Tonemapping
  color = reinhard2(color);

  // Gamma correction
  color = clamp(color, vec3(0), vec3(1));
  color = pow(color, vec3(1.0/2.4));

#ifdef ENABLE_DITHERING
  // Dithering
  float ditherPatternSize = textureSize(u_dither_pattern, 0).x;
  float ditherValue = texture(u_dither_pattern, fragCoord / ditherPatternSize).r;
  color += (ditherValue - 0.5) / 255.;
#endif

#ifdef ENABLE_COLOR_MAPPING
  // Color Mapping
  // Last step because the lut is transformed based on the rendered image
  color = colorLookup(color);
#endif

  out_color = vec4(color, 1.);
}