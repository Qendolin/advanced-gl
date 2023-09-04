package ibl

import (
	"advanced-gl/Project03/libio"
	"advanced-gl/Project03/stbi"
	_ "embed"
	"fmt"
	"unsafe"

	"github.com/Qendolin/go-opencl/cl"
	"golang.org/x/exp/slices"
)

//go:embed convert.cl
var openclConvertSrc string

//go:embed convolve.cl
var openclConvolveSrc string

//go:embed brdf.cl
var openclBrdfSrc string

//go:embed resize.cl
var openclResizeSrc string

//go:embed shared.cl
var openclSharedSrc string

type clCore struct {
	context *cl.Context
	queue   *cl.CommandQueue
	program *cl.Program
}

type clConverter struct {
	clCore
	kernel *cl.Kernel
}

type clDiffuseConvolver struct {
	clCore
	kernel  *cl.Kernel
	samples *cl.MemObject
}

type clSpecularConvolver struct {
	clCore
	kernel       *cl.Kernel
	samples      *cl.MemObject
	samplesIndex [][2]int
	levels       int
	resizer      *clResizer
}

type clResizer struct {
	clCore
	kernel  *cl.Kernel
	samples *cl.MemObject
}

type DeviceType = cl.DeviceType

const (
	DeviceTypeCPU         = DeviceType(cl.DeviceTypeCPU)
	DeviceTypeGPU         = DeviceType(cl.DeviceTypeGPU)
	DeviceTypeAccelerator = DeviceType(cl.DeviceTypeAccelerator)
)

func newClCore(preferredDevice DeviceType, programs ...string) (core *clCore, err error) {
	platforms, err := cl.GetPlatforms()
	if err != nil {
		return nil, err
	}

	var devices []*cl.Device
	for _, p := range platforms {
		devs, err := p.GetDevices(cl.DeviceTypeAll)
		if err != nil {
			continue
		}
		devices = append(devices, devs...)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no opencl devices found")
	}

	slices.SortFunc(devices, func(a, b *cl.Device) int {
		if a.Type() == preferredDevice && b.Type() != preferredDevice {
			return -1
		}
		if a.Type() != preferredDevice && b.Type() == preferredDevice {
			return 1
		}

		aPower := a.MaxComputeUnits() * a.MaxClockFrequency()
		bPower := b.MaxComputeUnits() * b.MaxClockFrequency()

		return aPower - bPower
	})

	device := devices[0]

	ctx, err := cl.CreateContext([]*cl.Device{device})
	if err != nil {
		return nil, err
	}

	queue, err := ctx.CreateCommandQueue(device, 0)
	if err != nil {
		return nil, err
	}

	prog, err := ctx.CreateProgramWithSource(programs)
	if err != nil {
		return nil, err
	}
	err = prog.BuildProgram(nil, "")
	if err != nil {
		return nil, err
	}

	return &clCore{
		context: ctx,
		queue:   queue,
		program: prog,
	}, nil
}

func NewClConverter(preferredDevice DeviceType) (conv Converter, err error) {
	core, err := newClCore(preferredDevice, openclSharedSrc, openclConvertSrc)
	if err != nil {
		return nil, err
	}
	kernel, err := core.program.CreateKernel("reproject_environment")
	if err != nil {
		return nil, err
	}

	return &clConverter{
		clCore: *core,
		kernel: kernel,
	}, nil
}

