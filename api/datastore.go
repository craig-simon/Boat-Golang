package api

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"

	"cloud.google.com/go/datastore"
)

var apiContext context.Context
var datastoreClient *datastore.Client

func startDataStore() {
	os.Setenv("DATASTORE_EMULATOR_HOST", "localhost:8169")
	apiContext = context.Background()
	client, err := datastore.NewClient(apiContext, "boatfuji")
	if err != nil {
		panic(err)
	}
	datastoreClient = client
	makeStaffFirstTime()
	makeStandardOrgs()
}

func newQuery(kind string, filters map[string]interface{}) *datastore.Query {
	query := datastore.NewQuery(kind)
	for filterName, filterValue := range filters {
		switch filterName {
		case "order":
			query = query.Order(filterValue.(string))
		case "offset":
			query = query.Offset(filterValue.(int))
		case "limit":
			query = query.Limit(filterValue.(int))
		default:
			query = query.Filter(filterName, filterValue)
		}
	}
	return query
}

type datastorer interface {
	Get(ctx context.Context, key *datastore.Key, dst interface{}) error
	GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error)
	GetMulti(ctx context.Context, keys []*datastore.Key, dst interface{}) error
	Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error)
}

var mockDataStoreClient datastorer

func getAllX(kind string, filters map[string]interface{}, dst interface{}) ([]*datastore.Key, error) {
	if orFilters, ok := filters["or"]; ok {
		if len(filters) > 1 {
			return nil, errors.New("NeedOnlyOrFilter")
		}
		switch orFiltersValue := orFilters.(type) {
		case []map[string]interface{}:
			orKeys := []*datastore.Key{}
			for _, filters := range orFiltersValue {
				array := reflect.ValueOf(dst).Elem()
				entities := reflect.New(array.Type())
				keys, err := getAllX(kind, filters, entities.Interface())
				if err != nil {
					return nil, err
				}
				orKeys = append(orKeys, keys...)
				array.Set(reflect.AppendSlice(array, entities.Elem()))
			}
			return orKeys, nil
		default:
			return nil, errors.New("BadOrFilter")
		}
	}
	id, hasID := filters["ID="]
	if !hasID {
		id, hasID = filters[kind+"ID="]
	}
	if hasID {
		if len(filters) > 1 {
			return nil, errors.New("NeedOnlyIDFilter")
		}
		switch idValue := id.(type) {
		case int64:
			array := reflect.ValueOf(dst).Elem()
			entity := reflect.New(array.Type().Elem().Elem())
			array.Set(reflect.Append(array, entity))
			if err := getX(kind, idValue, entity.Interface()); err != nil {
				return nil, err
			}
			return []*datastore.Key{idKey(kind, idValue)}, nil
		case []int64:
			keys := []*datastore.Key{}
			for _, i := range idValue {
				keys = append(keys, idKey(kind, i))
			}
			size := len(keys)
			slice := reflect.MakeSlice(reflect.ValueOf(dst).Type().Elem(), size, size)
			var err error
			if mockDataStoreClient != nil {
				err = mockDataStoreClient.GetMulti(apiContext, keys, slice.Interface())
			} else {
				err = datastoreClient.GetMulti(apiContext, keys, slice.Interface())
			}
			if err == nil {
				array := reflect.ValueOf(dst).Elem()
				array.Set(reflect.AppendSlice(array, slice))
			}
			return keys, err
		default:
			return nil, errors.New("BadIDFilter")
		}
	}
	q := newQuery(kind, filters)
	if mockDataStoreClient != nil {
		return mockDataStoreClient.GetAll(apiContext, q, dst)
	}
	return datastoreClient.GetAll(apiContext, q, dst)
}

func getAllOrgs(filters map[string]interface{}, dst *[]*Org) ([]*datastore.Key, error) {
	return getAllX("Org", filters, dst)
}

func getAllUsers(filters map[string]interface{}, dst *[]*User) ([]*datastore.Key, error) {
	return getAllX("User", filters, dst)
}

