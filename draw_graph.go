package main

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"./stat"

	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

func init() {
	draw2d.SetFontFolder("./static")
}

//
// Histograms Charts
//
func dailyWordHistChart(filename string, width, height int, days []*stat.DailyUserStat, dateLine []time.Time) {

	dest := image.NewRGBA(image.Rect(0, 0, width, height))
	gc := draw2dimg.NewGraphicContext(dest)

	var minVal, maxVal float64 = 0, 0
	dayLen := len(days)
	xStep := float64(width) / float64(dayLen)
	baseLineY := float64(50)

	dayAdd := make([]float64, dayLen)
	daySub := make([]float64, dayLen)

	for i := 0; i < dayLen; i += 1 {
		if days[i] == nil {
			dayAdd[i] = 0
			daySub[i] = 0
		} else {
			dayAdd[i] = float64(days[i].WordAdd)
			daySub[i] = float64(0 - days[i].WordSub)

			if dayAdd[i] > maxVal {
				maxVal = dayAdd[i]
			}
			if daySub[i] > minVal {
				minVal = daySub[i]
			}
		}
	}

	hf := float64(height)
	wf := float64(width)

	for i := 0; i < dayLen; i += 1 {
		dayAdd[i] = (hf - baseLineY) * dayAdd[i] / maxVal
		daySub[i] = (baseLineY) * daySub[i] / minVal
	}

	// Draw Grid Lines
	{
		gc.SetFillColor(color.RGBA{0x44, 0x44, 0x44, 0xff})
		gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0xff})
		gc.SetLineWidth(0.2)

		yStep := 100.0
		if maxVal > 10000 {
			yStep = 1000.0
		} else if maxVal > 1000 {
			yStep = 250.0
		}

		gc.SetFontData(draw2d.FontData{Name: "Roboto", Family: draw2d.FontFamilySans})
		for y := yStep; y < maxVal; y += yStep {
			yf := (hf - baseLineY) - (hf-baseLineY)*y/maxVal
			gc.MoveTo(0, yf)
			gc.LineTo(wf, yf)
			gc.Stroke()

			gc.FillStringAt(fmt.Sprintf("%.0f", y), 10, yf)
		}

		yStep = 100.0
		if minVal > 10000 {
			yStep = 1000.0
		} else if minVal > 1000 {
			yStep = 250.0
		}

		for y := yStep; y < minVal; y += yStep {
			yf := (hf - baseLineY) + baseLineY*y/minVal
			gc.MoveTo(0, yf)
			gc.LineTo(wf, yf)
		}

		gc.Stroke()
	}

	// Add
	gc.SetFillColor(color.RGBA{0x44, 0xff, 0x44, 0xff})
	for i := 0; i < dayLen; i += 1 {
		x := float64(i) * xStep
		draw2dkit.Rectangle(gc, x, hf-baseLineY, x+xStep, hf-baseLineY-dayAdd[i])
	}
	gc.Fill()

	// Sub
	gc.SetFillColor(color.RGBA{0xff, 0x44, 0x44, 0xff})
	for i := 0; i < dayLen; i += 1 {
		x := float64(i) * xStep
		draw2dkit.Rectangle(gc, x, hf-baseLineY, x+xStep, hf-baseLineY+daySub[i])
	}
	gc.Fill()

	// Draw Base Line
	{
		gc.SetStrokeColor(color.RGBA{0x00, 0x00, 0x00, 0xff})
		gc.SetLineWidth(1)
		gc.MoveTo(0, 0)
		gc.LineTo(0, hf)
		gc.MoveTo(0, hf-baseLineY)
		gc.LineTo(wf, hf-baseLineY)
		gc.MoveTo(wf, 0)
		gc.LineTo(wf, hf)

		for i := 0; i < dayLen; i += 1 {
			if dateLine[i].Weekday() == time.Monday {
				x := float64(i) * xStep
				gc.MoveTo(x, hf-baseLineY-10)
				gc.LineTo(x, hf-baseLineY+10)
			}
		}

		gc.Stroke()
	}

	draw2dimg.SaveToPngFile(filename, dest)
}
