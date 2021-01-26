package sites

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
)

const nWm = 2
const wWm = 155
const hWm = 100
const basicGrayBound = 45
const yStartSumm = 7
const yFinishSumm = 49
const hSumm = yFinishSumm - yStartSumm

var dirX []int = []int{0, 1, 0, -1, -1, 1, 1, -1, -2, 0, 2, 0, -1, -2, -2, -1, 1, 2, 2, 1, -2, -2, 2, 2, 0, -3, 0, 3, 3, 1, -1, -3, -3, -1, 1, 3, 3, 2, -2, -3, -3, -2, 2, 3, 3, -3, -3, 3, 4, 0, -4, 0, 1, 4, 4, 1, -1, -4, -4, -1, 2, 4, 4, 2, -2, -2, -4, -4, -2, 3, 4, 4, 3, -3, -4, -4, -3, 4, 4, -4, -4, 0, 5, 0, -5, -5, -1, 1, 5, 5, 1, -1, -5, -5, -2, 2, 5, 5, 2, -2, -5, -5, -3, 3, 5, 5, 3, -3, -5, -5, -4, 4, 5, 5, 4, -4, -5, -5, 5, 5, -5, -6, 0, 6, 0, -1, -6, -6, -1, 1, 6, 6, 1, -2, -6, -6, -2, 2, 6, 6, 2, -3, -6, -6, -3, 3, 6, 6, 3, -4, -6, -6, -4, 4, 6, 6, 4, -5, -6, -6, -5, 5, 6, 6, 5, -6, -6, 6}
var dirY []int = []int{-1, 0, 1, 0, -1, -1, 1, 1, 0, -2, 0, 2, 2, 1, -1, -2, -2, -1, 1, 2, 2, -2, -2, 2, 3, 0, -3, 0, 1, 3, 3, 1, -1, -3, -3, -1, 2, 3, 3, 2, -2, -3, -3, -2, 3, 3, -3, -3, 0, 4, 0, -4, -4, -1, 1, 4, 4, 1, -1, -4, -4, -2, 2, 4, 4, 4, 2, -2, -4, -4, -3, 3, 4, 4, 3, -3, -4, -4, 4, 4, -4, -5, 0, 5, 0, -1, -5, -5, -1, 1, 5, 5, 1, -2, -5, -5, -2, 2, 5, 5, 2, -3, -5, -5, -3, 3, 5, 5, 3, -4, -5, -5, -4, 4, 5, 5, 4, -5, -5, 5, 5, 0, -6, 0, 6, 6, 1, -1, -6, -6, -1, 1, 6, 6, 2, -2, -6, -6, -2, 2, 6, 6, 3, -3, -6, -6, -3, 3, 6, 6, 4, -4, -6, -6, -4, 4, 6, 6, 5, -5, -6, -6, -5, 5, 6, 6, -6, -6}

var dets [nWm][wWm][hSumm]uint8
var watermarks [nWm][wWm][hWm]uint8
var dirs [nWm][wWm][hWm]int

func init() {
	// read watermark templates: 2 down (BoatBound, Boatsetter) by 3 across (dets, watermarks, dirs)
	infile, err := os.Open("sites/watermarks.png")
	if err != nil {
		panic("NeedWatermarksPng")
	}
	defer infile.Close()
	iniImg, _, _ := image.Decode(infile)
	for watermarkIndex := 0; watermarkIndex < 2; watermarkIndex++ {
		top := watermarkIndex * hWm
		for x := 0; x < wWm; x++ {
			for y := 0; y < hSumm; y++ {
				_, _, b, _ := iniImg.At(x, top+y+yStartSumm).RGBA()
				dets[watermarkIndex][x][y] = uint8(math.Round(float64(b) / 257))
			}
		}
		for x := 0; x < wWm; x++ {
			for y := 0; y < hWm; y++ {
				r, g, b, _ := iniImg.At(wWm+x, top+y).RGBA()
				if r == 255*257 && g == 0 && b == 0 {
					watermarks[watermarkIndex][x][y] = 0
				} else {
					watermarks[watermarkIndex][x][y] = uint8(math.Round(float64(b) / 257))
				}
			}
		}
		for x := 0; x < wWm; x++ {
			for y := 0; y < hWm; y++ {
				r, g, b, _ := iniImg.At(wWm*2+x, top+y).RGBA()
				if r == 255*257 && g == 0 {
					dirs[watermarkIndex][x][y] = int(math.Round(float64(b) / 257))
				} else {
					dirs[watermarkIndex][x][y] = -1
				}
			}
		}
	}
}

