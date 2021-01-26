package api

import (
	"testing"
	"time"

	"cloud.google.com/go/datastore"
)

func TestGetBoats(t *testing.T) {
	session := &Session{UserID: 123}
	testAPI(t, session, nil, "GetBoats", `{}`, `{"SubscriptionID":-1,"Boats":{"201":{"ID":201,"Make":"#201","Audit":{}},"202":{"ID":202,"Make":"#202","Audit":{}},"301":{"ID":301,"Make":"#301","Audit":{}},"302":{"ID":302,"Make":"#302","Audit":{}}}}`, []mockDataStoreCall{
		{
			name: "Get",
			key:  idKey("User", 123),
			dst:  User{Favorites: []int64{201, 202}},
		},
		{
			name:       "GetAll",
			q:          newQuery("Boat", map[string]interface{}{"UserID=": 123}),
			dst:        []*Boat{{Make: "#301"}, {Make: "#302"}},
			keysResult: []*datastore.Key{idKey("Boat", 301), idKey("Boat", 302)},
		},
		{
			name: "GetMulti",
			keys: []*datastore.Key{idKey("Boat", 201), idKey("Boat", 202)},
			dst:  []*Boat{{Make: "#201"}, {Make: "#202"}},
		},
	})
	testAPI(t, session, nil, "GetBoats", `{"Location":{"Lat":30,"Lng":-90},"StartDate":"2020-01-25T13:00:00.000Z","EndDate":"2020-01-25T17:00:00.000Z"}`, `{"SubscriptionID":-1,"Boats":{"101":{"ID":101,"Make":"#101","Rental":{"ListingTitle":"Super!","RentalIfCaptain":{"Start":"2020-05-07T13:00:00Z","End":"2020-05-07T17:00:00Z","Captain":"CaptainIncluded","Price":700,"InsureFee":140,"TowFee":35,"TransactionFee":70,"SalesTax":66.15,"Total":1011.15,"SecurityDeposit":500,"FuelPayer":"Owner","CancelCutOffs":[{"CutOff":"2020-05-06T13:00:00Z","Refund":1011.15}]},"RentalIfNoCaptain":{"Start":"2020-05-07T13:00:00Z","End":"2020-05-07T17:00:00Z","Captain":"NoCaptain","Price":600,"InsureFee":120,"TowFee":30,"TransactionFee":60,"SalesTax":56.7,"Total":866.7,"SecurityDeposit":500,"FuelPayer":"Renter","CancelCutOffs":[{"CutOff":"2020-05-06T13:00:00Z","Refund":866.7}]},"NextAvailable":["2020-05-07T13:00:00Z","2020-05-07T17:00:00Z"]},"Audit":{}}}}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("Boat", map[string]interface{}{"Location.Loc100KM=": 13320}),
			dst:        []*Boat{},
			keysResult: []*datastore.Key{},
		},
		{
			name: "GetAll",
			q:    newQuery("Boat", map[string]interface{}{"Location.Loc300KM=": 1503}),
			dst: []*Boat{
				{
					Make: "#101",
					Rental: &BoatRental{
						ListingTitle: "Super!",
						Seasons: []BoatRentalSeason{
							{
								StartDay: DateTime(2000, 1, 1, 0, 0, 0),
								EndDay:   DateTime(2000, 12, 31, 0, 0, 0),
								Pricing: []BoatRentalPricing{
									{
										Captain:        "NoCaptain",
										DailyPrice:     1000,
										HalfDailyPrice: 600,
										FuelPayer:      "Renter",
									},
									{
										Captain:        "CaptainIncluded",
										DailyPrice:     1200,
										HalfDailyPrice: 700,
										FuelPayer:      "Owner",
									},
								},
							},
						},
						NotAvailable: []time.Time{
							*DateTime(2020, 5, 5, 13, 0, 0), *DateTime(2020, 5, 5, 21, 0, 0),
							*DateTime(2020, 5, 6, 13, 0, 0), *DateTime(2020, 5, 6, 17, 0, 0),
							*DateTime(2020, 5, 8, 13, 0, 0), *DateTime(2020, 5, 8, 17, 0, 0),
						},
					},
				},
				{
					Make: "#102",
				},
			},
			keysResult: []*datastore.Key{idKey("Boat", 101), idKey("Boat", 102)},
		},
	})
}

func TestSetBoat(t *testing.T) {
}
