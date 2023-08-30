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
	IblEnvVersion1_002_000 = IblEnvVersion(1_002_000)
)

type IblEnvCompression uint32

const (
	IblEnvCompressionNone = IblEnvCompression(iota)
	IblEnvCompressionLZ4Fast
	IblEnvCompressionLZ4
)

type iblEnvHeader1_001_000 struct {
	Check       uint32
	Version     IblEnvVersion
	Compression IblEnvCompression
	Size        uint32
}

type iblEnvHeader1_002_000 struct {
	iblEnvHeader1_001_000
	Levels uint32
}

type IblEnvHeader struct {
	Check       uint32
	Version     IblEnvVersion
	Compression IblEnvCompression
	Size        uint32
	Levels      uint32
}

type IblEnv struct {
	Levels   int
	BaseSize int
	faces    [][6][]float32
	sizes    []int
	data     []float32
	levels   [][]float32
}

func NewIblEnv(data []float32, size int, levels int) *IblEnv {
	if levels == 0 {
		levels = 1
	}

	levelsConcat := make([][]float32, levels)
	faces := make([][6][]float32, levels)
	lvlsize := size
	sizes := make([]int, levels)
	for lvl := 0; lvl < levels; lvl++ {
		offset, end := calcCubeMapOffset(size, lvl)
		offset *= 3
		end *= 3
		stride := lvlsize * lvlsize * 3
		faces[lvl] = [6][]float32{
			data[offset+0*stride : offset+1*stride : offset+1*stride],
			data[offset+1*stride : offset+2*stride : offset+2*stride],
			data[offset+2*stride : offset+3*stride : offset+3*stride],
			data[offset+3*stride : offset+4*stride : offset+4*stride],
			data[offset+4*stride : offset+5*stride : offset+5*stride],
			data[offset+5*stride : offset+6*stride : offset+6*stride],
		}
		levelsConcat[lvl] = data[offset:end:end]
		sizes[lvl] = lvlsize
		lvlsize /= 2
	}

	return &IblEnv{
		Levels:   levels,
		BaseSize: size,
		faces:    faces,
		levels:   levelsConcat,
		data:     data,
		sizes:    sizes,
	}
}

func (env *IblEnv) All() []float32 {
	return env.data
}

func (env *IblEnv) Face(level int, face int) []float32 {
	return env.faces[level][face]
}

func (env *IblEnv) Level(level int) []float32 {
	return env.levels[level]
}

func (env *IblEnv) Size(level int) int {
	return env.sizes[level]
}

func calcCubeMapPixels(size int, levels int) int {
	sum := size * size * 6
	for i := 1; i < levels; i++ {
		size /= 2
		sum += size * size * 6
		if size == 1 {
			break
		}
	}
	return sum
}

func calcCubeMapOffset(size int, level int) (start, end int) {
	for i := 0; i <= level; i++ {
		len := size * size * 6
		end += len
		start = end - len
		size /= 2
		if size == 0 {
			break
		}
	}
	return
}
