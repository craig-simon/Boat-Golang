package api

import (
	"testing"

	"cloud.google.com/go/datastore"
)

func TestGetEvents(t *testing.T) {
	session := &Session{UserID: 123}
	// TODO
	testAPI(t, session, nil, "GetEvents", `{}`, `{"SubscriptionID":-1,"Boats":{"201":{"ID":201,"Make":"#201","Audit":{}},"202":{"ID":202,"Make":"#202","Audit":{}},"301":{"ID":301,"Make":"#301","Audit":{}},"302":{"ID":302,"Make":"#302","Audit":{}}}}`, []mockDataStoreCall{
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
}

func TestSetEvent(t *testing.T) {
}
