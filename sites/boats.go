package sites

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"boatfuji.com/api"
)

var (
	btBaseURL               = "https://www.boats.com/"
	sourceBaseURL           = "https://www.boattrader.com/"
	btURLPattern            = regexp.MustCompile(`^https://www\.boats\.com/boat/([0-9]*)`)
	btBaseDir               = "harvest/www.boats.com/"
	btBoatMap               = map[string]int64{}
	btBoatsJSONPattern      = regexp.MustCompile(`<script>var __REDUX_STATE__=(.*)<\/script>`)
	locationPattern         = regexp.MustCompile(`{location:{lat:'(-?[0-9]{1,3}\.[0-9]{1,10})',lng:'(-?[0-9]{1,3}\.[0-9]{1,10}).*'`)
	boatTypePattern         = regexp.MustCompile(`"type":"((.*?))"`)
	btBoatHorsepowerPattern = regexp.MustCompile(`^(\d+) hp$`)
)

func init() {
	api.Sites[btBaseURL] = &Boats{}
}

// Boats accesses https://www.boats.com/
type Boats struct {
	StoreData bool
	WriteSQL  bool
}

// Harvest gets data from the site
func (site *Boats) Harvest(url string) error {
	boatsByFilters = map[string][]string{}
	filtersJSON, err := ioutil.ReadFile(btBaseDir + "filters.json")
	if err == nil {
		err = json.Unmarshal(filtersJSON, &boatsByFilters)
		if err != nil {
			return err
		}
	}
	assetsdir := []string{"boat-sales", "boats", "boattrader"}
	for _, assetdir := range assetsdir {
		os.MkdirAll(btBaseDir+assetdir, 0755)
	}
	switch url {
	case btBaseURL:
		for _, filter := range []string{"type-power", "type-sail", "type-small", "type-pwc"} {
			for page := 1; page < 99999; page++ {
				boatsPage, err := getPage(fmt.Sprintf("%s/boat-sales/%d&%s.html", btBaseDir, page, filter), fmt.Sprintf("%s/boats/%s/page-%d/", sourceBaseURL, filter, page))
				if err != nil {
					return err
				}
				bsBoatIDs := boatsPage.FindN(nil, "//li/@data-listing-id", 0, 9999999, "", "")
				if len(bsBoatIDs) != 0 {
					filtersKey := strings.Split(filter, "-")[1]
					boatsByFilters[filtersKey] = append(bsBoatIDs, boatsByFilters[filter]...)
				} else {
					break
				}
			}
		}
		filtersJSON, _ := json.Marshal(boatsByFilters)
		ioutil.WriteFile(btBaseDir+"filters.json", filtersJSON, 0644)
		return nil
	}
	match := btURLPattern.FindStringSubmatch(url)
	if match == nil {
		return errors.New("BadURL")
	}
	_, err = site.harvestBoat(match[1])
	return err
}

