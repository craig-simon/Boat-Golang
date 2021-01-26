package api

import (
	"errors"
	"strconv"
	"strings"
)

// Marketplace has site-wide information
type Marketplace struct {
	ReviewCount     int64
	ReviewRatingSum int64
	IOs             MobileApp
	Android         MobileApp
	APIKeys         map[string]string
}

// MobileApp has information about the iOS or Android mobile app
type MobileApp struct {
	MinimumRequiredVersion  float32
	MinimumSuggestedVersion float32
	CurrentVersion          float32
}

// the cached Marketplace struct used whenever someone visits the website
var marketplaces = map[int]*Marketplace{}

func init() {
	marketplaces[1] = &Marketplace{
		ReviewCount:     123456, // TODO: move to datastore.go, and do a query here
		ReviewRatingSum: 555555,
		IOs:             mobileApp(Config.Env.IOSVersions), // TODO: watch so we don't need to restart server whenever mobile app version changes
		Android:         mobileApp(Config.Env.AndroidVersions),
		APIKeys: map[string]string{
			"GoogleAndroid":     Config.Env.GoogleAndroid,
			"GoogleIOS":         Config.Env.GoogleIOS,
			"GoogleWeb":         Config.Env.GoogleWeb,
			"StripePublishable": Config.Env.StripePublishable,
		},
	}
	apiHandlers["GetMarketplaces"] = GetMarketplaces
}

func mobileApp(versionsString string) MobileApp {
	versions := strings.Split(versionsString, ",")
	if len(versions) == 3 {
		v1, e1 := strconv.ParseFloat(versions[0], 32)
		v2, e2 := strconv.ParseFloat(versions[1], 32)
		v3, e3 := strconv.ParseFloat(versions[2], 32)
		if e1 == nil && e2 == nil && e3 == nil {
			return MobileApp{
				MinimumRequiredVersion:  float32(v1),
				MinimumSuggestedVersion: float32(v2),
				CurrentVersion:          float32(v3),
			}
		}
	}
	panic(errors.New("app-local.yaml must have ANDROID_VERSIONS and IOS_VERSIONS each with three versions; i.e., 1.0,1.7,1.71"))
}

// GetMarketplaces gets marketplaces
func GetMarketplaces(req *Request, pub *Publication) *Response {
	return &Response{
		SubscriptionID: -1,
		Marketplaces:   marketplaces,
	}
}
