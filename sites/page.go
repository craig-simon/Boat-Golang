package sites

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"boatfuji.com/api"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type page struct {
	HTML     string
	Doc      *html.Node
	Warnings string
}

// getPage gets from cache or does a GET for a url, and returns either the page or an error
func getPage(cachePath, url string) (*page, error) {
	// get html from cachePath, or if not there, then from url (and save to cachePath)
	var bytes []byte
	var err error
	if bytes, err = ioutil.ReadFile(cachePath); err != nil {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("URL does not exists")
		}
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		ioutil.WriteFile(cachePath, bytes, 0644)
	}
	htmlString := string(bytes)
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return nil, err
	}
	return &page{HTML: htmlString, Doc: doc}, nil
}

func (p *page) Find1ByRE(re *regexp.Regexp, group int, if0, ifN string) string {
	return p.FindNByRE(re, group, 1, 1, if0, ifN)[0]
}

func (p *page) Find0or1ByRE(re *regexp.Regexp, group int, if0, ifN string) string {
	return p.FindNByRE(re, group, 0, 1, if0, ifN)[0]
}

func (p *page) FindNByRE(re *regexp.Regexp, group, min, max int, ifLow, ifHigh string) []string {
	match := re.FindAllStringSubmatch(p.HTML, -1)
	ct := 0
	if match != nil {
		ct = len(match)
	}
	if ct >= min && ct <= max {
		if ct == 0 {
			return []string{ifLow}
		}
		result := make([]string, ct)
		for i, v := range match {
			result[i] = html.UnescapeString(v[group])
		}
		return result
	}
	p.Warnings += "Found " + strconv.Itoa(ct) + " " + re.String() + "\n"
	if ct < min {
		return []string{ifLow}
	}
	return []string{ifHigh}
}

func (p *page) Find1(node *html.Node, expr, if0, ifN string) string {
	return p.FindN(node, expr, 1, 1, if0, ifN)[0]
}

func (p *page) Find0or1(node *html.Node, expr, if0, ifN string) string {
	return p.FindN(node, expr, 0, 1, if0, ifN)[0]
}

func (p *page) FindN(node *html.Node, expr string, min, max int, ifLow, ifHigh string) []string {
	if node == nil {
		node = p.Doc
	}
	nodes := htmlquery.Find(node, expr)
	ct := len(nodes)
	if ct >= min && ct <= max {
		if ct == 0 {
			return []string{ifLow}
		}
		result := make([]string, ct)
		for i, v := range nodes {
			result[i] = htmlquery.InnerText(v)
		}
		return result
	}
	p.Warnings += "Found " + strconv.Itoa(ct) + " " + expr + "\n"
	if ct < min {
		return []string{ifLow}
	}
	return []string{ifHigh}
}

func (p *page) FindNodes(expr string) []*html.Node {
	return htmlquery.Find(p.Doc, expr)
}

var backgroundURLPattern = regexp.MustCompile(`^background-image: url\('(.*)'\)$`)

func changeIf(from, to, s string) string {
	if s == from {
		return to
	}
	return s
}

func (p *page) Int(s string, re *regexp.Regexp) int {
	if re != nil {
		match := re.FindStringSubmatch(s)
		if match == nil {
			p.Warn("NeedInt \"" + s + "\" " + re.String())
			return 0
		}
		s = match[1]
	}
	num, err := strconv.Atoi(s)
	if err != nil {
		p.Warn("NeedInt \"" + s + "\"")
	}
	return num
}

func (p *page) Float64(s string, re *regexp.Regexp) float64 {
	if re != nil {
		match := re.FindStringSubmatch(s)
		if match == nil {
			p.Warn("NeedFloat64 \"" + s + "\" " + re.String())
			return 0
		}
		s = match[1]
	}
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		p.Warn("NeedInt \"" + s + "\"")
	}
	return num
}

func (p *page) Image(url string, width, height int, cleanImage func(i image.Image)) *api.Image {
	if url == "" {
		return nil
	}
	// normalize url and verify
	match := backgroundURLPattern.FindStringSubmatch(url)
	if match != nil {
		url = match[1]
	}
	if !strings.HasPrefix(url, "https://") {
		p.Warnings += "Bad url \"" + url + "\"\n"
		return nil
	}
	// if it was in cache, return previous Image
	imgDir := "harvest/img"
	os.MkdirAll(imgDir, 0755)
	cachePath := validFilePath(imgDir, url+".json")
	if bytes, err := ioutil.ReadFile(cachePath); err == nil {
		if len(bytes) == 0 {
			return nil // AccessDenied from below
		}
		img := &api.Image{}
		json.Unmarshal(bytes, img)
		return img
	}
	// get image, crop/stretch, and save
	resp, err := http.Get(url)
	warnIfErr := "Get"
	if err == nil {
		defer resp.Body.Close()
		var bytes []byte
		bytes, err = ioutil.ReadAll(resp.Body)
		warnIfErr = "ReadAll"
		if err == nil {
			warnIfErr = "AccessDenied"
			if bytes[0] != '<' || !strings.Contains(string(bytes), "AccessDenied") {
				var img *api.Image
				img, err = uploadImage(url, bytes, width, height, cleanImage)
				warnIfErr = "UploadImage"
				if err == nil {
					// save in cache for next time
					bytes, _ = json.Marshal(img)
					ioutil.WriteFile(cachePath, bytes, 0644)
					return img
				}
			}
		}
	}
	// if any error, save in cache so it doesn't try again, and return nil (no image)
	p.Warnings += warnIfErr + " " + url + " " + err.Error() + "\n"
	ioutil.WriteFile(cachePath, []byte{}, 0644)
	return nil
}

func uploadImage(tag string, image []byte, width, height int, cleanImage func(i image.Image)) (*api.Image, error) {
	resp := api.UploadImage(&api.Request{
		Session: &api.Session{},
		Image: &api.Image{
			Tag:    tag,
			Width:  width,
			Height: height,
			Data:   "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image),
		},
		CleanImage: cleanImage,
	}, nil)
	if resp.ErrorCode != "" {
		return nil, api.Err(resp.ErrorCode, resp.ErrorDetails)
	}
	return resp.Image, nil
}

func (p *page) Warn(warning string) {
	p.Warnings += warning + "\n"
}

func (p *page) SaveWarnings(path string) {
	if p.Warnings != "" {
		ioutil.WriteFile(path, []byte(p.Warnings), 0644)
	} else {
		os.Remove(path)
	}
}

var invalidFilePathPattern = regexp.MustCompile(`[\\/:*?"<>|\0-\x1F\x7F-\xFF]`)

func validFilePath(dir, file string) string {
	path, _ := filepath.Abs(dir + "/" + invalidFilePathPattern.ReplaceAllLiteralString(file, " "))
	if len(path) > 260 {
		path = path[0:260]
	}
	return path
}
