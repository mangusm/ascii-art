package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

type RGB struct {
	r uint32
	g uint32
	b uint32
}

// these magic weights are how you convert color to grayscale, apparently
const rWeight float32 = 0.299
const gWeight float32 = 0.587
const bWeight float32 = 0.114

// takes an int _max_ and returns a slice of ints of length _width_ as evenly as possible
// ex: getSteps(10, 3) -> { 3, 7, 10 }
func getSteps(max int, width int) []int {
	// GOOD LUCK
	smallStep := max / width
	bigStep := smallStep + 1
	bigStepsNeeded := max - smallStep*width
	smallStepsNeeded := width - bigStepsNeeded
	steps := make([]int, width)
	sum := 0

	if bigStepsNeeded > 0 {
		smallStepSpacing := make([]float32, smallStepsNeeded+1)
		for i := 0; i < smallStepsNeeded+1; i++ {
			smallStepSpacing[i] = float32(i) / float32(smallStepsNeeded)
		}

		bigStepSpacing := make([]float32, bigStepsNeeded+1)
		for i := 0; i < bigStepsNeeded+1; i++ {
			bigStepSpacing[i] = float32(i) / float32(bigStepsNeeded)
		}

		smallStepIdx := 0
		bigStepIdx := 0

		for i := 0; i < width; i++ {
			if smallStepSpacing[smallStepIdx] <= bigStepSpacing[bigStepIdx] {
				sum += smallStep
				steps[i] = sum
				smallStepIdx++
			} else {
				sum += bigStep
				steps[i] = sum
				bigStepIdx++
			}
		}
	} else {
		for i := 0; i < width; i++ {
			sum += smallStep
			steps[i] = sum
		}
	}

	return steps
}

// Does exactly what it sounds like
func getAvgRgbOfChunk(stepsX []int, stepsY []int, ix int, iy int, m image.Image) (int, RGB) {
	// calculate the width of the chunk, _dx_
	startX := 0
	if ix > 0 {
		startX = stepsX[ix-1]
	}
	dx := stepsX[ix] - startX

	// calculate the height of the chunk, _dy_
	startY := 0
	if iy > 0 {
		startY = stepsY[iy-1]
	}
	dy := stepsY[iy] - startY

	// Iterate over the pixels in the chunk and calculate their (weighted?) average RGB value
	sum := 0
	rgb := RGB{0, 0, 0}
	for y := startY; y < stepsY[iy]; y++ {
		for x := startX; x < stepsX[ix]; x++ {
			r, g, b, _ := m.At(x, y).RGBA()
			rgb.r += r
			rgb.g += g
			rgb.b += b
			sum += int(rWeight*float32(r) + gWeight*float32(g) + bWeight*float32(b))
		}
	}

	numPixels := dx * dy * 255
	rgb.r /= uint32(numPixels)
	rgb.g /= uint32(numPixels)
	rgb.b /= uint32(numPixels)

	// This magic 257 is needed to convert the RGB values to 0-255 values because why not
	avg := sum / (dx * dy * 257)

	return avg, rgb
}

// Convert a chunk's average RGB value to a rune
func avgToChar(avg int, isInverted bool) rune {
	// avg ranges from 0-255
	// we want to map the avg to one of the 19 runes below
	// using too many different runes or weird runes makes the output look bad
	// these were stolen from https://www.ascii-art-generator.org/ after running a few pictures
	// through it to compare to and noticing it used only a few specific runes
	// tried to order them in descending order of "whitespace" for better results on a light background
	// run with the "invert" option for better results on a darker background
	runes := []rune{'W', 'M', 'N', 'X', 'K', 'O', '0', 'd', 'k', 'x', 'o', 'c', 'l', ';', ':', '\'', ',', '.', ' '}
	if isInverted {
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
	}

	// Determine which rune best represents the chunk
	for i := 0; i < len(runes); i++ {
		if (float32(255)/float32(len(runes)))*float32(i+1) >= float32(avg) {
			return runes[i]
		}
	}
	// this should be unreachable
	return '+'
}

// replaced checkArgs with flag-based parser
func parseFlags() (string, int, bool, bool) {
	var file string
	var width int
	var invert bool
	var color bool

	flag.StringVar(&file, "file", "", "image file to convert")
	flag.StringVar(&file, "f", "", "image file to convert (shorthand)")
	flag.IntVar(&width, "width", 0, "output width in characters")
	flag.IntVar(&width, "w", 0, "output width in characters (shorthand)")
	flag.BoolVar(&invert, "invert", false, "invert brightness mapping")
	flag.BoolVar(&invert, "i", false, "invert brightness mapping (shorthand)")
	flag.BoolVar(&color, "color", false, "use color output")
	flag.BoolVar(&color, "c", false, "use color output (shorthand)")

	flag.Parse()

	if file == "" {
		log.Fatal("A filename must be given (use --file or -f)")
	}
	if width <= 0 {
		log.Fatal("Width must be > 0 (use --width or -w)")
	}

	return file, width, invert, color
}

func main() {
	// Check that the command line arguments are valid
	fileName, width, isInverted, useColor := parseFlags()

	// Decode the JPEG data
	reader, err := os.Open(fileName)

	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	m, _, err := image.Decode(reader)

	if err != nil {
		log.Fatal(err)
	}

	bounds := m.Bounds()

	// Split the image's width into _width_ intervals that are as even as possible
	stepsX := getSteps(bounds.Max.X, width)
	// Split the image's height into a proportional number of intervals that are as even as possible
	stepsY := getSteps(bounds.Max.Y, int(float32(width)*float32(bounds.Max.Y)/float32(bounds.Max.X)/2))

	// This is the list of outRunes that will be appended to as chunks of the image are calculated
	outRunes := []rune{}
	outColors := []RGB{}

	if len(stepsX) > 0 && len(stepsY) > 0 {
		// Iterate over each chunk, calculate its average RGB value,
		// convert that to a rune, and add it to the output
		for iy := range stepsY {
			for ix := range stepsX {
				avg, rgb := getAvgRgbOfChunk(stepsX, stepsY, ix, iy, m)
				rune := avgToChar(avg, isInverted)
				outRunes = append(outRunes, rune)
				// Euclidean Distance in RGB Space
				// distance = sqrt((R2 - R1)^2 + (G2 - G1)^2 + (B2 - B1)^2)
				outColors = append(outColors, rgb)
			}
		}
		// Print out each character of the output and a newline every _width_ runes
		for i := 0; i < len(outRunes); i++ {
			if i%(width) == 0 {
				fmt.Println()
			}
			r := outColors[i].r
			g := outColors[i].g
			b := outColors[i].b

			if useColor {
				coloredRune := fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, string(outRunes[i]))
				fmt.Print(coloredRune)
				continue
			}
			fmt.Print(string(outRunes[i]))
		}
		fmt.Println()
	} else {
		fmt.Println("Step too small")
	}
}
