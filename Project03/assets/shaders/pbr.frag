#version 450 core
//meta:name geometry_frag

layout(early_fragment_tests) in;

layout(location = 0) in vec3 in_world_position;
layout(location = 1) in vec2 in_uv;
layout(location = 2) in mat3 in_tbn;

layout(location = 0) out vec4 out_color;

layout(binding = 0) uniform sampler2D u_albedo;
layout(binding = 1) uniform sampler2D u_normal;
layout(binding = 2) uniform sampler2D u_orm;
layout(binding = 3) uniform samplerCube u_environment_diffuse;
layout(binding = 4) uniform samplerCube u_environment_specualr;
layout(binding = 5) uniform sampler2D u_environment_brdf_lut;
uniform vec3 u_camera_position;
uniform vec3[4] u_light_positions;
uniform vec3[4] u_light_colors;

const float PI = 3.14159265359;

vec3 transformNormal(vec3 tN) {
    return normalize(in_tbn * tN);
}

// Based on https://media.contentapi.ea.com/content/dam/eacom/frostbite/files/course-notes-moving-frostbite-to-pbr-v2.pdf page 92
float adjustRoughness(vec3 tN, float roughness) {
    float r = length(tN);
    if (r < 1.0) {
        float kappa = (3.0 * r - r * r * r) / (1.0 - r * r);
        float variance = 1.0 / kappa;
        // Why is it ok for the roughness to be > 1 ?
        return sqrt(roughness * roughness + variance);
    }
    return roughness;
}

float DistributionGGX(vec3 N, vec3 H, float roughness)
{
    float a = roughness*roughness;
    float a2 = a*a;
    float NdotH = max(dot(N, H), 0.0);
    float NdotH2 = NdotH*NdotH;

    float nom   = a2;
    float denom = (NdotH2 * (a2 - 1.0) + 1.0);
    denom = PI * denom * denom;

    return nom / denom;
}

float GeometrySchlickGGX(float NdotV, float roughness)
{
    float r = (roughness + 1.0);
    float k = (r*r) / 8.0;

    float nom   = NdotV;
    float denom = NdotV * (1.0 - k) + k;

    return nom / denom;
}

float GeometrySmith(vec3 N, vec3 V, vec3 L, float roughness)
{
    // + 5e-6 to prevent artifacts, value is from https://google.github.io/filament/Filament.html#materialsystem/specularbrdf:~:text=float%20NoV%20%3D%20abs(dot(n%2C%20v))%20%2B%201e%2D5%3B
    float NdotV = max(dot(N, V), 0.0) * (1.0 - 5e-6) + 5e-6;
    float NdotL = max(dot(N, L), 0.0);
    float ggx2 = GeometrySchlickGGX(NdotV, roughness);
    float ggx1 = GeometrySchlickGGX(NdotL, roughness);

    return ggx1 * ggx2;
}

vec3 fresnelSchlick(float cosTheta, vec3 F0)
{
    return F0 + (1.0 - F0) * pow(clamp(1.0 - cosTheta, 0.0, 1.0), 5.0);
}

vec3 fresnelSchlickRoughness(float cosTheta, vec3 F0, float roughness)
{
    return F0 + (max(vec3(1.0 - roughness), F0) - F0) * pow(clamp(1.0 - cosTheta, 0.0, 1.0), 5.0);
}


vec3 sampleAmbient(vec3 N, vec3 V, vec3 R, vec3 F0, float roughness, vec3 albedo, float ao)
{
    vec3 F = fresnelSchlickRoughness(max(dot(N, V), 0.0), F0, roughness);
    vec3 kS = F;
    vec3 kD = 1.0 - kS;
    vec3 irradiance = texture(u_environment_diffuse, N).rgb;
    vec3 diffuse    = irradiance * albedo;

    const float MAX_REFLECTION_LOD = 5.0;
    vec3 reflection = textureLod(u_environment_specualr, R, roughness * MAX_REFLECTION_LOD).rgb;   
    vec2 envBRDF  = texture(u_environment_brdf_lut, vec2(max(dot(N, V), 0.0), roughness)).rg;
    vec3 specular = reflection * (F * envBRDF.x + envBRDF.y);

    return (kD * diffuse + specular) * ao; 
}

vec3 reinhard2(vec3 x) {
  const float L_white = 4.0;

  return (x * (1.0 + x / (L_white * L_white))) / (1.0 + x);
}

void main()
{
    vec3 albedo     = pow(texture(u_albedo, in_uv).rgb, vec3(2.2));
    vec3 orm        = texture(u_orm, in_uv).xyz;
    float metallic  = orm.z;
    float roughness = orm.y;
    float ao        = orm.x;

    vec3 tN = texture(u_normal, in_uv).xyz * 2.0 - 1.0;
    float br = roughness;
    roughness = adjustRoughness(tN, roughness);
    vec3 N = transformNormal(tN);
    vec3 V = normalize(u_camera_position - in_world_position);
    vec3 R = reflect(-V, N);

    // calculate reflectance at normal incidence; if dia-electric (like plastic) use F0
    // of 0.04 and if it's a metal, use the albedo color as F0 (metallic workflow)
    vec3 F0 = vec3(0.04);
    F0 = mix(F0, albedo, metallic);

    // reflectance equation
    vec3 Lo = vec3(0.0);
    for(int i = 0; i < 4; ++i)
    {
        // calculate per-light radiance
        vec3 L = normalize(u_light_positions[i] - in_world_position);
        vec3 H = normalize(V + L);
        float distance = length(u_light_positions[i] - in_world_position);
        float attenuation = 1.0 / (distance * distance);
        vec3 radiance = u_light_colors[i] * attenuation;

        // Cook-Torrance BRDF
        float NDF = DistributionGGX(N, H, roughness);
        float G   = GeometrySmith(N, V, L, roughness);
        vec3 F    = fresnelSchlick(max(dot(H, V), 0.0), F0);

        vec3 numerator    = NDF * G * F;
        float denominator = 4.0 * max(dot(N, V), 0.0) * max(dot(N, L), 0.0) + 1e-5; // + 1e-5 to prevent divide by zero
        vec3 specular = numerator / denominator;

        // kS is equal to Fresnel
        vec3 kS = F;
        // for energy conservation, the diffuse and specular light can't
        // be above 1.0 (unless the surface emits light); to preserve this
        // relationship the diffuse component (kD) should equal 1.0 - kS.
        vec3 kD = vec3(1.0) - kS;
        // multiply kD by the inverse metalness such that only non-metals
        // have diffuse lighting, or a linear blend if partly metal (pure metals
        // have no diffuse light).
        kD *= 1.0 - metallic;

        // scale light by NdotL
        float NdotL = max(dot(N, L), 0.0);

        // The ao term doesn't really belong here, but I like it better that way
        float occlusion = mix(ao, 1.0, NdotL);

        // add to outgoing radiance Lo
        Lo += (kD * albedo / PI + specular) * radiance * NdotL * occlusion;  // note that we already multiplied the BRDF by the Fresnel (kS) so we won't multiply by kS again
    }

    // ambient lighting (note that the next IBL tutorial will replace
    // this ambient lighting with environment lighting).
    vec3 ambient = sampleAmbient(N, V, R, F0, roughness, albedo, ao);

    vec3 color = ambient + Lo;

    // HDR tonemapping
    color = reinhard2(color);
    // gamma correct
    color = pow(color, vec3(1.0/2.2));

    out_color = vec4(color, 1.0);
}