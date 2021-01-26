package api

import (
	"log"
	"math"
	"regexp"
	"time"
)

// Boat is a boat for sale or rental, etc.
type Boat struct {
	ID                 int64             `json:",omitempty" datastore:"-"`
	UserID             int64             `json:",omitempty" datastore:",omitempty"`
	User               *User             `json:",omitempty" datastore:"-"`
	OrgID              int64             `json:",omitempty" datastore:",omitempty"`
	Org                *Org              `json:",omitempty" datastore:"-"`
	URLs               []string          `json:",omitempty" datastore:",omitempty"`
	HullID             string            `json:",omitempty" datastore:",omitempty,noindex"`
	Name               string            `json:",omitempty" datastore:",omitempty,noindex"`
	MakeID             int               `json:",omitempty" datastore:",omitempty,noindex"`
	MakeDetailID       int               `json:",omitempty" datastore:",omitempty,noindex"`
	OptionIDs          []int             `json:",omitempty" datastore:",omitempty,noindex"`
	Year               int               `json:",omitempty" datastore:",omitempty,noindex"`
	Make               string            `json:",omitempty" datastore:",omitempty,noindex"`
	Model              string            `json:",omitempty" datastore:",omitempty,noindex"`
	Designer           string            `json:",omitempty" datastore:",omitempty,noindex"`
	Category           string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Aft Cabin, Airboat, Aluminum Fishing, Angler, Antique And Classic, Barge, Bass Boat, Bay Boat, Beach Catamaran, Bluewater Fishing, Bow Rider, Canal And River Cruiser, Canoe/Kayak, Cargo Ship, Catamaran, Center Cockpit, Center Console, Classic, Commercial, Convertible, Cruise ship, Cruiser, Cruiser Racer, Cuddy Cabin, Cutter, Daysailer & Weekender, Deck Boat, Deck Saloon, Dinghy, Dive Boat, Downeast, Dragger, Dual Console, Duck Boat, Electric, Express Cruiser, Fish And Ski, Flats Boat, Flybridge, Freshwater Fishing, Gulet, High Performance, Houseboat, Inflatable, Inflatable Outboard, Jet Boat, Jet Ski/Personal Water Craft, Jon Boat, Ketch, Lobster, Mega Yacht, Motorsailer, Motor Yacht, Multi-Hull, Narrow Boat, Offshore Sport Fishing, Other, Passenger, Performance, Performance Fishing, Personal Watercraft, Pilothouse, Pontoon, Power Catamaran, Racer, Racer/Cruiser, Rigid Inflatable, Runabout, Saltwater Fishing, Schooner, Ski And Fish, Ski And Wakeboard, Skiff, Sloop, Sports Cruiser, Sports Fishing, Submersible, Tender, Trawler, Trimaran, Troller, Tug, Utility, Walkaround, Weekender, Yawl"`
	Type               string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Air Boats, Bay Launch, Catamaran, Houseboats, Hovercraft Boats, Inboard Boats, Inflatable Boats, Jet Drive Boats, L-Drive Boats, Monohull Sailboats, Outboard Boats, Pontoon Boats, Power Cat, Rowboats Driftboats Etc, Sea Drive Boats, Stern Drive Power Boat, Surface Drive Boats, Trimaran Boats, Utility/Jon, VDR"`
	HomeMade           bool              `json:",omitempty" datastore:",omitempty,noindex"`
	Condition          string            `json:",omitempty" datastore:",omitempty,noindex" enum:"New, Used"`
	Currency           string            `json:",omitempty" datastore:",omitempty,noindex"`
	UseMetric          bool              `json:",omitempty" datastore:",omitempty,noindex"`
	Length             float32           `json:",omitempty" datastore:",omitempty,noindex"`
	Beam               float32           `json:",omitempty" datastore:",omitempty,noindex"`
	Draft              float32           `json:",omitempty" datastore:",omitempty,noindex"`
	BridgeClearance    float32           `json:",omitempty" datastore:",omitempty,noindex"`
	Weight             float32           `json:",omitempty" datastore:",omitempty,noindex"`
	HullMaterials      []string          `json:",omitempty" datastore:",omitempty,noindex" enum:"Aluminum, Carbon Fiber, Composite, Epoxy, Ferro Cement, Fiberglass, Foam, Graphite Composite, Hypalon, Kevlar, Neoprene, Nylon, Plastic, Plywood, Polyester, Polyethylene, Polypropylene, Polyurethene, PVC, Resin Transfer Molding, Resitex, Roplene, Rubber, Sheet Molded Compound, Steel, Strongnan, Wood"`
	Keel               string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Bulb, Canting, Centerboard, Fin, Full, Wing"`
	Passengers         int               `json:",omitempty" datastore:",omitempty,noindex"`
	Sleeps             int               `json:",omitempty" datastore:",omitempty,noindex"`
	Rooms              int               `json:",omitempty" datastore:",omitempty,noindex"`
	Heads              int               `json:",omitempty" datastore:",omitempty,noindex"`
	FreshWaterCapacity int               `json:",omitempty" datastore:",omitempty,noindex"`
	GrayWaterCapacity  int               `json:",omitempty" datastore:",omitempty,noindex"`
	Locomotion         string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Power, Sail, Unpowered"`
	CruisingSpeed      float32           `json:",omitempty" datastore:",omitempty,noindex"`
	TopSpeed           float32           `json:",omitempty" datastore:",omitempty,noindex"`
	EngineCount        int               `json:",omitempty" datastore:",omitempty,noindex"`
	EnginePower        int               `json:",omitempty" datastore:",omitempty,noindex"`
	EngineYear         int               `json:",omitempty" datastore:",omitempty,noindex"`
	EngineMake         string            `json:",omitempty" datastore:",omitempty,noindex"`
	EngineModel        string            `json:",omitempty" datastore:",omitempty,noindex"`
	FuelType           string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Unknown, Gas, Diesel, Electric, Other"`
	FuelCapacity       int               `json:",omitempty" datastore:",omitempty,noindex"`
	FuelConsumption    int               `json:",omitempty" datastore:",omitempty,noindex"`
	FuelCost           float32           `json:",omitempty" datastore:",omitempty,noindex"`
	Trailer            BoatTrailer       `json:",omitempty" datastore:",omitempty,noindex"`
	Images             []Image           `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	LocationType       string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Unknown, Marina Slip, Marina Dry Storage, Marina Rack Storage, Marina Mooring, Residence Trailer, Residence Slip, Residence Mooring, Storage Facility, Storage Trailer"`
	Location           *Contact          `json:",omitempty" datastore:",omitempty"`
	Activities         []string          `json:",omitempty" datastore:",omitempty,noindex" enum:"Fishing, Celebrating, Sailing, Watersports, Cruising, PWC"`
	Amenities          []string          `json:",omitempty" datastore:",omitempty,noindex" enum:"Air Conditioning, Anchor, Anchor Windlass, Autopilot, Bathroom, Bimini Top, Bluetooth Audio, Bow Thruster, Chart Plotter, Child Life Jackets, Cooler/Ice Chest, Deck Shower, Depth Finder, Fish Finder, Fishing Gear, Floating Island, Floating Mat, Galley, GPS, Grill, Head, Inflatable Toys, Jet Ski, Kayaks, Live Aboard Allowed, Livewell/Baitwell, Microwave, Paddleboards, Pets Allowed, Radar, Refrigerator, Rod Holders, Seabob, Shower, Sink, Smoking Allowed, Snokeling Gear, Sonar, Stereo, Stereo Aux Input, Suitable for Meetings, Swim Ladder, Tender, Trolling Motor, Tubes Inflatables, TV/DVD, VHF Radio, Wakeboard, Wakeboard Tower, Waterskis, Wifi"`
	Ownership          string            `json:",omitempty" datastore:",omitempty,noindex" enum:"Own, Lease"`
	OriginalOwner      bool              `json:",omitempty" datastore:",omitempty,noindex"`
	Acquired           *time.Time        `json:",omitempty" datastore:",omitempty,noindex"`
	OtherOwners        []User            `json:",omitempty" datastore:",omitempty,noindex"`
	MarketValue        float32           `json:",omitempty" datastore:",omitempty,noindex"`
	Liens              []Lien            `json:",omitempty" datastore:",omitempty,noindex"`
	InsurancePolicies  []InsurancePolicy `json:",omitempty" datastore:",omitempty"`
	Rental             *BoatRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Cruise             *BoatRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Ride               *BoatRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Sale               *BoatSale         `json:",omitempty" datastore:",omitempty,noindex"`
	Audit              *Audit            `json:",omitempty" datastore:",omitempty"`
}

