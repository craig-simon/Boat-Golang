package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"cloud.google.com/go/datastore"
)

type mockDataStore struct {
	t     *testing.T
	pos   int
	calls []mockDataStoreCall
}

type mockDataStoreCall struct {
	name       string // Get, GetAll, or Put
	key        *datastore.Key
	keys       []*datastore.Key
	q          *datastore.Query
	src        interface{}
	srcJSON    string
	dst        interface{}
	keyResult  *datastore.Key
	keysResult []*datastore.Key
}

func (call *mockDataStoreCall) Serialize() string {
	s := ""
	switch call.name {
	case "Get":
		s = fmt.Sprintf("Get(ctx, %+v, dst)", call.key)
	case "GetAll":
		s = fmt.Sprintf("GetAll(ctx, %+v, dst)", call.q)
	case "GetMulti":
		s = fmt.Sprintf("GetMulti(ctx, %+v, dst)", call.keys)
	case "Put":
		s = fmt.Sprintf("Put(ctx, %+v, %s)", call.key, call.srcJSON)
	default:
		s = fmt.Sprintf("%s(...)", call.name)
	}
	return s
}

func (mds *mockDataStore) Do(actual *mockDataStoreCall) *mockDataStoreCall {
	if mds.pos >= len(mds.calls) {
		mds.t.Fatalf("mockDataStoreCall called too many times\n  actual:%s\n", actual.Serialize())
		return nil
	}
	expected := mds.calls[mds.pos]
	mds.pos++
	if actual.Serialize() != expected.Serialize() {
		mds.t.Fatalf("mockDataStore wrong call\n  actual:%s\n  expect:%s\n", actual.Serialize(), expected.Serialize())
		return nil
	}
	return &expected
}

func (mds *mockDataStore) Done() {
	if mds.pos < len(mds.calls) {
		mds.t.Fatalf("mockDataStoreCall not called enough\n")
	}
}

func (mds *mockDataStore) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	if dst == nil { // get catches nil interfaces; we need to catch nil ptr here
		return datastore.ErrInvalidEntityType
	}
	call := mds.Do(&mockDataStoreCall{name: "Get", key: key})
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(call.dst))
	return nil
}

func (mds *mockDataStore) GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error) {
	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return nil, datastore.ErrInvalidEntityType
	}
	call := mds.Do(&mockDataStoreCall{name: "GetAll", q: q})
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(call.dst))
	return call.keysResult, nil
}

func (mds *mockDataStore) GetMulti(ctx context.Context, keys []*datastore.Key, dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Slice {
		return datastore.ErrInvalidEntityType
	}
	if len(keys) != v.Len() {
		return errors.New("datastore: keys and dst slices have different length")
	}
	call := mds.Do(&mockDataStoreCall{name: "GetMulti", keys: keys})
	reflect.Copy(reflect.ValueOf(dst), reflect.ValueOf(call.dst))
	return nil
}

var redactPassword = regexp.MustCompile(`"PasswordHashCrypt":"[^"]+"`)

func (mds *mockDataStore) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	srcJSON, _ := json.Marshal(src)
	call := mds.Do(&mockDataStoreCall{
		name:    "Put",
		key:     key,
		src:     src,
		srcJSON: redactPassword.ReplaceAllString(string(srcJSON), `"PasswordHashCrypt":"REDACTED"`),
	})
	return call.keyResult, nil
}

