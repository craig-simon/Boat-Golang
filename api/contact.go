package api

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexmo-community/nexmo-go"
	"google.golang.org/appengine"
)

// Contact has an address, email, phone, or other contact type
type Contact struct {
	Type       string              `json:",omitempty" datastore:",omitempty,noindex" enum:"Address, Email, Apple, Facebook, Google, LinkedIn, Phone, URL"`
	SubType    string              `json:",omitempty" datastore:",omitempty,noindex" enum:"Home, Work, Billing, Mailing, Shipping, Fax, Mobile"`
	Line1      string              `json:",omitempty" datastore:",omitempty,noindex"`
	Line2      string              `json:",omitempty" datastore:",omitempty,noindex"`
	City       string              `json:",omitempty" datastore:",omitempty,noindex"`
	County     string              `json:",omitempty" datastore:",omitempty,noindex"`
	State      string              `json:",omitempty" datastore:",omitempty,noindex"`
	Postal     string              `json:",omitempty" datastore:",omitempty,noindex"`
	Country    string              `json:",omitempty" datastore:",omitempty,noindex"`
	Residence  Residence           `json:",omitempty" datastore:",omitempty,noindex"`
	Location   *appengine.GeoPoint `json:",omitempty" datastore:",omitempty,noindex"`
	Loc100KM   []int               `json:",omitempty" datastore:",omitempty"`
	Loc300KM   []int               `json:",omitempty" datastore:",omitempty"`
	Email      string              `json:",omitempty" datastore:",omitempty"`
	Phone      string              `json:",omitempty" datastore:",omitempty"`
	Extension  string              `json:",omitempty" datastore:",omitempty,noindex"`
	OAuthID    string              `json:",omitempty" datastore:",omitempty"`
	OAuthToken string              `json:",omitempty" datastore:",omitempty"`
	URL        string              `json:",omitempty" datastore:",omitempty,noindex"`
	VerifyCode string              `json:",omitempty" datastore:",omitempty,noindex"`
	Verifying  *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
	Verified   *time.Time          `json:",omitempty" datastore:",omitempty,noindex"`
}

// Residence has additional information about an address
type Residence struct {
	Status string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Own, Rent"`
	Since  *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Until  *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Value  float32    `json:",omitempty" datastore:",omitempty,noindex"`
	Liens  []Lien     `json:",omitempty" datastore:",omitempty,noindex"`
}

func init() {
	rand.Seed(now().UnixNano())
	addEnumsFor(Contact{})
}

func getContacts(contacts []Contact) {
	for i := range contacts {
		contacts[i].Loc100KM = nil
		contacts[i].Loc300KM = nil
		if contacts[i].VerifyCode != "" {
			// obscure the actual code that was sent during MFA
			contacts[i].VerifyCode = "SENT"
		}
	}
}

var emailPattern = regexp.MustCompile(`^[^ ,;@]+@([a-z0-9-]+\.)+[a-z]{2,63}$`)
var phonePattern = regexp.MustCompile(`^ *(\+(\d{1,3})[ \.\-/])?((\(\d{2,5}\) *)?\d{2,}([ \.\-/] *\d{2,})*) *$`)
var nonDigitPattern = regexp.MustCompile(`\D`)
var verifyCodePattern = regexp.MustCompile(`^\d{4}$`)