func (conv *clConverter) Convert(hdri *stbi.RgbaHdr, size int) (*IblEnv, error) {
	srcImage, err := conv.context.CreateImage(cl.MemReadOnly|cl.MemCopyHostPtr, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:   cl.MemObjectTypeImage2D,
		Width:  hdri.Rect.Dx(),
		Height: hdri.Rect.Dy(),
	}, len(hdri.Pix)*4, unsafe.Pointer(&hdri.Pix[0]))
	if err != nil {
		return nil, err
	}
	defer srcImage.Release()

	dstImage, err := conv.context.CreateImage(cl.MemWriteOnly, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:      cl.MemObjectTypeImage2DArray,
		Width:     size,
		Height:    size,
		ArraySize: 6,
	}, size*size*6*4*4, nil)
	if err != nil {
		return nil, err
	}
	defer dstImage.Release()

	err = conv.kernel.SetArgBuffer(0, srcImage)
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgBuffer(1, dstImage)
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgInt32(2, int32(size))
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgFloat32(3, 1.0/float32(size))
	if err != nil {
		return nil, err
	}

	localWorkSize := []int{32, 32, 1}
	globalWorkSize := []int{roundUpKernelSize(localWorkSize[0], size), roundUpKernelSize(localWorkSize[1], size), 6}

	_, err = conv.queue.EnqueueNDRangeKernel(conv.kernel, []int{0, 0, 0}, globalWorkSize, localWorkSize, nil)
	if err != nil {
		return nil, err
	}

	result := make([]float32, size*size*6*4)
	_, err = conv.queue.EnqueueReadImage(dstImage, true, [3]int{}, [3]int{size, size, 6}, 0, 0, unsafe.Pointer(&result[0]), nil)
	if err != nil {
		return nil, err
	}

	// compact RGBA to RGB
	for i := 0; i < len(result)/4; i++ {
		result[i*3+0] = result[i*4+0]
		result[i*3+1] = result[i*4+1]
		result[i*3+2] = result[i*4+2]
	}
	result = result[: size*size*6*3 : size*size*6*3]

	iblEnv := NewIblEnv(result, size, 1)

	return iblEnv, nil
}

func (conv *clConverter) Release() {
	conv.kernel.Release()
	conv.program.Release()
	conv.queue.Release()
	conv.context.Release()
}

func NewClDiffuseConvolver(preferredDevice DeviceType, quality int) (conv Convolver, err error) {
	core, err := newClCore(preferredDevice, openclSharedSrc, openclConvolveSrc)
	if err != nil {
		return nil, err
	}
	kernel, err := core.program.CreateKernel("convolve_diffuse")
	if err != nil {
		return nil, err
	}

	samples := generateDiffuseConvolutionSamples(quality)
	sampleBuf, err := core.context.CreateBuffer(cl.MemReadOnly|cl.MemCopyHostPtr, len(samples)*int(unsafe.Sizeof(samples[0])), unsafe.Pointer(&samples[0]))
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgBuffer(4, sampleBuf)
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgInt32(5, int32(len(samples)))
	if err != nil {
		return nil, err
	}

	return &clDiffuseConvolver{
		clCore:  *core,
		kernel:  kernel,
		samples: sampleBuf,
	}, nil
}

func (conv *clDiffuseConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
	bpp := 4 * 4

	rgbaData := make([]float32, env.BaseSize*env.BaseSize*6*4)
	rgbData := env.All()
	for i := 0; i < env.BaseSize*env.BaseSize*6; i++ {
		rgbaData[i*4+0] = rgbData[i*3+0]
		rgbaData[i*4+1] = rgbData[i*3+1]
		rgbaData[i*4+2] = rgbData[i*3+2]
		rgbaData[i*4+3] = 1.0
	}

	srcImage, err := conv.context.CreateImage(cl.MemReadOnly|cl.MemCopyHostPtr, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:      cl.MemObjectTypeImage2DArray,
		Width:     env.BaseSize,
		Height:    env.BaseSize,
		ArraySize: 6,
	}, env.BaseSize*env.BaseSize*6*bpp, unsafe.Pointer(&rgbaData[0]))
	if err != nil {
		return nil, err
	}
	defer srcImage.Release()

	dstImage, err := conv.context.CreateImage(cl.MemWriteOnly, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:      cl.MemObjectTypeImage2DArray,
		Width:     size,
		Height:    size,
		ArraySize: 6,
	}, size*size*6*bpp, nil)
	if err != nil {
		return nil, err
	}
	defer dstImage.Release()

	err = conv.kernel.SetArgBuffer(0, srcImage)
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgBuffer(1, dstImage)
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgInt32(2, int32(size))
	if err != nil {
		return nil, err
	}
	err = conv.kernel.SetArgFloat32(3, 1.0/float32(size))
	if err != nil {
		return nil, err
	}

	localWorkSize := []int{32, 32, 1}
	globalWorkSize := []int{roundUpKernelSize(localWorkSize[0], size), roundUpKernelSize(localWorkSize[1], size), 6}

	_, err = conv.queue.EnqueueNDRangeKernel(conv.kernel, []int{0, 0, 0}, globalWorkSize, localWorkSize, nil)
	if err != nil {
		return nil, err
	}

	result := make([]float32, size*size*6*4)
	_, err = conv.queue.EnqueueReadImage(dstImage, true, [3]int{}, [3]int{size, size, 6}, 0, 0, unsafe.Pointer(&result[0]), nil)
	if err != nil {
		return nil, err
	}

	// compact RGBA to RGB
	for i := 0; i < len(result)/4; i++ {
		result[i*3+0] = result[i*4+0]
		result[i*3+1] = result[i*4+1]
		result[i*3+2] = result[i*4+2]
	}
	result = result[: size*size*6*3 : size*size*6*3]

	iblEnv := NewIblEnv(result, size, 1)

	return iblEnv, nil
}