// BoatRental is how a boat is available for rental
type BoatRental struct {
	ListingTitle       string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	ListingDescription string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	ListingSummary     string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Rules              string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	AllowTwoHalfDays   bool               `json:",omitempty" datastore:",omitempty,noindex"`
	InstantBook        bool               `json:",omitempty" datastore:",omitempty,noindex"`
	CancelPolicy       string             `json:",omitempty" datastore:",omitempty,noindex" enum:"Flexible, Moderate, Strict"`
	ReviewCount        int                `json:",omitempty" datastore:",omitempty,noindex"`
	ReviewRatingSum    int                `json:",omitempty" datastore:",omitempty,noindex"`
	Approvals          []UserApproval     `json:",omitempty" datastore:",omitempty"`
	Seasons            []BoatRentalSeason `json:",omitempty" datastore:",omitempty,noindex"`
	RentalIfCaptain    *EventRental       `json:",omitempty" datastore:",omitempty,noindex"`
	RentalIfNoCaptain  *EventRental       `json:",omitempty" datastore:",omitempty,noindex"`
	NotAvailable       []time.Time        `json:",omitempty" datastore:",omitempty,noindex"`
	NextAvailable      []time.Time        `json:",omitempty" datastore:",omitempty,noindex"`
}

// BoatRentalSeason is how a boat rental is priced different times of year
type BoatRentalSeason struct {
	StartDay *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
	EndDay   *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
	Pricing  []BoatRentalPricing `json:",omitempty" datastore:",omitempty,noindex"`
}

