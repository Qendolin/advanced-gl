package main

import (
	"advanced-gl/Project03/libio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

type command struct {
	Run   func(self *command)
	Name  string
	Help  string
	Flags *flag.FlagSet
}

var commands = []*command{}

type generateArgs struct {
	size  int
	out   string
	flipY bool
}

func createGenerateCommand() *command {
	args := &generateArgs{
		size:  32,
		out:   "standard_lut.png",
		flipY: false,
	}

	flags := flag.NewFlagSet("generate", flag.ExitOnError)
	flags.IntVar(&args.size, "size", args.size, "size of the lut")
	flags.StringVar(&args.out, "out", args.out, "output filename")
	flags.BoolVar(&args.flipY, "flipY", args.flipY, "flip output vertically")

	return &command{
		Help:  "generates a standard lut",
		Name:  "generate",
		Flags: flags,
		Run: func(self *command) {
			runGenerateCommand(args)
		},
	}
}

func runGenerateCommand(args *generateArgs) {
	pix := generateStandedLut(args.size, args.flipY)
	saveLut(pix, args.size, args.out)
}

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

func main() {
	commands = append(commands, createGenerateCommand())

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
}

func saveLut(pix []uint8, size int, out string) {
	img := libio.NewIntImage(pix, 4, size, size*size)

	file, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	harderr(err)
	defer file.Close()

	err = png.Encode(file, img.ToRGBA())
	harderr(err)
}

func loadLut(filename string) (pix []uint8, size int) {
	file, err := os.Open(filename)
	harderr(err)
	img, err := png.Decode(file)
	harderr(err)

	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Rect, img, image.Point{}, draw.Over)

	return rgba.Pix, rgba.Rect.Dx()
}

func generateStandedLut(size int, flipY bool) []uint8 {
	pix := make([]uint8, size*size*size*4)

	for z := 0; z < size; z++ {
		for x := 0; x < size; x++ {
			for y := 0; y < size; y++ {
				i := x + size*y + size*size*z
				r := float32(x) / float32(size-1)
				g := float32(y) / float32(size-1)
				b := float32(z) / float32(size-1)
				if flipY {
					g = 1.0 - g
					b = 1.0 - b
				}
				pix[i*4+0] = uint8(0xff * r)
				pix[i*4+1] = uint8(0xff * g)
				pix[i*4+2] = uint8(0xff * b)
				pix[i*4+3] = 0xff
			}
		}
	}

	return pix
}

func harderr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
