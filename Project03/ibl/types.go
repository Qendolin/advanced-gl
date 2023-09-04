package ibl

import "advanced-gl/Project03/stbi"

type Convolver interface {
	Convolve(env *IblEnv, size int) (*IblEnv, error)
	Release()
}

type Resizer interface {
	Resize(env *IblEnv, size int) (*IblEnv, error)
	Release()
}

type Converter interface {
	Convert(hdr *stbi.RgbaHdr, size int) (*IblEnv, error)
	Release()
}
