package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strconv"
)

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
func getAvgRgbOfChunk(stepsX []int, stepsY []int, ix int, iy int, m image.Image) int {
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
	for y := startY; y < stepsY[iy]; y++ {
		for x := startX; x < stepsX[ix]; x++ {
			r, g, b, _ := m.At(x, y).RGBA()
			sum += int(rWeight*float32(r) + gWeight*float32(g) + bWeight*float32(b))
		}
	}

	// This magic 257 is needed to convert the RGB values to 0-255 values because why not
	avg := sum / (dx * dy * 257)

	return avg
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

// check if the command line arguments are valid
func checkArgs(args []string) (string, int, bool) {
	if len(os.Args) < 2 {
		panic("A filename must be given")
	}

	// The file to convert to ascii art
	var fileName = os.Args[1]
	if len(os.Args) < 3 {
		panic("A width must be given")
	}

	// the number of characters wide that the output will be
	var width, err = strconv.Atoi(os.Args[2])
	if err != nil {
		panic(fmt.Sprintf("Invalid width: '%s'", os.Args[2]))
	}
	var isInverted bool = len(os.Args) == 4 && os.Args[3] == "invert"
	return fileName, width, isInverted
}

func main() {
	// Check that the command line arguments are valid
	fileName, width, isInverted := checkArgs(os.Args)

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

	// This is the list of runes that will be appended to as chunks of the image are calculated
	out := []rune{}

	if len(stepsX) > 0 && len(stepsY) > 0 {
		// Iterate over each chunk, calculate its average RGB value,
		// convert that to a rune, and add it to the output
		for iy := range stepsY {
			for ix := range stepsX {
				avg := getAvgRgbOfChunk(stepsX, stepsY, ix, iy, m)
				rune := avgToChar(avg, isInverted)
				out = append(out, rune)
			}
		}
		// Print out each character of the output and a newline every _width_ runes
		for i := 0; i < len(out); i++ {
			if i%(width) == 0 {
				fmt.Println()
			}
			fmt.Print(string(out[i]))
		}
		fmt.Println()
	} else {
		fmt.Println("Step too small")
	}
}
