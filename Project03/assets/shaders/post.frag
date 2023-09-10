#version 450 core


layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_color;
layout(binding = 1) uniform sampler2D u_bloom;

uniform float u_bloom_factor;
uniform float u_exposure;

const float WhiteLevel = 11.0;

float luminance(vec3 v)
{
    return dot(v, vec3(0.2126, 0.7152, 0.0722));
}

vec3 change_luminance(vec3 c_in, float l_out)
{
    float l_in = luminance(c_in);
    return c_in * (l_out / l_in);
}

vec3 reinhard2(vec3 x) {
  return (x * (1.0 + x / (WhiteLevel * WhiteLevel))) / (1.0 + x);
}

vec3 reinhard2_lumi(vec3 v) {
  float l_old = luminance(v);
  float numerator = l_old * (1.0 + (l_old / (WhiteLevel * WhiteLevel)));
  float l_new = numerator / (1.0 + l_old);
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
    float A = 0.15;
    float B = 0.50;
    float C = 0.10;
    float D = 0.20;
    float E = 0.02;
    float F = 0.30;
    return ((x*(A*x+C*B)+D*E)/(x*(A*x+B)+D*F))-E/F;
}

vec3 uncharted2_filmic(vec3 v)
{
    float exposure_bias = 2.0;
    vec3 curr = uncharted2_tonemap_partial(v * exposure_bias);

    vec3 W = vec3(11.2);
    vec3 white_scale = vec3(1.0) / uncharted2_tonemap_partial(W);
    return curr * white_scale;
}

vec3 aces_approx(vec3 v)
{
    v *= 0.6;
    float a = 2.51;
    float b = 0.03;
    float c = 2.43;
    float d = 0.59;
    float e = 0.14;
    return clamp((v*(a*v+b))/(v*(c*v+d)+e), 0.0, 1.0);
}

// https://github.com/Unity-Technologies/Graphics/blob/9e7b41b807a6c18597e2cdb64c693c02e08d5ab/com.unity.postprocessing/PostProcessing/Shaders/Colors.hlsl#L291C1-L321C1
// Neutral tonemapping (Hable/Hejl/Frostbite)
// Input is linear RGB
vec3 neutralCurve(vec3 x, float a, float b, float c, float d, float e, float f)
{
    return ((x * (a * x + c * b) + d * e) / (x * (a * x + b) + d * f)) - e / f;
}

vec3 neutralTonemap(vec3 x)
{
    // Tonemap
    float a = 0.2;
    float b = 0.29;
    float c = 0.24;
    float d = 0.272;
    float e = 0.02;
    float f = 0.3;
    float whiteClip = 0.95;

    vec3 whiteScale = vec3(1.0) / neutralCurve(vec3(WhiteLevel), a, b, c, d, e, f);
    x = neutralCurve(x * whiteScale, a, b, c, d, e, f);
    x *= whiteScale;

    // Post-curve white point adjustment
    x /= whiteClip.xxx;

    return x;
}


void main() {
	vec3 color = texture(u_color, in_uv).rgb;
	color += texture(u_bloom, in_uv).rgb * u_bloom_factor;

	// Tonemapping
  color = neutralTonemap(color);
  // color = reinhard2_lumi(color);

  // Gamma correction
  color = clamp(color, vec3(0), vec3(1));
  color = pow(color, vec3(1.0/2.4));

  out_color = vec4(color, 1.);
}