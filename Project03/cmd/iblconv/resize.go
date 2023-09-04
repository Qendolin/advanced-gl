package main

import (
	"advanced-gl/Project03/ibl"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type resizeArgs struct {
	commonArgs
	sizeImplArgs
	samples int
}

func createResizeCommand() *command {
	args := resizeArgs{
		commonArgs: commonArgs{
			ext:      ".iblenv",
			suffix:   "_resized",
			compress: 2,
		},
		sizeImplArgs: sizeImplArgs{
			impl: implCl,
			size: size{
				unit:    unitPercent,
				percent: 100,
			},
		},
		samples: 5,
	}

	flags := flag.NewFlagSet("resize", flag.ExitOnError)

	registerCommonFlags(flags, &args.commonArgs)
	registerSizeImplFlag(flags, &args.sizeImplArgs)
	flags.IntVar(&args.samples, "samples", args.samples, "supersampling samples")

	return &command{
		Name: "resize",
		Help: "resize ibl environments",
		Run: func(self *command) {
			if self.Flags.NArg() < 1 || args.compress < 0 || args.compress > 10 {
				printCommandUsage(self, " file-glob...")
			}
			setCommonArgs(&args.commonArgs)

			runResize(args, gatherInputFiles(self.Flags.Args()))
		},
		Flags: flags,
	}
}

func runResize(args resizeArgs, inputFiles []string) {
	var err error
	var resizer ibl.Resizer

	switch args.impl {
	case implCl:
		resizer, err = ibl.NewClResizer(args.device.clDevice(), args.samples)
		if err == nil {
			defer resizer.Release()
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
		resizer = ibl.NewSwResizer(args.samples)
		if !cargs.quiet {
			fmt.Println("Using software implementation")
		}
	}

	ext := cargs.suffix + cargs.ext
	success := 0
	start := time.Now()
	for i, p := range inputFiles {
		if !cargs.quiet {
			fmt.Printf("Processing file %d/%d %q ...\n", i+1, len(inputFiles), filepath.ToSlash(filepath.Clean(p)))
		}
		err := resizeFile(args, p, ext, resizer)
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

func resizeFile(args resizeArgs, p string, ext string, resizer ibl.Resizer) error {
	inFile, err := os.Open(p)
	if err != nil {
		return err
	}
	defer close(inFile)

	hdri, err := ibl.DecodeIblEnv(inFile)
	if err != nil {
		return err
	}

	srcSize := hdri.BaseSize
	dstSize := args.size.Calc(hdri.BaseSize)
	if !cargs.quiet {
		fmt.Printf("Resizing from %dx%d to %dx%d cubemap ...\n", srcSize, srcSize, dstSize, dstSize)
	}

	outFilename := filepath.Join(cargs.out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+ext)
	outFile, err := os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer close(outFile)

	result, err := resizer.Resize(hdri, dstSize)
	if err != nil {
		return err
	}

	if !cargs.quiet {
		fmt.Printf("Writing %q ...\n", filepath.ToSlash(filepath.Clean(outFilename)))
	}

	err = ibl.EncodeIblEnv(outFile, result, ibl.OptCompress(cargs.compress-1))
	if err != nil {
		outFile.Close()
		os.Remove(outFilename)
		return err
	}

	return nil
}