func (conv *clDiffuseConvolver) Release() {
	conv.kernel.Release()
	conv.program.Release()
	conv.queue.Release()
	conv.context.Release()
	conv.samples.Release()
}

func NewClSpecularConvolver(preferredDevice DeviceType, quality, levels int) (conv Convolver, err error) {
	core, err := newClCore(preferredDevice, openclSharedSrc, openclConvolveSrc, openclResizeSrc)
	if err != nil {
		return nil, err
	}
	kernel, err := core.program.CreateKernel("convolve_specular")
	if err != nil {
		return nil, err
	}

	samples := generateSpecularConvolutionSamples(quality, levels)

	sampleCount := len(samples[0])
	samplesIndex := make([][2]int, levels)
	samplesIndex[0] = [2]int{0, len(samples[0])}
	for lvl := 1; lvl < levels; lvl++ {
		prev := samplesIndex[lvl-1]
		samplesIndex[lvl] = [2]int{prev[0] + prev[1], len(samples[lvl])}
		sampleCount += len(samples[lvl])
	}

	sampleBuf, err := core.context.CreateBuffer(cl.MemReadOnly|cl.MemCopyHostPtr, sampleCount*int(unsafe.Sizeof(samples[0][0])), unsafe.Pointer(&samples[0][0]))
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgBuffer(4, sampleBuf)
	if err != nil {
		return nil, err
	}

	resizer, err := newClResizer(core, 11)
	if err != nil {
		return nil, err
	}

	return &clSpecularConvolver{
		clCore:       *core,
		kernel:       kernel,
		samples:      sampleBuf,
		samplesIndex: samplesIndex,
		levels:       levels,
		resizer:      resizer,
	}, nil
}

func (conv *clSpecularConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
	srcImage, err := iblEnvToClBuffer(env, conv.context)
	if err != nil {
		return nil, err
	}
	defer srcImage.Release()

	err = conv.kernel.SetArgBuffer(0, srcImage)
	if err != nil {
		return nil, err
	}

	err = conv.resizer.kernel.SetArgBuffer(0, srcImage)
	if err != nil {
		return nil, err
	}

	pixels := calcCubeMapPixels(size, conv.levels)
	result := make([]float32, pixels*4)
	lvlsize := size
	resizeLevelCl(conv.clCore, conv.resizer.kernel, result, lvlsize)
	lvlsize /= 2

	for lvl := 1; lvl < conv.levels; lvl++ {
		dstImage, err := conv.context.CreateImage(cl.MemWriteOnly, cl.ImageFormat{
			ChannelOrder:    cl.ChannelOrderRGBA,
			ChannelDataType: cl.ChannelDataTypeFloat,
		}, cl.ImageDescription{
			Type:      cl.MemObjectTypeImage2DArray,
			Width:     lvlsize,
			Height:    lvlsize,
			ArraySize: 6,
		}, lvlsize*lvlsize*6*4*4, nil)
		if err != nil {
			return nil, err
		}
		defer dstImage.Release()

		err = conv.kernel.SetArgBuffer(1, dstImage)
		if err != nil {
			return nil, err
		}
		err = conv.kernel.SetArgInt32(2, int32(lvlsize))
		if err != nil {
			return nil, err
		}
		err = conv.kernel.SetArgFloat32(3, 1.0/float32(lvlsize))
		if err != nil {
			return nil, err
		}
		err = conv.kernel.SetArgInt32(5, int32(conv.samplesIndex[lvl][0]))
		if err != nil {
			return nil, err
		}
		err = conv.kernel.SetArgInt32(6, int32(conv.samplesIndex[lvl][1]))
		if err != nil {
			return nil, err
		}

		localWorkSize := []int{32, 32, 1}
		globalWorkSize := []int{roundUpKernelSize(localWorkSize[0], lvlsize), roundUpKernelSize(localWorkSize[1], lvlsize), 6}

		_, err = conv.queue.EnqueueNDRangeKernel(conv.kernel, []int{0, 0, 0}, globalWorkSize, localWorkSize, nil)
		if err != nil {
			return nil, err
		}

		lvlStart, lvlEnd := calcCubeMapOffset(size, lvl)
		lvlResult := result[lvlStart*4 : lvlEnd*4]

		_, err = conv.queue.EnqueueReadImage(dstImage, true, [3]int{}, [3]int{lvlsize, lvlsize, 6}, 0, 0, unsafe.Pointer(&lvlResult[0]), nil)
		if err != nil {
			return nil, err
		}

		dstImage.Release()
		lvlsize /= 2
	}

	// compact RGBA to RGB
	for i := 0; i < len(result)/4; i++ {
		result[i*3+0] = result[i*4+0]
		result[i*3+1] = result[i*4+1]
		result[i*3+2] = result[i*4+2]
	}
	result = result[: pixels*3 : pixels*3]

	iblEnv := NewIblEnv(result, size, conv.levels)

	return iblEnv, nil
}

