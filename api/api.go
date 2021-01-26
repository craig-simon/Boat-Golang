package api

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"google.golang.org/appengine"
	"gopkg.in/yaml.v2"
)

// Config is loaded from app-local.yaml
var Config yamlFile

type yamlFile struct {
	Env yamlEnv `yaml:"env_variables"`
}

type yamlEnv struct {
	JWTKey            string `yaml:"JWT_KEY"`
	EmailHost         string `yaml:"EMAIL_HOST"`
	EmailPort         string `yaml:"EMAIL_PORT"`
	EmailUser         string `yaml:"EMAIL_USER"`
	EmailPass         string `yaml:"EMAIL_PASS"`
	NexmoKey          string `yaml:"NEXMO_KEY"`
	NexmoSecret       string `yaml:"NEXMO_SECRET"`
	NexmoFrom         string `yaml:"NEXMO_FROM"`
	GoogleAndroid     string `yaml:"GOOGLE_API_KEY_ANDROID"`
	GoogleIOS         string `yaml:"GOOGLE_API_KEY_IOS"`
	GoogleWeb         string `yaml:"GOOGLE_API_KEY_WEB"`
	StripePublishable string `yaml:"STRIPE_PUBLISHABLE_KEY"`
	StripeSecret      string `yaml:"STRIPE_SECRET_KEY"`
	StripeWebhook     string `yaml:"STRIPE_WEBHOOK_SECRET"`
	AndroidVersions   string `yaml:"ANDROID_VERSIONS"`
	IOSVersions       string `yaml:"IOS_VERSIONS"`
}

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// in case we're doing a test inside /api or another folder
	if !strings.HasSuffix(strings.ToLower(dir), "boatfuji") {
		err := os.Chdir("..")
		if err != nil {
			panic(err)
		}
	}
	yamlText, err := ioutil.ReadFile("app-local.yaml")
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(yamlText, &Config); err != nil {
		panic(err)
	}
	go rollIPAPIActivity()
	apiHandlers["Search"] = search
	apiHandlers["Log"] = logMessage
}

// Start does things after each init() but before first call to DispatchToAPIHandler; skipped when running any tests
func Start() {
	startDataStore()
	startMake()
}

// Request is a superset of information that each API handler needs
type Request struct {
	Session        *Session            `json:"-" datastore:",omitempty"` // this is set from the Authorization header, not from the POST content
	Subscription   *subscription       `json:"-" datastore:",omitempty"` // this is nil when called on API, or defined when some other datastore change triggers a subscription update
	Subscribe      bool                `json:",omitempty" datastore:",omitempty"`
	SubscriptionID int64               `json:",omitempty" datastore:",omitempty"`
	QA             bool                `json:",omitempty" datastore:",omitempty"`
	OrgID          int64               `json:",omitempty" datastore:",omitempty"`
	UserID         int64               `json:",omitempty" datastore:",omitempty"`
	BoatID         int64               `json:",omitempty" datastore:",omitempty"`
	DealID         int64               `json:",omitempty" datastore:",omitempty"`
	EventID        int64               `json:",omitempty" datastore:",omitempty"`
	Year           int                 `json:",omitempty" datastore:",omitempty"`
	MakeID         int                 `json:",omitempty" datastore:",omitempty"`
	MakeDetailID   int                 `json:",omitempty" datastore:",omitempty"`
	Location       *appengine.GeoPoint `json:",omitempty" datastore:",omitempty"`
	KMRadius       int                 `json:",omitempty" datastore:",omitempty"`
	StartDate      *time.Time          `json:",omitempty" datastore:",omitempty"`
	EndDate        *time.Time          `json:",omitempty" datastore:",omitempty"`
	OrgTypes       []string            `json:",omitempty" datastore:",omitempty" enum:"Marketplace, Crew, Dealer, Financer, Insurer, Manufacturer, Servicer, Tax Authority, Transporter"`
	EventTypes     []string            `json:",omitempty" datastore:",omitempty" enum:"Message, Payment, Rental, Review"`
	UseMetric      bool                `json:",omitempty" datastore:",omitempty"`
	Unread         bool                `json:",omitempty" datastore:",omitempty"`
	Org            *Org                `json:",omitempty" datastore:",omitempty"`
	User           *User               `json:",omitempty" datastore:",omitempty"`
	Boat           *Boat               `json:",omitempty" datastore:",omitempty"`
	Deal           *Deal               `json:",omitempty" datastore:",omitempty"`
	Event          *Event              `json:",omitempty" datastore:",omitempty"`
	Image          *Image              `json:",omitempty" datastore:",omitempty"`
	Crop           *image.Rectangle    `json:",omitempty" datastore:",omitempty"`
	CleanImage     func(i image.Image) `json:",omitempty" datastore:",omitempty"`
	Language       string              `json:",omitempty" datastore:",omitempty"`
	Summary        string              `json:",omitempty" datastore:",omitempty"`
	Details        string              `json:",omitempty" datastore:",omitempty"`
	Text           string              `json:",omitempty" datastore:",omitempty"`
}

