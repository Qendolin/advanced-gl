package ibl

import (
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

type clCore struct {
	context *cl.Context
	queue   *cl.CommandQueue
	program *cl.Program
}

type clConverter struct {
	clCore
	kernel *cl.Kernel
}

type clConvolver struct {
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

func newClCore(preferredDevice DeviceType, program string) (core *clCore, err error) {
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

	prog, err := ctx.CreateProgramWithSource([]string{program})
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
	core, err := newClCore(preferredDevice, openclConvertSrc)
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

func NewClConvolver(preferredDevice DeviceType, quality int) (conv Convolver, err error) {
	core, err := newClCore(preferredDevice, openclConvolveSrc)
	if err != nil {
		return nil, err
	}
	kernel, err := core.program.CreateKernel("convolve_cubemap")
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

	return &clConvolver{
		clCore:  *core,
		kernel:  kernel,
		samples: sampleBuf,
	}, nil
}

func (conv *clConvolver) Convolve(env *IblEnv, size int) (*IblEnv, error) {
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

func (conv *clConvolver) Release() {
	conv.kernel.Release()
	conv.program.Release()
	conv.queue.Release()
	conv.context.Release()
	conv.samples.Release()
}

func roundUpKernelSize(groupSize, globalSize int) int {
	r := globalSize % groupSize
	if r == 0 {
		return globalSize
	}
	return globalSize + groupSize - r
}
