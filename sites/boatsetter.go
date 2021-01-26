package sites

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"boatfuji.com/api"
)

var bsBaseURL = "https://www.boatsetter.com/"
var bsBaseDir = "harvest/www.boatsetter.com/"
var bsURLPattern = regexp.MustCompile(`^https://www\.boatsetter\.com/(users|boats)/(\w+)\??$`)
var bsUserMap = map[string]int64{} // from id like "abcdefg" to UserID
var bsBoatMap = map[string]int64{} // from id like "abcdefg" to BoatID

// Start does nothing currently, but is needed to make main dependent on sites
func Start() {
}

func init() {
	api.Sites[bsBaseURL] = &Boatsetter{}
}

var boatsByFilters map[string][]string

// Boatsetter accesses https://www.boatsetter.com/
type Boatsetter struct {
	StoreData bool
	WriteSQL  bool
}

// Harvest gets data from the site
func (site *Boatsetter) Harvest(url string) error {
	boatsByFilters = map[string][]string{}
	filtersJSON, err := ioutil.ReadFile(bsBaseDir + "filters.json")
	if err == nil {
		err = json.Unmarshal(filtersJSON, &boatsByFilters)
		if err != nil {
			return err
		}
	}
	os.MkdirAll(bsBaseDir+"boat-rentals", 0755)
	os.MkdirAll(bsBaseDir+"boats", 0755)
	os.MkdirAll(bsBaseDir+"users", 0755)
	if url == "" {
		// loop through users and boats folders
		for _, dir := range []string{"boats", "users"} {
			files, err := ioutil.ReadDir(bsBaseDir + dir)
			if err != nil {
				return err
			}
			oldPercent := ""
			for fileIndex, file := range files {
				newPercent := strconv.Itoa(100 * fileIndex / len(files))
				if newPercent != oldPercent {
					log.Println(dir + " " + newPercent + "%")
				}
				oldPercent = newPercent
				var fileName = file.Name()
				if strings.HasSuffix(fileName, ".htm") {
					id := strings.Split(filepath.Base(fileName), ".")[0]
					if _, err = site.harvestX(dir, id); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if url == bsBaseURL {
		boatsByFilters = map[string][]string{}
		for _, filter := range []string{"&boat_types=power", "&boat_types=sail", "&activity=fishing", "&activity=celebrating", "&activity=sailing", "&activity=watersports", "&activity=cruising", ""} {
			for page := 1; page < 99999; page++ {
				boatsPage, err := getPage(bsBaseDir+"boat-rentals/"+strconv.Itoa(page)+filter+".htm", "https://www.boatsetter.com/boat-rentals?latLngNe=89.99%2C0&latLngSw=-90%2C-180&page="+strconv.Itoa(page)+filter)
				if err != nil {
					return err
				}
				if boatsPage.Find1(nil, "//div/@data-total-search-results", "0", "0") == "0" {
					break
				}
				bsBoatIDs := boatsPage.FindN(nil, "//div/@data-boat-public-id", 0, 99999, "", "")
				if filter != "" {
					filtersKey := strings.Split(filter, "=")[1]
					if page == 1 {
						boatsByFilters[filtersKey] = bsBoatIDs
					} else {
						boatsByFilters[filtersKey] = append(boatsByFilters[filtersKey], bsBoatIDs...)
					}
					continue
				}
				/*
					for _, bsBoatID := range bsBoatIDs {
						_, err = harvestBoat(bsBoatID, 0)
						if err != nil {
							return err
						}
					}
				*/
			}
		}
		filtersJSON, _ := json.Marshal(boatsByFilters)
		ioutil.WriteFile(bsBaseDir+"filters.json", filtersJSON, 0644)
		return nil
	}
	match := bsURLPattern.FindStringSubmatch(url)
	if match == nil {
		return errors.New("BadURL")
	}
	_, err = site.harvestX(match[1], match[2])
	return err
}

func (site *Boatsetter) harvestX(x, id string) (int64, error) {
	if x == "users" {
		return site.harvestUser(id, true, nil)
	}
	return site.harvestBoat(id, 0, nil)
}

var bsAnalyticsPattern = regexp.MustCompile(`<script>\n +analytics\.identify\("[^"]+", \{"initial_page_route":"/(boat-rentals|boats/\w+)"\}\);\n</script>`)
var bsAvgResponseTimePattern = regexp.MustCompile(`>\nAvg\. response time\n<span class='[^']*'>(?:N/A|(&[lg]t;) (\d+) (min|hour))`)
var bsResponseRatePattern = regexp.MustCompile(`>\nResponse rate\n<div class='[^']*'>(?:N/A|(\d+)%)`)
var bsBoatPassengersPattern = regexp.MustCompile(`^ *Up to (\d+) people$`)
var bsBoatHorsepowerPattern = regexp.MustCompile(`^(\d+) hp$`)
var bsBoatCityPattern = regexp.MustCompile(`^([^,]*), (\w{2})?$`)
var bsBoatLocationPattern = regexp.MustCompile(`var evergage_boatLatitude = "([^"]+)";\n *var evergage_boatLongitude = "([^"]+)";`)
var bsBoatPackagesPattern = regexp.MustCompile(`packages: (.*?),\n`)

type bsPackage struct {
	ID              int       `json:"id"`
	PublicID        string    `json:"public_id"`
	IsDefault       bool      `json:"is_default"`
	Type            string    `json:"type" enum:"bareboat, captained"`
	InstantBookable bool      `json:"instant_bookable"`
	IsCharter       bool      `json:"is_charter"`
	OwnerInsured    bool      `json:"owner_provides_insurance"`
	Prices          []bsPrice `json:"prices"`
}

type bsPrice struct {
	BoatPrice           string           `json:"boat_price"`
	ServiceFee          string           `json:"service_fee"`
	TowingFee           string           `json:"towing_fee"`
	BoatingCreditsValue string           `json:"boating_credits_value"`
	CouponValue         string           `json:"coupon_value"`
	SalesTax            string           `json:"sales_tax"`
	TotalPrice          string           `json:"total_price"`
	SecurityDeposit     string           `json:"security_deposit"`
	CalendarDays        int              `json:"calendar_days"`
	RangeAvailable      bool             `json:"range_available"`
	DatePriceAdjustment string           `json:"date_price_adjustment"`
	CaptainPrice        string           `json:"captain_price"`
	RegularPrice        string           `json:"regular_price"`
	CaptainFee          string           `json:"captain_fee"`
	Value               string           `json:"value"`
	IsDefault           bool             `json:"is_default"`
	Duration            string           `json:"duration" enum:"all_day, half_day"`
	FuelPolicy          string           `json:"fuel_policy" enum:"owner_pays, renter_pays"`
	SpecialPrices       []bsSpecialPrice `json:"special_prices"`
}

type bsSpecialPrice struct {
}

// harvestBoat will harvest the given boat (i.e., id=abcdefg), either from the harvest folder or from the website
func (site *Boatsetter) harvestBoat(bsBoatID string, userIDIfUnavailable int64, onlyGetUserInfo *api.User) (int64, error) {
	if boatID, ok := bsBoatMap[bsBoatID]; ok && onlyGetUserInfo == nil {
		// commentSQL("cached boat " + bsBoatID)
		return boatID, nil
	}
	// commentSQL("starting boat " + bsBoatID)
	url := "https://www.boatsetter.com/boats/" + bsBoatID
	boat := api.Boat{URLs: []string{url}, Rental: &api.BoatRental{}}
	boatFileWithoutExt := bsBaseDir + "boats/" + bsBoatID
	boatPage, err := getPage(boatFileWithoutExt+".htm", url)
	if err != nil {
		return 0, err
	}
	// see if this boat wasn't availble so it redirected to boat-rentals
	available := boatPage.Find1ByRE(bsAnalyticsPattern, 1, "boat-rentals", "boat-rentals") != "boat-rentals"
	if boatPage.Find0or1(nil, "//h1[text()='BLIMEY!']", "", "") == "BLIMEY!" {
		available = false
	}
	userInfo := onlyGetUserInfo
	if userInfo == nil {
		userInfo = &api.User{}
	}
	if available {
		// get user response rate and time
		userInfo.RequestCount = 1000
		userInfo.ResponseCount = 10 * boatPage.Int(changeIf("", "0", boatPage.Find1ByRE(bsResponseRatePattern, 1, "0", "0")), nil)
		avgResponseSecs, _ := strconv.Atoi(changeIf("", "0", changeIf("min", "60", changeIf("hour", "3600", boatPage.Find1ByRE(bsAvgResponseTimePattern, 3, "", "")))))
		if avgResponseSecs > 0 {
			avgResponseSecs *= boatPage.Int(boatPage.Find1ByRE(bsAvgResponseTimePattern, 2, "", ""), nil)
			if boatPage.Find1ByRE(bsAvgResponseTimePattern, 1, "", "") == "<" {
				avgResponseSecs--
			} else {
				avgResponseSecs++
			}
		}
		userInfo.ResponseSecSum = 1000 * avgResponseSecs
	}
	if onlyGetUserInfo != nil {
		return 0, nil
	}
	// get UserID
	if !available {
		// it redirected to a boats list because boat is not currently available
		boat.UserID = userIDIfUnavailable
	} else {
		bsUserID := strings.TrimPrefix(boatPage.Find1(nil, "//a[text()='View profile']/@href", "", ""), "/users/")
		userID, err := site.harvestUser(bsUserID, true, userInfo)
		if err != nil {
			return 0, err
		}
		boat.UserID = userID
		// get boat info
		fieldXPath := func(name string) string {
			return `//p[@class='u-sm-flex u-sm-flexJustifyBetween'][span[text()='` + name + `']]/span[@class='u-textBold u-ml1 u-sm-textRight']`
		}
		boat.Year = boatPage.Int(boatPage.Find1(nil, fieldXPath("Year"), "0", "0"), nil)
		boat.Make = boatPage.Find0or1(nil, fieldXPath("Manufacturer"), "", "")
		boat.Model = boatPage.Find0or1(nil, fieldXPath("Model"), "", "")
		make := api.LookupMake(boat.Year, 0, 0, boat.Make) // TODO
		if make != nil {
			boat.MakeID = make.ID
			for _, detail := range make.Details {
				boat.Type = detail.Type
				break
			}
		}
		category := boatPage.Find1(nil, fieldXPath("Boat type"), "", "")
		if category != "" {
			_, cats := api.Enums(api.Boat{}, "Category")
			for code, label := range cats {
				if strings.ToUpper(category) == strings.ToUpper(label) {
					boat.Category = code
				}
			}
			if boat.Category == "" {
				boatPage.Warn("BadCategory \"" + category + "\"")
			}
		}
		boat.Length = float32(boatPage.Int(boatPage.Find1(nil, fieldXPath("Length"), "0", "0"), nil))
		boat.Passengers = boatPage.Int(boatPage.Find1(nil, fieldXPath("Passenger capacity"), "Up to 0 people", "Up to 0 people"), bsBoatPassengersPattern)
		boat.Sleeps = boatPage.Int(boatPage.Find0or1(nil, fieldXPath("Sleeps"), "0", "0"), nil)
		boat.Rooms = boatPage.Int(boatPage.Find0or1(nil, fieldXPath("Staterooms"), "0", "0"), nil)
		if api.StringInArray(bsBoatID, boatsByFilters["power"]) {
			boat.Locomotion = "Power"
		} else if api.StringInArray(bsBoatID, boatsByFilters["sail"]) {
			boat.Locomotion = "Sail"
		} else {
			boat.Locomotion = "Power"
			// boatPage.Warn("BadLocomotion")
		}
		boat.EngineCount = boatPage.Int(changeIf("", "0", boatPage.Find0or1(nil, fieldXPath("Number of engines"), "0", "0")), nil)
		boat.EnginePower = boatPage.Int(boatPage.Find0or1(nil, fieldXPath("Horsepower"), "0 hp", "0 hp"), bsBoatHorsepowerPattern)
		boat.Location = &api.Contact{
			Type: "Address",
			Location: api.LatLng(
				boatPage.Float64(boatPage.Find1ByRE(bsBoatLocationPattern, 1, "0", "0"), nil),
				boatPage.Float64(boatPage.Find1ByRE(bsBoatLocationPattern, 2, "0", "0"), nil)),
		}
		keyInfos := boatPage.FindN(nil, `//h3[@class='u-fsBase u-textSemiBold']`, 3, 4, "", "")
		for keyInfoIndex, keyInfo := range keyInfos {
			// usually it will be citystate, optionally instant bookable, captain, passengers
			if keyInfoIndex == 0 {
				match := bsBoatCityPattern.FindStringSubmatch(keyInfo)
				if match != nil {
					boat.Location.City = match[1]
					boat.Location.State = match[2]
					boat.Location.Country = "US"
				} else {
					// TODO: call Google API for something like "Cancún, QROO"
					// TODO: boatPage.Warn("BadCityState \"" + keyInfo + "\"")
				}
			} else if keyInfo == "Instant bookable" {
				boat.Rental.InstantBook = true
			}
		}
		features := boatPage.FindN(nil, "//div[@data-remodal-id='js-modal-features']/div/div/div[@class='u-textRegular']", 0, 99999, "", "")
		if features[0] != "" {
			boat.Amenities = []string{}
			_, amenities := api.Enums(api.Boat{}, "Amenities")
			for _, feature := range features {
				found := false
				for code, label := range amenities {
					if strings.ToUpper(feature) == strings.ToUpper(label) {
						boat.Amenities = append(boat.Amenities, code)
						found = true
					}
				}
				if !found {
					boatPage.Warn("BadAmenity: " + feature)
				}
			}
			sort.Strings(boat.Amenities)
		}
		boat.Activities = []string{}
		for _, filter := range []string{"fishing", "celebrating", "sailing", "watersports", "cruising"} {
			if api.StringInArray(bsBoatID, boatsByFilters[filter]) {
				boat.Activities = append(boat.Activities, strings.Title(filter))
			}
		}
		sort.Strings(boat.Activities)
		// get boat images
		imageURLs := boatPage.FindN(nil, "//a[@data-fresco-group='boat-photos']/@href", 0, 99999, "", "")
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
		// get rental info
		boat.Rental.ListingTitle = boatPage.Find1(nil, "//h1", "", "")
		boat.Rental.ListingDescription = strings.TrimSpace(boatPage.Find1(nil, "//div[@class='u-mb1 js-show-more-content']", "", ""))
		boat.Rental.ListingSummary = "" // TODO
		boat.Rental.Rules = ""          // TODO
		boat.Rental.CancelPolicy = boatPage.Find0or1(nil, "//div[h3[text()='Cancellation policy']]/div", "Moderate", "Moderate")
		// count reviews and stars
		reviewStars := boatPage.FindN(nil, "//div[@data-remodal-id='js-modal-reviews']//span[@class='u-hiddenVisually']", 0, 99999, "", "")
		for _, stars := range reviewStars {
			if strings.HasSuffix(stars, "/5 stars") {
				rating, _ := strconv.Atoi(strings.TrimSuffix(stars, "/5 stars"))
				boat.Rental.ReviewCount++
				boat.Rental.ReviewRatingSum += rating
			} else if stars != "" {
				boatPage.Warn("BadReviewStars " + stars)
			}
		}
		// get boat packages and pricing
		rentalPricing := []api.BoatRentalPricing{}
		packagesJSON := boatPage.Find1ByRE(bsBoatPackagesPattern, 1, "[]", "[]")
		packages := []bsPackage{}
		if err := json.Unmarshal([]byte(packagesJSON), &packages); err != nil {
			boatPage.Warn("BadPackages: " + packagesJSON)
		}
		for _, pkg := range packages {
			for _, price := range pkg.Prices {
				captain := "NoCaptain"
				if pkg.Type == "captained" {
					if price.CaptainPrice == "0.00" {
						captain = "CaptainIncluded"
					} else {
						captain = "CaptainExtra"
					}
				}
				rentalPricingIndex := -1
				for i, item := range rentalPricing {
					if item.Captain == captain {
						rentalPricingIndex = i
					}
				}
				if rentalPricingIndex == -1 {
					rentalPricingIndex = len(rentalPricing)
					rentalPricing = append(rentalPricing, api.BoatRentalPricing{
						Captain: captain,
					})
				}
				boatPrice, _ := strconv.ParseFloat(price.BoatPrice, 32)
				switch price.Duration {
				case "all_day":
					rentalPricing[rentalPricingIndex].DailyPrice = float32(boatPrice)
				case "half_day":
					rentalPricing[rentalPricingIndex].HalfDailyPrice = float32(boatPrice)
				default:
					boatPage.Warn("BadDuration: " + price.Duration)
				}
				switch price.FuelPolicy {
				case "owner", "owner pays", "owner_pays":
					rentalPricing[rentalPricingIndex].FuelPayer = "owner"
				case "renter", "renter pays", "renter_pays":
					rentalPricing[rentalPricingIndex].FuelPayer = "renter"
				default:
					// boatPage.Warn("BadFuelPolicy: " + price.FuelPolicy)
				}
			}
		}
		boat.Rental.Seasons = []api.BoatRentalSeason{
			{
				StartDay: api.Date(2000, 1, 1),
				EndDay:   api.Date(2000, 12, 31),
				Pricing:  rentalPricing,
			},
		}
	}
	// save record and warnings
	var boatID int64
	if site.StoreData {
		resp := api.SetBoat(&api.Request{Session: &api.Session{IsGod: true}, Boat: &boat}, nil)
		boatID = resp.ID
		if boatID == 0 {
			boatPage.Warn(fmt.Sprintf("SetBoat: %v", resp))
		}
	} else {
		boatID = codeToInt64(bsBoatID)
	}
	boat.ID = boatID
	if site.WriteSQL {
		writeBoatSQL(&boat)
	}
	boatJSON, _ := json.Marshal(boat)
	ioutil.WriteFile(boatFileWithoutExt+".json", boatJSON, 0644)
	boatPage.SaveWarnings(boatFileWithoutExt + ".txt")
	bsBoatMap[bsBoatID] = boatID
	// commentSQL("caching boat " + bsBoatID)
	return boatID, nil
}

var bsAboardSincePattern = regexp.MustCompile(`^Aboard since (\d{4})$`)
var bsUserCityPattern = regexp.MustCompile(`^From ([^,]*), (\w{2})?$`)
var bsReviewCtPattern = regexp.MustCompile(`^\n(\d+) reviews?\n$`)
var bsDatePattern = regexp.MustCompile(`^(\w{3})\. (\d\d)(st|nd|rd|th)$`)

var nextDealID int64 = 1

func (site *Boatsetter) harvestUser(bsUserID string, harvestDeep bool, userInfoFromBoat *api.User) (int64, error) {
	if userID, ok := bsUserMap[bsUserID]; ok {
		// commentSQL("cached user " + bsUserID)
		return userID, nil
	}
	// commentSQL("starting user " + bsUserID)
	url := "https://www.boatsetter.com/users/" + bsUserID
	user := api.User{URLs: []string{url}}
	userFileWithoutExt := bsBaseDir + "users/" + bsUserID
	userPage, err := getPage(userFileWithoutExt+".htm", url)
	if err != nil {
		return 0, err
	}
	// harvest name, description, start year, city, and profile image
	user.GivenName = strings.TrimSpace(userPage.Find1(nil, "//h1", "Missing", "Ambiguous"))
	details := userPage.FindN(nil, "//div[@class='Panel Panel--arrowTopLeft']/p", 2, 3, "", "")
	if strings.HasPrefix(details[0], `“`) && strings.HasSuffix(details[0], `”`) {
		user.Description = strings.Trim(details[0], `“”`)
	} else if !strings.HasSuffix(details[0], "hasn't completed their profile yet.") {
		userPage.Warn("NeedDescription: " + details[0])
	}
	if len(details) > 1 {
		match := bsAboardSincePattern.FindStringSubmatch(details[1])
		if match == nil {
			userPage.Warn("NeedCreated: " + details[1])
		} else {
			year, _ := strconv.Atoi(match[1])
			user.Audit = &api.Audit{Created: api.Date(year, 1, 1)}
		}
	}
	if len(details) > 2 {
		match := bsUserCityPattern.FindStringSubmatch(details[2])
		if match == nil {
			userPage.Warn("NeedCityState: " + details[2])
		} else if match[1] != "" || match[2] != "" {
			user.Contacts = []api.Contact{{Type: "Address", City: match[1], State: match[2]}}
		}
	}
	image := userPage.Image(userPage.Find1(nil, "//span[@class='UserPic UserPic--lg UserPic--withBorder']/@style", "", ""), 200, 200, nil)
	if image != nil {
		user.Images = []api.Image{*image}
	}
	if userInfoFromBoat == nil {
		userInfoFromBoat = &api.User{}
		boatHrefs := userPage.FindN(nil, "//a[@class='u-textGrayDark']/@href", 0, 99999, "", "")
		for _, href := range boatHrefs {
			if href != "" {
				match := bsURLPattern.FindStringSubmatch(href)
				if match == nil || match[1] != "boats" {
					userPage.Warn("BadBoatURL")
				} else if _, err := site.harvestBoat(match[2], 0, userInfoFromBoat); err != nil {
					return 0, err
				}
				break
			}
		}
	}
	user.RequestCount = userInfoFromBoat.RequestCount
	user.ResponseCount = userInfoFromBoat.ResponseCount
	user.ResponseSecSum = userInfoFromBoat.ResponseSecSum
	// save record
	var userID int64
	if site.StoreData {
		resp := api.SetUser(&api.Request{Session: &api.Session{IsGod: true}, User: &user}, nil)
		userID = resp.ID
		if userID == 0 {
			userPage.Warn(fmt.Sprintf("SetUser: %v", resp))
		}
	} else {
		userID = codeToInt64(bsUserID)
	}
	user.ID = userID
	if site.WriteSQL {
		writeUserSQL(&user)
	}
	userJSON, _ := json.Marshal(user)
	ioutil.WriteFile(userFileWithoutExt+".json", userJSON, 0644)
	bsUserMap[bsUserID] = userID
	if !harvestDeep {
		// commentSQL("caching user " + bsUserID + " but not going deep")
		return userID, nil
	}
	// commentSQL("caching user " + bsUserID + " and going deep")
	// harvest boats
	boatHrefs := userPage.FindN(nil, "//a[@class='u-textGrayDark']/@href", 0, 99999, "", "")
	for _, href := range boatHrefs {
		if href != "" {
			match := bsURLPattern.FindStringSubmatch(href)
			if match == nil || match[1] != "boats" {
				userPage.Warn("BadBoatURL")
			} else if _, err := site.harvestBoat(match[2], userID, nil); err != nil {
				return userID, err
			}
		}
	}
	// harvest reviews
	reviewNodes := userPage.FindNodes("//div[@class='Arrange-sizeFill'][div[@class='Arrange']]")
	deals := make(map[int64]api.Deal, len(reviewNodes))
	events := make(map[int64]api.Event, len(reviewNodes))
	reviewsWithBoats := 0
	for reviewIndex, reviewNode := range reviewNodes {
		deal := api.Deal{UserID: userID, Rental: &api.EventRental{Status: "Booked"}}
		event := api.Event{UserID: userID, Review: &api.EventReview{}}
		// there are actually 3 <a href>, but the first two are identical so they are combined by html
		// if boat was deleted, then only 1
		hrefs := userPage.FindN(reviewNode, "//a/@href", 1, 2, "", "")
		if !strings.HasPrefix(hrefs[0], "/users/") {
			userPage.Warn("BadReviewUser " + strconv.Itoa(reviewIndex))
		} else {
			reviewUserID, err := site.harvestUser(strings.TrimPrefix(hrefs[0], "/users/"), harvestDeep, nil)
			if err != nil {
				return userID, err
			}
			event.FromUserID = reviewUserID
		}
		if len(hrefs) > 1 {
			if !strings.HasPrefix(hrefs[1], "/boats/") {
				userPage.Warn("BadReviewBoat " + strconv.Itoa(reviewIndex))
			} else {
				reviewBoatID, err := site.harvestBoat(strings.TrimPrefix(hrefs[1], "/boats/"), userID, nil)
				if err != nil {
					return userID, err
				}
				deal.BoatID = reviewBoatID
				event.BoatID = reviewBoatID
				reviewsWithBoats++
			}
		}
		stars := userPage.Find1(reviewNode, "//span[@class='u-hiddenVisually']", "", "")
		if strings.HasSuffix(stars, "/5 stars") {
			rating, _ := strconv.Atoi(strings.TrimSuffix(stars, "/5 stars"))
			event.Review.Rating = rating
		} else {
			userPage.Warn("BadReviewStars " + strconv.Itoa(reviewIndex))
		}
		dateString := userPage.Find1(reviewNode, "//div[@class='u-fsSm u-textSemiBold']", "Missing", "TooMany")
		if dateString != "" {
			match := bsDatePattern.FindStringSubmatch(dateString)
			if match == nil {
				userPage.Warn("BadReviewDate " + strconv.Itoa(reviewIndex) + ": \"" + dateString + "\"")
			} else {
				month := strings.Index("   JanFebMarAprMayJunJulAugSepOctNovDec", match[1]) / 3
				day, _ := strconv.Atoi(strings.TrimLeft(match[2], "0"))
				date := api.Date(1970, month, day)
				deal.Rental.Start = date
				deal.Rental.End = date
			}
		}
		event.Review.Text = strings.Join(userPage.FindN(reviewNode, "//div[@class='u-fsSm u-textGrayMedium']/p", 0, 99999, "", ""), "\n")
		var dealID, eventID int64
		if site.StoreData {
			resp := api.SetDeal(&api.Request{Session: &api.Session{IsGod: true}, Deal: &deal}, nil)
			dealID = resp.ID
			if dealID == 0 {
				userPage.Warn(fmt.Sprintf("SetDeal: %v", resp))
			}
		} else {
			dealID = nextDealID
			nextDealID++
		}
		event.DealID = dealID
		if site.StoreData {
			resp := api.SetEvent(&api.Request{Session: &api.Session{IsGod: true}, Event: &event}, nil)
			eventID = resp.ID
			if eventID == 0 {
				userPage.Warn(fmt.Sprintf("SetEvent: %v", resp))
			}
		} else {
			eventID = dealID
		}
		deals[dealID] = deal
		events[eventID] = event
		event.Deal = &deal
	}
	reviewCtString := userPage.Find1(nil, "//span[@class='u-lg-sizeFull u-fsSm u-textSemiBold']", "", "")
	match := bsReviewCtPattern.FindStringSubmatch(reviewCtString)
	if match == nil {
		userPage.Warn("BadReviewCt")
	} else {
		reviewCt, _ := strconv.Atoi(match[1])
		if reviewCt != reviewsWithBoats && reviewCt != len(reviewNodes) {
			userPage.Warn("WrongReviewCt")
		}
	}
	if site.WriteSQL {
		writeReviewsSQL(events)
	}
	if len(deals) == 0 {
		os.Remove(userFileWithoutExt + "_deals.json")
	} else {
		dealsJSON, _ := json.Marshal(deals)
		ioutil.WriteFile(userFileWithoutExt+"_deals.json", dealsJSON, 0644)
	}
	if len(events) == 0 {
		os.Remove(userFileWithoutExt + "_events.json")
	} else {
		eventsJSON, _ := json.Marshal(events)
		ioutil.WriteFile(userFileWithoutExt+"_events.json", eventsJSON, 0644)
	}
	// save warnings
	userPage.SaveWarnings(userFileWithoutExt + ".txt")
	return userID, nil
}

// codeToInt64 converts a 5-7 character lowercase code like "bcdefgh" to a positive integer not more than 10 digits
func codeToInt64(code string) int64 {
	var result int64 = 0
	for _, c := range code {
		result = result*26 + int64(c-'a')
	}
	return result
}