// BoatRentalPricing is how a boat rental is priced within a season, but for different Captain situations
type BoatRentalPricing struct {
	Captain        string  `json:",omitempty" datastore:",omitempty,noindex" enum:"No Captain, Captain Included, Captain Extra"`
	BasePrice      float32 `json:",omitempty" datastore:",omitempty,noindex"`
	HourlyPrice    float32 `json:",omitempty" datastore:",omitempty,noindex"`
	DailyPrice     float32 `json:",omitempty" datastore:",omitempty,noindex"`
	HalfDailyPrice float32 `json:",omitempty" datastore:",omitempty,noindex"`
	WeeklyPrice    float32 `json:",omitempty" datastore:",omitempty,noindex"`
	FuelPayer      string  `json:",omitempty" datastore:",omitempty,noindex" enum:"Renter, Owner"`
}

// BoatSale is how a boat is available for sale
type BoatSale struct {
	ListingTitle       string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	ListingDescription string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	ListingSummary     string             `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Price              float32            `json:",omitempty" datastore:",omitempty,noindex"`
	Fractions          []BoatSaleFraction `json:",omitempty" datastore:",omitempty,noindex"`
	SoldDate           *time.Time         `json:",omitempty" datastore:",omitempty,noindex"`
	CloseDate          *time.Time         `json:",omitempty" datastore:",omitempty,noindex"`
}

// BoatSaleFraction is for fractional ownership
type BoatSaleFraction struct {
	Fraction float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Price    float32 `json:",omitempty" datastore:",omitempty,noindex"`
}

// BoatTrailer has information about the boat trailer
type BoatTrailer struct {
	Year  int    `json:",omitempty" datastore:",omitempty,noindex"`
	Make  string `json:",omitempty" datastore:",omitempty,noindex"`
	Model string `json:",omitempty" datastore:",omitempty,noindex"`
	Axles int    `json:",omitempty" datastore:",omitempty,noindex"`
}

// InsurancePolicy has a boat insurance policy
type InsurancePolicy struct {
	Insurer      *Org       `json:",omitempty" datastore:",omitempty,noindex"`
	Number       string     `json:",omitempty" datastore:",omitempty,noindex"`
	Type         string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Personal, Charter, Commercial"`
	IssueDate    *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	ExpirDate    *time.Time `json:",omitempty" datastore:",omitempty"`
	InsuredValue float32    `json:",omitempty" datastore:",omitempty,noindex"`
}