func getAllBoats(filters map[string]interface{}, dst *[]*Boat) ([]*datastore.Key, error) {
	return getAllX("Boat", filters, dst)
}

func getAllDeals(filters map[string]interface{}, dst *[]*Deal) ([]*datastore.Key, error) {
	return getAllX("Deal", filters, dst)
}

func getAllEvents(filters map[string]interface{}, dst *[]*Event) ([]*datastore.Key, error) {
	return getAllX("Event", filters, dst)
}

func idKey(kind string, id int64) *datastore.Key {
	// if id == 0 {
	// 	return datastore.IncompleteKey(kind, nil)
	// }
	return datastore.IDKey(kind, id, nil)
}

func getX(kind string, id int64, dst interface{}) error {
	if id == 0 {
		return nil
	}
	key := idKey(kind, id)
	var err error
	if mockDataStoreClient != nil {
		err = mockDataStoreClient.Get(apiContext, key, dst)
	} else {
		err = datastoreClient.Get(apiContext, key, dst)
	}
	if err == datastore.ErrNoSuchEntity {
		return errors.New("AccessDenied")
	}
	if err == nil {
		reflect.ValueOf(dst).Elem().FieldByName("ID").SetInt(id)
	}
	return err
}

func getOrg(id int64) (*Org, error) {
	dst := &Org{}
	return dst, getX("Org", id, dst)
}

func getUser(id int64) (*User, error) {
	dst := &User{}
	return dst, getX("User", id, dst)
}
func getBoat(id int64) (*Boat, error) {
	dst := &Boat{}
	return dst, getX("Boat", id, dst)
}
func getDeal(id int64) (*Deal, error) {
	dst := &Deal{}
	return dst, getX("Deal", id, dst)
}
func getEvent(id int64) (*Event, error) {
	dst := &Event{}
	return dst, getX("Event", id, dst)
}

func putX(key *datastore.Key, src interface{}, level int) (*datastore.Key, error) {
	if mockDataStoreClient != nil {
		return mockDataStoreClient.Put(apiContext, key, src)
	}
	key, err := datastoreClient.Put(apiContext, key, src)
	// srcJSON, _ := json.Marshal(src)
	// log.Printf("Info: Put%s %s => %d %v", key.Kind, string(srcJSON), key.ID, err)
	sseSink <- &Publication{SetLevel: level}
	return key, err
}

func putOrg(src *Org) (*datastore.Key, error) {
	return putX(idKey("Org", src.ID), src, 1)
}

func putUser(src *User) (*datastore.Key, error) {
	return putX(idKey("User", src.ID), src, 2)
}

func putBoat(src *Boat) (*datastore.Key, error) {
	return putX(idKey("Boat", src.ID), src, 3)
}

func putDeal(src *Deal) (*datastore.Key, error) {
	return putX(idKey("Deal", src.ID), src, 4)
}

func putEvent(src *Event) (*datastore.Key, error) {
	return putX(idKey("Event", src.ID), src, 5)
}

