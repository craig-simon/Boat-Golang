package api

import (
	"testing"
)

func TestSetOrg(t *testing.T) {
	testAPI(t, &Session{}, nil, "SetOrg", `{}`, `{"ErrorCode":"MustVerify"}`, nil)
	session := &Session{UserID: 123, Verified: true}
	testAPI(t, session, nil, "SetOrg", `{}`, `{"ErrorCode":"NeedOrg"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{}}`, `{"ErrorCode":"NeedOrgTypes"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{"Types":[]}}`, `{"ErrorCode":"NeedOrgTypes"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{"Types":["Crew","Marketplace"]}}`, `{"ErrorCode":"BadOrgTypes"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{"Types":["Insurer"]}}`, `{"ErrorCode":"BadOrgTypes"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{"Types":["Crew"],"Name":"Acme"}}`, `{"ID":124}`, []mockDataStoreCall{
		{
			name:      "Put",
			key:       idKey("Org", 0),
			src:       []*Org{},
			srcJSON:   `{"Types":["Crew"],"Audit":{"Created":"2020-05-05T05:05:05Z","QANeeded":"2020-05-05T05:05:05Z","QAFields":["Org.Name"],"Org":{"Name":"Acme"}}}`,
			keyResult: idKey("Org", 124),
		},
	})
	testAPI(t, session, nil, "SetOrg", `{"Org":{"ID":125,"Types":["Crew"],"Name":"Acme 2"}}`, `{"ErrorCode":"AccessDenied"}`, nil)
	testAPI(t, session, nil, "SetOrg", `{"Org":{"ID":124,"Types":["Crew"],"Name":"Acme 2"}}`, `{"ID":124}`, []mockDataStoreCall{
		{
			name: "Get",
			key:  idKey("Org", 124),
			dst:  Org{Types: []string{"Crew"}, Audit: &Audit{Created: DateTime(2020, 5, 5, 5, 5, 5), QANeeded: DateTime(2020, 5, 5, 5, 5, 5), QAFields: []string{"Org.Name"}, Org: &Org{Name: "Acme"}}},
		},
		{
			name:      "Put",
			key:       idKey("Org", 124),
			src:       []*Org{},
			srcJSON:   `{"ID":124,"Types":["Crew"],"Audit":{"Created":"2020-05-05T05:05:05Z","Updated":"2020-05-05T05:05:05Z","QANeeded":"2020-05-05T05:05:05Z","QAFields":["Org.Name"],"Org":{"Name":"Acme 2"}}}`,
			keyResult: idKey("Org", 124),
		},
	})
}
