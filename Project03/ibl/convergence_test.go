package ibl_test

import (
	"advanced-gl/Project03/ibl"
	"testing"
)

func TestRandomSequence(t *testing.T) {
	*ibl.SampleSequenceImplementation = ibl.RandomSequence
	runSpecularConvolution(t)
}

func TestHammersleySequence(t *testing.T) {
	*ibl.SampleSequenceImplementation = ibl.HammersleySequence
	runSpecularConvolution(t)
}

func TestRobertsSequence(t *testing.T) {
	*ibl.SampleSequenceImplementation = ibl.RobertsSequence
	runSpecularConvolution(t)
}

func runSpecularConvolution(t *testing.T) {
	conv, err := ibl.NewClSpecularConvolver(ibl.DeviceTypeGPU, 4096, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer conv.Release()

	result, err := conv.Convolve(testdata.iblStudioSmall, 128)
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), result)

	var sum float32
	for i := 0; i < len(result.Level(1)); i++ {
		diff := result.Level(1)[i] - testdata.iblStudioSmallSpecularReference.Level(1)[i]
		sum += diff * diff
	}
	mse := sum / float32(len(result.Level(1)))
	t.Logf("%v mse: %f\n", t.Name(), mse)
}
