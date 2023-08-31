float4 importanceSampleGGX(float2 uv, float4 N, float roughness) {
  float a = roughness * roughness;

  float phi = 2.0 * M_PI_F * uv.x;
  float cosTheta = sqrt((1.0 - uv.y) / (1.0 + (a * a - 1.0) * uv.y));
  float sinTheta = sqrt(1.0 - cosTheta * cosTheta);

  // from spherical coordinates to cartesian coordinates
  float4 H;
  H.x = cos(phi) * sinTheta;
  H.y = sin(phi) * sinTheta;
  H.z = cosTheta;
  H.w = 0.0;

  // from tangent-space vector to world-space sample vector
  float4 up = fabs(N.z) < 0.999 ? (float4)(0.0, 0.0, 1.0, 0.0)
                                : (float4)(1.0, 0.0, 0.0, 0.0);
  float4 tangent = normalize(cross(up, N));
  float4 bitangent = cross(N, tangent);

  float4 sampleVec = tangent * H.x + bitangent * H.y + N * H.z;
  return normalize(sampleVec);
}

float geometrySchlickGGX(float NdotV, float roughness) {
  float a = roughness;
  float k = (a * a) / 2.0f;

  float nom = NdotV;
  float denom = NdotV * (1.0f - k) + k;

  return nom / denom;
}

float geometrySmith(float4 N, float4 V, float4 L, float roughness) {
  float NdotV = fmax(dot(N, V), 0.0f);
  float NdotL = fmax(dot(N, L), 0.0f);
  float ggx2 = geometrySchlickGGX(NdotV, roughness);
  float ggx1 = geometrySchlickGGX(NdotL, roughness);

  return ggx1 * ggx2;
}

__kernel void integrate_brdf(__write_only image2d_t dstImage, int size,
                             float sizefac, __global float2 *hammersleySeq,
                             int sequenceSize) {
  int outu = get_global_id(0);
  int outv = get_global_id(1);

  // This check is probably unnecessary
  if (outu >= size || outv >= size) {
    return;
  }

  float NdotV = ((float)(outu) + 0.5) * sizefac;
  float roughness = ((float)(outv) + 0.5) * sizefac;

  float4 V;
  V.x = sqrt(1.0f - NdotV * NdotV);
  V.y = 0.0f;
  V.z = NdotV;

  float A = 0.0f;
  float B = 0.0f;

  float4 N = (float4)(0.0f, 0.0f, 1.0f, 0.0f);

  for (int i = 0; i < sequenceSize; ++i) {
    float2 hs = hammersleySeq[i];
    float4 H = importanceSampleGGX(hs, N, roughness);
    float4 L = normalize(2.0f * dot(V, H) * H - V);

    float NdotL = fmax(L.z, 0.0f);
    float NdotH = fmax(H.z, 0.0f);
    float VdotH = fmax(dot(V, H), 0.0f);

    if (NdotL > 0.0f) {
      float g = geometrySmith(N, V, L, roughness);
      float gvis = (g * VdotH) / (NdotH * NdotV);
      float Fc = pow(1.0f - VdotH, 5.0f);

      A += (1.0f - Fc) * gvis;
      B += Fc * gvis;
    }
  }
  A /= (float)(sequenceSize);
  B /= (float)(sequenceSize);

  write_imagef(dstImage, (int2)(outu, outv), (float4)(A, B, 0.0f, 1.0f));
}