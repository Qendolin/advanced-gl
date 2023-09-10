#version 450 core


layout(location = 0) in vec2 in_uv;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_color;
uniform vec4 u_threshold;
uniform int u_first_pass;


float luminance(vec3 v)
{
    return dot(v, vec3(0.2126f, 0.7152f, 0.0722f));
}

float karisAverage(vec3 col)
{
    float luma = luminance(col);
    return 1.0 / (1.0 + luma);
}

// Better, temporally stable box filtering
// [Jimenez14] http://goo.gl/eomGso
// . . . . . . .
// . A . B . C .
// . . D . E . .
// . F . G . H .
// . . I . J . .
// . K . L . M .
// . . . . . . .
vec4 sampleBox13Tap(vec2 uv, vec2 texelSize) {
    vec4 a = texture(u_color, uv + vec2(-1.0, -1.0) * texelSize);
    vec4 b = texture(u_color, uv + vec2( 0.0, -1.0) * texelSize);
    vec4 c = texture(u_color, uv + vec2( 1.0, -1.0) * texelSize);
    vec4 d = texture(u_color, uv + vec2(-0.5, -0.5) * texelSize);
    vec4 e = texture(u_color, uv + vec2( 0.5, -0.5) * texelSize);
    vec4 f = texture(u_color, uv + vec2(-1.0,  0.0) * texelSize);
    vec4 g = texture(u_color, uv);
    vec4 h = texture(u_color, uv + vec2( 1.0,  0.0) * texelSize);
    vec4 i = texture(u_color, uv + vec2(-0.5,  0.5) * texelSize);
    vec4 j = texture(u_color, uv + vec2( 0.5,  0.5) * texelSize);
    vec4 k = texture(u_color, uv + vec2(-1.0,  1.0) * texelSize);
    vec4 l = texture(u_color, uv + vec2( 0.0,  1.0) * texelSize);
    vec4 m = texture(u_color, uv + vec2( 1.0,  1.0) * texelSize);

    if (u_first_pass == 0) {
        vec4 result = (d + e + i + j) * 0.25 * 0.5;
        result += (a + b + g + f) * 0.25 * 0.125;
        result += (b + c + h + g) * 0.25 * 0.125;
        result += (f + g + l + k) * 0.25 * 0.125;
        result += (g + h + m + l) * 0.25 * 0.125;
        return result;
    }

    // Note: This is the correct way to apply the karis average
    // The implementation by https://learnopengl.com/Guest-Articles/2022/Phys.-Based-Bloom is incorrect
    // as it is not energy preserving
    vec4 result0 = (d + e + i + j) * 0.25 * 4.0;
    vec4 result1 = (a + b + g + f) * 0.25;
    vec4 result2 = (b + c + h + g) * 0.25;
    vec4 result3 = (f + g + l + k) * 0.25;
    vec4 result4 = (g + h + m + l) * 0.25;

    float weight0 = karisAverage(result0.rgb);
    float weight1 = karisAverage(result1.rgb);
    float weight2 = karisAverage(result2.rgb);
    float weight3 = karisAverage(result3.rgb);
    float weight4 = karisAverage(result4.rgb);

    return ( 
        result0 * weight0 + 
        result1 * weight1 + 
        result2 * weight2 + 
        result3 * weight3 + 
        result4 * weight4) / (
            weight0 + weight1 + weight2 + weight3 + weight4
        );
}

vec4 quadraticThreshold(vec4 color, float threshold, vec3 curve) {
	// Pixel brightness
    float br = max(color.r, max(color.g, color.b));

    // Under-threshold part: quadratic curve
    float rq = clamp(br - curve.x, 0.0, curve.y);
    rq = curve.z * rq * rq;

    // Combine and apply the brightness response curve.
    color *= max(rq, br - threshold) / max(br, 1.0e-4);

    return color;
}

void main() {
  out_color = sampleBox13Tap(in_uv, vec2(1) / textureSize(u_color, 0));
  if(u_first_pass == 1) {
  	out_color = quadraticThreshold(out_color, u_threshold.x, u_threshold.yzw);
  }
}