// Lien has a boat loan
type Lien struct {
	LienHolder *Org    `json:",omitempty" datastore:",omitempty,noindex"`
	Number     string  `json:",omitempty" datastore:",omitempty,noindex"`
	Balance    float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Monthly    float32 `json:",omitempty" datastore:",omitempty,noindex"`
}

func init() {
	addEnumsFor(Boat{})
	addEnumsFor(BoatRental{})
	addEnumsFor(BoatRentalPricing{})
	apiHandlers["GetBoats"] = GetBoats
	apiHandlers["SetBoat"] = SetBoat
}

func getPublicBoat(boatID int64) *Boat {
	if boatID == 0 {
		return nil
	}
	boat, err := getBoat(boatID)
	if err != nil {
		log.Printf("getBoat(%d) => %s", boatID, err.Error())
		return nil
	}
	if boat.Rental == nil {
		boat.Rental = &BoatRental{}
	}
	return &Boat{
		Rental: &BoatRental{ListingTitle: boat.Rental.ListingTitle},
		Audit: &Audit{
			Created: boat.Audit.Created,
		},
	}
}

// GetBoats gets boats by QA, OrgID, UserID, BoatID, Location, StartDate, EndDate, or none (signed-in UserID)
func GetBoats(req *Request, pub *Publication) *Response {
	filters, staff, resp := makeFilters(req, "Location")
	if resp != nil {
		return resp
	}
	if !req.QA && req.OrgID == 0 && (req.UserID == 0 || req.UserID == req.Session.UserID) && req.BoatID == 0 && req.Location == nil {
		// include favorites in boat list
		user, err := getUser(req.Session.UserID)
		if err != nil {
			return errResponse(err)
		}
		if user.Favorites != nil {
			filters = map[string]interface{}{"or": []map[string]interface{}{
				filters,
				{"ID=": user.Favorites},
			}}
		}
	}
	resp = &Response{SubscriptionID: -1, Boats: map[int64]*Boat{}}
	var boats []*Boat
	keys, err := getAllBoats(filters, &boats)
	if err != nil {
		return errResponse(err)
	}
	// if searching in 50 km radius but not enough found, go to 150 km radius
	if req.Location != nil && len(boats) < 10 && req.KMRadius != 150 {
		req.KMRadius = 150
		return GetBoats(req, pub)
	}
	// process each boat found
	for index, key := range keys {
		boat := boats[index]
		boat.ID = key.ID
		// omit pending boats if searching by location
		if req.Location != nil &&
			(boat.Rental == nil || boat.Rental.ListingTitle == "") &&
			(boat.Cruise == nil || boat.Cruise.ListingTitle == "") &&
			(boat.Ride == nil || boat.Ride.ListingTitle == "") &&
			(boat.Sale == nil || boat.Sale.ListingTitle == "") {
			continue
		}
		// add User and Org
		boat.User = getPublicUser(boat.UserID)
		boat.Org = getPublicOrg((boat.OrgID))
		// TODO
		boat.FuelCost = 0
		// compute rental details
		if boat.Rental != nil {
			// get start and end, defaulting to tomorrow full day
			startTime := req.StartDate
			if startTime == nil {
				tomorrow := now().Add(24 * time.Hour)
				timeZone := req.Session.TimeZone
				if timeZone == nil {
					timeZone = time.UTC
				}
				tomorrowMorning := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 8, 0, 0, 0, timeZone)
				startTime = &tomorrowMorning
			}
			endTime := req.EndDate
			if endTime == nil {
				later := startTime.Add(8 * time.Hour)
				endTime = &later
			}
			// if desired time is in past or not available, find next available date at same times of day
			boat.Rental.NextAvailable = nil
			if startTime.Before(*now()) {
				postpone := time.Hour * time.Duration(24*math.Ceil(now().Sub(*startTime).Hours()/24))
				*startTime = startTime.Add(postpone)
				*endTime = endTime.Add(postpone)
				boat.Rental.NextAvailable = []time.Time{*startTime, *endTime}
			}
			if boat.Rental.NotAvailable != nil {
				// it will be a list of start, end, start, end, etc. (even number of items) in ascending order
				// see if the desired startTime..endTime range intersects with any NotAvailable range
				for pos, tm := range boat.Rental.NotAvailable {
					if pos%2 == 0 {
						if endTime.After(tm) {
							continue
						}
						// no intersection, so it's available
						break
					} else {
						if startTime.After(tm) {
							continue
						}
						// not available, so find next availability on future date at same times of day
						postpone := time.Hour * time.Duration(24*math.Ceil(tm.Sub(*startTime).Hours()/24))
						*startTime = startTime.Add(postpone)
						*endTime = endTime.Add(postpone)
						boat.Rental.NextAvailable = []time.Time{*startTime, *endTime}
						// continue to make sure it doesn't intersect other future NotAvailable ranges
					}
				}
				boat.Rental.NotAvailable = nil // only used by the server
			}
			boat.Rental.RentalIfNoCaptain = boatRental(boat, startTime, endTime, 0)
			boat.Rental.RentalIfCaptain = boatRental(boat, startTime, endTime, 1)
		}
		// if it's not my boat and it's not my org's boat, and I'm not staff, sanitize record
		if boat.UserID != req.Session.UserID && (boat.OrgID == 0 || boat.OrgID != req.Session.OrgID) && !staff {
			boat.HullID = ""
			boat.InsurancePolicies = nil
			boat.Liens = nil
			if boat.Location != nil {
				boat.Location = &Contact{City: boat.Location.City, State: boat.Location.State, Country: boat.Location.Country, Location: boat.Location.Location}
			}
			boat.URLs = nil
			if boat.Rental != nil {
				boat.Rental.Approvals = nil
				boat.Rental.Seasons = nil
			}
			getAudit(req, boat)
		} else if boat.Location != nil {
			boat.Location.Loc100KM = nil
			boat.Location.Loc300KM = nil
		}
		resp.Boats[key.ID] = boat
	}
	return resp
}

