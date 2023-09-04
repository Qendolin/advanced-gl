constant sampler_t srcSampler =
    CLK_NORMALIZED_COORDS_TRUE | CLK_ADDRESS_CLAMP_TO_EDGE | CLK_FILTER_LINEAR;

// These transforms specify the directions based on the cube face
// They are based on
// https://www.khronos.org/opengl/wiki_opengl/images/CubeMapAxes.png The order
// is +X, -X, +Y, -Y, +Z, -Z Using the reference image the transforms are
// (horizontal face axis, vertical face axis, face direction)
// Using float4 because float3 were getting a 1 float padding on my gpu which
// messed up the indexing
__constant float4 xTransforms[6] = {
    (float4)(0.0f, 0.0f, 1.0f, 0.0f), (float4)(0.0f, 0.0f, -1.0f, 0.0f),
    (float4)(1.0f, 0.0f, 0.0f, 0.0f), (float4)(1.0f, 0.0f, 0.0f, 0.0f),
    (float4)(1.0f, 0.0f, 0.0f, 0.0f), (float4)(-1.0f, 0.0f, 0.0f, 0.0f)};
__constant float4 yTransforms[6] = {
    (float4)(0.0f, -1.0f, 0.0f, 0.0f), (float4)(0.0f, -1.0f, 0.0f, 0.0f),
    (float4)(0.0f, 0.0f, 1.0f, 0.0f),  (float4)(0.0f, 0.0f, -1.0f, 0.0f),
    (float4)(0.0f, -1.0f, 0.0f, 0.0f), (float4)(0.0f, -1.0f, 0.0f, 0.0f)};
__constant float4 zTransforms[6] = {
    (float4)(-1.0f, 0.0f, 0.0f, 0.0f), (float4)(1.0f, 0.0f, 0.0f, 0.0f),
    (float4)(0.0f, 1.0f, 0.0f, 0.0f),  (float4)(0.0f, -1.0f, 0.0f, 0.0f),
    (float4)(0.0f, 0.0f, 1.0f, 0.0f),  (float4)(0.0f, 0.0f, -1.0f, 0.0f)};

// https://www.gamedev.net/forums/topic/687535-implementing-a-cube-map-lookup-function/
float4 projectCubeMap(const float4 v) {
  float4 vAbs = fabs(v);
  float ma;
  float4 uv = (float4)(0.0f, 0.0f, 0.0f, 0.0f);
  if (vAbs.z >= vAbs.x && vAbs.z >= vAbs.y) {
    uv.z = v.z < 0.0f ? 5.0f : 4.0f;
    ma = 0.5f / vAbs.z;
    uv.xy = (float2)(v.z < 0.0f ? -v.x : v.x, -v.y);
  } else if (vAbs.y >= vAbs.x) {
    uv.z = v.y < 0.0f ? 3.0f : 2.0f;
    ma = 0.5f / vAbs.y;
    uv.xy = (float2)(v.x, v.y < 0.0f ? -v.z : v.z);
  } else {
    uv.z = v.x < 0.0f ? 1.0f : 0.0f;
    ma = 0.5f / vAbs.x;
    uv.xy = (float2)(v.x < 0.0f ? v.z : -v.z, -v.y);
  }
  uv.xy = uv.xy * ma + (float2)(0.5f, 0.5f);
  return uv;
}