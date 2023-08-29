package ibl

const MagicNumberIBLENV = 0x78b85411

type CubeMapFace int

const (
	CubeMapRight = CubeMapFace(iota)
	CubeMapLeft
	CubeMapTop
	CubeMapBottom
	CubeMapBack
	CubeMapFront
)
const (
	CubeMapPositiveX = CubeMapFace(iota)
	CubeMapNegativeX
	CubeMapPositiveY
	CubeMapNegativeY
	CubeMapPositiveZ
	CubeMapNegativeZ
)

type IblEnvVersion uint32

const (
	IblEnvVersion1_001_000 = IblEnvVersion(1_001_000)
)

type IblEnvCompression uint32

const (
	IblEnvCompressionNone = IblEnvCompression(iota)
	IblEnvCompressionLZ4Fast
	IblEnvCompressionLZ4
)

type IblEnvHeader struct {
	Check       uint32
	Version     IblEnvVersion
	Compression IblEnvCompression
	Size        uint32
}

type IblEnv struct {
	Faces [6][]float32
	Size  int
	data  []float32
}

func NewIblEnv(data []float32, size int) *IblEnv {
	o := size * size * 3

	faces := [6][]float32{
		data[0*0 : 1*o : 1*o],
		data[1*o : 2*o : 2*o],
		data[2*o : 3*o : 3*o],
		data[3*o : 4*o : 4*o],
		data[4*o : 5*o : 5*o],
		data[5*o : 6*o : 6*o],
	}

	return &IblEnv{
		Size:  size,
		data:  data,
		Faces: faces,
	}
}

func (env *IblEnv) Concat() []float32 {
	return env.data
}

func (env *IblEnv) Side(face int) []float32 {
	return env.Faces[face]
}
