package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSetContacts(t *testing.T) {
	testTime = DateTime(2020, 5, 5, 5, 5, 5)
	testVerifyCode = "1234"
	test := func(newContacts, oldContacts []Contact, expect string) {
		err := setContacts(newContacts, oldContacts, nil)
		actual, _ := json.Marshal(newContacts)
		if err != nil {
			actual = []byte(err.Error())
		}
		if string(actual) != expect {
			t.Errorf("setContacts wrong result\n  actual:%s\n  expect:%s\n", actual, expect)
		}
	}
	test([]Contact{{}}, nil, "NeedType")
	test([]Contact{{Type: "Email", SubType: "Fax"}}, nil, "BadSubType")
	test([]Contact{{Type: "Email", Extension: "123"}}, nil, "ExtraContactData")
	test([]Contact{{Type: "Address", Country: "Canada"}}, nil, "BigAddress")
	// test([]Contact{{Type: "Address", Country: "US"}}, nil, "NeedLocation")
	test([]Contact{{Type: "Address", Location: LatLng(91, 0)}}, nil, "BadLat")
	test([]Contact{{Type: "Address", Location: LatLng(27.4097155, -80.329127)}}, nil, `[{"Type":"Address","Location":{"Lat":27.4097155,"Lng":-80.329127},"Loc100KM":[11728,11729,12128,12129],"Loc300KM":[1239,1240,1372,1373]}]`)
	test([]Contact{{Type: "Email", Email: "example@gmail"}}, nil, "BadEmail")
	test([]Contact{{Type: "Email", Email: "Example@gmail.com"}}, nil, `[{"Type":"Email","Email":"example@gmail.com"}]`)
	test([]Contact{{Type: "Phone", Phone: "407-555-"}}, nil, "BadPhone")
	test([]Contact{{Type: "Phone", Phone: " +1 (407) 555-1212 "}}, nil, `[{"Type":"Phone","Phone":"407-555-1212"}]`)
	test([]Contact{{Type: "Phone", Phone: "+40 12345678901234567890 "}}, nil, "BigPhone")
	test([]Contact{{Type: "Phone", Phone: " +40 (31) 123 45 67 "}}, nil, `[{"Type":"Phone","Phone":"+40 (31) 123 45 67"}]`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", Extension: "123456789"}}, nil, "BigExtension")
	test([]Contact{{Type: "Phone", Phone: "4075551212", Extension: "ABC"}}, nil, "BadExtension")
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "SEND"}}, nil, `[{"Type":"Phone","Phone":"407-555-1212","VerifyCode":"1234","Verifying":"2020-05-05T05:05:05Z"}]`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "SEND"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1111", Verifying: DateTime(2020, 5, 5, 5, 5, 0)}}, `MustWaitToResendCode`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "SEND"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1111", Verifying: DateTime(2020, 5, 5, 5, 4, 0)}}, `[{"Type":"Phone","Phone":"407-555-1212","VerifyCode":"1234","Verifying":"2020-05-05T05:05:05Z"}]`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "SENT"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1234", Verifying: DateTime(2020, 5, 5, 5, 5, 0)}}, `[{"Type":"Phone","Phone":"407-555-1212","VerifyCode":"1234","Verifying":"2020-05-05T05:05:00Z"}]`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "BAD"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1234", Verifying: DateTime(2020, 5, 5, 5, 5, 0)}}, `BadVerifyCode`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "4321"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1234", Verifying: DateTime(2020, 5, 5, 5, 5, 0)}}, `WrongVerifyCode`)
	test([]Contact{{Type: "Phone", Phone: "4075551212", VerifyCode: "1234"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", VerifyCode: "1234", Verifying: DateTime(2020, 5, 5, 5, 5, 0)}}, `[{"Type":"Phone","Phone":"407-555-1212","Verified":"2020-05-05T05:05:05Z"}]`)
	test([]Contact{{Type: "Phone", Phone: "4075555555", VerifyCode: "SEND"}}, []Contact{{Type: "Phone", Phone: "407-555-1212", Verified: DateTime(2020, 5, 5, 5, 5, 5)}}, `[{"Type":"Phone","Phone":"407-555-5555","VerifyCode":"1234","Verifying":"2020-05-05T05:05:05Z"}]`)
	test([]Contact{{Type: "URL", URL: strings.Repeat("-", 3000)}}, nil, "BigURL")
}

func TestVerifyCode(t *testing.T) {
	// _, err := verifyCode("dave.lampert@rpm6.com", "")
	// _, err := verifyCode("", "407-834-8834")
	// if err != nil {
	// 	t.Error(err)
	// }
}
