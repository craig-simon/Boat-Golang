package api

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"regexp"

	// register gif format
	_ "image/gif"
	// register png format
	_ "image/png"
	"io/ioutil"
	"math"
	"strings"

	// register bmp format
	_ "golang.org/x/image/bmp"
	// register tiff format
	_ "golang.org/x/image/tiff"

	"github.com/nfnt/resize"
)

// Image has information about an uploaded photo, logo, or other image
type Image struct {
	Tag    string `json:",omitempty" datastore:",omitempty,noindex"`
	URL    string `json:",omitempty" datastore:",omitempty,noindex"`
	Width  int    `json:",omitempty" datastore:",omitempty,noindex"`
	Height int    `json:",omitempty" datastore:",omitempty,noindex"`
	// Data will be like "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQ..."
	Data   string `json:",omitempty" datastore:",omitempty,noindex"`
	Camera bool   `json:",omitempty" datastore:",omitempty,noindex"`
	Video  bool   `json:",omitempty" datastore:",omitempty,noindex"`
}

func init() {
	apiHandlers["UploadImage"] = UploadImage
}

var imagePrefixPattern = regexp.MustCompile(`^data:image/(bmp|gif|jpeg|png|tiff);base64$`)

// UploadImage uploads an image
func UploadImage(req *Request, pub *Publication) *Response {
	if req.Image == nil {
		return &Response{ErrorCode: "NeedImage"}
	}
	if req.Image.Width == 0 || req.Image.Height == 0 {
		return &Response{ErrorCode: "NeedImageWidthHeight"}
	}
	// only accepts 200x200 or 600x400
	if (req.Image.Width != 200 || req.Image.Height != 200) && (req.Image.Width != 600 || req.Image.Height != 400) {
		return &Response{ErrorCode: "BadImageWidthHeight"}
	}
	if req.Image.Data == "" {
		return &Response{ErrorCode: "NeedImageData"}
	}
	prefixAndBase64 := strings.SplitN(req.Image.Data, ",", 2)
	if !imagePrefixPattern.MatchString(prefixAndBase64[0]) {
		return &Response{ErrorCode: "BadImageDataType"}
	}
	data, err := base64.StdEncoding.DecodeString(prefixAndBase64[1])
	if err != nil {
		return &Response{ErrorCode: "BadImageDataBase64"}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return &Response{ErrorCode: "BadImageDataDecode", ErrorDetails: map[string]string{"Reason": err.Error()}}
	}
	// remove watermarks if needed
	if req.CleanImage != nil {
		req.CleanImage(img)
	}
	// crop and resize image as requested
	oldAspect := float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
	newAspect := float64(req.Image.Width) / float64(req.Image.Height)
	if math.Abs(newAspect-oldAspect) > 0.01 || req.Crop != nil {
		// crop given image to yield aspect ratio of newAspect
		// either:
		// ______
		// |____|    __________
		// |    | or | |    | | or by req.Crop
		// |____|    |_|____|_|
		// |____|
		crop := image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy())
		if newAspect > oldAspect {
			crop.Min.Y = int((1.0 - oldAspect/newAspect) / 2 * float64(crop.Max.Y))
			crop.Max.Y -= crop.Min.Y
		} else {
			crop.Min.X = int((1.0 - newAspect/oldAspect) / 2 * float64(crop.Max.X))
			crop.Max.X -= crop.Min.X
		}
		if req.Crop != nil {
			cropAspect := float64(req.Crop.Dx()) / float64(req.Crop.Dy())
			if math.Abs(newAspect-cropAspect) > 0.01 {
				return &Response{ErrorCode: "BadCropAspect"}
			}
			crop = *req.Crop
		}
		cropImg := image.NewRGBA(crop)
		draw.Draw(cropImg, crop, img, crop.Min, draw.Src)
		img = cropImg
	}
	img = resize.Resize(uint(req.Image.Width), uint(req.Image.Height), img, resize.Lanczos3)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, img, nil)
	if err != nil {
		return &Response{ErrorCode: "JPEGEncode", ErrorDetails: map[string]string{"Error": err.Error()}}
	}
	imageBytes := buf.Bytes()
	req.Image.URL = fmt.Sprintf("/i/%x.jpg", md5.Sum(imageBytes))
	req.Image.Data = ""
	if err := ioutil.WriteFile("www"+req.Image.URL, imageBytes, 0644); err != nil {
		return &Response{ErrorCode: "WriteFile"}
	}
	// TODO: imbed Latitude and Longitude tags from original image
	return &Response{Image: req.Image}
}
