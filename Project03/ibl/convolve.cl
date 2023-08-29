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

// https://www.gamedev.net/forums/topic/687535-implementing-a-cube-map-lookup-function/
float4 projectCubeMap(const float4 v) {
  float4 vAbs = fabs(v);
  float ma;
  float4 uv = (float4)(0.0, 0.0, 0.0, 0.0);
  if (vAbs.z >= vAbs.x && vAbs.z >= vAbs.y) {
    uv.z = v.z < 0.0 ? 5.0 : 4.0;
    ma = 0.5 / vAbs.z;
    uv.xy = (float2)(v.z < 0.0 ? -v.x : v.x, -v.y);
  } else if (vAbs.y >= vAbs.x) {
    uv.z = v.y < 0.0 ? 3.0 : 2.0;
    ma = 0.5 / vAbs.y;
    uv.xy = (float2)(v.x, v.y < 0.0 ? -v.z : v.z);
  } else {
    uv.z = v.x < 0.0 ? 1.0 : 0.0;
    ma = 0.5 / vAbs.x;
    uv.xy = (float2)(v.x < 0.0 ? v.z : -v.z, -v.y);
  }
  uv.xy = uv.xy * ma + (float2)(0.5, 0.5);
  return uv;
}

// The kernel is invoked for every pixel on every face of the cubemap
// 'size' is the size of a cube map face
// 'sizefac' is 1/size precomputed
__kernel void convolve_cubemap(__read_only image2d_array_t srcImage,
                               __write_only image2d_array_t dstImage, int size,
                               float sizefac, __global float4 *samples,
                               int count) {
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

  float4 normal = normalize((float4)(x, y, z, 0.0));
  float4 up = (float4)(0.0, 1.0, 0.0, 0.0);
  float4 right = normalize(cross(up, normal));
  up = normalize(cross(normal, right));

  float4 cumulative = (float4)(0.0, 0.0, 0.0, 0.0);

  int n = 0;
  for (int i = 0; i < count; i++) {
    float4 sample = samples[i];
    float4 dir = sample.x * right + sample.y * up + sample.z * normal;
    if (dir.x == 0.0 && dir.y == 0.0 && dir.z == 0.0) {
      continue;
    }
    float4 uv = projectCubeMap(dir);
    float4 color = read_imagef(srcImage, srcSampler, uv);
    cumulative += color * sample.w;
    n++;
  }
  float4 color = M_PI_F * cumulative / (float)(n);

  // the cube map faces are stacked vertically
  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}