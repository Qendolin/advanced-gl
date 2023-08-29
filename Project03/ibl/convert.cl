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
    (float4)(0.0, -1.0f, 0.0f, 0.0f),  (float4)(0.0f, -1.0f, 0.0f, 0.0f),
    (float4)(0.0f, 0.0f, 1.0f, 0.0f),  (float4)(0.0f, 0.0f, -1.0f, 0.0f),
    (float4)(0.0f, -1.0f, 0.0f, 0.0f), (float4)(0.0f, -1.0f, 0.0f, 0.0f)};
__constant float4 zTransforms[6] = {
    (float4)(-1.0f, 0.0f, 0.0f, 0.0f), (float4)(1.0f, 0.0f, 0.0f, 0.0f),
    (float4)(0.0f, 1.0f, 0.0f, 0.0f),  (float4)(0.0f, -1.0f, 0.0f, 0.0f),
    (float4)(0.0f, 0.0f, 1.0f, 0.0f),  (float4)(0.0f, 0.0f, -1.0f, 0.0f)};

float2 projectSphericalMap(float4 dir) {
  float2 uv = (float2)(atan2pi(dir.z, dir.x) * 0.5, asinpi(dir.y));
  uv += (float2)(0.5f, 0.5f);
  return uv;
}

// The kernel is invoked for every pixel on every face of the cubemap
// 'size' is the size of a cube map face
// 'sizefac' is 1/size precomputed
__kernel void reproject_environment(__read_only image2d_t srcImage,
                                    __write_only image2d_array_t dstImage,
                                    int size, float sizefac) {
  int outu = get_global_id(0);
  int outv = get_global_id(1);
  int face = get_global_id(2);

  // This check is probably unnecessary
  if (outu >= size || outv >= size || face >= 6) {
    return;
  }

  // The value range is (-1, 1)
  float horizontal = (float)(2 * outu + 1) * sizefac - 1.0;
  float vertical = (float)(2 * outv + 1) * sizefac - 1.0;

  float4 vec = (float4)(horizontal, vertical, 1.0f, 0.0f);

  float x = dot(vec, xTransforms[face]);
  float y = dot(vec, yTransforms[face]);
  float z = dot(vec, zTransforms[face]);

  float4 dir = normalize((float4)(x, y, z, 0.0));

  float2 uv = projectSphericalMap(normalize(dir));
  float4 color = read_imagef(srcImage, srcSampler, uv);
  // the cube map faces are stacked vertically
  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}