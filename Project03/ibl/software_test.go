package ibl_test

import (
	"advanced-gl/Project03/ibl"
	"math"
	"testing"
)

func TestConvertSw(t *testing.T) {
	conv := ibl.NewSwConverter()

	hdri, err := conv.Convert(testdata.hdr, testdata.hdr.Rect.Dx()/4)
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)

	expected := []float32{0.22190419, 0.17548445, 0.12154484, 0.20652103, 0.20577157, 0.17545599}

	for i := 0; i < 6; i++ {
		is := hdri.Face(0, i)[len(hdri.Face(0, i))-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.0001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestConvolvDiffuseeSw(t *testing.T) {
	conv := ibl.NewSwDiffuseConvolver(48)
	hdri, err := conv.Convolve(testdata.iblEnv, 32)
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)

	expected := []float32{0.09986539, 0.09226462, 0.088060774, 0.10043078, 0.09293459, 0.09899382}

	for i := 0; i < 6; i++ {
		is := hdri.Face(0, i)[len(hdri.Face(0, i))-1]
		should := expected[i]
		if math.Abs(float64(is-should)) > 0.0001 {
			t.Errorf("conversion result incorrect for face %d, should be: %.4f but is %.4f\n", i, should, is)
		}
	}
}

func TestConvolveSpecularSw(t *testing.T) {
	conv := ibl.NewSwSpecularConvolver(2048, 5)
	hdri, err := conv.Convolve(testdata.iblEnv, 128)
	if err != nil {
		t.Fatal(err)
	}

	saveResultIbl(t.Name(), hdri)
}
