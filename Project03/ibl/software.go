package ibl

import (
	"advanced-gl/Project03/stbi"

	"github.com/chewxy/math32"
)

type swConverter struct{}

func NewSwConverter() (conv Converter) {
	return &swConverter{}
}

func (*swConverter) Convert(image *stbi.RgbaHdr, size int) (*IblEnv, error) {

	result := make([]float32, 6*size*size*3)

	forEachCubeMapPixel(size, func(face, pu, pv int, cx, cy, cz float32, i int) {
		rx, ry, rz := cx, cy, cz
		l := math32.Sqrt(rx*rx + ry*ry + rz*rz)
		rx /= l
		ry /= l
		rz /= l
		su, sv := sampleSphericalMap(rx, ry, rz)

		sr, sg, sb := sampleBilinear(image.Rect.Dx(), image.Rect.Dy(), 4, image.Pix, su, sv)

		result[i*3+0] = sr
		result[i*3+1] = sg
		result[i*3+2] = sb
	})

	return NewIblEnv(result, size, 1), nil
}

// 1/(2pi), 1/pi
var invAtan [2]float32 = [2]float32{0.15915494309, 0.31830988618}

func sampleSphericalMap(rx, ry, rz float32) (u, v float32) {
	u, v = math32.Atan2(rz, rx), math32.Asin(ry)
	u = u*invAtan[0] + 0.5
	v = v*invAtan[1] + 0.5
	return u, v
}

func sampleBilinear(w, h int, channels int, pix []float32, u, v float32) (r, g, b float32) {
	// -0.5 to adjust for the pixel center offset
	u = u*float32(w) - 0.5
	v = v*float32(h) - 0.5
	ufloor, ufrac := math32.Modf(u)
	vfloor, vfrac := math32.Modf(v)
	ufloori, vfloori := int(ufloor), int(vfloor)
	uceili, vceili := ufloori+1, vfloori+1

	if ufloori < 0 {
		ufloori = 0
	}
	if vfloori < 0 {
		vfloori = 0
	}

	if uceili >= w {
		uceili = w - 1
	}
	if ufloori >= uceili {
		ufloori = uceili
		ufrac = 0.0
	}
	if vceili >= h {
		vceili = h - 1
	}
	if vfloori >= vceili {
		vfloori = vceili
		vfrac = 0.0
	}

	colstride := channels
	rowstride := channels * w

	o00 := vfloori*rowstride + ufloori*colstride
	o10 := vfloori*rowstride + uceili*colstride
	o01 := vceili*rowstride + ufloori*colstride
	o11 := vceili*rowstride + uceili*colstride

	r00, g00, b00 := pix[o00+0], pix[o00+1], pix[o00+2]
	r10, g10, b10 := pix[o10+0], pix[o10+1], pix[o10+2]
	r01, g01, b01 := pix[o01+0], pix[o01+1], pix[o01+2]
	r11, g11, b11 := pix[o11+0], pix[o11+1], pix[o11+2]

	rh0 := r00*(1.0-ufrac) + r10*ufrac
	gh0 := g00*(1.0-ufrac) + g10*ufrac
	bh0 := b00*(1.0-ufrac) + b10*ufrac

	rh1 := r01*(1.0-ufrac) + r11*ufrac
	gh1 := g01*(1.0-ufrac) + g11*ufrac
	bh1 := b01*(1.0-ufrac) + b11*ufrac

	rhh := rh0*(1.0-vfrac) + rh1*vfrac
	ghh := gh0*(1.0-vfrac) + gh1*vfrac
	bhh := bh0*(1.0-vfrac) + bh1*vfrac

	return rhh, ghh, bhh
}

func (conv *swConverter) Release() {
}

type swDiffuseConvolver struct {
	samples []sample
}

func NewSwDiffuseConvolver(quality int) (conv Convolver) {
	return &swDiffuseConvolver{
		samples: generateDiffuseConvolutionSamples(quality),
	}
}

func (conv *swDiffuseConvolver) Release() {
}

type sample struct {
	// z is 'up'
	x, y, z float32
	weight  float32
}

