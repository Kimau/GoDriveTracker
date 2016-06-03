package main

import (
	"image"
	"image/color"
	"log"
	"testing"

	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

func TestImage(t *testing.T) {
	DrawTest()
}

func DrawTest() {
	// Initialize the graphic context on an RGBA image
	dest := image.NewRGBA(image.Rect(0, 0, 297, 210.0))
	gc := draw2dimg.NewGraphicContext(dest)

	DrawHello(gc, "Hello World")

	// Save to png
	err := draw2dimg.SaveToPngFile("_testHello.png", dest)
	if err != nil {
		log.Fatalln("Saving failed:", err)
	}
}

// Draw "Hello World"
func DrawHello(gc draw2d.GraphicContext, text string) {
	draw2d.SetFontFolder("static")
	gc.SetFontData(draw2d.FontData{
		Name: "Roboto",
	})

	gc.SetFontSize(14)
	l, t, r, b := gc.GetStringBounds(text)
	//log.Println(t, l, r, b)

	draw2dkit.Rectangle(gc, 0, 0, r-l+30, b-t+30)
	gc.SetFillColor(image.White)
	gc.FillStroke()

	draw2dkit.Rectangle(gc, 10, 10, r-l+20, b-t+20)
	gc.SetFillColor(image.White)
	gc.FillStroke()

	gc.SetFillColor(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	gc.FillStringAt(text, 15-l, 15-t)

}
