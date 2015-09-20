package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	"./stat"

	"github.com/vdobler/chart"
	"github.com/vdobler/chart/imgg"
)

type GraphImg struct {
	W, H    int
	I       *image.RGBA
	imgFile *os.File
}

func NewGraphImg(name string, w, h int) *GraphImg {
	var err error
	gImg := GraphImg{W: w, H: h}

	gImg.imgFile, err = os.Create(name + ".png")
	if err != nil {
		panic(err)
	}

	gImg.I = image.NewRGBA(image.Rect(0, 0, w, h))
	bg := image.NewUniform(color.RGBA{0xff, 0xff, 0xff, 0xff})
	draw.Draw(gImg.I, gImg.I.Bounds(), bg, image.ZP, draw.Src)

	return &gImg
}

func (d *GraphImg) Close() {
	png.Encode(d.imgFile, d.I)
	d.imgFile.Close()
}

func (d *GraphImg) Plot(c chart.Chart) {

	igr := imgg.AddTo(d.I, 0, 0, d.W, d.H, color.RGBA{0xff, 0xff, 0xff, 0xff}, nil, nil)
	c.Plot(igr)
}

//
// Histograms Charts
//
func dailyWordHistChart(days []*stat.DailyStat, dateLine []time.Time) {
	gImg := NewGraphImg("dayHist", 800, 300)
	defer gImg.Close()

	hc := chart.BarChart{
		Title:        "Last 100 Days",
		SameBarWidth: true,
		Stacked:      true,
		Key: chart.Key{
			Hide: true,
		},
	}

	green := chart.Style{Symbol: 'x', LineColor: color.NRGBA{0x00, 0xaa, 0x00, 0xff}, LineWidth: 0, FillColor: color.NRGBA{0x40, 0xff, 0x40, 0xff}}
	red := chart.Style{Symbol: '%', LineColor: color.NRGBA{0xcc, 0x00, 0x00, 0xff}, LineWidth: 0, FillColor: color.NRGBA{0xff, 0x40, 0x40, 0xff}}

	dayLen := len(days)
	dayAdd := make([]float64, dayLen)
	daySub := make([]float64, dayLen)
	dateLineFlt := make([]float64, dayLen)

	hc.XRange = chart.Range{
		Label:      "Days",
		Time:       false,
		ShowLimits: false,
		ShowZero:   false,
		MinMode:    chart.RangeMode{Fixed: true, Constrained: true, Expand: chart.ExpandTight, Value: float64(-dayLen)},
		MaxMode:    chart.RangeMode{Fixed: true, Constrained: true, Expand: chart.ExpandTight, Value: 0},
		DataMin:    float64(dateLine[0].Unix()),
		DataMax:    float64(dateLine[dayLen-1].Unix()),
		TicSetting: chart.TicSetting{
			Hide: true,
		},
	}

	hc.YRange = chart.Range{
		Label:      "",
		ShowLimits: false,
		ShowZero:   true,
		Category:   []string{"Add", "Sub"},
		TicSetting: chart.TicSetting{
			Hide: true,
		},
	}

	for i := 0; i < dayLen; i += 1 {
		dateLineFlt[i] = float64(i - dayLen)
		if days[i] == nil {
			dayAdd[i] = 0
			daySub[i] = 0
		} else {
			dayAdd[i] = float64(days[i].WordAdd)
			daySub[i] = float64(days[i].WordSub)
		}
	}
	dayAdd[0] = 1000
	dayAdd[dayLen-1] = 1000
	daySub[0] = -1000
	daySub[dayLen-1] = -1000

	hc.AddDataPair("Add", dateLineFlt, dayAdd, green)
	hc.AddDataPair("Sub", dateLineFlt, daySub, red)
	gImg.Plot(&hc)
}
