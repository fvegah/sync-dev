//go:build ignore
// +build ignore

// This program generates the tray icons for macOS menu bar.
// Run with: go run generate.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 22

func main() {
	// Template icons: black (#000000) with alpha transparency
	// macOS automatically inverts for dark mode
	black := color.RGBA{0, 0, 0, 255}

	// Generate idle icon (circular arrows / sync complete)
	generateIdleIcon(black)

	// Generate syncing icon (arrows with motion)
	generateSyncingIcon(black)

	// Generate error icon (warning triangle)
	generateErrorIcon(black)
}

func generateIdleIcon(c color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw two circular arrows (refresh symbol)
	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size)/2 - 4

	// Draw circle arc (not complete)
	for angle := 0.0; angle < 270; angle += 3 {
		rad := angle * math.Pi / 180
		x := cx + radius*math.Cos(rad)
		y := cy + radius*math.Sin(rad)
		drawThickPoint(img, int(x), int(y), c, 2)
	}

	// Arrow heads at ends
	drawArrowHead(img, int(cx+radius), int(cy), 90, c)
	drawArrowHead(img, int(cx-radius*0.7), int(cy+radius*0.7), -135, c)

	saveIcon(img, "tray-idle.png")
}

func generateSyncingIcon(c color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw two opposing curved arrows (active sync)
	cx, cy := float64(size)/2, float64(size)/2
	radius := float64(size)/2 - 4

	// Upper arc
	for angle := 180.0; angle < 360; angle += 3 {
		rad := angle * math.Pi / 180
		x := cx + radius*math.Cos(rad)
		y := cy + radius*0.6*math.Sin(rad) - 2
		drawThickPoint(img, int(x), int(y), c, 2)
	}

	// Lower arc
	for angle := 0.0; angle < 180; angle += 3 {
		rad := angle * math.Pi / 180
		x := cx + radius*math.Cos(rad)
		y := cy + radius*0.6*math.Sin(rad) + 2
		drawThickPoint(img, int(x), int(y), c, 2)
	}

	// Arrow heads
	drawArrowHead(img, int(cx+radius), int(cy-2), 90, c)
	drawArrowHead(img, int(cx-radius), int(cy+2), -90, c)

	saveIcon(img, "tray-syncing.png")
}

func generateErrorIcon(c color.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw warning triangle
	top := image.Point{size / 2, 2}
	left := image.Point{2, size - 3}
	right := image.Point{size - 3, size - 3}

	// Draw triangle outline
	drawLine(img, top.X, top.Y, left.X, left.Y, c, 2)
	drawLine(img, left.X, left.Y, right.X, right.Y, c, 2)
	drawLine(img, right.X, right.Y, top.X, top.Y, c, 2)

	// Draw exclamation mark
	cx := size / 2
	// Vertical line of exclamation
	for y := 8; y <= 14; y++ {
		drawThickPoint(img, cx, y, c, 1)
	}
	// Dot of exclamation
	drawThickPoint(img, cx, 17, c, 1)

	saveIcon(img, "tray-error.png")
}

func drawThickPoint(img *image.RGBA, x, y int, c color.RGBA, thickness int) {
	for dx := -thickness; dx <= thickness; dx++ {
		for dy := -thickness; dy <= thickness; dy++ {
			if dx*dx+dy*dy <= thickness*thickness {
				px, py := x+dx, y+dy
				if px >= 0 && px < size && py >= 0 && py < size {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA, thickness int) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps == 0 {
		drawThickPoint(img, x1, y1, c, thickness)
		return
	}

	xInc := dx / float64(steps)
	yInc := dy / float64(steps)

	x, y := float64(x1), float64(y1)
	for i := 0; i <= steps; i++ {
		drawThickPoint(img, int(x), int(y), c, thickness)
		x += xInc
		y += yInc
	}
}

func drawArrowHead(img *image.RGBA, x, y, angle int, c color.RGBA) {
	rad := float64(angle) * math.Pi / 180
	length := 4.0

	// Two lines forming arrow head
	for i := -1; i <= 1; i += 2 {
		headAngle := rad + float64(i)*math.Pi/4
		ex := float64(x) - length*math.Cos(headAngle)
		ey := float64(y) - length*math.Sin(headAngle)
		drawLine(img, x, y, int(ex), int(ey), c, 1)
	}
}

func saveIcon(img *image.RGBA, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