func boatRental(boat *Boat, startTime, endTime *time.Time, captain int) *EventRental {
	duration := endTime.Sub(*startTime)
	bigBoat := 0.0
	if boat.Length >= 20 {
		bigBoat = 1
	}
	if boat == nil || boat.Rental == nil || boat.Rental.Seasons == nil {
		return nil
	}
	for _, season := range boat.Rental.Seasons {
		if season.Pricing == nil {
			continue
		}
		// TODO: check if startTime falls within season
		for _, pricing := range season.Pricing {
			if (pricing.Captain == "NoCaptain") != (captain == 0) {
				continue
			}
			price := float64(pricing.HalfDailyPrice)
			captainFee := 200.00 + bigBoat*150.00
			if price == 0 || duration.Hours() > 5 {
				numDays := math.Ceil((duration.Hours() + 8) / 24)
				numWeeks := math.Ceil(numDays / 7)
				priceByDay := float64(pricing.DailyPrice) * numDays
				priceByWeek := float64(pricing.WeeklyPrice) * numWeeks
				if priceByDay == 0 {
					price = priceByWeek
				} else if priceByWeek == 0 {
					price = priceByDay
				} else {
					price = math.Min(priceByDay, priceByWeek)
				}
				captainFee = (300.00 + bigBoat*300.00) * numDays
			}
			if pricing.Captain != "CaptainExtra" {
				captainFee = 0
			}
			percent := func(p int) float32 {
				return float32(math.Round(price * float64(p) / 100))
			}
			rental := &EventRental{
				Start:           startTime,
				End:             endTime,
				CancelPolicy:    boat.Rental.CancelPolicy,
				Currency:        boat.Currency,
				Captain:         pricing.Captain,
				Price:           float32(price),
				CaptainFee:      float32(captainFee),
				InsureFee:       percent(20),
				TowFee:          percent(5),
				TransactionFee:  percent(10),
				RewardsDiscount: 0,
				SecurityDeposit: float32(500.00 + bigBoat*500.00),
				FuelPayer:       pricing.FuelPayer,
			}
			subtotal := rental.Price + rental.CaptainFee + rental.InsureFee + rental.TowFee + rental.TransactionFee - rental.RewardsDiscount
			// TODO: hook up with Avalara
			rental.SalesTax = float32(math.Round(float64(subtotal)*7) / 100)
			rental.Total = subtotal + rental.SalesTax
			fullRefund := rental.Total
			halfRefund := math.Round(float64(fullRefund)/2*100) / 100
			// Flexible is full refund with 24 hours
			daysBackFullRefund := 1
			daysBackHalfRefund := 0
			switch boat.Rental.CancelPolicy {
			case "Moderate":
				daysBackFullRefund = 5
				daysBackHalfRefund = 2
			case "Strict":
				daysBackFullRefund = 30
				daysBackHalfRefund = 14
			}
			rental.CancelCutOffs = []EventRentalCancel{}
			if daysBackFullRefund != 0 {
				cutoff := startTime.AddDate(0, 0, -daysBackFullRefund)
				rental.CancelCutOffs = append(rental.CancelCutOffs, EventRentalCancel{CutOff: &cutoff, Refund: float32(fullRefund)})
			}
			if daysBackHalfRefund != 0 {
				cutoff := startTime.AddDate(0, 0, -daysBackHalfRefund)
				rental.CancelCutOffs = append(rental.CancelCutOffs, EventRentalCancel{CutOff: &cutoff, Refund: float32(halfRefund)})
			}
			return rental
		}
	}
	return nil
}

