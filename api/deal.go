package api

import "log"

// Deal is a deal for a boat, such as a rental, sale, etc.
type Deal struct {
	ID          int64           `json:",omitempty" datastore:"-"`
	BoatID      int64           `json:",omitempty" datastore:",omitempty"`
	Boat        *Boat           `json:",omitempty" datastore:"-"`
	UserID      int64           `json:",omitempty" datastore:",omitempty"`
	User        *User           `json:",omitempty" datastore:"-"`
	OrgID       int64           `json:",omitempty" datastore:",omitempty"`
	Org         *Org            `json:",omitempty" datastore:"-"`
	CustomerIDs []int           `json:",omitempty" datastore:",omitempty"`
	Rental      *EventRental    `json:",omitempty" datastore:",omitempty"`
	Cruise      *EventRental    `json:",omitempty" datastore:",omitempty"`
	Ride        *EventRental    `json:",omitempty" datastore:",omitempty"`
	Sale        *EventSale      `json:",omitempty" datastore:",omitempty"`
	Finance     *EventFinance   `json:",omitempty" datastore:",omitempty"`
	Insure      *EventInsure    `json:",omitempty" datastore:",omitempty"`
	Transport   *EventTransport `json:",omitempty" datastore:",omitempty"`
	Service     *EventService   `json:",omitempty" datastore:",omitempty"`
	Crew        *EventCrew      `json:",omitempty" datastore:",omitempty"`
	Audit       *Audit          `json:",omitempty" datastore:",omitempty"`
}

func init() {
	apiHandlers["GetDeals"] = GetDeals
	apiHandlers["SetDeal"] = SetDeal
}

func getPublicDeal(dealID int64) *Deal {
	if dealID == 0 {
		return nil
	}
	deal, err := getDeal(dealID)
	if err != nil {
		log.Printf("getDeal(%d) => %s", dealID, err.Error())
		return nil
	}
	if deal.Rental == nil {
		deal.Rental = &EventRental{}
	}
	return &Deal{
		Rental: &EventRental{Start: deal.Rental.Start, End: deal.Rental.End},
		Audit: &Audit{
			Created: deal.Audit.Created,
		},
	}
}

// GetDeals gets deals
func GetDeals(req *Request, pub *Publication) *Response {
	result := map[int64]*Deal{}
	// TODO
	return &Response{
		SubscriptionID: -1,
		Deals:          result,
	}
}

// SetDeal sets a deal
func SetDeal(req *Request, pub *Publication) *Response {
	if req.Deal == nil {
		return &Response{ErrorCode: "NeedDeal"}
	}
	if err := validate(req.Deal); err != nil {
		return errResponse(err)
	}
	_, err := singleField(req.Deal, []string{"Rental"})
	if err != nil {
		return errResponse(err)
	}
	staff := isStaff(req)
	oldDeal, err := getDeal(req.Deal.ID)
	if err != nil {
		return errResponse(err)
	}
	// TODO: check BoatID, UserID, OrgID, CustomerIDs, Rental
	req.Deal.Boat = nil
	req.Deal.User = nil
	req.Deal.Org = nil
	if req.Deal.Rental != nil {
		rental := req.Deal.Rental
		rental.CaptainUser = nil
	}
	// finalize and save
	setAudit(staff, req.Deal, oldDeal)
	key, err := putDeal(req.Deal)
	if err != nil {
		return errResponse(err)
	}
	return &Response{
		ID: key.ID,
	}
}
