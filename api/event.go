package api

import (
	"time"
)

// Event is an event for a deal, such as a delivery, message, payment, rental, review, etc.
type Event struct {
	ID           int64              `json:",omitempty" datastore:"-"`
	DealID       int64              `json:",omitempty" datastore:",omitempty"`
	Deal         *Deal              `json:",omitempty" datastore:"-"`
	BoatID       int64              `json:",omitempty" datastore:",omitempty"`
	Boat         *Boat              `json:",omitempty" datastore:"-"`
	UserID       int64              `json:",omitempty" datastore:",omitempty"`
	User         *User              `json:",omitempty" datastore:"-"`
	OrgID        int64              `json:",omitempty" datastore:",omitempty"`
	Org          *Org               `json:",omitempty" datastore:"-"`
	FromUserID   int64              `json:",omitempty" datastore:",omitempty"`
	FromUser     *User              `json:",omitempty" datastore:"-"`
	UnreadByIDs  []int64            `json:",omitempty" datastore:",omitempty"`
	UserIDs      []int64            `json:",omitempty" datastore:",omitempty"`
	OrgIDs       []int64            `json:",omitempty" datastore:",omitempty"`
	Crew         *EventCrew         `json:",omitempty" datastore:",omitempty,noindex"`
	Cruise       *EventRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Delivery     *EventDelivery     `json:",omitempty" datastore:",omitempty,noindex"`
	Finance      *EventFinance      `json:",omitempty" datastore:",omitempty,noindex"`
	Insure       *EventInsure       `json:",omitempty" datastore:",omitempty,noindex"`
	Message      *EventMessage      `json:",omitempty" datastore:",omitempty,noindex"`
	Notification *EventNotification `json:",omitempty" datastore:",omitempty,noindex"`
	Payment      *EventPayment      `json:",omitempty" datastore:",omitempty,noindex"`
	Rental       *EventRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Review       *EventReview       `json:",omitempty" datastore:",omitempty,noindex"`
	Ride         *EventRental       `json:",omitempty" datastore:",omitempty,noindex"`
	Sale         *EventSale         `json:",omitempty" datastore:",omitempty,noindex"`
	Service      *EventService      `json:",omitempty" datastore:",omitempty,noindex"`
	Transport    *EventTransport    `json:",omitempty" datastore:",omitempty,noindex"`
	Audit        *Audit             `json:",omitempty" datastore:",omitempty"`
}

