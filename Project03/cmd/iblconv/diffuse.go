package main

import (
	"advanced-gl/Project03/ibl"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type convolveArgs struct {
	commonArgs
	sizeImplArgs
	samples int
}

func createConvolveCommand() *command {
	args := convolveArgs{
		commonArgs: commonArgs{
			ext:      ".iblenv",
			suffix:   "_diffuse",
			compress: 2,
		},
		sizeImplArgs: sizeImplArgs{
			impl: implCl,
			size: size{
				unit:  unitPixel,
				pixel: 32,
			},
			device: deviceGpu,
		},
		samples: 128,
	}

	flags := flag.NewFlagSet("diffuse", flag.ExitOnError)

	registerCommonFlags(flags, &args.commonArgs)
	registerSizeImplFlag(flags, &args.sizeImplArgs)

	flags.IntVar(&args.samples, "samples", args.samples, "number of samples used for convolution")

	return &command{
		Name: "diffuse",
		Help: "create diffuse irradiance map",
		Run: func(self *command) {
			if self.Flags.NArg() < 1 || args.compress < 0 || args.compress > 10 {
				printCommandUsage(self, " file-glob...")
			}
			setCommonArgs(&args.commonArgs)

			runConvolve(args, gatherInputFiles(self.Flags.Args()))
		},
		Flags: flags,
	}
}

func runConvolve(args convolveArgs, inputFiles []string) {
	runtime.LockOSThread()

	ext := cargs.suffix + cargs.ext

	var err error
	var conv ibl.Convolver

	switch args.impl {
	case implCl:
		conv, err = ibl.NewClDiffuseConvolver(args.device.clDevice(), args.samples)
		if err == nil {
			defer conv.Release()
			if !cargs.quiet {
				fmt.Println("Using OpenCL implementation")
			}
			break
		}
		softerr(err)
		if !cargs.quiet {
			fmt.Println("Falling back to software implementation")
		}
		fallthrough
	case implSw:
		conv = ibl.NewSwDiffuseConvolver(args.samples)
		if !cargs.quiet {
			fmt.Println("Using software implementation")
		}
	}

	success := 0
	start := time.Now()
	for i, p := range inputFiles {
		if !cargs.quiet {
			fmt.Printf("Processing file %d/%d %q ...\n", i+1, len(inputFiles), filepath.ToSlash(filepath.Clean(p)))
		}
		err := convolveFile(args, p, ext, conv)
		softerr(err)
		if err == nil {
			success++
		}
	}
	if !cargs.quiet {
		took := float32(time.Since(start).Milliseconds()) / 1000
		fmt.Printf("Convolved %d/%d files in %.3f seconds\n", success, len(inputFiles), took)
	}
}

func convolveFile(args convolveArgs, p string, ext string, conv ibl.Convolver) error {
	inFile, err := os.Open(p)
	if err != nil {
		return err
	}
	defer close(inFile)

	src, err := ibl.DecodeIblEnv(inFile)
	if err != nil {
		return err
	}

	outFilename := filepath.Join(cargs.out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+ext)
	outFile, err := os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer close(outFile)

	var dst io.Writer = outFile

	if src.BaseSize == 0 {
		return fmt.Errorf("image has zero size")
	}

	size := args.size.Calc(src.BaseSize)
	if !cargs.quiet {
		fmt.Printf("Convolving to %dx%d cubemap ...\n", size, size)
	}

	iblEnv, err := conv.Convolve(src, size)

	if err != nil {
		return err
	}

	if !cargs.quiet {
		fmt.Printf("Writing %q ...\n", filepath.ToSlash(filepath.Clean(outFilename)))
	}

	err = ibl.EncodeIblEnv(dst, iblEnv, ibl.OptCompress(cargs.compress-1))
	if err != nil {
		outFile.Close()
		os.Remove(outFilename)
		return err
	}

	return nil
}
