package main

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/libio"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type previewArgs struct {
	commonArgs
	gamma    float64
	scale    float64
	device   device
	reinhard bool
}

func createPreviewCommand() *command {

	args := previewArgs{
		commonArgs: commonArgs{
			ext: ".png",
		},
		gamma:    2.2,
		scale:    1.0,
		device:   deviceGpu,
		reinhard: false,
	}

	flags := flag.NewFlagSet("preview", flag.ExitOnError)

	registerCommonFlags(flags, &args.commonArgs)

	flags.Float64Var(&args.gamma, "gamma", args.gamma, "gamma correction value")
	flags.Float64Var(&args.scale, "scale", args.scale, "brightness scale factor")
	flags.Var(&args.device, "device", "the preferred opencl deivce; gpu or cpu")
	flags.BoolVar(&args.reinhard, "reinhard", args.reinhard, "apply reinhard tonemapping")

	return &command{
		Name: "preview",
		Help: "render ibl environments to png",
		Run: func(self *command) {
			if self.Flags.NArg() < 1 || args.compress < 0 || args.compress > 10 {
				printCommandUsage(self, " file-glob...")
			}
			setCommonArgs(&args.commonArgs)

			runPreview(args, gatherInputFiles(self.Flags.Args()))
		},
		Flags: flags,
	}
}

func runPreview(args previewArgs, inputFiles []string) {
	ext := cargs.suffix + cargs.ext
	success := 0
	start := time.Now()
	for i, p := range inputFiles {
		if !cargs.quiet {
			fmt.Printf("Processing file %d/%d %q ...\n", i+1, len(inputFiles), filepath.ToSlash(filepath.Clean(p)))
		}
		err := previewFile(args, p, ext)
		softerr(err)
		if err == nil {
			success++
		}
	}
	if !cargs.quiet {
		took := float32(time.Since(start).Milliseconds()) / 1000
		fmt.Printf("Converted %d/%d files in %.3f seconds\n", success, len(inputFiles), took)
	}
}

func previewFile(args previewArgs, p string, ext string) error {
	inFile, err := os.Open(p)
	if err != nil {
		return err
	}
	defer close(inFile)

	hdri, err := ibl.DecodeIblEnv(inFile)
	if err != nil {
		return err
	}

	size := hdri.BaseSize
	if !cargs.quiet {
		fmt.Printf("Converting to %dx%d png ...\n", size, size*6)
	}

	for i := 0; i < hdri.Levels; i++ {
		outFilename := filepath.Join(cargs.out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+fmt.Sprintf("_%d", i)+ext)
		outFile, err := os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer close(outFile)

		pix := hdri.Level(i)
		fimg := libio.NewFloatImage(pix, 3, hdri.Size(i), hdri.Size(i)*6)
		if args.reinhard {
			for i := 0; i < fimg.Count(); i++ {
				fimg.Pix[i*3+0] = fimg.Pix[i*3+0] / (1 + fimg.Pix[i*3+0])
				fimg.Pix[i*3+1] = fimg.Pix[i*3+1] / (1 + fimg.Pix[i*3+1])
				fimg.Pix[i*3+2] = fimg.Pix[i*3+2] / (1 + fimg.Pix[i*3+2])
			}
		}
		rgba := fimg.ToIntImage(float32(args.gamma), float32(args.scale)).ToRGBA()

		if !cargs.quiet {
			fmt.Printf("Writing %q ...\n", filepath.ToSlash(filepath.Clean(outFilename)))
		}

		err = png.Encode(outFile, rgba)
		if err != nil {
			return err
		}
	}

	return nil
}
