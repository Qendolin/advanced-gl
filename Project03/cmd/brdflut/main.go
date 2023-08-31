package main

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var args = struct {
	samples int
	size    int
}{
	samples: 1024,
	size:    512,
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

	flag.Parse()

	if flag.NArg() != 1 {
		printGeneralUsage()
	}

	img, err := ibl.GenerateClBrdfLut(ibl.DeviceTypeGPU, args.size, args.samples)
	harderr(err)

	file, err := os.OpenFile(flag.Arg(0), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	harderr(err)

	err = libio.EncodeFloatImage(file, img, libio.FloatImageCompressionFixedPoint16Lz4)
	harderr(err)
}

func harderr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