// Response is a superset of all API handler responses
type Response struct {
	Bearer         string                 `json:",omitempty" datastore:",omitempty"`
	ExpiresIn      int                    `json:",omitempty" datastore:",omitempty"`
	SubscriptionID int64                  `json:",omitempty" datastore:",omitempty"`
	ID             int64                  `json:",omitempty" datastore:",omitempty"`
	Marketplaces   map[int]*Marketplace   `json:",omitempty" datastore:",omitempty"`
	Makes          map[int]*Make          `json:",omitempty" datastore:",omitempty"`
	Orgs           map[int64]*Org         `json:",omitempty" datastore:",omitempty"`
	Users          map[int64]*User        `json:",omitempty" datastore:",omitempty"`
	Boats          map[int64]*Boat        `json:",omitempty" datastore:",omitempty"`
	Deals          map[int64]*Deal        `json:",omitempty" datastore:",omitempty"`
	Events         map[int64]*Event       `json:",omitempty" datastore:",omitempty"`
	Options        map[string]interface{} `json:",omitempty" datastore:",omitempty"`
	Image          *Image                 `json:",omitempty" datastore:",omitempty"`
	ErrorCode      string                 `json:",omitempty" datastore:",omitempty"`
	ErrorDetails   map[string]string      `json:",omitempty" datastore:",omitempty"`
}

// Err makes an error using a code and details
func Err(code string, details map[string]string) error {
	detailsJSONBytes, _ := json.Marshal(details)
	detailsJSON := string(detailsJSONBytes)
	if details == nil {
		detailsJSON = ""
	}
	return errors.New(code + detailsJSON)
}

func errResponse(err error) *Response {
	codeAndDetails := err.Error()
	pos := strings.Index(codeAndDetails, "{")
	if pos < 0 {
		return &Response{ErrorCode: codeAndDetails}
	}
	code := codeAndDetails[:pos]
	detailsJSON := []byte(codeAndDetails[pos:])
	details := map[string]string{}
	json.Unmarshal(detailsJSON, &details)
	return &Response{ErrorCode: code, ErrorDetails: details}
}

// a registry of all handlers by their name; i.e., "GetUser"
var apiHandlers = make(map[string]func(req *Request, pub *Publication) *Response)

var ipAPIActivityMutex sync.Mutex
var ipAPIActivity map[string]*activity = map[string]*activity{}

type activity struct {
	reqCt0to5MinAgo  int
	reqCt5to10MinAgo int
	logged           bool
}

var alotOfRequests = 100

// rollIPAPIActivity checks every 5 minutes and rolls counters over
func rollIPAPIActivity() {
	for {
		time.Sleep(5 * time.Minute)
		ipAPIActivityMutex.Lock()
		for ipAPI, act := range ipAPIActivity {
			if act.reqCt0to5MinAgo+act.reqCt5to10MinAgo > alotOfRequests {
				log.Printf("throttling %s (%d+%d requests)", ipAPI, act.reqCt0to5MinAgo, act.reqCt5to10MinAgo)
			}
			act.reqCt5to10MinAgo = act.reqCt0to5MinAgo
			act.reqCt0to5MinAgo = 0
		}
		ipAPIActivityMutex.Unlock()
	}
}

