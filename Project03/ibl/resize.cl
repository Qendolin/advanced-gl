// The kernel is invoked for every pixel on every face of the cubemap
// 'size' is the size of a cube map face
// 'sizefac' is 1/size precomputed
// 'samplesSize' is the number of samples
__kernel void resize_environment(__read_only image2d_array_t srcImage,
                                 __write_only image2d_array_t dstImage,
                                 int size, float sizefac,
                                 __global float2 *samples, int samplesSize) {
  int outu = get_global_id(0);
  int outv = get_global_id(1);
  int face = get_global_id(2);

  // This check is probably unnecessary
  if (outu >= size || outv >= size || face >= 6) {
    return;
  }

  float4 cumulative = (float4)(0.0f, 0.0f, 0.0f, 0.0f);

  // The value range is (-1, 1)
  float horizontal = (float)(2 * outu + 1) * sizefac - 1.0f;
  float vertical = (float)(2 * outv + 1) * sizefac - 1.0f;

  for (int i = 0; i < samplesSize; i++) {
    float2 sample = samples[i];
    float su = horizontal + sample.x * sizefac;
    float sv = vertical + sample.y * sizefac;

    float4 vec = (float4)(su, sv, 1.0f, 0.0f);

    float x = dot(vec, xTransforms[face]);
    float y = dot(vec, yTransforms[face]);
    float z = dot(vec, zTransforms[face]);

    float4 dir = normalize((float4)(x, y, z, 0.0f));
    float4 uv = projectCubeMap(dir);
    float4 color = read_imagef(srcImage, srcSampler, uv);
    cumulative += color;
  }

  float4 color = cumulative / (float)(samplesSize);

  write_imagef(dstImage, (int4)(outu, outv, face, 0), color);
}