#version 450 core


layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_color;
layout(binding = 1) uniform sampler2D u_bloom;

uniform float u_bloom_factor;
uniform float u_exposure;

float luminance(vec3 v)
{
    return dot(v, vec3(0.2126f, 0.7152f, 0.0722f));
}

vec3 change_luminance(vec3 c_in, float l_out)
{
    float l_in = luminance(c_in);
    return c_in * (l_out / l_in);
}

vec3 reinhard2(vec3 x) {
  const float L_white = 4.0;

  return (x * (1.0 + x / (L_white * L_white))) / (1.0 + x);
}

vec3 reinhard2_lumi(vec3 v) {
  const float L_white = 4.0;
  float l_old = luminance(v);
  float numerator = l_old * (1.0f + (l_old / (L_white * L_white)));
  float l_new = numerator / (1.0f + l_old);
  return change_luminance(v, l_new);
}

vec3 reinhard_jodie(vec3 v)
{
    float l = luminance(v);
    vec3 tv = v / (1.0 + v);
    return mix(v / (1.0 + l), tv, tv);
}

vec3 reinhard(vec3 x) {
  return x / (4.0 + x);
}

vec3 exposure(vec3 x) {
  return vec3(1.0) - exp(-x * u_exposure);
}

vec3 uncharted2_tonemap_partial(vec3 x)
{
    float A = 0.15f;
    float B = 0.50f;
    float C = 0.10f;
    float D = 0.20f;
    float E = 0.02f;
    float F = 0.30f;
    return ((x*(A*x+C*B)+D*E)/(x*(A*x+B)+D*F))-E/F;
}

vec3 uncharted2_filmic(vec3 v)
{
    float exposure_bias = 2.0f;
    vec3 curr = uncharted2_tonemap_partial(v * exposure_bias);

    vec3 W = vec3(11.2f);
    vec3 white_scale = vec3(1.0f) / uncharted2_tonemap_partial(W);
    return curr * white_scale;
}

vec3 aces_approx(vec3 v)
{
    v *= 0.6f;
    float a = 2.51f;
    float b = 0.03f;
    float c = 2.43f;
    float d = 0.59f;
    float e = 0.14f;
    return clamp((v*(a*v+b))/(v*(c*v+d)+e), 0.0f, 1.0f);
}

void main() {
	vec3 color = texture(u_color, in_uv).rgb;
	color += texture(u_bloom, in_uv).rgb * u_bloom_factor;

	// Tonemapping
  color = reinhard2_lumi(color);

  // Gamma correction
  color = clamp(color, vec3(0), vec3(1));
  color = pow(color, vec3(1.0/2.4));

  out_color = vec4(color, 1.);
}