func setContacts(newContacts, oldContacts []Contact, req *Request) error {
	if newContacts == nil {
		return nil
	}
	if oldContacts == nil {
		oldContacts = []Contact{}
	}
	oldUnmatched := make([]Contact, len(oldContacts))
	copy(oldUnmatched, oldContacts)
	for contactIndex := range newContacts {
		contact := &newContacts[contactIndex]
		// check Type and SubType
		if contact.Type == "" {
			return errors.New("NeedType")
		}
		if contact.Type == "Address" && !StringInArray(contact.SubType, []string{"", "Home", "Work", "Billing", "Mailing", "Shipping"}) ||
			(contact.Type == "Email" || contact.Type == "URL") && !StringInArray(contact.SubType, []string{"", "Home", "Work"}) ||
			(contact.Type == "Facebook" || contact.Type == "Google") && contact.SubType != "" ||
			contact.Type == "Phone" && !StringInArray(contact.SubType, []string{"", "Home", "Work", "Fax", "Mobile"}) {
			return errors.New("BadSubType")
		}
		// error if extra data in contact fields
		if contact.Type != "Address" && (contact.Line1+contact.Line2+contact.City+contact.County+contact.State+contact.Postal+contact.Country != "" ||
			contact.Location != nil || contact.Loc100KM != nil || contact.Loc300KM != nil) ||
			contact.Type != "Email" && contact.Email != "" ||
			contact.Type != "Phone" && contact.Phone+contact.Extension != "" ||
			contact.Type != "Facebook" && contact.Type != "Google" && contact.Type != "URL" && contact.URL != "" ||
			contact.Type != "Email" && contact.Type != "Phone" && (contact.VerifyCode != "" || contact.Verifying != nil || contact.Verified != nil) {
			return errors.New("ExtraContactData")
		}
		switch contact.Type {
		case "Address":
			// assume Line1, Line2, City, County, State, Postal, and Country are valid, and only check lengths
			if len(contact.Line1) > 100 || len(contact.Line2) > 100 || len(contact.City) > 100 || len(contact.County) > 20 ||
				len(contact.State) > 20 || len(contact.Postal) > 20 || len(contact.Country) > 2 {
				return errors.New("BigAddress")
			}
			contact.Loc100KM = nil
			contact.Loc300KM = nil
			if contact.Location != nil {
				for i, km := range []float64{100.0, 300.0} {
					if loc, err := geoSquare(contact.Location.Lat, contact.Location.Lng, km, km/2); err != nil {
						return err
					} else if i == 0 {
						contact.Loc100KM = loc
					} else if i == 1 {
						contact.Loc300KM = loc
					}
				}
			}
		case "Email", "Phone":
			if contact.Type == "Email" {
				contact.Email = strings.ToLower(contact.Email)
				if emailPattern.FindString(contact.Email) == "" {
					return errors.New("BadEmail")
				}
			}
			if contact.Type == "Phone" {
				phone, extension, err := normalizePhone(contact.Phone, contact.Extension)
				if err != nil {
					return err
				}
				contact.Phone = phone
				contact.Extension = extension
			}
			// find match from old record
			oldContact := Contact{}
			for i, old := range oldUnmatched {
				if old.Type == contact.Type && (contact.Type == "Email" && old.Email == contact.Email ||
					contact.Type == "Phone" && old.Phone == contact.Phone) {
					oldContact = old
					oldUnmatched[i] = Contact{}
					break
				}
			}
			if contact.Email != oldContact.Email || contact.Phone != oldContact.Phone {
				// email or phone (ignoring extension) has changed, so need to verify
				oldContact.VerifyCode = ""
				oldContact.Verifying = nil
				oldContact.Verified = nil
			}
			switch contact.VerifyCode {
			case "", "SENT":
				// do nothing, and keep same as before
				contact.VerifyCode = oldContact.VerifyCode
				contact.Verifying = oldContact.Verifying
				contact.Verified = oldContact.Verified
			case "SEND":
				// send new 4-digit code, unless it's been under 30 seconds since last send
				if oldContact.Verifying != nil && now().Sub(*oldContact.Verifying).Seconds() < 30 {
					return errors.New("MustWaitToResendCode")
				}
				code, err := verifyCode(9999, contact.Email+contact.Phone)
				if err != nil {
					return err
				}
				contact.VerifyCode = code
				contact.Verifying = now()
				contact.Verified = nil
			default:
				// should be 4-digit code
				if verifyCodePattern.FindString(contact.VerifyCode) == "" {
					return errors.New("BadVerifyCode")
				}
				if contact.VerifyCode != oldContact.VerifyCode {
					throttle("VerifyCode", req)
					return errors.New("WrongVerifyCode")
				}
				unthrottle("VerifyCode", req)
				contact.VerifyCode = ""
				contact.Verifying = nil
				contact.Verified = now()
			}
		case "Facebook", "Google", "URL":
			// assume URL is valid, and only check length
			if len(contact.URL) > 2000 {
				return errors.New("BigURL")
			}
		}
	}
	return nil
}

func geoSquare(lat, lng, kmSize, kmRadius float64) ([]int, error) {
	// the surface of the Earth is divided into geoSquares of approximate width and height of kmSize
	// square numbering starts at the antimeridian and equator, so that to the northeast is 0, and to the east of that is 1, 2, 3, etc.
	// the length of the equator is 40,075 km, so if kmSize = 400.75 km, square 99 would be northwest of the antimeridian and equator
	// square 100 would be north of square 0, square 200 would be north of 100, etc.
	// however, because the circumference gets shorter at higher latitudes, there may be some numbers skipped, such as 399, 497, 498, 499, 596, etc.
	// if kmRadius = 0, the result is a single number; this number is used in the query filter
	// if kmRadius = kmSize/2, the result will usually be 4 numbers:
	// _____    _____
	// |_|_| or |_|_|
	// |_|_|     |_|_|
	//
	// if kmRadius = kmSize, the result will usually be up to 9 numbers:
	// _______
	// |_|_|_|
	// |_|_|_| or etc.
	// |_|_|_|
	//
	// if kmRadius = kmSize*2, the result will usually be up to 21 numbers:
	//   _______
	//  _|_|_|_|_
	// |_|_|_|_|_|
	// |_|_|_|_|_| or etc.
	// |_|_|_|_|_|
	//   |_|_|_|
	// the best query is where each record has about 4 numbers, and the query filter has a single number
	// for example, if you want to search for locations in a 20 km radius, and if not enough are found, you search in a 100 km radius,
	// then each record should have two fields, one for 40 km numbering and one for 200 km numbering, so each field has about 4 numbers
	if lat < -90 || lat > 90 {
		return nil, errors.New("BadLat")
	}
	if lng < -180 || lng > 180 {
		return nil, errors.New("BadLng")
	}
	if kmSize <= 0 {
		return nil, errors.New("BadKMSize")
	}
	if kmRadius < 0 {
		return nil, errors.New("BadKMRadius")
	}
	result := []int{}
	kmMeridian := 20003.93
	kmEquator := 40075.02
	squaresPerBand := (int)(math.Ceil(kmEquator / kmSize))
	minBandNorthOfEquator := (int)(lat/180*kmMeridian/kmSize - kmRadius/kmSize)
	maxBandNorthOfEquator := (int)(lat/180*kmMeridian/kmSize + kmRadius/kmSize)
	for band := minBandNorthOfEquator; band <= maxBandNorthOfEquator; band++ {
		kmBand := kmEquator * math.Cos(float64(band)*kmSize/kmMeridian*math.Pi)
		// instead of kmRadius below, it should use sqrt(kmRadius^2-kmNorthOfPt^2)
		min := band*squaresPerBand + (int)(kmBand/kmSize*(lng+180)/360-kmRadius/kmSize)
		max := band*squaresPerBand + (int)(kmBand/kmSize*(lng+180)/360+kmRadius/kmSize)
		for num := min; num <= max; num++ {
			result = append(result, num)
		}
	}
	return result, nil
}