func (conv *clSpecularConvolver) Release() {
	conv.kernel.Release()
	conv.program.Release()
	conv.queue.Release()
	conv.context.Release()
	conv.samples.Release()
	conv.resizer.Release()
}

func GenerateClBrdfLut(preferredDevice DeviceType, size, quality int) (*libio.FloatImage, error) {
	core, err := newClCore(preferredDevice, openclSharedSrc, openclBrdfSrc)
	if err != nil {
		return nil, err
	}
	defer core.context.Release()
	defer core.program.Release()
	defer core.queue.Release()
	kernel, err := core.program.CreateKernel("integrate_brdf")
	if err != nil {
		return nil, err
	}
	defer kernel.Release()

	samples := generateHammersleySequence(quality)
	sampleBuf, err := core.context.CreateBuffer(cl.MemReadOnly|cl.MemCopyHostPtr, len(samples)*int(unsafe.Sizeof(samples[0])), unsafe.Pointer(&samples[0]))
	if err != nil {
		return nil, err
	}
	defer sampleBuf.Release()

	dstImage, err := core.context.CreateImage(cl.MemWriteOnly, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRG,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:   cl.MemObjectTypeImage2D,
		Width:  size,
		Height: size,
	}, size*size*2*4, nil)
	if err != nil {
		return nil, err
	}
	defer dstImage.Release()

	err = kernel.SetArgBuffer(0, dstImage)
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgInt32(1, int32(size))
	if err != nil {
		return nil, err
	}
	err = kernel.SetArgFloat32(2, 1.0/float32(size))
	if err != nil {
		return nil, err
	}
	err = kernel.SetArgBuffer(3, sampleBuf)
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgInt32(4, int32(len(samples)))
	if err != nil {
		return nil, err
	}

	localWorkSize := []int{32, 32, 1}
	globalWorkSize := []int{roundUpKernelSize(localWorkSize[0], size), roundUpKernelSize(localWorkSize[1], size), 6}

	_, err = core.queue.EnqueueNDRangeKernel(kernel, []int{0, 0, 0}, globalWorkSize, localWorkSize, nil)
	if err != nil {
		return nil, err
	}

	result := make([]float32, size*size*2*4)

	_, err = core.queue.EnqueueReadImage(dstImage, true, [3]int{}, [3]int{size, size, 1}, 0, 0, unsafe.Pointer(&result[0]), nil)
	if err != nil {
		return nil, err
	}

	return libio.NewFloatImage(result, 2, size, size), nil
}

func NewClResizer(preferredDevice DeviceType, supersample int) (resizer Resizer, err error) {
	core, err := newClCore(preferredDevice, openclSharedSrc, openclResizeSrc)
	if err != nil {
		return nil, err
	}

	return newClResizer(core, supersample)
}

func newClResizer(core *clCore, supersample int) (resizer *clResizer, err error) {
	kernel, err := core.program.CreateKernel("resize_environment")
	if err != nil {
		return nil, err
	}

	samples := generateSuperSamples(supersample)

	sampleBuf, err := core.context.CreateBuffer(cl.MemReadOnly|cl.MemCopyHostPtr, len(samples)*int(unsafe.Sizeof(samples[0])), unsafe.Pointer(&samples[0]))
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgBuffer(4, sampleBuf)
	if err != nil {
		return nil, err
	}

	err = kernel.SetArgInt32(5, int32(len(samples)))
	if err != nil {
		return nil, err
	}

	return &clResizer{
		clCore:  *core,
		kernel:  kernel,
		samples: sampleBuf,
	}, nil
}

