#include <stdint.h>
#include "float_ops.h"

// compile with `clang -masm=intel -mno-red-zone -mavx2 -mfma -mstackrealign -mllvm -inline-threshold=1000 -fno-asynchronous-unwind-tables -fno-exceptions -S -Ofast -o c/encode._s -c c/encode.c`
// then convert to go asm with `c2goasm -a c/encode._s encode_amd64.s`

// https://github.com/KnightOS/libc/blob/master/src/gpl/frexpf.c
__attribute__((always_inline)) static inline float fast_frexpf(float x, int *pw2)
{
	ieee_float_shape_type fl;
	long int i;

	fl.value = x;
	/* Find the exponent (power of 2) */
	i = (fl.word >> 23) & 0x000000ff;
	i -= 0x7e;
	*pw2 = i;
	fl.word &= 0x807fffff; /* strip all exponent bits */
	fl.word |= 0x3f000000; /* mantissa between 0.5 and 1 */
	return (fl.value);
}

uint64_t EncodeRgbe(uint64_t components, uint64_t len, float *src, uint8_t *dst)
{
	if (components != 3 && components != 4)
		return 0;

	uint64_t n = 0;
	for (uint32_t i = 0; i < len / components; i++)
	{
		float r = src[i * components + 0];
		float g = src[i * components + 1];
		float b = src[i * components + 2];

		float max = r;
		if (g > max)
		{
			max = g;
		}
		if (b > max)
		{
			max = b;
		}

		if (max < 1e-32)
		{
			dst[i * 4 + 0] = 0;
			dst[i * 4 + 1] = 0;
			dst[i * 4 + 2] = 0;
			dst[i * 4 + 3] = 0;
		}
		else
		{
			int exp;
			float frac = fast_frexpf(max, &exp);
			float f = frac * 256.0 / max;
			dst[i * 4 + 0] = (uint8_t)(r * f);
			dst[i * 4 + 1] = (uint8_t)(g * f);
			dst[i * 4 + 2] = (uint8_t)(b * f);
			dst[i * 4 + 3] = (uint8_t)(exp + 128);
		}
		n += 4;
	}
	return n;
}
