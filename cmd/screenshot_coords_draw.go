package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/mj1618/desktop-cli/internal/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// LabelMode controls what text is drawn on each annotated element.
type LabelMode int

const (
	// LabelCoords draws "(x,y)" screen-absolute center coordinates.
	LabelCoords LabelMode = iota
	// LabelIDs draws "[id]" element IDs.
	LabelIDs
)

// AnnotateScreenshot draws bounding boxes and coordinate labels on an image.
// windowBounds is [x, y, w, h] of the captured window in screen points.
// Element bounds are screen-absolute points; we convert to window-relative
// image pixels using the ratio of image dimensions to window dimensions.
func AnnotateScreenshot(img image.Image, elements []model.Element, windowBounds [4]int) (image.Image, error) {
	return AnnotateScreenshotWithMode(img, elements, windowBounds, LabelCoords)
}

// AnnotateScreenshotWithMode is like AnnotateScreenshot but allows choosing
// between coordinate labels and element-ID labels.
func AnnotateScreenshotWithMode(img image.Image, elements []model.Element, windowBounds [4]int, mode LabelMode) (image.Image, error) {
	// Convert to RGBA for drawing
	rgba := ImageToRGBA(img)

	imgBounds := img.Bounds()
	imgW := float64(imgBounds.Dx())
	imgH := float64(imgBounds.Dy())
	winW := float64(windowBounds[2])
	winH := float64(windowBounds[3])

	// Compute scale from screen points to image pixels.
	// This automatically accounts for Retina (2x) and the screenshot scale factor.
	var scaleX, scaleY float64
	if winW > 0 {
		scaleX = imgW / winW
	} else {
		scaleX = 1.0
	}
	if winH > 0 {
		scaleY = imgH / winH
	} else {
		scaleY = 1.0
	}

	// Colors for drawing
	boxColor := color.RGBA{R: 255, G: 0, B: 0, A: 100}      // Red with transparency
	textColor := color.RGBA{R: 255, G: 255, B: 255, A: 255} // White
	outlineColor := color.RGBA{R: 0, G: 0, B: 0, A: 200}    // Black

	// Draw each element's bounding box and label
	for _, el := range elements {
		drawElementBoxWithMode(rgba, el, windowBounds[0], windowBounds[1], scaleX, scaleY, boxColor, textColor, outlineColor, mode)
	}

	return rgba, nil
}

// ImageToRGBA converts any image to RGBA
func ImageToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// drawElementBoxWithMode draws a bounding box and label for a single element.
// winX, winY are the window origin in screen points.
// scaleX, scaleY convert from points to image pixels.
func drawElementBoxWithMode(img *image.RGBA, el model.Element, winX, winY int, scaleX, scaleY float64, boxColor, textColor, outlineColor color.Color, mode LabelMode) {
	bounds := el.Bounds
	// Convert from screen-absolute points to window-relative image pixels
	x := int(float64(bounds[0]-winX) * scaleX)
	y := int(float64(bounds[1]-winY) * scaleY)
	w := int(float64(bounds[2]) * scaleX)
	h := int(float64(bounds[3]) * scaleY)

	// Center point in image pixels (for drawing position)
	centerX := x + w/2
	centerY := y + h/2

	// Draw bounding box (rectangle)
	drawRectangle(img, x, y, x+w, y+h, boxColor)

	// Label depends on mode
	var label string
	switch mode {
	case LabelIDs:
		label = fmt.Sprintf("[%d]", el.ID)
	default: // LabelCoords
		origCenterX := bounds[0] + bounds[2]/2
		origCenterY := bounds[1] + bounds[3]/2
		label = fmt.Sprintf("(%d,%d)", origCenterX, origCenterY)
	}
	drawTextWithOutline(img, label, centerX, centerY, textColor, outlineColor)
}

// isWithinBounds checks if a point is within the image bounds
func isWithinBounds(bounds image.Rectangle, x, y int) bool {
	return x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y
}

// drawRectangle draws a rectangle outline on the image
func drawRectangle(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	bounds := img.Bounds()

	// Clamp to image bounds
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}

	if x2 <= x1 || y2 <= y1 {
		return // Empty rectangle
	}

	// Draw top and bottom lines
	for x := x1; x < x2; x++ {
		if isWithinBounds(bounds, x, y1) {
			img.Set(x, y1, c)
		}
		if isWithinBounds(bounds, x, y2-1) {
			img.Set(x, y2-1, c)
		}
	}

	// Draw left and right lines
	for y := y1; y < y2; y++ {
		if isWithinBounds(bounds, x1, y) {
			img.Set(x1, y, c)
		}
		if isWithinBounds(bounds, x2-1, y) {
			img.Set(x2-1, y, c)
		}
	}
}

// drawTextWithOutline draws text with an outline for better visibility
func drawTextWithOutline(img *image.RGBA, text string, x, y int, textColor, outlineColor color.Color) {
	// basicfont.Face7x13 character dimensions
	// Each character is approximately 7 pixels wide
	// Font height is approximately 13 pixels
	textWidth := len(text) * 7
	textHeight := 13

	// Offset position to center the text at (x, y)
	offsetX := x - textWidth/2
	offsetY := y - textHeight/2

	// Draw outline (4 directions around the text)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue // Skip center, we'll draw it as main text
			}
			p := fixed.Point26_6{
				X: fixed.Int26_6((offsetX + dx) * 64),
				Y: fixed.Int26_6((offsetY + dy) * 64),
			}
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(outlineColor),
				Face: basicfont.Face7x13,
				Dot:  p,
			}
			d.DrawString(text)
		}
	}

	// Draw main text
	point := fixed.Point26_6{
		X: fixed.Int26_6(offsetX * 64),
		Y: fixed.Int26_6(offsetY * 64),
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)
}
