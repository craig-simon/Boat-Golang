package sites

import (
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"boatfuji.com/api"
)

var makentMakes map[string]int = map[string]int{}
var makentModels map[string]int = map[string]int{}
var makentUsers map[int64]int = map[int64]int{}
var makentBoats map[int64]int = map[int64]int{}

func makentUserID(userID int64) int {
	if userID == 0 {
		return 10001
	}
	if id, ok := makentUsers[userID]; ok {
		return id
	}
	id := len(makentUsers) + 10002
	makentUsers[userID] = id
	return id
}

func makentBoatID(boatID int64) int {
	if boatID == 0 {
		return 10001
	}
	if id, ok := makentBoats[boatID]; ok {
		return id
	}
	id := len(makentBoats) + 10001
	makentBoats[boatID] = id
	return id
}

func writeBoatSQL(boat *api.Boat) {
	userID := makentUserID(boat.UserID)
	boatID := makentBoatID(boat.ID)
	bookingType := "request_to_book"
	if boat.Rental.InstantBook {
		bookingType = "instant_book"
	}
	perDay := float32(0.00)
	captainFee := float32(0.00)
	fuelPayer := "owner_pays"
	if boat.Rental.Seasons != nil && len(boat.Rental.Seasons) > 0 {
		for _, pricing := range boat.Rental.Seasons[0].Pricing {
			if pricing.DailyPrice > 0 {
				perDay = pricing.DailyPrice
			} else if pricing.HalfDailyPrice > 0 {
				perDay = pricing.HalfDailyPrice * 2
			}
			if pricing.Captain == "CaptainExtra" {
				captainFee = 300
				if boat.Length >= 20 {
					captainFee = 600
				}
			}
			if pricing.FuelPayer == "Renter" {
				fuelPayer = "renter_pays"
			}
		}
	}
	makeID, ok := makentMakes[boat.Make]
	if !ok {
		makeID = len(makentMakes) + 1
		makentMakes[boat.Make] = makeID
		insertSQL("boat_make", "id,name,description,icon,status", map[string]interface{}{
			"id":          makeID,
			"name":        boat.Make,
			"description": boat.Make,
			"icon":        "abbott.png",
			"status":      "Active",
		})
	}
	modelID, ok := makentModels[boat.Model]
	if !ok {
		modelID = len(makentModels) + 1
		makentModels[boat.Model] = modelID
		insertSQL("boat_model", "id,boat_make_id,name,description,status", map[string]interface{}{
			"id":           modelID,
			"boat_make_id": makeID,
			"name":         truncateString(boat.Model, 50),
			"description":  boat.Model,
			"status":       "Active",
		})
	}
	insertSQL("boats", "id,user_id,name,sub_name,summary,trip_start_time,trip_finish_time,booking_message,boat_type,boat_category,boat_make,boat_model,fuel_type,fuel_capacity,passengers_capacity,boat_year,sleeps,rooms,horsepower,length,fuel_policy,fuel_consumption,engine_type,engine_year,nof_engine,top_speed,hull_material,amenities,activities,video,calendar_type,booking_type,cancel_policy,popular,started,recommended,views_count,status,created_at,updated_at,deleted_at", map[string]interface{}{
		"id":                  boatID,
		"user_id":             userID,
		"name":                truncateString(boat.Rental.ListingTitle, 35),
		"sub_name":            boat.Rental.ListingTitle,
		"summary":             boat.Rental.ListingSummary,
		"trip_start_time":     "Flexible",
		"trip_finish_time":    "Flexible",
		"booking_message":     boat.URLs[0],
		"boat_type":           enumInt(boat.Locomotion, []string{"Power", "Sail"}),
		"boat_category":       enumInt(boat.Category, []string{"Airboat", "Aluminum fishing", "Bass boat", "Bow rider", "Dive boat", "Duck boat", "Electric", "House boat", "Jet boat", "Motor yacht", "Tug boat", "Trimaran", "Jon boat", "Steam boat", "Rigid-hulled inflatable b", "Scow", "Galway hooker"}),
		"boat_make":           makeID,
		"boat_model":          modelID,
		"fuel_type":           enumInt(boat.FuelType, []string{"Diesel", "Petrol", "Gas", "Electric"}),
		"fuel_capacity":       boat.FuelCapacity,
		"passengers_capacity": tinyInt(boat.Passengers),
		"boat_year":           boat.Year,
		"sleeps":              tinyInt(boat.Sleeps),
		"rooms":               tinyInt(boat.Rooms),
		"horsepower":          boat.EnginePower,
		"length":              boat.Length,
		"fuel_policy":         fuelPayer,
		"fuel_consumption":    boat.FuelConsumption,
		"engine_type":         nil,
		"engine_year":         nil,
		"nof_engine":          boat.EngineCount,
		"top_speed":           boat.TopSpeed,
		"hull_material":       nil,
		"amenities":           enumInts(boat.Amenities, []string{"Air Conditioning", "Anchor", "Kitchen", "Internet", "Wireless Internet", "Tv Dvd", "Gym", "Anchor Windlass", "Autopilot", "Bimini Top", "Chart Plotter", "Cooler", "Live Bait Well", "Microwave", "Trolling Motor", "Tubes Inflatables", "Wakeboard Tower", "Waterskis", "Wakeboard", "Vhf Radio", "Depth Finder", "Fish Finder", "Galley", "Gps", "Grill", "Live Aboard Allowed", "Radar", "Refrigeration", "Rod Holders", "Roller Furling", "Shower", "Smoking Allowed", "Suitable For Meetings", "Stereo", "Stereo Aux Input", "Sonar", "Spinnaker", "Swim Ladder", "Sink", "Pets Allowed", "First Aid Kit", "Safety Card"}),
		"activities":          enumInts(boat.Activities, []string{"Fishing", "Celebrating", "Sailing", "Watersports", "Cruising"}),
		"video":               "",
		"calendar_type":       "Always",
		"booking_type":        bookingType,
		"cancel_policy":       changeIf("", "Strict", boat.Rental.CancelPolicy),
		"popular":             "No",
		"started":             "No",
		"recommended":         "No",
		"views_count":         0,
		"status":              "Listed",
		"created_at":          nil,
		"updated_at":          nil,
		"deleted_at":          nil,
	})
	if boat.Location != nil {
		insertSQL("boats_address", "boat_id,address_line_1,address_line_2,city,state,country,postal_code,latitude,longitude", map[string]interface{}{
			"boat_id":        boatID,
			"address_line_1": boat.Location.Line1,
			"address_line_2": boat.Location.Line2,
			"city":           boat.Location.City,
			"state":          boat.Location.State,
			"country":        changeIf("", "US", boat.Location.Country),
			"postal_code":    boat.Location.Postal,
			"latitude":       boat.Location.Location.Lat,
			"longitude":      boat.Location.Location.Lng,
		})
	}
	/*
		insertSQL("boats_availability_rules", "id,boat_id,type,minimum_day,maximum_day,trip_start,trip_finish", map[string]interface{}{
			"id":          nil,
			"boat_id":     boatID,
			"type":        "custom",
			"minimum_day": nil,
			"maximum_day": nil,
			"trip_start":  "",
			"trip_finish": "",
		})
	*/
	insertSQL("boats_description", "boat_id,boat,access,interaction,notes,boat_rules,area_overview,transit", map[string]interface{}{
		"boat_id":       boatID,
		"boat":          boat.Rental.ListingDescription,
		"access":        "",
		"interaction":   "",
		"notes":         "",
		"boat_rules":    boat.Rental.Rules,
		"area_overview": "",
		"transit":       "",
	})
	if boat.Images != nil {
		for _, img := range boat.Images {
			insertSQL("boats_photos", "id,boat_id,name,highlights,featured", map[string]interface{}{
				"id":         nil,
				"boat_id":    boatID,
				"name":       img.URL,
				"highlights": img.Tag,
				"featured":   "Yes",
			})
		}
	}
	insertSQL("boats_price", "boat_id,per_day,captain_fee,cleaning,additional_passenger,passengers,security,weekend,minimum_day,maximum_day,currency_code", map[string]interface{}{
		"boat_id":              boatID,
		"per_day":              perDay,
		"captain_fee":          captainFee,
		"cleaning":             0,
		"additional_passenger": 0,
		"passengers":           0,
		"security":             500,
		"weekend":              0,
		"minimum_day":          nil,
		"maximum_day":          nil,
		"currency_code":        "USD",
	})
	insertSQL("boats_steps_status", "boat_id,basics,description,location,photos,pricing,calendar", map[string]interface{}{
		"boat_id":     boatID,
		"basics":      1,
		"description": 1,
		"location":    1,
		"photos":      1,
		"pricing":     1,
		"calendar":    1,
	})
}

