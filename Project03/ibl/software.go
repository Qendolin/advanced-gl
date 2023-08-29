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

	return NewIblEnv(result, size), nil
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

type swConvolver struct {
	samples []sample
}

func NewSwConvolver(quality int) (conv Convolver) {
	return &swConvolver{
		samples: generateConvolutionSamples(quality),
	}
}

func (conv *swConvolver) Release() {
}

type sample struct {
	// z is 'up'
	x, y, z float32
	weight  float32
}

// quality >= 0
func generateConvolutionSamples(quality int) []sample {

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

func (conv *swConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
	result := make([]float32, size*size*6*3)

	forEachCubeMapPixel(size, func(face, pu, pv int, cx, cy, cz float32, i int) {
		clen := math32.Sqrt(cx*cx + cy*cy + cz*cz)
		nx := cx / clen
		ny := cy / clen
		nz := cz / clen

		// cross(normal, (0, 1, 0))
		rx := ny*0 - nz*1
		ry := nz*0 - nx*0
		rz := nx*1 - ny*0
		rlen := math32.Sqrt(rx*rx + ry*ry + rz*rz)
		rx /= rlen
		ry /= rlen
		rz /= rlen

		// corss(normal, right)
		ux := ny*rz - nz*ry
		uy := nz*rx - nx*rz
		uz := nx*ry - ny*rx
		ulen := math32.Sqrt(ux*ux + uy*uy + uz*uz)
		ux /= ulen
		uy /= ulen
		uz /= ulen

		var cr, cg, cb float32
		var count int
		for _, s := range conv.samples {

			dx := (s.x * rx) + (s.y * ux) + (s.z * nx)
			dy := (s.x * ry) + (s.y * uy) + (s.z * ny)
			dz := (s.x * rz) + (s.y * uz) + (s.z * nz)

			if dx == 0.0 && dy == 0.0 && dz == 0.0 {
				continue
			}

			sface, su, sv := sampleCubeMap(dx, dy, dz)
			sr, sg, sb := sampleBilinear(env.Size, env.Size, 3, env.Faces[sface], su, sv)

			cr += sr * s.weight
			cg += sg * s.weight
			cb += sb * s.weight

			count++
		}

		result[i*3+0] = cr * math32.Pi / float32(count)
		result[i*3+1] = cg * math32.Pi / float32(count)
		result[i*3+2] = cb * math32.Pi / float32(count)
	})

	return NewIblEnv(result, size), nil
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