// EventCrew is when someone needs a captain or other crewmember for their boat
type EventCrew struct {
	Start *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	End   *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Notes string     `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
}

// EventDelivery is when a renter checks out the boat before rental and an owner checks in the boat after rental
type EventDelivery struct {
	Sequence          int                    `json:",omitempty" datastore:",omitempty,noindex"`
	RenterIDPhoto     Image                  `json:",omitempty" datastore:",omitempty,noindex"`
	FuelLevel         float32                `json:",omitempty" datastore:",omitempty,noindex"`
	OilLevel          float32                `json:",omitempty" datastore:",omitempty,noindex"`
	LifeJackets       int                    `json:",omitempty" datastore:",omitempty,noindex"`
	FireExtinguishers int                    `json:",omitempty" datastore:",omitempty,noindex"`
	Notes             []EventDeliveryNote    `json:",omitempty" datastore:",omitempty,noindex"`
	Charges           []EventDeliveryCharges `json:",omitempty" datastore:",omitempty,noindex"`
	Completed         bool                   `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventDeliveryCharges has any additional charges (i.e., fuel or gratuity) or adjustments
type EventDeliveryCharges struct {
	Type     string  `json:",omitempty" datastore:",omitempty,noindex" enum:"Fuel, Gratuity, Other"`
	Quantity float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Rate     float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Charge   float32 `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventDeliveryNote is a note during check-out or check-in that may be associated with a spot on the hull or just in general
type EventDeliveryNote struct {
	Type   string  `json:",omitempty" datastore:",omitempty,noindex" enum:"Fuel, Oil, Hull Condition, Life Jackets, Furnishing / Seat Coverings, Fire Extinguishers, Flares, Charts / Navigation, Radio / Electronics, Engine / Propellers, Lines / Masts / Sails, Anchor, Other"`
	NA     bool    `json:",omitempty" datastore:",omitempty,noindex"`
	Point  []int   `json:",omitempty" datastore:",omitempty,noindex"`
	Note   string  `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Images []Image `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventFinance is to finance or re-finance a boat
type EventFinance struct {
	Status         string  `json:",omitempty" datastore:",omitempty,noindex" enum:"New, Used, Refinance"`
	Applicants     []User  `json:",omitempty" datastore:",omitempty,noindex"`
	UseAsResidence bool    `json:",omitempty" datastore:",omitempty,noindex"`
	BoatsFinanced  []Boat  `json:",omitempty" datastore:",omitempty,noindex"`
	BoatsPrevious  []Boat  `json:",omitempty" datastore:",omitempty,noindex"`
	BoatsTraded    []Boat  `json:",omitempty" datastore:",omitempty,noindex"`
	Price          float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Tax            float32 `json:",omitempty" datastore:",omitempty,noindex"`
	CashDown       float32 `json:",omitempty" datastore:",omitempty,noindex"`
	TradeAllowance float32 `json:",omitempty" datastore:",omitempty,noindex"`
	TradePayoffs   float32 `json:",omitempty" datastore:",omitempty,noindex"`
	AmountFinanced float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Term           int     `json:",omitempty" datastore:",omitempty,noindex"`
	APR            float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Monthly        float32 `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventInsure is to get insurance on a boat
type EventInsure struct {
	Use       string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Pleasure use exclusively, Racing/speed contests, Business/commercial use, Rented or leased to others, Primary residence"`
	IssueDate *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Boats     []Boat     `json:",omitempty" datastore:",omitempty,noindex"`
	Users     []User     `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventMessage is a private message from one user to another
type EventMessage struct {
	Text string `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
}

// EventNotification is for system-generated messages
type EventNotification struct {
	Text   string `json:",omitempty" datastore:",omitempty,noindex"`
	Action string `json:",omitempty" datastore:",omitempty,noindex" enum:"CheckOut, CheckIn, Review"`
}

// EventPayment is a rental or purchase payment, partial or full, made from the renter/buyer or to the owner/seller or to a tax authority
type EventPayment struct {
	IsDeposit bool    `json:",omitempty" datastore:",omitempty,noindex"`
	Currency  string  `json:",omitempty" datastore:",omitempty,noindex"`
	Amount    float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Token     string  `json:",omitempty" datastore:",omitempty,noindex"`
	Approval  string  `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventRental is when the renter makes an offer to rent, or changes that offer (i.e., new rental date or cancel), or when owner accepts or counters
type EventRental struct {
	Locations       []Contact           `json:",omitempty" datastore:",omitempty,noindex"`
	Start           *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
	End             *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
	CancelPolicy    string              `json:",omitempty" datastore:",omitempty,noindex" enum:"Flexible, Moderate, Strict"`
	Currency        string              `json:",omitempty" datastore:",omitempty,noindex"`
	Captain         string              `json:",omitempty" datastore:",omitempty,noindex" enum:"No Captain, Captain Included, Captain Extra"`
	Price           float32             `json:",omitempty" datastore:",omitempty,noindex"`
	CaptainFee      float32             `json:",omitempty" datastore:",omitempty,noindex"`
	CaptainUserID   int                 `json:",omitempty" datastore:",omitempty,noindex"`
	CaptainUser     *User               `json:",omitempty" datastore:",omitempty,noindex"`
	InsureFee       float32             `json:",omitempty" datastore:",omitempty,noindex"`
	TowFee          float32             `json:",omitempty" datastore:",omitempty,noindex"`
	TransactionFee  float32             `json:",omitempty" datastore:",omitempty,noindex"`
	RewardsDiscount float32             `json:",omitempty" datastore:",omitempty,noindex"`
	SalesTax        float32             `json:",omitempty" datastore:",omitempty,noindex"`
	TaxAuthority    string              `json:",omitempty" datastore:",omitempty,noindex"`
	Total           float32             `json:",omitempty" datastore:",omitempty,noindex"`
	SecurityDeposit float32             `json:",omitempty" datastore:",omitempty,noindex"`
	FuelPayer       string              `json:",omitempty" datastore:",omitempty,noindex" enum:"Renter, Owner"`
	Status          string              `json:",omitempty" datastore:",omitempty,noindex" enum:"Interested, Requested, Booked, Canceled, Blocked"`
	CancelCutOffs   []EventRentalCancel `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventRentalCancel shows how much is refunded if cancellation occurs before a CutOff date/time
type EventRentalCancel struct {
	CutOff *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Refund float32    `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventReview is a public review of a rental or sale
type EventReview struct {
	Text   string  `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Images []Image `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Rating int     `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventSale is when the buyer makes an offer to buy, or changes that offer (i.e., new price or cancel), or when owner accepts or counters
type EventSale struct {
	Fraction float32 `json:",omitempty" datastore:",omitempty,noindex"`
	Price    float32 `json:",omitempty" datastore:",omitempty,noindex"`
}

// EventService is to repair, maintain, or get other service for a boat
type EventService struct {
	Notes string `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
}

// EventTransport is when someone wants to transport a boat from one location to another
type EventTransport struct {
	Types          []string   `json:",omitempty" datastore:",omitempty,noindex" enum:"Open Transport, Tow-Away Service, Flatbed Transport Service, Enclosed Transport, In-water Delivery Service, Ocean Freight Container Service"`
	PickUpAfter    *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	PickUpBefore   *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	DeliverAfter   *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	DeliverBefore  *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Destination    *Contact   `json:",omitempty" datastore:",omitempty,noindex"`
	Notes          string     `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	TransportOrgID int        `json:",omitempty" datastore:",omitempty,noindex"`
	Price          float32    `json:",omitempty" datastore:",omitempty,noindex"`
	SalesTax       float32    `json:",omitempty" datastore:",omitempty,noindex"`
	TaxAuthority   string     `json:",omitempty" datastore:",omitempty,noindex"`
	Total          float32    `json:",omitempty" datastore:",omitempty,noindex"`
}

func init() {
	addEnumsFor(EventRental{})
	apiHandlers["GetEvents"] = GetEvents
	apiHandlers["SetEvent"] = SetEvent
	apiHandlers["ReadEvent"] = ReadEvent
}

// GetEvents gets events
func GetEvents(req *Request, pub *Publication) *Response {
	filters, _, resp := makeFilters(req, "")
	if resp != nil {
		return resp
	}
	resp = &Response{SubscriptionID: -1, Events: map[int64]*Event{}}
	var events []*Event
	keys, err := getAllEvents(filters, &events)
	if err != nil {
		return errResponse(err)
	}
	// process each event found
	for index, key := range keys {
		event := events[index]
		event.ID = key.ID
		// filter out by EventTypes
		if req.EventTypes != nil && (!StringInArray(getEventType(event), req.EventTypes)) {
			continue
		}
		// add User, Org, Boat, Deal, and FromUser
		event.User = getPublicUser(event.UserID)
		event.Org = getPublicOrg(event.OrgID)
		event.Boat = getPublicBoat(event.BoatID)
		event.Deal = getPublicDeal(event.DealID)
		event.FromUser = getPublicUser(event.FromUserID)
		resp.Events[key.ID] = event
	}
	return resp
}

func getEventType(event *Event) string {
	if event.Delivery != nil {
		return "Delivery"
	}
	if event.Message != nil {
		return "Message"
	}
	if event.Notification != nil {
		return "Notification"
	}
	if event.Payment != nil {
		return "Payment"
	}
	if event.Rental != nil {
		return "Rental"
	}
	if event.Review != nil {
		return "Review"
	}
	if event.Sale != nil {
		return "Sale"
	}
	return ""
}

// SetEvent sets an event
func SetEvent(req *Request, pub *Publication) *Response {
	e := req.Event
	if e == nil {
		return &Response{ErrorCode: "NeedEvent"}
	}
	if err := validate(e); err != nil {
		return errResponse(err)
	}
	eventKind, err := singleField(e, []string{"Delivery", "Message", "Notification", "Payment", "Rental", "Review", "Sale"})
	if err != nil {
		return errResponse(err)
	}
	messageToStaff := eventKind == "Message" && e.DealID == 0 && e.BoatID == 0 && e.UserID == 0 && e.UserIDs == nil && e.OrgIDs != nil
	if !messageToStaff && !isVerifiedUser(req) {
		return mustVerifyResp()
	}
	staff := isStaff(req)
	oldEvent, err := getEvent(e.ID)
	if err != nil {
		return errResponse(err)
	}
	// TODO: check DealID, BoatID, UserID, OrgID, etc.
	if e.UserID == 0 {
		e.UserID = req.Session.UserID // TODO
	}
	e.Deal = nil
	e.Boat = nil
	e.User = nil
	e.Org = nil
	if e.Rental != nil {
		rental := e.Rental
		rental.CaptainUser = nil
	}
	// finalize and save
	setAudit(staff, e, oldEvent)
	key, err := putEvent(e)
	if err != nil {
		return errResponse(err)
	}
	return &Response{
		ID: key.ID,
	}
}

// ReadEvent marks an event as read
func ReadEvent(req *Request, pub *Publication) *Response {
	// TODO
	return &Response{}
}