func writeUserSQL(user *api.User) {
	userID := makentUserID(user.ID)
	insertSQL("users", "id,first_name,last_name,email,password,remember_token,dob,gender,live,about,school,work,timezone,languages,fb_id,google_id,linkedin_id,currency_code,status,created_at,updated_at,deleted_at", map[string]interface{}{
		"id":             userID,
		"first_name":     user.GivenName,
		"last_name":      user.FamilyName,
		"email":          strconv.Itoa(userID) + "@example.org", // user.Contacts...
		"password":       user.URLs[0],
		"remember_token": nil,
		"dob":            "1970-01-01", // user.Birthdate,
		"gender":         nil,          // user.Gender,
		"live":           "",
		"about":          user.Description,
		"school":         "",
		"work":           "",
		"timezone":       "",
		"languages":      "",
		"fb_id":          nil,
		"google_id":      nil,
		"linkedin_id":    nil,
		"currency_code":  nil,
		"status":         "Active",
		"created_at":     nil,
		"updated_at":     nil,
		"deleted_at":     nil,
	})
	if user.Images != nil && len(user.Images) > 0 {
		insertSQL("profile_picture", "user_id,src,photo_source", map[string]interface{}{
			"user_id":      userID,
			"src":          user.Images[0].URL,
			"photo_source": "Local",
		})
	}
	// users_phone_numbers
	// users_verification
}