var hullIDPattern = regexp.MustCompile(`^([A-Z]{2}-)?[A-Z0-9]{3}\d{5}(0[1-9]\d\d|1[0-2]\d\d|M\d\d[A-L]|[A-L]\d\d\d)$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// SetBoat sets a boat
func SetBoat(req *Request, pub *Publication) *Response {
	if !isVerifiedUser(req) {
		return mustVerifyResp()
	}
	if req.Boat == nil {
		return &Response{ErrorCode: "NeedBoat"}
	}
	if err := validate(req.Boat); err != nil {
		return errResponse(err)
	}
	staff := isStaff(req)
	oldBoat, err := getBoat(req.Boat.ID)
	if err != nil {
		return errResponse(err)
	}
	// check UserID and OrgID and omit User and Org
	if req.Boat.UserID == 0 {
		req.Boat.UserID = req.Session.UserID
	}
	if req.Boat.OrgID == 0 && req.Boat.UserID == req.Session.UserID {
		req.Boat.OrgID = req.Session.OrgID
	}
	if oldBoat != nil && oldBoat.UserID != 0 {
		// only staff can change UserID or OrgID
		if (req.Boat.UserID != oldBoat.UserID || req.Boat.OrgID != oldBoat.OrgID) && !staff {
			return &Response{ErrorCode: "StaffOnlyToChangeBoatOwner"}
		}
	}
	req.Boat.User = nil
	req.Boat.Org = nil
	if (req.Boat.UserID != req.Session.UserID || req.Boat.OrgID != req.Session.OrgID) && !staff {
		return &Response{ErrorCode: "StaffOnlyToSetBoatOwner"}
	}
	// check HullID
	if req.Boat.HullID != "" && !hullIDPattern.MatchString(req.Boat.HullID) {
		return &Response{ErrorCode: "BadHullID"}
	}
	// check Currency
	if req.Boat.Currency != "" && !currencyPattern.MatchString(req.Boat.Currency) {
		return &Response{ErrorCode: "BadCurrency"}
	}
	req.Boat.FuelCost = 0
	// TODO: check Rental
	if req.Boat.Rental != nil {
		rental := req.Boat.Rental
		rental.RentalIfCaptain = nil
		rental.RentalIfNoCaptain = nil
		rental.NextAvailable = nil
	}
	// finalize and save
	setAudit(staff, req.Boat, oldBoat)
	if req.Boat.Location != nil {
		locations := []Contact{*req.Boat.Location}
		if err := setContacts(locations, nil, req); err != nil {
			return errResponse(err)
		}
		req.Boat.Location = &locations[0]
	}
	key, err := putBoat(req.Boat)
	if err != nil {
		return errResponse(err)
	}
	return &Response{
		ID: key.ID,
	}
}
