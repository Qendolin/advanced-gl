package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type impl string

const (
	implCl impl = "opencl"
	implSw impl = "software"
)

func (i *impl) String() string {
	return string(*i)
}

func (i *impl) Set(s string) error {
	switch impl(s) {
	case implCl:
		*i = implCl
	case implSw:
		*i = implSw
	default:
		return fmt.Errorf("%s is not a valid implementation", s)
	}
	return nil
}

type sizeUnit string

const (
	unitPixel   = "px"
	unitPercent = "%"
)

type size struct {
	unit    sizeUnit
	pixel   int32
	percent float64
}

func (sz *size) String() string {
	switch sz.unit {
	case unitPercent:
		return fmt.Sprintf("%s%%", strconv.FormatFloat(sz.percent, 'f', -1, 64))
	case unitPixel:
		return fmt.Sprintf("%dpx", sz.pixel)
	default:
		return ""
	}
}

func (sz *size) Set(s string) error {
	s = strings.TrimSpace(s)
	var err error
	var px int64
	if strings.HasSuffix(s, unitPercent) {
		sz.unit = unitPercent
		sz.percent, err = strconv.ParseFloat(strings.TrimSuffix(s, unitPercent), 64)
	} else if strings.HasSuffix(s, unitPixel) {
		sz.unit = unitPixel
		px, err = strconv.ParseInt(strings.TrimSuffix(s, unitPixel), 10, 32)
		sz.pixel = int32(px)
	}
	return err
}

func (sz *size) Calc(width int) int {
	switch sz.unit {
	case unitPercent:
		return int(math.Round(sz.percent / 100 * float64(width)))
	case unitPixel:
		return int(sz.pixel)
	}
	return 0
}

type commonArgs struct {
	compress int
	out      string
	quiet    bool
	supress  bool
	ext      string
	suffix   string
}

type sizeImplArgs struct {
	size size
	impl impl
}

var cargs *commonArgs

type command struct {
	Run   func(self *command)
	Name  string
	Help  string
	Flags *flag.FlagSet
}

var commands = []*command{}

func printGeneralUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [arguments]\n\n", exe)
	fmt.Fprintf(os.Stderr, "The commands are:\n\n")
	longest := slices.MaxFunc(commands, func(a, b *command) int {
		return len(a.Name) - len(b.Name)
	})
	for _, c := range commands {
		fmt.Fprintf(os.Stderr, "    %*s%s\n", -len(longest.Name)-4, c.Name, c.Help)
	}
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}

func printCommandUsage(cmd *command, suffix string) {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s %s [arguments]%s\n\n", exe, cmd.Name, suffix)
	fmt.Fprintf(os.Stderr, "The arguments are:\n\n")
	cmd.Flags.SetOutput(os.Stderr)
	cmd.Flags.PrintDefaults()
	os.Exit(1)
}

func main() {
	commands = append(commands, createConvertCommand())
	commands = append(commands, createConvolveCommand())
	commands = append(commands, createUpdateCommand())
	commands = append(commands, createPrefilterCommand())

	slices.SortFunc(commands, func(a, b *command) int {
		return strings.Compare(a.Name, b.Name)
	})

	if len(os.Args) < 2 {
		printGeneralUsage()
	}

	var cmd *command
	for _, c := range commands {
		if strings.EqualFold(c.Name, os.Args[1]) {
			cmd = c
			break
		}
	}
	if cmd == nil {
		printGeneralUsage()
	}

	err := cmd.Flags.Parse(os.Args[2:])
	harderr(err)

	cmd.Run(cmd)
}

func registerCommonFlags(flags *flag.FlagSet, args *commonArgs) {
	flags.IntVar(&args.compress, "compress", args.compress, "the compression level from 0 (none) to 10 (high)")
	flags.IntVar(&args.compress, "c", args.compress, "shorthand for compress")
	flags.StringVar(&args.out, "out", args.out, "the output directory")
	flags.StringVar(&args.out, "o", args.out, "shorthand for out")
	flags.BoolVar(&args.quiet, "quiet", args.quiet, "disables informational logging")
	flags.BoolVar(&args.quiet, "q", args.quiet, "shorthand for quiet")
	flags.BoolVar(&args.supress, "supress", args.supress, "disables soft error logging")
	flags.StringVar(&args.ext, "ext", args.ext, "the result file extension")
	flags.StringVar(&args.suffix, "suffix", args.suffix, "the result file suffix")

}

func registerSizeImplFlag(flags *flag.FlagSet, args *sizeImplArgs) {
	flags.Var(&args.size, "size", "the cubemap face resolution, either % of the input width or absolute px")
	flags.Var(&args.size, "s", "shorthand for size")
	flags.Var(&args.impl, "impl", "the conversion implementation; opencl, opengl or software")
}

func setCommonArgs(args *commonArgs) {
	cargs = args
	if args.out == "" {
		var err error
		args.out, err = os.Getwd()
		harderr(err)
	}

	_, err := os.Stat(args.out)
	if err != nil {
		harderr(fmt.Errorf("cannon stat output directory: %w", err))
	}
}

func gatherInputFiles(globs []string) []string {
	matched := []string{}

	for _, g := range globs {
		m, err := filepath.Glob(g)
		softerr(err)
		matched = append(matched, m...)
	}

	return matched
}

func close(closer io.Closer) {
	closer.Close()
}

func softerr(err error) bool {
	if err != nil && !cargs.supress {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return true
	}
	return false
}

func harderr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