func writeReviewsSQL(reviews map[int64]api.Event) {
	for _, review := range reviews {
		dealID := review.DealID
		userID := makentUserID(review.UserID)
		boatID := makentBoatID(review.BoatID)
		reviewerID := makentUserID(review.FromUserID)
		tripDate := api.Date(1999, 1, 1)
		if review.Deal != nil && review.Deal.Rental != nil && review.Deal.Rental.Start != nil {
			tripDate = review.Deal.Rental.Start
		}
		insertSQL("reservation", "id,code,boat_id,owner_id,renter_id,trip_start,trip_finish,start_time,end_time,number_of_passengers,days,per_day,subtotal,cleaning,additional_passenger,insurance_fee,security,captain_fee,service,owner_fee,total,coupon_code,coupon_amount,base_per_day,length_of_rent_type,length_of_rent_discount,length_of_rent_discount_price,booked_period_type,booked_period_discount,booked_period_discount_price,currency_code,paypal_currency,transaction_id,paymode,cancellation,first_name,last_name,postal_code,country,status,type,friends_email,cancelled_by,cancelled_reason,decline_reason,owner_remainder_email_sent,special_offer_id,accepted_at,expired_at,declined_at,cancelled_at,created_at,updated_at,date_check", map[string]interface{}{
			"id":                            dealID,
			"code":                          "",
			"boat_id":                       boatID,
			"owner_id":                      userID,
			"renter_id":                     reviewerID,
			"trip_start":                    tripDate,
			"trip_finish":                   tripDate,
			"start_time":                    "00:00",
			"end_time":                      "00:00",
			"number_of_passengers":          0,
			"days":                          0,
			"per_day":                       0,
			"subtotal":                      0,
			"cleaning":                      0,
			"additional_passenger":          0,
			"insurance_fee":                 0,
			"security":                      0,
			"captain_fee":                   0,
			"service":                       0,
			"owner_fee":                     0,
			"total":                         0,
			"coupon_code":                   0,
			"coupon_amount":                 0,
			"base_per_day":                  0,
			"length_of_rent_type":           "custom",
			"length_of_rent_discount":       0,
			"length_of_rent_discount_price": 0,
			"booked_period_type":            "last_min",
			"booked_period_discount":        0,
			"booked_period_discount_price":  0,
			"currency_code":                 "USD",
			"paypal_currency":               "",
			"transaction_id":                "",
			"paymode":                       "PayPal",
			"cancellation":                  "Strict",
			"first_name":                    "",
			"last_name":                     "",
			"postal_code":                   "",
			"country":                       "US",
			"status":                        "trip_finish",
			"type":                          "reservation",
			"friends_email":                 "",
			"cancelled_by":                  nil,
			"cancelled_reason":              "",
			"decline_reason":                "",
			"owner_remainder_email_sent":    0,
			"special_offer_id":              0,
			"accepted_at":                   "1970-01-01",
			"expired_at":                    "1970-01-01",
			"declined_at":                   "1970-01-01",
			"cancelled_at":                  "1970-01-01",
			"created_at":                    nil,
			"updated_at":                    nil,
			"date_check":                    "",
		})
		insertSQL("reviews", "id,reservation_id,boat_id,user_from,user_to,review_by,comments,private_feedback,love_comments,improve_comments,rating,accuracy,accuracy_comments,cleanliness,cleanliness_comments,trip_start,trip_start_comments,amenities,amenities_comments,communication,communication_comments,location,location_comments,value,value_comments,respect_boat_rules,recommend,created_at,updated_at", map[string]interface{}{
			"id":                     nil,
			"reservation_id":         dealID,
			"boat_id":                boatID,
			"user_from":              reviewerID,
			"user_to":                userID,
			"review_by":              "renter",
			"comments":               review.Review.Text,
			"private_feedback":       "",
			"love_comments":          "",
			"improve_comments":       "",
			"rating":                 review.Review.Rating,
			"accuracy":               0,
			"accuracy_comments":      "",
			"cleanliness":            0,
			"cleanliness_comments":   "",
			"trip_start":             0,
			"trip_start_comments":    "",
			"amenities":              0,
			"amenities_comments":     "",
			"communication":          0,
			"communication_comments": "",
			"location":               0,
			"location_comments":      "",
			"value":                  0,
			"value_comments":         "",
			"respect_boat_rules":     0,
			"recommend":              0,
			"created_at":             nil,
			"updated_at":             nil,
		})
	}
}

