package api

import (
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestErrResponse(t *testing.T) {
	test := func(err string, expect *Response) {
		r := errResponse(errors.New(err))
		if !reflect.DeepEqual(r, expect) {
			actualJSON, _ := json.Marshal(r)
			expectJSON, _ := json.Marshal(expect)
			t.Errorf("errResponse wrong result\n  actual:%s\n  expect:%s\n", actualJSON, expectJSON)
		}
		e := Err(expect.ErrorCode, expect.ErrorDetails).Error()
		if e != err {
			t.Errorf("Err wrong result\n  actual:%s\n  expect:%s\n", e, err)
		}
	}
	test("Test", &Response{ErrorCode: "Test"})
	test(`Test{"A":"1","B":"2"}`, &Response{ErrorCode: "Test", ErrorDetails: map[string]string{"A": "1", "B": "2"}})
}

func testAPI(t *testing.T, session *Session, pub *Publication, apiName, reqJSON, respJSON string, dbCalls []mockDataStoreCall) {
	// respJSON may be like `{"ErrorCode":"BadStuff","ErrorDetails":/.*/}`
	// so fragments within slashes can be regular expressions
	testTime = DateTime(2020, 5, 5, 5, 5, 5)
	if session == nil {
		session = &Session{}
	}
	if dbCalls == nil {
		dbCalls = []mockDataStoreCall{}
	}
	mockDataStoreClient = &mockDataStore{
		t:     t,
		calls: dbCalls,
	}
	req := &Request{}
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		t.Errorf("Bad request: %s", reqJSON)
	}
	req.Session = session
	req.Subscription = nil
	if handler, ok := apiHandlers[apiName]; ok {
		resp := handler(req, pub)
		actualJSONBytes, _ := json.Marshal(resp)
		actualJSON := string(actualJSONBytes)
		if !matchString(actualJSON, respJSON) {
			t.Errorf("Wrong %s response\nActual %s\nExpect %s\n", apiName, actualJSON, respJSON)
		}
		mockDataStoreClient.(*mockDataStore).Done()
	}
}

var slashPattern = regexp.MustCompile(`/[^/]+/`)

func matchString(subject, pattern string) bool {
	if !strings.Contains(pattern, "/") {
		return subject == pattern
	}
	pattern = "^" + slashPattern.ReplaceAllStringFunc("/"+pattern+"/", func(nonre string) string {
		return regexp.QuoteMeta(strings.Trim(nonre, "/"))
	}) + "$"
	return regexp.MustCompile(pattern).MatchString(subject)
}