// quality >= 0
func generateDiffuseConvolutionSamples(quality int) []sample {

	if quality == 0 {
		// only one sample directly upwards
		return []sample{{x: 0, y: 0, z: 1, weight: 1}}
	}

	rings := quality + 1
	segments := quality * 4

	dPhi := (2.0 * math32.Pi) / float32(segments)
	dTheta := (math32.Pi / 2.0) / float32(rings)

	samples := make([]sample, (rings-1)*segments+1)
	i := 0
	for ring := 0; ring < rings-1; ring++ {
		theta := (float32(ring) + 0.5) * dTheta
		for segment := 0; segment < segments; segment++ {
			phi := (float32(segment) + 0.5) * dPhi
			x := math32.Sin(theta) * math32.Cos(phi)
			y := math32.Sin(theta) * math32.Sin(phi)
			z := math32.Cos(theta)

			samples[i].x = x
			samples[i].y = y
			samples[i].z = z
			samples[i].weight = math32.Cos(theta) * math32.Sin(theta)
			i++
		}
	}

	poleTheta := (float32(rings-1) + 0.5) * dTheta
	// the last 'ring' accounts for the the segments around the 'pole cap'
	samples[i].x = 0
	samples[i].y = 0
	samples[i].z = 1
	samples[i].weight = float32(segments) * math32.Cos(poleTheta) * math32.Sin(poleTheta)

	return samples
}

// Based on: https://www.gamedev.net/forums/topic/687535-implementing-a-cube-map-lookup-function/5337472/
// Cube map face reference: https://www.khronos.org/opengl/wiki_opengl/images/CubeMapAxes.png
func sampleCubeMap(rx, ry, rz float32) (face int, u, v float32) {
	ax := math32.Abs(rx)
	ay := math32.Abs(ry)
	az := math32.Abs(rz)

	// this normalizes the uvs
	var uvfac float32

	if ax >= ay && ax >= az {
		if rx >= 0 {
			face = 0
			u = -rz
		} else {
			face = 1
			u = rz
		}
		uvfac = 0.5 / ax
		v = -ry
	} else if ay >= ax && ay >= az {
		if ry >= 0 {
			face = 2
			v = rz
		} else {
			face = 3
			v = -rz
		}
		uvfac = 0.5 / ay
		u = rx
	} else {
		if rz >= 0 {
			face = 4
			u = rx
		} else {
			face = 5
			u = -rx
		}
		uvfac = 0.5 / az
		v = -ry
	}

	u = u*uvfac + 0.5
	v = v*uvfac + 0.5

	return
}

func normalize(x, y, z float32) (float32, float32, float32) {
	len := math32.Sqrt(x*x + y*y + z*z)
	return x / len, y / len, z / len
}

func cross(ax, ay, az, bx, by, bz float32) (float32, float32, float32) {
	x := ay*bz - az*by
	y := az*bx - ax*bz
	z := ax*by - ay*bx
	return x, y, z
}

func dot(ax, ay, az, bx, by, bz float32) float32 {
	return ax*bx + ay*by + az*bz
}

func transform(vx, vy, vz, xx, xy, xz, yx, yy, yz, zx, zy, zz float32) (float32, float32, float32) {
	x := (vx * xx) + (vy * yx) + (vz * zx)
	y := (vx * xy) + (vy * yy) + (vz * zy)
	z := (vx * xz) + (vy * yz) + (vz * zz)
	return x, y, z
}