func TestMakeStaffFirstTime(t *testing.T) {
	testTime = DateTime(2020, 5, 5, 5, 5, 5)
	mockDataStoreClient = &mockDataStore{
		t: t,
		calls: []mockDataStoreCall{
			{
				name:       "GetAll",
				q:          newQuery("Org", map[string]interface{}{"Types=": "Marketplace"}),
				dst:        []*Org{},
				keysResult: []*datastore.Key{},
			},
			{
				name:      "Put",
				key:       idKey("Org", 0),
				src:       []*Org{},
				srcJSON:   `{"Types":["Marketplace"],"Name":"Boat Fuji Inc.","Description":"The marketplace operator","Contacts":[{"Type":"Address","SubType":"Work","Line1":"3101 South US Highway 1","City":"Fort Pierce","County":"St. Lucie","State":"FL","Postal":"34982-6337","Country":"US","Location":{"Lat":27.4097155,"Lng":-80.329127},"Loc100KM":[11728,11729,12128,12129],"Loc300KM":[1239,1240,1372,1373]},{"Type":"Email","SubType":"Work","Email":"info@boatfuji.com"}],"EIN":"84-4243290","Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
				keyResult: idKey("Org", 1),
			},
			{
				name:      "Put",
				key:       idKey("User", 0),
				src:       []*User{},
				srcJSON:   `{"OrgID":1,"PasswordHashCrypt":"REDACTED","NameOrder":"GivenFamily","GivenName":"Dave","FamilyName":"Lampert","Gender":"Male","Images":[{"URL":"/i/EDC4B5EAFE5C91AE0614E21AE34137BF.jpg","Width":200,"Height":200}],"Contacts":[{"Type":"Email","SubType":"Work","Email":"dave.lampert@boatfuji.com"},{"Type":"Phone","SubType":"Mobile","Phone":"407-834-8834"}],"Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
				keyResult: idKey("User", 2),
			},
			{
				name:      "Put",
				key:       idKey("User", 0),
				src:       []*User{},
				srcJSON:   `{"OrgID":1,"PasswordHashCrypt":"REDACTED","NameOrder":"GivenFamily","GivenName":"Erik","FamilyName":"Breckenfelder","Gender":"Male","Images":[{"URL":"/i/78601FF342A85F39AE36B45FF480D19A.jpg","Width":200,"Height":200}],"Contacts":[{"Type":"Email","SubType":"Work","Email":"erik.breckenfelder@boatfuji.com"},{"Type":"Phone","SubType":"Mobile","Phone":"630-222-7505"}],"Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
				keyResult: idKey("User", 3),
			},
			{
				name:      "Put",
				key:       idKey("User", 0),
				src:       []*User{},
				srcJSON:   `{"PasswordHashCrypt":"REDACTED","Birthdate":"1988-06-02T00:00:00Z","NameOrder":"FamilyGiven","GivenName":"Jiazhen","FamilyName":"Lin","Description":"Follow your dreams!","Gender":"Female","Languages":["en-us","zh"],"Images":[{"URL":"/i/052DCC6A7CC5117A6122F615E19DCE56.jpg","Width":200,"Height":200}],"Contacts":[{"Type":"Address","SubType":"Work","Line1":"123 5th Avenue","City":"New York City","State":"NY","Postal":"10003","Location":{"Lat":40.7391967,"Lng":-73.9930489},"Loc100KM":[17734,17735,18134,18135],"Loc300KM":[1906,1907,2039,2040]},{"Type":"Address","SubType":"Home","Line1":"500 N Sweetzer Avenue","Line2":"Box 123","City":"Los Angeles","State":"CA","Postal":"90048","Location":{"Lat":34.0802574,"Lng":-118.3722444},"Loc100KM":[14893,14894,15294,15295],"Loc300KM":[1626,1627,1760,1761]},{"Type":"Email","SubType":"Work","Email":"info@awkwafina.com"},{"Type":"Email","SubType":"Home","Email":"awkwafina@gmail.com"},{"Type":"Phone","SubType":"Mobile","Phone":"213-555-1212"}],"Notifications":["RentalStartEnd","MessageReceived","SpecialOffers","News","Tips","UpcomingRentals","UserReviews","ReviewReminder","BookingExpired"],"Currency":"USD","BankAccounts":[{"Type":"Checking","Routing":"021000322","Account":"1234567890"}],"CreditCards":[{"NickName":"Gold Card","Last4":"1234","Token":"..."},{"NickName":"Cash Back","Last4":"9876","Token":"..."}],"Audit":{"Created":"2020-01-02T03:04:05Z","Updated":"2020-05-05T05:05:05Z","QANeeded":"2020-02-03T03:04:05Z","QAFields":["User.Description"],"User":{"Description":"Follow your dreams even if the world tells you you can't!"}}}`,
				keyResult: idKey("User", 4),
			},
			{
				name:      "Put",
				key:       idKey("Org", 0),
				src:       []*Org{},
				srcJSON:   `{"Types":["Crew"],"Name":"Yo Soy Capitan","Description":"Let us captain your boats so renters won't capsize them!","Contacts":[{"Type":"Address","SubType":"Work","Line1":"123 Brickell Ave","City":"Miami","County":"Miami-Dade","State":"FL","Postal":"33129","Country":"US","Location":{"Lat":25.7467903,"Lng":-80.2113866},"Loc100KM":[11328,11329,11728,11729],"Loc300KM":[1239,1240,1372,1373]},{"Type":"Email","SubType":"Work","Email":"captainnotcapsize@gmail.com"}],"EIN":"59-1234567","Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
				keyResult: idKey("Org", 5),
			},
			{
				name:      "Put",
				key:       idKey("User", 0),
				src:       []*User{},
				srcJSON:   `{"PasswordHashCrypt":"REDACTED","Birthdate":"1988-06-02T00:00:00Z","NameOrder":"GivenFamily","GivenName":"Bligh","FamilyName":"Blue","Description":"Captain! Not capsize!","Gender":"Male","Languages":["en-us","es"],"Images":[{"URL":"/i/75FF04C131EE08005781A097795AF6AA.jpg","Width":200,"Height":200}],"Contacts":[{"Type":"Address","SubType":"Work","Line1":"123 Brickell Ave","City":"Miami","County":"Miami-Dade","State":"FL","Postal":"33129","Country":"US","Location":{"Lat":25.7467903,"Lng":-80.2113866},"Loc100KM":[11328,11329,11728,11729],"Loc300KM":[1239,1240,1372,1373]},{"Type":"Email","SubType":"Work","Email":"captainnotcapsize@gmail.com"},{"Type":"Phone","SubType":"Mobile","Phone":"305-555-1212"}],"Notifications":["RentalStartEnd","MessageReceived","SpecialOffers","News","Tips","UpcomingRentals","UserReviews","ReviewReminder","BookingExpired"],"Currency":"USD","BankAccounts":[{"Type":"Checking","Routing":"021000322","Account":"314159265"}],"Audit":{"Created":"2020-05-05T05:05:05Z"}}`,
				keyResult: idKey("User", 6),
			},
		},
	}
	makeStaffFirstTime()
	mockDataStoreClient.(*mockDataStore).Done()
}

// func TestInteractiveData(t *testing.T) {
// 	os.Setenv("DATASTORE_EMULATOR_HOST", "localhost:8169")
// 	apiContext = context.Background()
// 	client, err := datastore.NewClient(apiContext, "boatfuji")
// 	if err != nil {
// 		panic(err)
// 	}
// 	datastoreClient = client
// 	// var dst []*User
// 	// query := datastore.NewQuery("User").Filter("OrgID=", 1).Limit(10)
// 	var dst []*Boat
// 	query := datastore.NewQuery("Boat").Filter("Location.Loc100KM=", 10526).Limit(10)
// 	// var dst []*Event
// 	// query := datastore.NewQuery("Event").Filter("BoatID=", 2470).Limit(1000)
// 	keys, err := datastoreClient.GetAll(apiContext, query, &dst)
// 	if err != nil {
// 		fmt.Printf("error: %s\n", err.Error())
// 	} else {
// 		fmt.Printf("%d record(s)\n", len(keys))
// 		for i, key := range keys {
// 			// fmt.Printf("%d\t%s\t%v\n", key.ID, dst[i].GivenName, dst[i].PasswordHashCrypt)
// 			fmt.Printf("%d\t%s\t%s\t%v\n", key.ID, dst[i].URLs[0], dst[i].Location.City, dst[i].Location.Loc100KM)
// 			// fmt.Printf("%d\t%d\t%d\t%v\n", key.ID, dst[i].UserID, dst[i].BoatID, len(dst[i].Review.Text))
// 		}
// 	}
// }
