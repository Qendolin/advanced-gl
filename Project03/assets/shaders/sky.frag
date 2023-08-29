#version 450 core

layout(location = 0) out vec3 out_color;
layout(location = 0) in vec3 in_dir;

layout(binding = 0) uniform samplerCube u_hdri;

const float pi = 3.14159265359;


void main()
{
    out_color = texture(u_hdri, normalize(in_dir)).rgb;

	out_color = out_color / (out_color + vec3(1.0));
    out_color = pow(out_color, vec3(1.0/2.2)); 

}