func (conv *swDiffuseConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
	result := make([]float32, calcCubeMapPixels(size, 1)*3)

	forEachCubeMapPixel(size, func(face, pu, pv int, cx, cy, cz float32, i int) {
		nx, ny, nz := normalize(cx, cy, cz)

		var upx, upy, upz float32 = 0.0, 1.0, 0.0
		if math32.Abs(ny) >= 0.999 {
			upx, upy, upz = 0.0, 0.0, 1.0
		}

		// tangent = cross(up, normal)
		tx, ty, tz := normalize(cross(upx, upy, upz, nx, ny, nz))
		// bitangent = corss(normal, right)
		bx, by, bz := normalize(cross(nx, ny, nz, tx, ty, tz))

		var cr, cg, cb float32
		var count int
		for _, s := range conv.samples {

			dx, dy, dz := transform(s.x, s.y, s.z, tx, ty, tz, bx, by, bz, nx, ny, nz)

			if dx == 0.0 && dy == 0.0 && dz == 0.0 {
				continue
			}

			sface, su, sv := sampleCubeMap(dx, dy, dz)
			sr, sg, sb := sampleBilinear(env.BaseSize, env.BaseSize, 3, env.Face(0, sface), su, sv)

			cr += sr * s.weight
			cg += sg * s.weight
			cb += sb * s.weight

			count++
		}

		result[i*3+0] = cr * math32.Pi / float32(count)
		result[i*3+1] = cg * math32.Pi / float32(count)
		result[i*3+2] = cb * math32.Pi / float32(count)
	})

	return NewIblEnv(result, size, 1), nil
}

func forEachCubeMapPixel(resolution int, cb func(face, pu, pv int, cx, cy, cz float32, i int)) {
	face := 0
	index := 0
	// Cube map face reference: https://www.khronos.org/opengl/wiki_opengl/images/CubeMapAxes.png
	for cx := float32(1.0); cx >= -1.0; cx -= 2.0 {
		for dy := 0; dy < resolution; dy++ {
			for dz := 0; dz < resolution; dz++ {
				// (2x+1)/r - 1 is the correct formular to get the center coords of the pixels
				// e.g. for r=3
				// x=0: 1/3 - 1 = 0.33 - 1 = -0.66
				// x=1: 3/3 - 1 = 1.00 - 1 =  0.0
				// x=2: 5/3 - 1 = 1.66 - 1 =  0.66
				cy := (2.0*float32(dy)+1.0)/float32(resolution) - 1.0
				cz := (2.0*float32(dz)+1.0)/float32(resolution) - 1.0
				cy *= -1
				if cx == 1.0 {
					cz *= -1
				}

				cb(face, dz, dy, cx, cy, cz, index)
				index++
			}
		}
		face++
	}

	for cy := float32(1.0); cy >= -1.0; cy -= 2.0 {
		for dz := 0; dz < resolution; dz++ {
			for dx := 0; dx < resolution; dx++ {
				cx := (2.0*float32(dx)+1.0)/float32(resolution) - 1.0
				cz := (2.0*float32(dz)+1.0)/float32(resolution) - 1.0
				if cy == -1.0 {
					cz *= -1
				}

				cb(face, dx, dz, cx, cy, cz, index)
				index++
			}
		}
		face++
	}

	for cz := float32(1.0); cz >= -1.0; cz -= 2.0 {
		for dy := 0; dy < resolution; dy++ {
			for dx := 0; dx < resolution; dx++ {
				cx := (2.0*float32(dx)+1.0)/float32(resolution) - 1.0
				cy := (2.0*float32(dy)+1.0)/float32(resolution) - 1.0
				cy *= -1
				if cz == -1.0 {
					cx *= -1
				}

				cb(face, dx, dy, cx, cy, cz, index)
				index++
			}
		}
		face++
	}
}

type swSpecularConvolver struct {
	samples [][]sample
	levels  int
}

func NewSwSpecularConvolver(quality int, levels int) (conv Convolver) {
	return &swSpecularConvolver{
		samples: generateSpecularConvolutionSamples(quality, levels),
		levels:  levels,
	}
}

func (conv *swSpecularConvolver) Release() {
}

func generateHammersleySequence(count int) [][2]float32 {
	samples := make([][2]float32, count)
	for i := 0; i < count; i++ {
		su, sv := hammersley(uint32(i), uint32(count))
		samples[i][0] = su
		samples[i][1] = sv
	}

	return samples
}