// DispatchToAPIHandler is the main entry for all API calls
func DispatchToAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept,Accept-Language,Content-Language,Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Vary", "Accept-Encoding, Origin")
		w.Header().Set("Keep-Alive", "timeout=2, max=100 ")
		w.Header().Set("Connection", "Keep-Alive")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	startTime := now()
	// get authorization from either the querystring or the header, if it's there
	auth := r.URL.Query().Get("auth")
	if auth == "" {
		auth = r.Header.Get("Authorization")
	}
	ip := strings.Split(r.RemoteAddr, ":")[0]
	session := getSession(auth, ip, r.UserAgent())
	apiName := r.URL.Path[len("/api/"):] // i.e., "GetUser"
	req := &Request{}
	resp := &Response{}
	if len(apiName) == 0 {
		resp.ErrorCode = "NeedAPIName"
	} else if apiName != "SignIn" && apiName != "Log" && session.ID == 0 {
		resp.ErrorCode = "MustSignIn"
	} else if apiName == "SSE" {
		handleSSE(w, r, session)
		return
	} else if handler, ok := apiHandlers[apiName]; ok {
		// throttle based on IP and APIName
		ipAPIActivityMutex.Lock()
		act, ok := ipAPIActivity[ip+apiName]
		if !ok {
			act = &activity{}
			ipAPIActivity[ip+apiName] = act
		}
		ipAPIActivityMutex.Unlock()
		act.reqCt0to5MinAgo++
		reqCt := act.reqCt0to5MinAgo + act.reqCt5to10MinAgo
		if reqCt > alotOfRequests {
			if !act.logged {
				act.logged = true
				log.Printf("throttle %s from %s (%d+%d requests)", apiName, ip, act.reqCt0to5MinAgo, act.reqCt5to10MinAgo)
			}
			time.Sleep(time.Duration(reqCt/alotOfRequests) * time.Second) // so it progressively slows down with more requests
		}
		// get POST JSON content
		if r.Method != http.MethodPost || r.ContentLength == 0 || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			resp.ErrorCode = "MustPostJSON"
		} else {
			if err := json.NewDecoder(r.Body).Decode(req); err != nil {
				if ute, ok := err.(*json.UnmarshalTypeError); ok {
					resp.ErrorCode = "BadJSON"
					resp.ErrorDetails = map[string]string{
						"Offset": strconv.FormatInt(ute.Offset, 10),
						"Field":  ute.Field,
						"Value":  ute.Value,
						"Type":   ute.Type.Name(),
					}
				} else if se, ok := err.(*json.SyntaxError); ok {
					resp.ErrorCode = "BadJSON"
					resp.ErrorDetails = map[string]string{
						"Offset": strconv.FormatInt(se.Offset, 10),
						"Error":  se.Error(),
					}
				} else {
					resp.ErrorCode = "BadJSON"
					resp.ErrorDetails = map[string]string{
						"Error": err.Error(),
					}
				}
			} else {
				req.Session = session
				req.Subscription = nil
				// call the specific handler with the request, and get back the response
				resp = handler(req, nil)
				if resp.SubscriptionID == -1 {
					// this means that the handler supports subscriptions
					resp.SubscriptionID = 0
					if req.Subscribe {
						subscribe(req, handler, resp)
					}
				}
			}
		}
	} else {
		resp.ErrorCode = "BadAPIName"
		resp.ErrorDetails = map[string]string{
			"Name": apiName,
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if resp.ErrorCode != "" {
		if resp.ErrorCode == "AccessDenied" || resp.ErrorCode == "NeedPasswordHash" {
			w.WriteHeader(http.StatusUnauthorized)
		} else if resp.ErrorCode == "MustWaitToResendCode" {
			w.WriteHeader(http.StatusTooEarly)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
	json.NewEncoder(w).Encode(resp)
	endMS := now().Sub(*startTime).Milliseconds()
	reqJSON, _ := json.Marshal(req)
	respJSON, _ := json.Marshal(resp)
	sessionLog(req, "Debug", "Call %s %d ms %s => %s", apiName, endMS, reqJSON, respJSON)
}

func makeFilters(req *Request, locationKind string) (map[string]interface{}, bool, *Response) {
	staff := isStaff(req)
	filters := map[string]interface{}{}
	filtersSafe := false
	if req.QA {
		if !staff {
			return filters, staff, staffOnly()
		}
		filters["Audit.QANeeded>"] = new(time.Time)
		filtersSafe = true
	}
	if req.OrgID != 0 {
		if !staff && req.OrgID != req.Session.OrgID {
			return filters, staff, accessDenied()
		}
		filters["OrgID="] = req.OrgID
		filtersSafe = true
	}
	if req.UserID != 0 {
		if !staff && req.UserID != req.Session.UserID {
			return filters, staff, accessDenied()
		}
		filters["UserID="] = req.UserID
		filtersSafe = true
	}
	if req.BoatID != 0 {
		filters["BoatID="] = req.BoatID
		filtersSafe = true
	}
	if req.DealID != 0 {
		filters["DealID="] = req.DealID
	}
	if req.EventID != 0 {
		filters["EventID="] = req.EventID
	}
	if req.Location != nil {
		if locationKind == "" {
			return filters, staff, &Response{ErrorCode: "OmitLocation"}
		}
		if req.KMRadius != 150 {
			req.KMRadius = 50
		}
		loc, err := geoSquare(req.Location.Lat, req.Location.Lng, float64(req.KMRadius*2), 0)
		if err != nil {
			return filters, staff, errResponse(err)
		}
		filters[locationKind+".Loc"+strconv.Itoa(req.KMRadius*2)+"KM="] = loc[0]
		filtersSafe = true
	}
	if req.EventTypes != nil && len(req.EventTypes) == 1 && req.EventTypes[0] == "Review" {
		// it will do the filtering later
		filtersSafe = true
	}
	if req.Unread {
		filters["UnreadByIDs="] = req.Session.UserID // TODO: must union with query of filter filters["UnreadByIDs="] = req.Session.OrgID
	}
	if !filtersSafe {
		filters["UserID="] = req.Session.UserID
	}
	return filters, staff, nil
}

// StringInArray returns true if string is in an array
func StringInArray(s string, a []string) bool {
	for _, i := range a {
		if i == s {
			return true
		}
	}
	return false
}

var testTime *time.Time

func now() *time.Time {
	// TODO: round to second so it's like "2020-05-05T05:05:05Z" instead of including fractional second and timezone offset
	if testTime != nil {
		return testTime
	}
	d := time.Now()
	return &d
}

// Date returns a *time.Date for a year, month, and day
func Date(year, month, day int) *time.Time {
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &d
}

// DateTime returns a *time.Date for a date and time
func DateTime(year, month, day, hour, minute, second int) *time.Time {
	d := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return &d
}

// LatLng returns a *GeoPoint
func LatLng(lat, lng float64) *appengine.GeoPoint {
	return &appengine.GeoPoint{
		Lat: lat,
		Lng: lng,
	}
}

// GetHTML does a GET for a url, and returns either the HTML string or an error
func GetHTML(cachePath, url string) (string, error) {
	// get html from cachePath, or if not there, then from url (and save to cachePath)
	if bytes, err := ioutil.ReadFile(cachePath); err == nil {
		return string(bytes), nil
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ioutil.WriteFile(cachePath, bytes, 0644)
	return string(bytes), nil
}

// GetHTMLDoc parses HTML after calling GET for a url
func GetHTMLDoc(cachePath, url string) (*html.Node, error) {
	htmlString, err := GetHTML(cachePath, url)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func md5Lower(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

var sqlPattern = regexp.MustCompile(`^select \* from (orgs|users|boats|deals|events)( where (.*?))?( order by ([\w\.]+)( desc)?)?( offset (\d+))?( limit (\d+))?$`)
var sqlWherePattern = regexp.MustCompile(`^([\w\.]+(=|<|<=|>|>=))('[^']*'|\d+)$`)

func search(req *Request, pub *Publication) *Response {
	if !isStaff(req) {
		return accessDenied()
	}
	if req.Text == "" {
		return &Response{ErrorCode: "NeedText"}
	}
	sql := sqlPattern.FindStringSubmatch(req.Text)
	if sql == nil {
		// TODO: handle full text search
		return &Response{ErrorCode: "NotImplemented"}
	}
	kind := sql[1]
	filters := map[string]interface{}{}
	if sql[2] != "" {
		wheres := strings.Split(sql[3], " and ")
		for _, where := range wheres {
			sqlWhere := sqlWherePattern.FindStringSubmatch(where)
			if sqlWhere == nil {
				return &Response{ErrorCode: "BadSQL"}
			}
			filterKey := sqlWhere[1]
			filterValue := sqlWhere[3]
			if filterValue[0] == '\'' {
				filters[filterKey] = strings.Trim(filterValue, "'")
			} else {
				filters[filterKey], _ = strconv.ParseInt(filterValue, 10, 64)
			}
		}
	}
	if sql[4] != "" {
		order := sql[5]
		if sql[6] == " desc" {
			order = "-" + order
		}
		filters["order"] = order
	}
	if sql[7] != "" {
		filters["offset"], _ = strconv.Atoi(sql[8])
	}
	if sql[9] != "" {
		filters["limit"], _ = strconv.Atoi(sql[10])
	}
	resp := &Response{SubscriptionID: -1}
	switch kind {
	case "orgs":
		resp.Orgs = map[int64]*Org{}
		var recs []*Org
		keys, err := getAllOrgs(filters, &recs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			recs[index].ID = key.ID
			resp.Orgs[key.ID] = recs[index]
		}
	case "users":
		resp.Users = map[int64]*User{}
		var recs []*User
		keys, err := getAllUsers(filters, &recs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			recs[index].ID = key.ID
			resp.Users[key.ID] = recs[index]
		}
	case "boats":
		resp.Boats = map[int64]*Boat{}
		var recs []*Boat
		keys, err := getAllBoats(filters, &recs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			recs[index].ID = key.ID
			resp.Boats[key.ID] = recs[index]
		}
	case "deals":
		resp.Deals = map[int64]*Deal{}
		var recs []*Deal
		keys, err := getAllDeals(filters, &recs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			recs[index].ID = key.ID
			resp.Deals[key.ID] = recs[index]
		}
	case "events":
		resp.Events = map[int64]*Event{}
		var recs []*Event
		keys, err := getAllEvents(filters, &recs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			recs[index].ID = key.ID
			resp.Events[key.ID] = recs[index]
		}
	}
	return resp
}

func logMessage(req *Request, pub *Publication) *Response {
	if req.Summary == "" {
		return &Response{ErrorCode: "NeedSummary"}
	}
	if req.Details == "" {
		return &Response{ErrorCode: "NeedDetails"}
	}
	log.Printf("Log(%q, %q)", req.Summary, req.Details)
	return &Response{}
}
