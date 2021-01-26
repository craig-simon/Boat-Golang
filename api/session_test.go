package api

import (
	"testing"

	"cloud.google.com/go/datastore"
)

func TestSignIn(t *testing.T) {
	testAPI(t, nil, nil, "SignIn", `{}`, `{"ErrorCode":"NeedUser"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{}}`, `{"Bearer":"/[\w\.\-]+/","ExpiresIn":/\d+/}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"johndoe@example.org"}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Email=": "johndoe@example.org"}),
			dst:        []*User{},
			keysResult: []*datastore.Key{},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"+91 12345678901234567890"}}`, `{"ErrorCode":"BigPhone"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"(407) 555-1212"}}`, `{"ErrorCode":"NeedPasswordHash"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Phone=": "407-555-1212"}),
			dst:        []*User{{GivenName: "John"}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"(407) 555-1212","PasswordHash":"..."}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Phone=": "407-555-1212"}),
			dst:        []*User{{GivenName: "John"}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"(407) 555-1212","PasswordHash":"1e73a0bda445ec13d0cd82feaaf9ca9a"}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Phone=": "407-555-1212"}),
			dst:        []*User{{GivenName: "John", PasswordHashCrypt: ""}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"Dave.Lampert@boatfuji.com","PasswordHash":"19b39b361282dc1165b818e7a8a8cde1"}}`, `{"Bearer":"/[\w\.\-]+/","ExpiresIn":/\d+/,"ID":123}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Email=": "dave.lampert@boatfuji.com"}),
			dst:        []*User{{GivenName: "Dave", PasswordHashCrypt: "$2a$13$rEvw.Fy1.0Q7ENQ9Trn5FeP0V3AyoWxFBaw2VUcLIwR1oGdy5MZge"}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"Dave.Lampert@boatfuji.com","PasswordHash":"19b39b361282dc1165b818e7a8a8cde2"}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Email=": "dave.lampert@boatfuji.com"}),
			dst:        []*User{{GivenName: "Dave", PasswordHashCrypt: "$2a$13$rEvw.Fy1.0Q7ENQ9Trn5FeP0V3AyoWxFBaw2VUcLIwR1oGdy5MZge"}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"TOTP":"SEND"}}`, `{"ErrorCode":"NeedUserName"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"Dave.Lampert@boatfuji.com","TOTP":"SEND"}}`, `{"ErrorCode":"MustWaitToResendCode"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Email=": "dave.lampert@boatfuji.com"}),
			dst:        []*User{{GivenName: "Dave", TOTPSent: DateTime(2020, 5, 5, 5, 5, 0)}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testVerifyCode = "12345678"
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"Dave.Lampert@boatfuji.com","TOTP":"SEND"}}`, `{}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.Email=": "dave.lampert@boatfuji.com"}),
			dst:        []*User{{GivenName: "Dave", TOTPSent: DateTime(2020, 5, 5, 5, 4, 0)}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
		{
			name:      "Put",
			key:       idKey("User", 123),
			src:       []*User{},
			srcJSON:   `{"ID":123,"TOTP":"12345678","TOTPSent":"2020-05-05T05:05:05Z","GivenName":"Dave"}`,
			keyResult: idKey("User", 1),
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"UserName":"Dave.Lampert@boatfuji.com","TOTP":"12345678"}}`, `{"ErrorCode":"NeedNoUserName"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"TOTP":"12345678"}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"TOTP=": "12345678"}),
			dst:        []*User{},
			keysResult: []*datastore.Key{},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"TOTP":"12345678"}}`, `{"ErrorCode":"TOTPExpired"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"TOTP=": "12345678"}),
			dst:        []*User{{GivenName: "Dave", TOTP: "12345678", TOTPSent: DateTime(2020, 5, 5, 5, 0, 0)}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"TOTP":"12345678"}}`, `{"Bearer":"/[\w\.\-]+/","ExpiresIn":/\d+/,"ID":123}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"TOTP=": "12345678"}),
			dst:        []*User{{GivenName: "Dave", TOTP: "12345678", TOTPSent: DateTime(2020, 5, 5, 5, 5, 0)}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
		{
			name:      "Put",
			key:       idKey("User", 123),
			src:       []*User{},
			srcJSON:   `{"ID":123,"GivenName":"Dave"}`,
			keyResult: idKey("User", 1),
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[]}}`, `{"ErrorCode":"Need1Contact"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[{}]}}`, `{"ErrorCode":"NeedOAuthID"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[{"OAuthID":"2734407573470227"}]}}`, `{"ErrorCode":"NeedOAuthToken"}`, nil)
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[{"OAuthID":"2734407573470227","OAuthToken":"..."}]}}`, `{"ErrorCode":"NeedOAuthType"}`, nil)
	testJSONFromURL = `{"name":"John Doe","id":"2734407573470227"}`
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[{"Type":"Facebook","OAuthID":"2734407573470227","OAuthToken":"..."}]}}`, `{"ErrorCode":"AccessDenied"}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.OAuthID=": "2734407573470227"}),
			dst:        []*User{},
			keysResult: []*datastore.Key{},
		},
	})
	testAPI(t, nil, nil, "SignIn", `{"User":{"Contacts":[{"Type":"Facebook","OAuthID":"2734407573470227","OAuthToken":"..."}]}}`, `{"Bearer":"/[\w\.\-]+/","ExpiresIn":/\d+/,"ID":123}`, []mockDataStoreCall{
		{
			name:       "GetAll",
			q:          newQuery("User", map[string]interface{}{"Contacts.OAuthID=": "2734407573470227"}),
			dst:        []*User{{GivenName: "Dave"}},
			keysResult: []*datastore.Key{{Kind: "User", ID: 123}},
		},
	})
}
