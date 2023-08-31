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

// The kernel is invoked for every pixel on every face of the cubemap
// 'size' is the size of a cube map face
// 'sizefac' is 1/size precomputed
// 'samplesSize' is the number of samples
__kernel void convolve_diffuse(__read_only image2d_array_t srcImage,
                               __write_only image2d_array_t dstImage, int size,
                               float sizefac, __global float4 *samples,
                               int samplesSize) {
  int outu = get_global_id(0);
  int outv = get_global_id(1);
  int face = get_global_id(2);

  // This check is probably unnecessary
  if (outu >= size || outv >= size || face >= 6) {
    return;
  }

  // The value range is (-1, 1)
  float horizontal = (float)(2 * outu + 1) * sizefac - 1.0f;
  float vertical = (float)(2 * outv + 1) * sizefac - 1.0f;

  float4 vec = (float4)(horizontal, vertical, 1.0f, 0.0f);

  float x = dot(vec, xTransforms[face]);
  float y = dot(vec, yTransforms[face]);
  float z = dot(vec, zTransforms[face]);

  float4 normal = normalize((float4)(x, y, z, 0.0f));
  float4 up = (float4)(0.0f, 1.0f, 0.0f, 0.0f);
  if (fabs(normal.y) >= 0.999f) {
    up = (float4)(0.0f, 0.0f, 1.0f, 0.0f);
  }
  float4 tangent = normalize(cross(up, normal));
  float4 bitangent = normalize(cross(normal, tangent));

  float4 cumulative = (float4)(0.0f, 0.0f, 0.0f, 0.0f);

  float totalWeight = 0.0f;
  for (int i = 0; i < samplesSize; i++) {
    float4 sample = samples[i];
    float4 dir = sample.x * tangent + sample.y * bitangent + sample.z * normal;
    if (dir.x == 0.0f && dir.y == 0.0f && dir.z == 0.0f) {
      continue;
    }
    dir = normalize(dir);
    float4 uv = projectCubeMap(dir);
    float4 color = read_imagef(srcImage, srcSampler, uv);
    cumulative += color * sample.w;
    totalWeight += 1;
  }
  float4 color = M_PI_F * cumulative / totalWeight;

  // the cube map faces are stacked vertically
  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}

// The kernel is invoked for every pixel on every face of the cubemap
// 'size' is the size of a cube map face
// 'sizefac' is 1/size precomputed
// 'samplesStart' is the first sample index
// 'samplesSize' is the number of samples
__kernel void convolve_specular(__read_only image2d_array_t srcImage,
                                __write_only image2d_array_t dstImage, int size,
                                float sizefac, __global float4 *samples,
                                int samplesStart, int samplesSize) {
  int outu = get_global_id(0);
  int outv = get_global_id(1);
  int face = get_global_id(2);

  // This check is probably unnecessary
  if (outu >= size || outv >= size || face >= 6) {
    return;
  }

  // The value range is (-1, 1)
  float horizontal = (float)(2 * outu + 1) * sizefac - 1.0f;
  float vertical = (float)(2 * outv + 1) * sizefac - 1.0f;

  float4 vec = (float4)(horizontal, vertical, 1.0f, 0.0f);

  float x = dot(vec, xTransforms[face]);
  float y = dot(vec, yTransforms[face]);
  float z = dot(vec, zTransforms[face]);

  float4 normal = normalize((float4)(x, y, z, 0.0f));
  float4 view = normal;
  float4 up = (float4)(0.0f, 0.0f, 1.0f, 0.0f);
  if (fabs(normal.z) >= 0.999f) {
    up = (float4)(1.0f, 0.0f, 0.0f, 0.0f);
  }
  float4 tangent = normalize(cross(up, normal));
  float4 bitangent = normalize(cross(normal, tangent));

  float4 cumulative = (float4)(0.0f, 0.0f, 0.0f, 0.0f);
  float totalWeight = 0.0f;
  for (int i = 0; i < samplesSize; i++) {
    float4 sample = samples[samplesStart + i];
    float4 dir = sample.x * tangent + sample.y * bitangent + sample.z * normal;
    if (dir.x == 0.0f && dir.y == 0.0f && dir.z == 0.0f) {
      continue;
    }
    dir = normalize(dir);
    float4 l = normalize(2.0f * dot(view, dir) * dir - view);
    float ndotl = fmax(dot(normal, l), 0.0f);
    if (ndotl > 0.0f) {
      float4 uv = projectCubeMap(l);
      float4 color = read_imagef(srcImage, srcSampler, uv);
      cumulative += color * ndotl;
      totalWeight += ndotl;
    }
  }
  float4 color = cumulative / totalWeight;

  // the cube map faces are stacked vertically
  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}