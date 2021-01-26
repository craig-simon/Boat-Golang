package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGetAudit(t *testing.T) {
	req := &Request{Session: &Session{UserID: 123}}
	ptr := &Boat{
		UserID: 123,
		Year:   2020,
		Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}},
		Rental: &BoatRental{
			ListingTitle:       "My boat title",
			ListingDescription: "My boat description",
			ListingSummary:     "My boat summary",
			Rules:              "My boat rules",
		},
		Audit: &Audit{
			Created:  DateTime(2020, 1, 2, 3, 4, 5),
			Updated:  DateTime(2020, 1, 2, 3, 4, 35),
			QANeeded: DateTime(2020, 1, 2, 3, 4, 35),
			QAFields: []string{"Boat.Images", "Boat.Rental.Rules"},
			Boat: &Boat{
				Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}, {URL: "/i/3.jpg"}},
				Rental: &BoatRental{
					Rules: "My improved boat rules",
				},
			},
		},
	}
	getAudit(req, ptr)
	expect := &Boat{
		UserID: 123,
		Year:   2020,
		Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}, {URL: "/i/3.jpg"}},
		Rental: &BoatRental{
			ListingTitle:       "My boat title",
			ListingDescription: "My boat description",
			ListingSummary:     "My boat summary",
			Rules:              "My improved boat rules",
		},
		Audit: &Audit{
			Created: DateTime(2020, 1, 2, 3, 4, 5),
			Updated: DateTime(2020, 1, 2, 3, 4, 35),
		},
	}
	if !reflect.DeepEqual(ptr, expect) {
		actualJSON, _ := json.Marshal(ptr)
		expectJSON, _ := json.Marshal(expect)
		t.Errorf("getAudit wrong result\n  actual:%s\n  expect:%s\n", actualJSON, expectJSON)
	}
}

func TestSetAudit(t *testing.T) {
	testTime = DateTime(2020, 5, 5, 5, 5, 5)
	oldPtr := &Boat{
		UserID: 123,
		Year:   2020,
		Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}},
		Rental: &BoatRental{
			ListingTitle:       "My boat title",
			ListingDescription: "My boat description",
			ListingSummary:     "My boat summary",
			Rules:              "My boat rules",
		},
		Audit: &Audit{
			Created: DateTime(2020, 1, 2, 3, 4, 5),
			Updated: DateTime(2020, 1, 2, 3, 4, 35),
		},
	}
	newPtr := &Boat{
		UserID: 123,
		Year:   2020,
		Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}, {URL: "/i/3.jpg"}},
		Rental: &BoatRental{
			ListingTitle:       "My boat title",
			ListingDescription: "My boat description",
			ListingSummary:     "My boat summary",
			Rules:              "My improved boat rules",
		},
		Audit: &Audit{
			Created: DateTime(2020, 1, 2, 3, 4, 5),
			Updated: DateTime(2020, 1, 2, 3, 4, 35),
		},
	}
	setAudit(false, newPtr, oldPtr)
	expect := &Boat{
		UserID: 123,
		Year:   2020,
		Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}},
		Rental: &BoatRental{
			ListingTitle:       "My boat title",
			ListingDescription: "My boat description",
			ListingSummary:     "My boat summary",
			Rules:              "My boat rules",
		},
		Audit: &Audit{
			Created:  DateTime(2020, 1, 2, 3, 4, 5),
			Updated:  DateTime(2020, 5, 5, 5, 5, 5),
			QANeeded: DateTime(2020, 5, 5, 5, 5, 5),
			QAFields: []string{"Boat.Images", "Boat.Rental.Rules"},
			Boat: &Boat{
				Images: []Image{{URL: "/i/1.jpg"}, {URL: "/i/2.jpg"}, {URL: "/i/3.jpg"}},
				Rental: &BoatRental{
					Rules: "My improved boat rules",
				},
			},
		},
	}
	if !reflect.DeepEqual(newPtr, expect) {
		actualJSON, _ := json.Marshal(newPtr)
		expectJSON, _ := json.Marshal(expect)
		t.Errorf("setAudit wrong result\n  actual:%s\n  expect:%s\n", actualJSON, expectJSON)
	}
}
