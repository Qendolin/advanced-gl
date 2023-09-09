package libutil

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	Rad2Deg = float32(180 / math.Pi)
	Deg2Rad = float32(math.Pi / 180)
)

const InvalidAddress uintptr = 0xffff_ffff_ffff_ffff

type Deleter interface {
	Delete()
}

func Hsl2rgb(hsl mgl32.Vec3) mgl32.Vec3 {
	var q, p, r, g, b float32

	h, s, l := hsl[0], hsl[1], hsl[2]

	if s == 0 {
		r, g, b = l, l, l // achromatic
	} else {
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p = 2*l - q
		r = hue2rgb(p, q, h+1./3.)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1./3.)
	}
	return mgl32.Vec3{r, g, b}
}

func hue2rgb(p, q, h float32) float32 {
	if h < 0 {
		h += 1
	} else if h > 1 {
		h -= 1
	}

	if 6*h < 1 {
		return p + ((q - p) * 6 * h)
	}
	if 2*h < 1 {
		return q
	}
	if 3*h < 2 {
		return p + ((q - p) * 6 * ((2. / 3.) - h))
	}

	return p
}

// https://math.stackexchange.com/a/1681815/1014081
func Perpendicular(v mgl32.Vec3) mgl32.Vec3 {
	lx := v[0] * v[0]
	ly := v[1] * v[1]
	lz := v[2] * v[2]

	smallest := lx
	index := 0
	if smallest > ly {
		smallest = ly
		index = 1
	}
	if smallest > lz {
		index = 2
	}
	e := mgl32.Vec3{}
	e[index] = 1
	return v.Cross(e)
}

func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