func generateSpecularConvolutionSamples(count int, levels int) [][]sample {
	// store all samples in contiguous memory
	samples := make([]sample, count*(levels-1)+1)
	slicedSamples := make([][]sample, levels)
	// roughtness 0 only requires a single sample
	samples[0].x = 0
	samples[0].y = 0
	samples[0].z = 1
	samples[0].weight = 1.0
	slicedSamples[0] = samples[0:1:1]
	i := 1

	hammersleySeq := generateHammersleySequence(count)

	for l := 1; l < levels; l++ {
		start := i
		roughness := float32(l) / float32(levels-1)
		for si := 0; si < count; si++ {
			hs := hammersleySeq[si]
			hx, hy, hz := importanceSampleGGX(hs[0], hs[1], roughness)
			samples[i].x = hx
			samples[i].y = hy
			samples[i].z = hz
			samples[i].weight = 1.0
			i++
		}
		slicedSamples[l] = samples[start:i:i]
	}

	return slicedSamples
}

func radicalInverseVdC(bits uint32) float32 {
	bits = (bits << 16) | (bits >> 16)
	bits = ((bits & 0x55555555) << 1) | ((bits & 0xAAAAAAAA) >> 1)
	bits = ((bits & 0x33333333) << 2) | ((bits & 0xCCCCCCCC) >> 2)
	bits = ((bits & 0x0F0F0F0F) << 4) | ((bits & 0xF0F0F0F0) >> 4)
	bits = ((bits & 0x00FF00FF) << 8) | ((bits & 0xFF00FF00) >> 8)
	return float32(bits) * 2.3283064365386963e-10 // / 0x100000000
}

func hammersley(i, N uint32) (x, y float32) {
	return float32(i) / float32(N), radicalInverseVdC(i)
}

func importanceSampleGGX(su, sv float32, roughness float32) (x, y, z float32) {
	a := roughness * roughness

	phi := 2.0 * math32.Pi * su
	cosTheta := math32.Sqrt((1.0 - sv) / (1.0 + (a*a-1.0)*sv))
	sinTheta := math32.Sqrt(1.0 - cosTheta*cosTheta)

	// from spherical coordinates to cartesian coordinates
	x = math32.Cos(phi) * sinTheta
	y = math32.Sin(phi) * sinTheta
	z = cosTheta

	return
}

func (conv *swSpecularConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
	result := make([]float32, calcCubeMapPixels(size, conv.levels)*3)
	lvlsize := size
	for lvl := 0; lvl < conv.levels; lvl++ {
		lvlStart, lvlEnd := calcCubeMapOffset(size, lvl)
		lvlResult := result[lvlStart*3 : lvlEnd*3]
		forEachCubeMapPixel(lvlsize, func(face, pu, pv int, cx, cy, cz float32, i int) {
			nx, ny, nz := normalize(cx, cy, cz)
			vx, vy, vz := nx, ny, nz
			// from tangent-space vector to world-space sample vector
			var upx, upy, upz float32 = 0.0, 0.0, 1.0
			if math32.Abs(nz) >= 0.999 {
				upx, upy, upz = 1.0, 0.0, 0.0
			}
			tx, ty, tz := normalize(cross(upx, upy, upz, nx, ny, nz))
			bx, by, bz := cross(nx, ny, nz, tx, ty, tz)

			var cr, cg, cb float32
			var totalWeight float32
			for _, s := range conv.samples[lvl] {

				hx, hy, hz := normalize(transform(s.x, s.y, s.z, tx, ty, tz, bx, by, bz, nx, ny, nz))
				vdoth := 2 * dot(vx, vy, vz, hx, hy, hz)
				lx, ly, lz := normalize(vdoth*hx-vx, vdoth*hy-vy, vdoth*hz-vz)

				ndotl := math32.Max(dot(nx, ny, nz, lx, ly, lz), 0.0)
				if ndotl > 0 {
					sface, su, sv := sampleCubeMap(lx, ly, lz)
					sr, sg, sb := sampleBilinear(env.BaseSize, env.BaseSize, 3, env.Face(0, sface), su, sv)

					cr += sr * ndotl
					cg += sg * ndotl
					cb += sb * ndotl

					totalWeight += ndotl
				}
			}

			lvlResult[i*3+0] = cr / totalWeight
			lvlResult[i*3+1] = cg / totalWeight
			lvlResult[i*3+2] = cb / totalWeight
		})
		lvlsize /= 2
	}

	return NewIblEnv(result, size, conv.levels), nil
}