func (resizer *clResizer) Resize(env *IblEnv, size int) (*IblEnv, error) {

	srcImage, err := iblEnvToClBuffer(env, resizer.context)
	if err != nil {
		return nil, err
	}
	defer srcImage.Release()

	err = resizer.kernel.SetArgBuffer(0, srcImage)
	if err != nil {
		return nil, err
	}

	lvlsize := size
	result := make([]float32, calcCubeMapPixels(size, env.Levels)*4)
	for lvl := 1; lvl < env.Levels; lvl++ {
		lvlStart, lvlEnd := calcCubeMapOffset(size, lvl)
		lvlResult := result[lvlStart*4 : lvlEnd*4]

		err = resizeLevelCl(resizer.clCore, resizer.kernel, lvlResult, lvlsize)
		if err != nil {
			return nil, err
		}
		lvlsize /= 2
	}

	// compact RGBA to RGB
	for i := 0; i < len(result)/4; i++ {
		result[i*3+0] = result[i*4+0]
		result[i*3+1] = result[i*4+1]
		result[i*3+2] = result[i*4+2]
	}
	result = result[: size*size*6*3 : size*size*6*3]

	iblEnv := NewIblEnv(result, size, 1)

	return iblEnv, nil
}

func resizeLevelCl(core clCore, kernel *cl.Kernel, result []float32, size int) error {
	dstImage, err := core.context.CreateImage(cl.MemWriteOnly, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:      cl.MemObjectTypeImage2DArray,
		Width:     size,
		Height:    size,
		ArraySize: 6,
	}, size*size*6*4*4, nil)
	if err != nil {
		return err
	}
	defer dstImage.Release()

	err = kernel.SetArgBuffer(1, dstImage)
	if err != nil {
		return err
	}
	err = kernel.SetArgInt32(2, int32(size))
	if err != nil {
		return err
	}
	err = kernel.SetArgFloat32(3, 1.0/float32(size))
	if err != nil {
		return err
	}

	localWorkSize := []int{32, 32, 1}
	globalWorkSize := []int{roundUpKernelSize(localWorkSize[0], size), roundUpKernelSize(localWorkSize[1], size), 6}

	_, err = core.queue.EnqueueNDRangeKernel(kernel, []int{0, 0, 0}, globalWorkSize, localWorkSize, nil)
	if err != nil {
		return err
	}

	_, err = core.queue.EnqueueReadImage(dstImage, true, [3]int{}, [3]int{size, size, 6}, 0, 0, unsafe.Pointer(&result[0]), nil)
	return err
}

func (resizer *clResizer) Release() {
	resizer.kernel.Release()
	resizer.program.Release()
	resizer.queue.Release()
	resizer.context.Release()
	resizer.samples.Release()
}

func roundUpKernelSize(groupSize, globalSize int) int {
	r := globalSize % groupSize
	if r == 0 {
		return globalSize
	}
	return globalSize + groupSize - r
}

func iblEnvToClBuffer(env *IblEnv, ctx *cl.Context) (*cl.MemObject, error) {
	bpp := 4 * 4

	rgbaData := make([]float32, env.BaseSize*env.BaseSize*6*4)
	rgbData := env.All()
	for i := 0; i < env.BaseSize*env.BaseSize*6; i++ {
		rgbaData[i*4+0] = rgbData[i*3+0]
		rgbaData[i*4+1] = rgbData[i*3+1]
		rgbaData[i*4+2] = rgbData[i*3+2]
		rgbaData[i*4+3] = 1.0
	}

	return ctx.CreateImage(cl.MemReadOnly|cl.MemCopyHostPtr, cl.ImageFormat{
		ChannelOrder:    cl.ChannelOrderRGBA,
		ChannelDataType: cl.ChannelDataTypeFloat,
	}, cl.ImageDescription{
		Type:      cl.MemObjectTypeImage2DArray,
		Width:     env.BaseSize,
		Height:    env.BaseSize,
		ArraySize: 6,
	}, env.BaseSize*env.BaseSize*6*bpp, unsafe.Pointer(&rgbaData[0]))
}
