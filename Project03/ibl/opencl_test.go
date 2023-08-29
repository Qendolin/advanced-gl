package ibl_test

import (
	"advanced-gl/Project03/ibl"
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

	var iblEnv *ibl.IblEnv
	onMain <- func() {
		iblEnv, err = conv.Convert(testdata.hdr, testdata.hdr.Rect.Dx()/4)
	}
	<-onMainDone
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), iblEnv)

	expected := []float32{0.22190419, 0.17548445, 0.12154484, 0.20652103, 0.20577157, 0.17545599}

	for i := 0; i < 6; i++ {
		is := iblEnv.Faces[i][len(iblEnv.Faces[i])-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestConvolveCl(t *testing.T) {
	var conv ibl.Convolver
	var err error

	onMain <- func() {
		conv, err = ibl.NewClConvolver(ibl.DeviceTypeGPU, 48)
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

	var iblEnv *ibl.IblEnv
	onMain <- func() {
		iblEnv, err = conv.Convolve(testdata.iblEnv, 32)
	}
	<-onMainDone
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), iblEnv)

	expected := []float32{0.09986539, 0.09226462, 0.088060774, 0.10043078, 0.09293459, 0.09899382}

	for i := 0; i < 6; i++ {
		is := iblEnv.Faces[i][len(iblEnv.Faces[i])-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}
