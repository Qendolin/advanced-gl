package ibl

type Convolver interface {
	Convolve(env *IblEnv, size int) (*IblEnv, error)
	Release()
}
