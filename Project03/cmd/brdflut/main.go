package main

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libio"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var args = struct {
	samples     int
	size        int
	preview     bool
	grayscale   bool
	compression int
}{
	samples:     1024,
	size:        512,
	preview:     false,
	grayscale:   false,
	compression: 1,
}

func printGeneralUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <out>\n\n", exe)
	fmt.Fprintf(os.Stderr, "The arguments are:\n\n")
	flag.CommandLine.SetOutput(os.Stderr)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.IntVar(&args.samples, "samples", args.samples, "samples of the integral")
	flag.IntVar(&args.size, "size", args.size, "size of the lut")
	flag.BoolVar(&args.preview, "preview", args.preview, "generate normalized preview png")
	flag.BoolVar(&args.grayscale, "grayscale", args.grayscale, "generate seperate grayscale images")
	flag.IntVar(&args.compression, "compression", args.compression, "0=none, 1=fixed-point + lz4-fast")

	flag.Parse()

	if flag.NArg() != 1 {
		printGeneralUsage()
	}

	img, err := ibl.GenerateClBrdfLut(ibl.DeviceTypeGPU, args.size, args.samples)
	harderr(err)

	fileext := path.Ext(flag.Arg(0))
	filename := strings.TrimSuffix(flag.Arg(0), fileext)

	if args.grayscale {
		rimg := img.Shuffle([]int{0})
		gimg := img.Shuffle([]int{1})
		saveFloatImage(rimg, filename+"_r", fileext)
		saveFloatImage(gimg, filename+"_g", fileext)
	} else {
		saveFloatImage(img, filename, fileext)
	}
}

func saveFloatImage(img *libio.FloatImage, filename, fileext string) {
	file, err := os.OpenFile(filename+fileext, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	harderr(err)
	defer file.Close()

	err = libio.EncodeFloatImage(file, img, libio.FloatImageCompression(args.compression))
	harderr(err)

	if args.preview {
		if args.grayscale {
			img = img.Shuffle([]int{0, 0, 0})
		}

		file, err = os.OpenFile(filename+".png", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		harderr(err)

		img.Normalize()
		rgba := img.ToIntImage().ToRGBA()
		err = png.Encode(file, rgba)
		harderr(err)
	}
}

func harderr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
