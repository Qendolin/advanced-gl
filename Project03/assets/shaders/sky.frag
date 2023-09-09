#version 450 core


layout(location = 0) out vec3 out_color;
layout(location = 0) in vec3 in_dir;

layout(binding = 0) uniform samplerCube u_hdri;

void main()
{
    out_color = texture(u_hdri, normalize(in_dir)).rgb; 
}
