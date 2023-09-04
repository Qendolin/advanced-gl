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

  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}