func removeBSWatermark(img image.Image) {
	bounds := img.Bounds()
	w := bounds.Max.X
	h := bounds.Max.Y
	if w < wWm || h < hWm {
		fmt.Println("wrong size: ", w, " x ", h)
	}
	// find offset from bottom right
	x0 := w - wWm
	y0 := h - hWm
	// figure out which watermark (BoatBound or Boatsetter) is best match
	watermarkIndex := 1
	if getDiff(img, dets[1], w, h, x0, y0) > getDiff(img, dets[0], w, h, x0, y0) {
		watermarkIndex = 0
	}
	// remove watermark
	cimg, _ := img.(Changeable)
	for x := 0; x < wWm; x++ {
		for y := 0; y < hWm; y++ {
			wm := watermarks[watermarkIndex][x][y]
			if watermarkIndex == 1 {
				v1 := float64(watermarks[1][x][y])
				v2 := 1 + v1/255
				r, g, b, _ := img.At(x+x0, y+y0).RGBA()
				component := func(c uint32) uint8 {
					c /= 257
					c1 := int(math.Round(v2*float64(c) - v1))
					if c1 > 255 {
						c1 = 255
					}
					if c1 < 0 {
						c1 = 0
					}
					return uint8(c1)
				}
				cimg.Set(x+x0, y+y0, color.RGBA{component(r), component(g), component(b), 255})
			} else if wm != 0 {
				r, g, b, _ := img.At(x+x0, y+y0).RGBA()
				component := func(c uint32) uint8 {
					c /= 257
					var c1 int
					if wm > basicGrayBound {
						strong := float64(wm-basicGrayBound) / 210
						c1 = (int)(math.Round((float64(c) - 255*strong) / (1 - strong)))
					} else {
						// dark
						strong := float64(basicGrayBound-wm) / basicGrayBound
						c1 = (int)(math.Round((float64(c)) / (1 - strong)))
					}
					c1 = (int)(math.Round(0.125 * float64((7*int(c) + c1))))
					if c1 > 255 {
						c1 = 255
					}
					if c1 < 0 {
						c1 = 0
					}
					return uint8(c1)
				}
				cimg.Set(x+x0, y+y0, color.RGBA{component(r), component(g), component(b), 255})
			}
		}
	}
	for x := 0; x < wWm; x++ {
		for y := 0; y < hWm; y++ {
			dir := dirs[watermarkIndex][x][y]
			if dir >= 0 {
				r, g, b, _ := img.At(x+x0+dirX[dir], y+y0+dirY[dir]).RGBA()
				cimg.Set(x+x0, y+y0, color.RGBA{uint8(r / 257), uint8(g / 257), uint8(b / 257), 255})
			}
		}
	}
}

func getDiff(img image.Image, det [wWm][hSumm]uint8, w int, h int, x0 int, y0 int) int64 {
	var res int64 = 0
	for x := 0; x < wWm; x++ {
		for y := 0; y < hSumm; y++ {
			detColor := int(det[x][y])
			r, g, b, _ := img.At(x+x0, y+yStartSumm+y0).RGBA()
			componentDiff := func(component uint32) int64 {
				diff := detColor - int(component/257)
				if diff < 0 {
					diff = -diff
				}
				return int64(diff * detColor)
			}
			res += componentDiff(r) + componentDiff(g) + componentDiff(b)
		}
	}
	return res
}

// Changeable has Set color
type Changeable interface {
	Set(x, y int, c color.Color)
}
