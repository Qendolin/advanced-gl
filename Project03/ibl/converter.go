package ibl

import (
	"advanced-gl/Project03/stbi"
)

type Converter interface {
	Convert(hdr *stbi.RgbaHdr, size int) (*IblEnv, error)
	Release()
}
