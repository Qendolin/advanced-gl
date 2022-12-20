#version 450 core
//meta:name ssao_blur_frag

layout(location = 0) in vec2 in_uv;

layout(location = 0) out float out_color;

layout(binding = 0) uniform sampler2D u_pos;
layout(binding = 1) uniform sampler2D u_ao;

uniform float u_edge_threshold = 0.1;

float distSq(vec3 a, vec3 b) {
    vec3 v = a - b;
    return dot(v, v);
}

void main() {
    vec2 texelSize = 1.0 / vec2(textureSize(u_ao, 0));
    vec3 p0 = texture(u_pos, in_uv).xyz;
    float sampleCount = 0;
    float result = 0;
    float threshold = u_edge_threshold * u_edge_threshold;
    for (int x = -2; x < 2; x++) {
        for (int y = -2; y < 2; y++) {
            vec2 offset = vec2(float(x), float(y)) * texelSize;
            vec3 p = texture(u_pos, in_uv + offset).xyz;
            float v = texture(u_ao, in_uv + offset).x;
            if(distSq(p0, p) < threshold) {
                result += v;
                sampleCount += 1;
            }
        }
    }
    out_color = result / sampleCount;
}