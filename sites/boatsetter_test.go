package sites

import (
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"boatfuji.com/api"
)

func TestBoatsetter(t *testing.T) {
	site := &Boatsetter{StoreData: true, WriteSQL: false}
	if site.StoreData {
		api.Start()
	}
	if site.WriteSQL {
		startSQL()
	}
	err := site.Harvest("")
	// err := site.Harvest("https://www.boatsetter.com/")
	// err := site.Harvest("https://www.boatsetter.com/boats/bbmzqtd")
	// err := site.Harvest("https://www.boatsetter.com/users/cmrqr")
	if err != nil {
		log.Printf(err.Error())
	}
	if site.WriteSQL {
		finishSQL()
	}
}

func TestBSWatermarks(t *testing.T) {
	dir := "../Boatsetter.com/RemoveWatermarks"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
		return
	}
	for _, file := range files {
		path := dir + "/" + file.Name()
		if (strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png")) && !strings.HasSuffix(path, ".new.png") {
			removeBSWatermarkFromFile(path)
		}
	}
}

func removeBSWatermarkFromFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	cimg := image.NewRGBA(img.Bounds())
	draw.Draw(cimg, img.Bounds(), img, image.Point{}, draw.Over)
	removeBSWatermark(cimg)
	// save with new suffix
	outfile, _ := os.Create(path + ".new.png")
	defer outfile.Close()
	png.Encode(outfile, cimg)
}