func makeStaffFirstTime() {
	var orgs []*Org
	if _, err := getAllOrgs(map[string]interface{}{"Types=": "Marketplace"}, &orgs); err != nil {
		panic(err)
	}
	if len(orgs) > 0 {
		return
	}
	godSession := &Session{IsGod: true}
	orgResp := SetOrg(&Request{Session: godSession, Org: &Org{
		Types:       []string{"Marketplace"},
		Name:        "Boat Fuji Inc.",
		Description: "The marketplace operator",
		Contacts: []Contact{
			{
				Type:     "Address",
				SubType:  "Work",
				Line1:    "3101 South US Highway 1",
				City:     "Fort Pierce",
				County:   "St. Lucie",
				State:    "FL",
				Postal:   "34982-6337",
				Country:  "US",
				Location: LatLng(27.4097155, -80.329127),
			},
			{
				Type:    "Email",
				SubType: "Work",
				Email:   "info@boatfuji.com",
			},
		},
		EIN:    "84-4243290",
		Images: []Image{},
	}}, nil)
	panicIfError(orgResp)
	for _, user := range []string{
		"EDC4B5EAFE5C91AE0614E21AE34137BF Dave Lampert dave.lampert@boatfuji.com 407-834-8834",
		"78601FF342A85F39AE36B45FF480D19A Erik Breckenfelder erik.breckenfelder@boatfuji.com 630-222-7505",
	} {
		fields := strings.Split(user, " ")
		panicIfError(SetUser(&Request{Session: godSession, User: &User{
			OrgID:        orgResp.ID,
			PasswordHash: userPasswordHash(fields[2] + "123!"),
			NameOrder:    "GivenFamily",
			GivenName:    fields[1],
			FamilyName:   fields[2],
			Gender:       "Male",
			Images:       []Image{userImageByMD5(fields[0])},
			Contacts: []Contact{
				{
					Type:     "Email",
					SubType:  "Work",
					Email:    fields[3],
					Verified: DateTime(2020, 1, 2, 3, 4, 5),
				},
				{
					Type:     "Phone",
					SubType:  "Mobile",
					Phone:    fields[4],
					Verified: DateTime(2020, 1, 2, 3, 4, 5),
				},
			},
		}}, nil))
	}
	panicIfError(SetUser(&Request{Session: godSession, User: &User{
		PasswordHash: userPasswordHash("Awkwafina123!"),
		Birthdate:    Date(1988, 6, 2),
		NameOrder:    "FamilyGiven",
		GivenName:    "Jiazhen",
		FamilyName:   "Lin",
		Description:  "Follow your dreams!",
		Gender:       "Female",
		Languages:    []string{"en-us", "zh"},
		Images:       []Image{userImageByMD5("052DCC6A7CC5117A6122F615E19DCE56")},
		Contacts: []Contact{
			{
				Type:     "Address",
				SubType:  "Work",
				Line1:    "123 5th Avenue",
				City:     "New York City",
				State:    "NY",
				Postal:   "10003",
				Location: LatLng(40.7391967, -73.9930489),
			},
			{
				Type:     "Address",
				SubType:  "Home",
				Line1:    "500 N Sweetzer Avenue",
				Line2:    "Box 123",
				City:     "Los Angeles",
				State:    "CA",
				Postal:   "90048",
				Location: LatLng(34.0802574, -118.3722444),
			},
			{
				Type:     "Email",
				SubType:  "Work",
				Email:    "info@awkwafina.com",
				Verified: DateTime(2020, 1, 2, 3, 4, 5),
			},
			{
				Type:     "Email",
				SubType:  "Home",
				Email:    "awkwafina@gmail.com",
				Verified: DateTime(2020, 1, 2, 3, 4, 5),
			},
			{
				Type:     "Phone",
				SubType:  "Mobile",
				Phone:    "213-555-1212",
				Verified: DateTime(2020, 1, 2, 3, 4, 5),
			},
		},
		Notifications: []string{"RentalStartEnd", "MessageReceived", "SpecialOffers", "News", "Tips", "UpcomingRentals", "UserReviews", "ReviewReminder", "BookingExpired"},
		Currency:      "USD",
		BankAccounts: []BankAccount{
			{
				Type:    "Checking",
				Routing: "021000322",
				Account: "1234567890",
			},
		},
		CreditCards: []CreditCard{
			{
				NickName: "Gold Card",
				Last4:    "1234",
				Token:    "...",
			},
			{
				NickName: "Cash Back",
				Last4:    "9876",
				Token:    "...",
			},
		},
		Audit: &Audit{
			Created:  DateTime(2020, 1, 2, 3, 4, 5),
			Updated:  DateTime(2020, 2, 3, 3, 4, 5),
			QANeeded: DateTime(2020, 2, 3, 3, 4, 5),
			QAFields: []string{"User.Description"},
			User: &User{
				Description: "Follow your dreams even if the world tells you you can't!",
			},
		},
	}}, nil))
	orgResp = SetOrg(&Request{Session: godSession, Org: &Org{
		Types:       []string{"Crew"},
		Name:        "Yo Soy Capitan",
		Description: "Let us captain your boats so renters won't capsize them!",
		Contacts: []Contact{
			{
				Type:     "Address",
				SubType:  "Work",
				Line1:    "123 Brickell Ave",
				City:     "Miami",
				County:   "Miami-Dade",
				State:    "FL",
				Postal:   "33129",
				Country:  "US",
				Location: LatLng(25.7467903, -80.2113866),
			},
			{
				Type:    "Email",
				SubType: "Work",
				Email:   "captainnotcapsize@gmail.com",
			},
		},
		EIN:    "59-1234567",
		Images: []Image{},
	}}, nil)
	panicIfError(orgResp)
	panicIfError(SetUser(&Request{Session: godSession, User: &User{
		PasswordHash: userPasswordHash("Bligh123!"),
		Birthdate:    Date(1988, 6, 2),
		NameOrder:    "GivenFamily",
		GivenName:    "Bligh",
		FamilyName:   "Blue",
		Description:  "Captain! Not capsize!",
		Gender:       "Male",
		Languages:    []string{"en-us", "es"},
		Images:       []Image{userImageByMD5("75FF04C131EE08005781A097795AF6AA")},
		Contacts: []Contact{
			{
				Type:     "Address",
				SubType:  "Work",
				Line1:    "123 Brickell Ave",
				City:     "Miami",
				County:   "Miami-Dade",
				State:    "FL",
				Postal:   "33129",
				Country:  "US",
				Location: LatLng(25.7467903, -80.2113866),
			},
			{
				Type:     "Email",
				SubType:  "Work",
				Email:    "captainnotcapsize@gmail.com",
				Verified: DateTime(2020, 1, 2, 3, 4, 5),
			},
			{
				Type:     "Phone",
				SubType:  "Mobile",
				Phone:    "305-555-1212",
				Verified: DateTime(2020, 1, 2, 3, 4, 5),
			},
		},
		Notifications: []string{"RentalStartEnd", "MessageReceived", "SpecialOffers", "News", "Tips", "UpcomingRentals", "UserReviews", "ReviewReminder", "BookingExpired"},
		Currency:      "USD",
		BankAccounts: []BankAccount{
			{
				Type:    "Checking",
				Routing: "021000322",
				Account: "314159265",
			},
		},
	}}, nil))
}