func (site *Boats) harvestBoat(id string) (int64, error) {
	if boatID, ok := btBoatMap[id]; ok {
		return boatID, nil
	}
	btURL := sourceBaseURL + "boat/" + id
	btBoatPage, err := getPage(btBaseDir+"boattrader/"+id+".htm", btURL)
	if err != nil {
		return 0, err
	}
	btboatsJSON := btBoatPage.Find1ByRE(btBoatsJSONPattern, 1, "0", "0")
	boatType := boatTypePattern.FindStringSubmatch(btboatsJSON)[1]
	boat := api.Boat{}
	var boatPage *page
	switch boatType {
	case "power":
		{
			url := btBaseURL + "power-boats/" + id
			boatPage, err = getPage(btBaseDir+"boats/"+id+".htm", url)
			if err != nil {
				return 0, err
			}
			boat = api.Boat{URLs: []string{url}, Sale: &api.BoatSale{}}
			boat.Locomotion = "Power"
		}
	case "sail":
		{
			url := btBaseURL + "sailing-boats/" + id
			boatPage, err = getPage(btBaseDir+"boats/"+id+".htm", url)
			if err != nil {
				return 0, err
			}
			boat = api.Boat{URLs: []string{url}, Sale: &api.BoatSale{}}
			boat.Locomotion = "Sail"
		}
	default:
		return 0, errors.New("boat not found")
	}

	// These functions are used to extract the features for the boat from the HTML.
	fieldXPath := func(name string) string {
		return `//div[@class='collapsible open']/table/tbody/tr/th[text()='` + name + `']/../td/text()`
	}
	fieldYPath := func(name string) string {
		return `//div[@class='collapsible']/table/tbody/tr/th[text()='` + name + `']/../td/text()`
	}
	fieldEPath := func(name string) string {
		return `//div[@id='propulsion']/div[@class='collapsible']/table[1]/tbody/tr/th[text()='` + name + `']/../td/text()`
	}

	// This function removes comma and currency symbols from price.
	calcaulePrice := func(price string) string {
		reg, _ := regexp.Compile("[^0-9]+")
		return reg.ReplaceAllString(price, "")
	}

	// This function is used to calculate the overall length and length.
	calculateFt := func(length string) float32 {
		var parsedTokens []float64
		reg := regexp.MustCompile("[0-9]+")
		filtered := reg.FindAllString(length, -1)
		for _, v := range filtered {
			k, _ := strconv.ParseFloat(v, 32)
			parsedTokens = append(parsedTokens, k)
		}
		if len(parsedTokens) > 1 {
			return float32(math.Floor((parsedTokens[0]+(parsedTokens[1]/12))*100) / 100)
		}
		return float32(parsedTokens[0])
	}
	// Extracting main features for the boat from boats.com page.
	boat.ID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, err
	}
	boat.Year = boatPage.Int(boatPage.Find1(nil, fieldXPath("Year"), "", ""), nil)
	boat.Make = boatPage.Find1(nil, fieldXPath("Make"), "", "")
	boat.Model = boatPage.Find1(nil, fieldXPath("Model"), "", "")
	boat.Condition = boatPage.Find1(nil, fieldXPath("Condition"), "", "")
	boat.Type = boatPage.Find1(nil, fieldXPath("Type"), "", "")
	boat.HullMaterials = []string{boatPage.Find1(nil, fieldXPath("Hull Material"), "", "")}
	boat.Category = boatPage.Find1(nil, fieldXPath("Class"), "", "")
	boat.FuelType = boatPage.Find1(nil, fieldXPath("Fuel Type"), "", "")
	boat.Length = calculateFt(boatPage.Find1(nil, fieldYPath("LOA"), "", ""))
	boat.Beam = calculateFt(boatPage.Find1(nil, fieldYPath("Beam"), "", ""))
	boat.EngineMake = boatPage.Find1(nil, fieldEPath("Engine Make"), "", "")
	boat.EngineModel = boatPage.Find1(nil, fieldEPath("Engine Model"), "", "")
	boat.EnginePower = boatPage.Int(boatPage.Find1(nil, fieldEPath("Power"), "", ""), btBoatHorsepowerPattern)

	location := strings.Split(strings.Trim(boatPage.Find1(nil, fieldXPath("Location"), "", ""), ","), " ")
	boat.Location = &api.Contact{
		Type:    "Address",
		City:    location[0],
		State:   location[1],
		Country: "US",
		Location: api.LatLng(
			boatPage.Float64(boatPage.Find1ByRE(locationPattern, 1, "0", "0"), nil),
			boatPage.Float64(boatPage.Find1ByRE(locationPattern, 2, "0", "0"), nil),
		),
	}
	listingDescription := strings.Join(boatPage.FindN(nil, `//div[@class='desc-text']/p/text()`, 0, 999, "", ""), " ")
	outsideSpace := regexp.MustCompile(`^[\s\p{Zs}]+|[\s\p{Zs}]+$`)
	insideSpace := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	final := outsideSpace.ReplaceAllString(listingDescription, " ")
	boat.Sale = &api.BoatSale{
		Price:              float32(boatPage.Float64(calcaulePrice(boatPage.Find1(nil, `//span[@class="price"]/text()`, "", "")), nil)),
		ListingDescription: strings.TrimPrefix(insideSpace.ReplaceAllString(final, " "), " "),
	}
	imageURLs := boatPage.FindN(nil, "//div[@class='carousel']/ul/li/@data-src_w0", 0, 99999, "", "")
	images := []api.Image{}
	for _, imageURL := range imageURLs {
		if imageURL != "" {
			image := boatPage.Image(imageURL, 600, 400, removeBSWatermark)
			if image != nil {
				images = append(images, *image)
			}
		}
	}
	if len(images) > 0 {
		boat.Images = images
	}
	api.SetBoat(&api.Request{Session: &api.Session{IsGod: true}, Boat: &boat}, nil)
	if site.WriteSQL {
		writeBoatSQL(&boat)
	}
	boatJSON, _ := json.Marshal(boat)
	ioutil.WriteFile(btBaseDir+"boats/"+id+".json", boatJSON, 0644)
	boatPage.SaveWarnings(btBaseDir + "boats/" + id + ".txt")
	btBoatMap[id] = boat.ID
	return boat.ID, nil
}