func normalizePhone(phone, extension string) (string, string, error) {
	match := phonePattern.FindStringSubmatch(phone)
	if match == nil {
		return "", "", errors.New("BadPhone")
	}
	if match[2] == "" || match[2] == "1" {
		// North American Numbering Plan number should be formatted ###-###-####
		phone = nonDigitPattern.ReplaceAllString(match[3], "")
		if len(phone) != 10 {
			return "", "", errors.New("BadPhone")
		}
		phone = fmt.Sprintf("%s-%s-%s", phone[0:3], phone[3:6], phone[6:10])
	} else {
		// international numbers retain formatting, but check length
		if len(phone) > 20 {
			return "", "", errors.New("BigPhone")
		}
		phone = strings.TrimSpace(phone)
	}
	if len(extension) > 8 {
		return "", "", errors.New("BigExtension")
	}
	if extension != "" && nonDigitPattern.FindString(extension) != "" {
		return "", "", errors.New("BadExtension")
	}
	return phone, extension, nil
}

var testVerifyCode string

func verifyCode(maxCode int, emailOrPhone string) (string, error) {
	if testVerifyCode != "" {
		return testVerifyCode, nil
	}
	// create random n-digit code
	code := strconv.Itoa(rand.Intn(maxCode+1) + maxCode + 1)[1:]
	// email or text code
	if emailPattern.MatchString(emailOrPhone) {
		log.Printf("Info: Email code %s to %s", code, emailOrPhone)
		return code, SendEmail("[Boat Fuji]Please verify your email address", "support@boatfuji.com", []string{emailOrPhone},
			`<div style='background:#0f233d;color:#fff;padding:40px 40px;font:bold 18px sans-serif'>
<img src='https://www.boatfuji.com/img/email-logo.png'>
<p>To verify your email address, please enter this code: <span style='color:#ff0'>`+code+`</span>.</p>
</div>`)
	}
	if phonePattern.MatchString(emailOrPhone) {
		return code, sendSMS(emailOrPhone, "Boat Fuji phone verification code "+code)
	}
	return code, nil
}

var nonPrintablePattern = regexp.MustCompile(`[\x00-\x1F\x80-\xFF]`)

// SendEmail sends an email message with a given subject, from, to, and html body
func SendEmail(subject, from string, to []string, htmlBody string) error {
	host := Config.Env.EmailHost
	port := Config.Env.EmailPort
	user := Config.Env.EmailUser
	pass := Config.Env.EmailPass
	msg := "Subject: " + nonPrintablePattern.ReplaceAllLiteralString(subject, "") + "\n" +
		"Reply-To: " + nonPrintablePattern.ReplaceAllLiteralString(from, "") + "\n" +
		"MIME-Version: 1.0;\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\n\n" +
		htmlBody
	return smtp.SendMail(host+":"+port, smtp.PlainAuth("", user, pass, host), user, to, []byte(msg))
}

func sendSMS(to, msg string) error {
	if to[0:1] != "+" {
		to = "1" + nonDigitPattern.ReplaceAllLiteralString(to, "")
	}
	auth := nexmo.NewAuthSet()
	auth.SetAPISecret(Config.Env.NexmoKey, Config.Env.NexmoSecret)
	client := nexmo.NewClient(http.DefaultClient, auth)
	smsContent := nexmo.SendSMSRequest{
		From: Config.Env.NexmoFrom,
		To:   to,
		Text: msg,
	}
	smsResp, _, err := client.SMS.SendSMS(smsContent)
	if err != nil {
		log.Printf("Error: NEXMO From %q To %q Text %q Resp %v Err %q", Config.Env.NexmoFrom, to, msg, smsResp, err.Error())
	} else {
		log.Printf("Info: NEXMO From %q To %q Text %q Resp %v", Config.Env.NexmoFrom, to, msg, smsResp)
	}
	return err
}