func enumInt(needle string, haystack []string) int {
	for i, s := range haystack {
		if needle == s {
			return i + 1
		}
	}
	// TODO: return 0
	return 1
}

func enumInts(needles []string, haystack []string) string {
	csv := ""
	for _, needle := range needles {
		csv += "," + strconv.Itoa(enumInt(needle, haystack))
	}
	if csv == "" {
		return csv
	}
	return csv[1:]
}

func tinyInt(i int) int {
	if i > 127 {
		return 127
	}
	return i
}

func truncateString(str string, num int) string {
	if len(str) <= num {
		return str
	}
	if num > 3 {
		num -= 3
	}
	return str[0:num] + "..."
}

var sqlFile *os.File

func startSQL() {
	os.Remove("harvest/makent.sql")
	sqlFile, _ = os.OpenFile("harvest/makent.sql", os.O_CREATE|os.O_WRONLY, 0644)
	sqlFile.WriteString("delete from reviews where id > 0;\n")
	sqlFile.WriteString("delete from reservation where id > 0;\n")
	sqlFile.WriteString("delete from boats_steps_status where boat_id > 0;\n")
	sqlFile.WriteString("delete from boats_price where boat_id > 0;\n")
	sqlFile.WriteString("delete from boats_photos where boat_id > 0;\n")
	sqlFile.WriteString("delete from boats_description where boat_id > 0;\n")
	sqlFile.WriteString("delete from boats_address where boat_id > 0;\n")
	sqlFile.WriteString("delete from boats where id > 0;\n")
	sqlFile.WriteString("delete from boat_model where id > 0;\n")
	sqlFile.WriteString("delete from boat_make where id > 0;\n")
	sqlFile.WriteString("delete from profile_picture where user_id <> 10001;\n")
	sqlFile.WriteString("delete from users where id <> 10001;\n")
}

var stripNonASCIIPattern = regexp.MustCompile("[[:^ascii:]]")

func insertSQL(table, columnsCSV string, columnValues map[string]interface{}) {
	nonNullCols := ""
	nonNullVals := ""
	for _, col := range strings.Split(columnsCSV, ",") {
		val := columnValues[col]
		if val != nil {
			if v, ok := val.(string); ok {
				nonNullVals += ",'" + strings.ReplaceAll(stripNonASCIIPattern.ReplaceAllLiteralString(v, ""), "'", "''") + "'"
			} else if v, ok := val.(int); ok {
				nonNullVals += "," + strconv.Itoa(v)
			} else if v, ok := val.(int64); ok {
				nonNullVals += "," + strconv.FormatInt(v, 10)
			} else if v, ok := val.(float32); ok {
				nonNullVals += "," + strconv.FormatFloat(float64(v), 'f', -1, 32)
			} else if v, ok := val.(float64); ok {
				nonNullVals += "," + strconv.FormatFloat(v, 'f', -1, 64)
			} else if v, ok := val.(*time.Time); ok {
				if v == nil {
					continue
				}
				nonNullVals += ",'" + v.Format("2006-01-02") + "'"
			} else {
				panic(errors.New("BadColType"))
			}
			nonNullCols += "," + col
		}
	}
	sql := "insert into " + table + " (" + nonNullCols[1:] + ") values (" + nonNullVals[1:] + ");\n"
	sqlFile.WriteString(sql)
}

func commentSQL(comment string) {
	sqlFile.WriteString("-- " + comment + "\n")
}

func finishSQL() {
	sqlFile.Close()
}
