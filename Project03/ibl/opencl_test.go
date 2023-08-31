package ibl_test

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libio"
	"math"
	"testing"
)

func TestConvertCl(t *testing.T) {
	var conv ibl.Converter
	var err error

	onMain <- func() {
		conv, err = ibl.NewClConverter(ibl.DeviceTypeCPU)
	}
	<-onMainDone

	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		onMain <- func() {
			conv.Release()
		}
		<-onMainDone
	}()

	var hdri *ibl.IblEnv
	onMain <- func() {
		hdri, err = conv.Convert(testdata.hdr, testdata.hdr.Rect.Dx()/4)
	}
	<-onMainDone
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)

	expected := []float32{0.22190419, 0.17548445, 0.12154484, 0.20652103, 0.20577157, 0.17545599}

	for i := 0; i < 6; i++ {
		is := hdri.Face(0, i)[len(hdri.Face(0, i))-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestConvolveDiffuseCl(t *testing.T) {
	var conv ibl.Convolver
	var err error

	onMain <- func() {
		conv, err = ibl.NewClDiffuseConvolver(ibl.DeviceTypeGPU, 48)
	}
	<-onMainDone

	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		onMain <- func() {
			conv.Release()
		}
		<-onMainDone
	}()

	var hdri *ibl.IblEnv
	onMain <- func() {
		hdri, err = conv.Convolve(testdata.iblEnv, 32)
	}
	<-onMainDone
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)

	expected := []float32{0.09986539, 0.09226462, 0.088060774, 0.10043078, 0.09293459, 0.09899382}

	for i := 0; i < 6; i++ {
		is := hdri.Face(0, i)[len(hdri.Face(0, i))-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestConvolveSpecularCl(t *testing.T) {
	var conv ibl.Convolver
	var err error

	onMain <- func() {
		conv, err = ibl.NewClSpecularConvolver(ibl.DeviceTypeGPU, 2048, 5)
	}
	<-onMainDone

	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		onMain <- func() {
			conv.Release()
		}
		<-onMainDone
	}()

	var hdri *ibl.IblEnv
	onMain <- func() {
		hdri, err = conv.Convolve(testdata.iblEnv, 128)
	}
	<-onMainDone
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)
}

func TestIntegrateBrdfSpecularCl(t *testing.T) {
	var err error
	var img *libio.FloatImage
	onMain <- func() {
		img, err = ibl.GenerateClBrdfLut(ibl.DeviceTypeGPU, 512, 1024)
	}
	<-onMainDone

	if err != nil {
		t.Fatal(err)
	}

	saveResultFloatImage(t.Name(), img, 1.0, 1.0)
}
