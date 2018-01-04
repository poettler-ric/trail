package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"log"
	"math"
	"os"
	"strconv"
)

type ElementType int
type Flag uint

type Element struct {
	Id        int
	Type      ElementType
	Length    float64
	Radius    float64
	Vp        int
	MinLength float64
	Errors    Flag
}

const (
	Straight ElementType = iota
	Clothoid
	Radius
)
const (
	EVpDiff Flag = 1 << iota
	EMinLength
)
const (
	MaxVp         int = 100
	MaxStraightVp int = 100
)

var (
	straightVps = map[int][]float64{
		40: []float64{30, 100, 180, 270, 380, 500},
		50: []float64{35, 120, 210, 320, 440},
		60: []float64{40, 140, 250, 370},
		70: []float64{50, 160, 280},
		80: []float64{60, 180},
		90: []float64{70},
	}

	typeTranslations = map[string]ElementType{
		"Gerade":    Straight,
		"Radius":    Radius,
		"Klothoide": Clothoid,
	}

	printAll = flag.Bool("all", false, "print all elemenets")
)

func printElements(elements []*Element) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Id",
		"Type",
		"Length",
		"Radius",
		"Vp",
		"MinLength",
		"Errors"})
	for _, e := range elements {
		table.Append([]string{
			strconv.Itoa(e.Id),
			strconv.Itoa(int(e.Type)),
			fmt.Sprintf("%.2f", e.Length),
			fmt.Sprintf("%.2f", e.Radius),
			strconv.Itoa(e.Vp),
			fmt.Sprintf("%.2f", e.MinLength),
			strconv.Itoa(int(e.Errors)),
		})
	}
	table.Render()
}

func readElement(row []string) *Element {
	result := new(Element)
	var err error

	result.Id, err = strconv.Atoi(row[0])
	if err != nil {
		log.Fatalf("couldn't convert %v to int %v", row[0], err)
	}

	result.Type = determineElementType(row[1])

	result.Length, err = strconv.ParseFloat(row[3], 64)
	if err != nil {
		log.Fatalf("couldn't convert %v to float %v", row[3], err)
	}

	if len(row[6]) > 0 {
		result.Radius, err = strconv.ParseFloat(row[6], 64)
		if err != nil {
			log.Fatalf("couldn't convert %v to float %v",
				row[6],
				err)
		}
	}

	return result
}

func determineElementType(s string) (result ElementType) {
	result, ok := typeTranslations[s]
	if !ok {
		log.Fatalf("unknown type: %v", s)
	}
	return
}

func determineRadiusVp(radius float64) (vp int) {
	radius = math.Abs(radius)
	if radius <= 30 {
		vp = 40
	} else if radius <= 40 {
		vp = 45
	} else if radius <= 50 {
		vp = 50
	} else if radius <= 60 {
		vp = 55
	} else if radius <= 80 {
		vp = 60
	} else if radius <= 100 {
		vp = 65
	} else if radius <= 130 {
		vp = 70
	} else if radius <= 160 {
		vp = 75
	} else if radius <= 200 {
		vp = 80
	} else if radius <= 250 {
		vp = 85
	} else if radius <= 300 {
		vp = 90
	} else if radius <= 350 {
		vp = 95
	} else if radius <= 430 {
		vp = 100
	} else if radius <= 530 {
		vp = 110
	} else if radius <= 670 {
		vp = 120
	} else {
		vp = 130
	}
	return
}

func determineStraightVp(radiusVp int, length float64) (vp int) {
	found := false
	vpAddition := radiusVp % 10
	vp = radiusVp - vpAddition
	for i, minLength := range straightVps[vp] {
		if length <= minLength {
			vp += 10*i + vpAddition
			found = true
			break
		}
	}
	if !found {
		vp = MaxStraightVp
	}
	return
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getNextRadius(elements []*Element, pos, increment int) (result *Element) {
	for i := pos + increment; i > 0 && i < len(elements); i += increment {
		if elements[i].Type == Radius {
			result = elements[i]
			break
		}
	}
	return
}

func drivingSecondLength(vp, seconds int) float64 {
	return float64(vp) / 3.6 * float64(seconds)
}

func main() {
	flag.Parse()

	file, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatalf("failed opening the file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	data, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed reading data: %v", err)
	}

	var elements []*Element

	for _, row := range data[3 : len(data)-1] {
		elements = append(elements, readElement(row))
	}

	// determine radius vp
	for _, e := range elements {
		if e.Type == Radius {
			e.Vp = min(MaxVp, determineRadiusVp(e.Radius))
		}
	}

	// determine straigth vp
	for i, e := range elements {
		if e.Type == Straight {
			radiusVp := 0
			if r := getNextRadius(elements, i, -1); r != nil {
				radiusVp = max(r.Vp, radiusVp)
			}
			if r := getNextRadius(elements, i, 1); r != nil {
				radiusVp = max(r.Vp, radiusVp)
			}
			e.Vp = determineStraightVp(radiusVp, e.Length)
		}
	}

	// determine minimum length of elements
	for i, e := range elements {
		if e.Type == Radius {
			e.MinLength = drivingSecondLength(e.Vp, 1)
		}
		if e.Type == Straight {
			e.MinLength = drivingSecondLength(e.Vp, 1)
			// radi in the same direction need 5 seconds
			p := getNextRadius(elements, i, -1)
			n := getNextRadius(elements, i, 1)
			if p != nil && n != nil {
				if p.Radius < 0 && n.Radius < 0 {
					e.MinLength = drivingSecondLength(e.Vp, 5)
				} else if p.Radius > 0 && n.Radius > 0 {
					e.MinLength = drivingSecondLength(e.Vp, 5)
				}
			}
		}
	}

	// check vp differences
	for i, e := range elements[:len(elements)-1] {
		n := elements[i+1]
		invalid := false
		if e.Vp == 100 || n.Vp == 100 {
			invalid = abs(e.Vp-n.Vp) >= 20
		} else {
			invalid = abs(e.Vp-n.Vp) > 20
		}
		if invalid {
			e.Errors |= EVpDiff
			n.Errors |= EVpDiff
		}
	}
	// check minimum lengths
	for _, e := range elements {
		if e.Length < e.MinLength {
			e.Errors |= EMinLength
		}
	}

	if *printAll {
		printElements(elements)
	} else {
		var invalid []*Element
		for _, e := range elements {
			if e.Errors != 0 {
				invalid = append(invalid, e)
			}
		}
		printElements(invalid)

	}
}
