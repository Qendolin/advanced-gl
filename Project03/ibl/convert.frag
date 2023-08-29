#version 330 core

out vec3 out_color;
in vec3 in_dir;

uniform sampler2D u_equirectangular_texture;

const float pi = 3.14159265359;

const vec2 invAtan = vec2(1.0/(2.0*pi), 1.0/pi);
vec2 sampleSphericalMap(vec3 v)
{
    vec2 uv = vec2(atan(v.z, v.x), asin(v.y));
    uv *= invAtan;
    uv += 0.5;
    return uv;
}

void main()
{		
    vec2 uv = sampleSphericalMap(normalize(in_dir)); // make sure to normalize localPos
    out_color = texture(u_equirectangular_texture, uv).rgb;
}
