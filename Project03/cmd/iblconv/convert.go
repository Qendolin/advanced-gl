package main

import (
	"advanced-gl/Project03/ibl"
	"advanced-gl/Project03/stbi"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func createConvertCommand() *command {

	args := convertArgs{
		commonArgs: commonArgs{
			ext:  ".iblenv",
			impl: implCl,
			size: size{
				unit:    unitPercent,
				percent: 25,
			},
		},
	}

	flags := flag.NewFlagSet("convert", flag.ExitOnError)

	registerCommonFlags(flags, &args.commonArgs)

	return &command{
		Name: "convert",
		Help: "convert radiance hdr images to ibl environments",
		Run: func(self *command) {
			if self.Flags.NArg() < 1 || args.compress < 0 || args.compress > 10 {
				printCommandUsage(self, " file-glob...")
			}
			setCommonArgs(&args.commonArgs)

			runConvert(args, gatherInputFiles(self.Flags.Args()))
		},
		Flags: flags,
	}
}

func runConvert(args convertArgs, inputFiles []string) {

	runtime.LockOSThread()

	ext := cargs.suffix + cargs.ext
	outFlags := os.O_CREATE | os.O_WRONLY
	if cargs.force {
		outFlags |= os.O_TRUNC
	}

	var err error
	var conv ibl.Converter

	switch cargs.impl {
	case implCl:
		conv, err = ibl.NewClConverter(ibl.DeviceTypeGPU)
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
		conv = ibl.NewSwConverter()
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
		err := convertFile(p, ext, outFlags, conv)
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

func convertFile(p string, ext string, outFlags int, conv ibl.Converter) error {
	inFile, err := os.Open(p)
	if err != nil {
		return err
	}
	defer close(inFile)

	stbi.Default.CopyData = false
	stbi.Default.FlipVertically = true
	hdr, err := stbi.LoadHdr(inFile)
	if err != nil {
		return err
	}
	defer close(hdr)

	outFilename := filepath.Join(cargs.out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+ext)
	outFile, err := os.OpenFile(outFilename, outFlags, 0666)
	if err != nil {
		return err
	}
	defer close(outFile)

	var dst io.Writer = outFile

	if hdr.Rect.Dx() == 0 || hdr.Rect.Dy() == 0 {
		return fmt.Errorf("image has zero size %dx%d", hdr.Rect.Dx(), hdr.Rect.Dy())
	}

	size := cargs.size.Calc(hdr.Rect.Dx())
	if !cargs.quiet {
		fmt.Printf("Converting to %dx%dx6 cubemap ...\n", size, size)
	}

	iblEnv, err := conv.Convert(hdr, size)

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