func panicIfError(resp *Response) {
	if resp.ErrorCode != "" {
		panic(errors.New(resp.ErrorCode))
	}
}

func userPasswordHash(clearPassword string) string {
	return md5Lower("NaCl:" + clearPassword)
}

func userImageByMD5(md5 string) Image {
	return Image{
		URL:    "/i/" + md5 + ".jpg",
		Width:  200,
		Height: 200,
	}
}

func makeStandardOrgs() {
	orgsToLoad := map[string][]string{
		"Insurer": {
			"Allianz",
			"AllState",
			"Auto-Owners",
			"BoatUS/GEICO",
			"C&L",
			"Chubb",
			"Foremost",
			"Great American",
			"Hagerty",
			"Liberty Mutual",
			"Markel",
			"MetLife",
			"NationWide",
			"Progressive",
			"SkiSafe",
			"StateFarm",
			"United Marine Underwriters",
			"Yachtinsure",
		},
		"TaxAuthority": {
			"Florida",
		},
	}
	godSession := &Session{IsGod: true}
	for orgType, names := range orgsToLoad {
		var orgs []*Org
		if _, err := getAllOrgs(map[string]interface{}{"Types=": orgType}, &orgs); err != nil {
			panic(err)
		}
		for _, name := range names {
			found := false
			for _, org := range orgs {
				if org.Name == name {
					found = true
					break
				}
			}
			if !found {
				panicIfError(SetOrg(&Request{Session: godSession, Org: &Org{
					Types: []string{orgType},
					Name:  name,
				}}, nil))
			}
		}
	}
}
