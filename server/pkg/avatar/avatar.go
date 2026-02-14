package avatar

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"unicode"
	"unicode/utf8"
)

var palette = []color.RGBA{
	{R: 229, G: 115, B: 115, A: 255}, // red
	{R: 149, G: 117, B: 205, A: 255}, // purple
	{R: 79, G: 195, B: 247, A: 255},  // blue
	{R: 129, G: 199, B: 132, A: 255}, // green
	{R: 255, G: 183, B: 77, A: 255},  // orange
	{R: 240, G: 98, B: 146, A: 255},  // pink
	{R: 77, G: 182, B: 172, A: 255},  // teal
	{R: 161, G: 136, B: 127, A: 255}, // brown
}

func extractInitials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}

	parts := strings.Fields(name)
	if len(parts) == 1 {
		r, _ := utf8.DecodeRuneInString(parts[0])
		return strings.ToUpper(string(r))
	}

	first, _ := utf8.DecodeRuneInString(parts[0])
	last, _ := utf8.DecodeRuneInString(parts[len(parts)-1])
	return strings.ToUpper(string(first) + string(last))
}

func colorFromName(name string) color.RGBA {
	h := fnv.New32a()
	h.Write([]byte(name))
	return palette[h.Sum32()%uint32(len(palette))]
}

func GenerateInitials(name string, size int) []byte {
	if size <= 0 {
		size = 128
	}

	initials := extractInitials(name)
	bg := colorFromName(name)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	drawInitials(img, initials, white, size)

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func drawInitials(img *image.RGBA, initials string, col color.RGBA, size int) {
	runes := []rune(initials)
	charCount := len(runes)
	if charCount == 0 {
		return
	}

	charWidth := size / 3
	charHeight := size * 2 / 3
	totalWidth := charWidth * charCount
	startX := (size - totalWidth) / 2
	startY := (size - charHeight) / 2

	for i, r := range runes {
		offsetX := startX + i*charWidth
		drawChar(img, r, offsetX, startY, charWidth, charHeight, col)
	}
}

func drawChar(img *image.RGBA, r rune, x, y, w, h int, clr color.RGBA) {
	if !unicode.IsPrint(r) {
		return
	}

	glyphs := builtinFont()
	pattern, ok := glyphs[r]
	if !ok {
		pattern = glyphs['?']
	}

	rows := len(pattern)
	if rows == 0 {
		return
	}
	cols := len(pattern[0])

	cellW := float64(w) / float64(cols)
	cellH := float64(h) / float64(rows)

	for row := 0; row < rows; row++ {
		for c := 0; c < cols; c++ {
			if pattern[row][c] == 1 {
				px := x + int(float64(c)*cellW)
				py := y + int(float64(row)*cellH)
				pw := int(float64(c+1)*cellW) - int(float64(c)*cellW)
				ph := int(float64(row+1)*cellH) - int(float64(row)*cellH)
				for dy := 0; dy < ph; dy++ {
					for dx := 0; dx < pw; dx++ {
						img.SetRGBA(px+dx, py+dy, clr)
					}
				}
			}
		}
	}
}
