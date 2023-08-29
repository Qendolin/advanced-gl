#include <stdint.h>
#include "float_ops.h"

// compile with `clang -masm=intel -mno-red-zone -mavx2 -mfma -mstackrealign -mllvm -inline-threshold=1000 -fno-asynchronous-unwind-tables -fno-exceptions -S -Ofast -o c/decode._s -c c/decode.c`
// then convert to go asm with `c2goasm -a c/decode._s decode_amd64.s`

// https://github.com/KnightOS/libc/blob/c1ab6948303ada88f6ab0923035413b8be92a209/src/gpl/ldexpf.c
__attribute__((always_inline)) static inline float fast_ldexpf(float a, int pw2)
{
	ieee_float_shape_type fl;
	uint32_t e;

	fl.value = a;

	e = (fl.word >> 23) & 0x000000ff;
	e += pw2;
	fl.word = ((e & 0xff) << 23) | (fl.word & 0x807fffff);

	return (fl.value);
}

uint64_t DecodeRgbe(uint64_t components, uint64_t len, uint8_t *src, float *dst)
{
	if (components != 3 && components != 4)
		return 0;

	uint64_t n = 0;
	for (uint32_t i = 0, j = 0; i < len; i += 4, j += components)
	{
		uint8_t r = src[i + 0];
		uint8_t g = src[i + 1];
		uint8_t b = src[i + 2];
		uint8_t e = src[i + 3];

		if (e == 0.0f)
		{
			if (components == 4)
			{
				dst[j + 0] = 0.0;
				dst[j + 1] = 0.0;
				dst[j + 2] = 0.0;
				dst[j + 3] = 0.0;
				n += 4;
			}
			else
			{
				dst[j + 0] = 0.0;
				dst[j + 1] = 0.0;
				dst[j + 2] = 0.0;
				n += 3;
			}
			continue;
		}

		float f = fast_ldexpf(1.0, (int)(e) - (128 + 8));

		if (components == 4)
		{
			dst[j + 0] = (float)(r)*f;
			dst[j + 1] = (float)(g)*f;
			dst[j + 2] = (float)(b)*f;
			dst[j + 3] = 1.0;
			n += 4;
		}
		else
		{
			dst[j + 0] = (float)(r)*f;
			dst[j + 1] = (float)(g)*f;
			dst[j + 2] = (float)(b)*f;
			n += 3;
		}
	}

	return n;
}
