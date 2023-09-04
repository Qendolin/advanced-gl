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

type updateArgs struct {
	commonArgs
}

func createUpdateCommand() *command {
	args := updateArgs{
		commonArgs: commonArgs{
			ext:      ".iblenv",
			suffix:   "_updated",
			compress: 2,
		},
	}

	flags := flag.NewFlagSet("update", flag.ExitOnError)

	registerCommonFlags(flags, &args.commonArgs)

	return &command{
		Name: "update",
		Help: "update ibl environment file to latest version",
		Run: func(self *command) {
			if self.Flags.NArg() < 1 || args.compress < 0 || args.compress > 10 {
				printCommandUsage(self, " file-glob...")
			}
			setCommonArgs(&args.commonArgs)

			runUpdate(args, gatherInputFiles(self.Flags.Args()))
		},
		Flags: flags,
	}
}

func runUpdate(args updateArgs, inputFiles []string) {
	ext := cargs.suffix + cargs.ext

	success := 0
	start := time.Now()
	for i, p := range inputFiles {
		if !cargs.quiet {
			fmt.Printf("Processing file %d/%d %q ...\n", i+1, len(inputFiles), filepath.ToSlash(filepath.Clean(p)))
		}
		err := updateFile(args, p, ext)
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

func updateFile(args updateArgs, p, ext string) error {
	inFile, err := os.Open(p)
	if err != nil {
		return err
	}
	defer close(inFile)

	src, err := ibl.DecodeOldIblEnv(inFile)
	if err != nil {
		return err
	}

	outFilename := filepath.Join(cargs.out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+ext)
	outFile, err := os.OpenFile(outFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer close(outFile)

	err = ibl.EncodeIblEnv(outFile, src, ibl.OptCompress(cargs.compress-1))
	if err != nil {
		outFile.Close()
		os.Remove(outFilename)
		return err
	}

	return nil
}
