package api

import (
	"testing"
)

func TestSetUser(t *testing.T) {
	session := &Session{}
	testAPI(t, session, nil, "SetUser", `{}`, `{"ErrorCode":"NeedUser"}`, nil)
	testAPI(t, session, nil, "SetUser", `{"User":{"PasswordHash":"...","GivenName":"Dave","Contacts":[{"Type":"Email","Email":"johndoe@example.org"}]}}`, `{"ID":123}`, []mockDataStoreCall{
		{
			name:      "Put",
			key:       idKey("User", 0),
			src:       []*User{},
			srcJSON:   `{"PasswordHashCrypt":"REDACTED","Contacts":[{"Type":"Email","Email":"johndoe@example.org"}],"Audit":{"Created":"2020-05-05T05:05:05Z","QANeeded":"2020-05-05T05:05:05Z","QAFields":["User.GivenName"],"User":{"GivenName":"Dave"}}}`,
			keyResult: idKey("User", 123),
		},
	})
	if session.Verified != false {
		t.Error("session.Verified should be false")
	}
	testVerifyCode = "1234"
	testAPI(t, session, nil, "SetUser", `{"User":{"ID":123,"GivenName":"Dave","Contacts":[{"Type":"Email","Email":"johndoe@example.org","VerifyCode":"SEND"}]}}`, `{"ID":123}`, []mockDataStoreCall{
		{
			name: "Get",
			key:  idKey("User", 123),
			dst:  User{PasswordHashCrypt: "REDACTED", Contacts: []Contact{{Type: "Email", Email: "johndoe@example.org"}}, Audit: &Audit{Created: DateTime(2020, 5, 5, 5, 5, 5), QANeeded: DateTime(2020, 5, 5, 5, 5, 5), QAFields: []string{"User.GivenName"}, User: &User{GivenName: "Dave"}}},
		},
		{
			name:      "Put",
			key:       idKey("User", 123),
			src:       []*User{},
			srcJSON:   `{"ID":123,"PasswordHashCrypt":"REDACTED","Contacts":[{"Type":"Email","Email":"johndoe@example.org","VerifyCode":"1234","Verifying":"2020-05-05T05:05:05Z"}],"Audit":{"Created":"2020-05-05T05:05:05Z","Updated":"2020-05-05T05:05:05Z","QANeeded":"2020-05-05T05:05:05Z","QAFields":["User.GivenName"],"User":{"GivenName":"Dave"}}}`,
			keyResult: idKey("User", 123),
		},
	})
	if session.Verified != false {
		t.Error("session.Verified should be false")
	}
	testAPI(t, session, nil, "SetUser", `{"User":{"ID":123,"GivenName":"Dave","Contacts":[{"Type":"Email","Email":"johndoe@example.org","VerifyCode":"1234"},{"Type":"Phone","Phone":"407-555-1212","VerifyCode":"5678"}]}}`, `{"ID":123}`, []mockDataStoreCall{
		{
			name: "Get",
			key:  idKey("User", 123),
			dst:  User{PasswordHashCrypt: "REDACTED", Contacts: []Contact{{Type: "Email", Email: "johndoe@example.org", VerifyCode: "1234", Verifying: DateTime(2020, 5, 5, 5, 5, 5)}, {Type: "Phone", Phone: "407-555-1212", VerifyCode: "5678", Verifying: DateTime(2020, 5, 5, 5, 5, 5)}}, Audit: &Audit{Created: DateTime(2020, 5, 5, 5, 5, 5), QANeeded: DateTime(2020, 5, 5, 5, 5, 5), QAFields: []string{"User.GivenName"}, User: &User{GivenName: "Dave"}}},
		},
		{
			name:      "Put",
			key:       idKey("User", 123),
			src:       []*User{},
			srcJSON:   `{"ID":123,"PasswordHashCrypt":"REDACTED","Contacts":[{"Type":"Email","Email":"johndoe@example.org","Verified":"2020-05-05T05:05:05Z"},{"Type":"Phone","Phone":"407-555-1212","Verified":"2020-05-05T05:05:05Z"}],"Audit":{"Created":"2020-05-05T05:05:05Z","Updated":"2020-05-05T05:05:05Z","QANeeded":"2020-05-05T05:05:05Z","QAFields":["User.GivenName"],"User":{"GivenName":"Dave"}}}`,
			keyResult: idKey("User", 123),
		},
	})
	if session.Verified != true {
		t.Error("session.Verified should be true")
	}
	session.OrgID = 124
	testAPI(t, session, nil, "SetUser", `{"User":{"ID":123,"OrgID":125}}`, `{"ErrorCode":"BadOrgID"}`, nil)
	testAPI(t, session, nil, "SetUser", `{"User":{"ID":123,"OrgID":124}}`, `{"ID":123}`, []mockDataStoreCall{
		{
			name: "Get",
			key:  idKey("User", 123),
			dst:  User{},
		},
		{
			name:      "Put",
			key:       idKey("User", 123),
			src:       []*User{},
			srcJSON:   `{"ID":123,"OrgID":124,"Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
			keyResult: idKey("User", 123),
		},
	})
}
