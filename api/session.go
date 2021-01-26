package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

var sessions = map[int64]*Session{}
var lastSessionID int64 = 0
var sessionsMutex sync.Mutex

// Session is a user sign-in (anonymous or using credentials) on a particular device
type Session struct {
	ID                 int64                   `json:",omitempty" datastore:"-"`
	UserID             int64                   `json:",omitempty" datastore:",omitempty"`
	OrgID              int64                   `json:",omitempty" datastore:",omitempty"`
	IsGod              bool                    `json:",omitempty" datastore:",omitempty"`
	OrgTypes           []string                `json:",omitempty" datastore:",omitempty"`
	OrgAccess          []string                `json:",omitempty" datastore:",omitempty"`
	Verified           bool                    `json:",omitempty" datastore:",omitempty"`
	IP                 string                  `json:",omitempty" datastore:",omitempty"`
	UserAgent          string                  `json:",omitempty" datastore:",omitempty"`
	TimeZone           *time.Location          `json:",omitempty" datastore:",omitempty"`
	Started            *time.Time              `json:",omitempty" datastore:",omitempty"`
	SubscriptionsMutex sync.RWMutex            `json:",omitempty" datastore:",omitempty"`
	Subscriptions      map[int64]*subscription `json:"-" datastore:"-"`
	LastSubscriptionID int64                   `json:"-" datastore:"-"`
	SSEConnection      chan *Publication       `json:"-" datastore:"-"`
}

type facebookOAuth2 struct {
	Name  string         `json:"name"`
	ID    string         `json:"id"`
	Error *facebookError `json:"error"`
}

type facebookError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	FBTraceID string `json:"fbtrace_id"`
}

type googleOAuth2 struct {
	IssuedTo         string `json:"issued_to"`
	Audience         string `json:"audience"`
	UserID           string `json:"user_id"`
	Scope            string `json:"scope"`
	ExpiresIn        int    `json:"expires_in"`
	Email            string `json:"email"`
	VerifiedEmail    bool   `json:"verified_email"`
	AccessType       string `json:"access_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func init() {
	apiHandlers["SignIn"] = SignIn
	apiHandlers["SignOut"] = SignOut
}

// getSession turns a valid authorization string into a Session (with ID, and with or without UserID), or with ID=0 otherwise
func getSession(auth, ip, userAgent string) *Session {
	if auth != "" {
		parser := jwt.Parser{UseJSONNumber: true}
		token, err := parser.Parse(strings.ReplaceAll(auth, "Bearer ", ""), func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("BadJWTSigningMethod")
			}
			return []byte(Config.Env.JWTKey), nil
		})
		if err == nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				sid, err1 := claims["sid"].(json.Number).Int64()
				exp, err2 := claims["exp"].(json.Number).Int64()
				if sid != 0 && err1 == nil && time.Unix(exp, 0).After(*now()) && err2 == nil {
					sessionsMutex.Lock()
					session, ok := sessions[sid]
					sessionsMutex.Unlock()
					if ok {
						if session.IP != ip {
							sessionLog(&Request{Session: session}, "Warn", "IP changed from %q to %q", session.IP, ip)
						}
						session.IP = ip
						if session.UserAgent != userAgent {
							sessionLog(&Request{Session: session}, "Warn", "UserAgent changed from %q to %q", session.UserAgent, userAgent)
						}
						session.UserAgent = userAgent
						return session
					}
				}
			}
		}
	}
	return &Session{
		IP:        ip,
		UserAgent: userAgent,
	}
}

func sessionLog(req *Request, level, format string, v ...interface{}) {
	msg := level + ": " + fmt.Sprintf(format, v...)
	if req == nil || req.Session == nil || req.Session.ID == 0 {
		log.Print(msg)
	} else {
		sessionsDir := "sessions"
		os.MkdirAll(sessionsDir, 0755)
		file := "sessions/" + strconv.FormatInt(req.Session.ID, 10) + ".log"
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Error: %s while opening %q to %s", err.Error(), msg, file)
		}
		defer f.Close()
		if _, err := f.WriteString(now().Format("2006/01/02 15:04:05 ") + msg + "\n"); err != nil {
			log.Printf("Error: %s while writing %q to %s", err.Error(), msg, file)
		}
	}
}

// SignIn checks for username/email/phone, or signs in, or sets TOTP in case of forgotten password
// {"User":{}} signs in as an anonymous user and returns {"Bearer":"...","ExpiresIn":...,"ID":...}
// {"User":{"UserName":"johndoe@example.org"}} checks if the user exists in database and returns {"ErrorCode":"AccessDenied"} or {"ErrorCode":"NeedPasswordHash"}
// {"User":{"UserName":"johndoe@example.org","PasswordHash":"..."}} signs in as user and returns {"Bearer":"...","ExpiresIn":...,"ID":...}
// {"User":{"UserName":"johndoe@example.org","TOTP":"SEND"}} is called if user forgot password and returns {}
// {"User":{"TOTP":"..."}} signs in as user after forgetting password and returns {"Bearer":"...","ExpiresIn":...,"ID":...}; client should then ask user to change password
// {"User":{"Contacts":[{"Type":"Facebook","OAuthID":"2734407573470227","OAuthToken":"..."}]}}
func SignIn(req *Request, pub *Publication) *Response {
	session := req.Session
	if req.User == nil {
		return &Response{ErrorCode: "NeedUser"}
	}
	expireSeconds := 365 * 24 * 3600
	if req.User.UserName != "" || req.User.TOTP != "" || req.User.Contacts != nil {
		// find user(s) by user name, email, or phone
		filterName := "UserName="
		filterValue := req.User.UserName
		oAuthType := ""
		if req.User.TOTP == "SEND" {
			if req.User.UserName == "" {
				return &Response{ErrorCode: "NeedUserName"}
			}
		} else if req.User.TOTP != "" {
			if req.User.UserName != "" {
				return &Response{ErrorCode: "NeedNoUserName"}
			}
			filterName = "TOTP="
			filterValue = req.User.TOTP
		} else if req.User.Contacts != nil {
			if len(req.User.Contacts) != 1 {
				return &Response{ErrorCode: "Need1Contact"}
			}
			contact := req.User.Contacts[0]
			if contact.OAuthID == "" {
				return &Response{ErrorCode: "NeedOAuthID"}
			}
			if contact.OAuthToken == "" {
				return &Response{ErrorCode: "NeedOAuthToken"}
			}
			oAuthType = contact.Type
			switch oAuthType {
			case "Facebook":
				oAuth2 := &facebookOAuth2{}
				if err := getJSONFromURL("https://graph.facebook.com/v2.3/me?access_token="+contact.OAuthToken, oAuth2); err != nil {
					return errResponse(err)
				}
				if oAuth2.Error != nil || oAuth2.ID != contact.OAuthID {
					sessionLog(req, "Warn", "BadOAuthToken %s ID=%q Token=%q Resp=%+v", oAuthType, contact.OAuthID, contact.OAuthToken, oAuth2.Error)
					return &Response{ErrorCode: "BadOAuthToken"}
				}
			case "Google":
				oAuth2 := &googleOAuth2{}
				if err := getJSONFromURL("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token="+contact.OAuthToken, oAuth2); err != nil {
					return errResponse(err)
				}
				if oAuth2.Error != "" || oAuth2.UserID != contact.OAuthID {
					sessionLog(req, "Warn", "BadOAuthToken %s ID=%q Token=%q Resp=%+v", oAuthType, contact.OAuthID, contact.OAuthToken, oAuth2)
					return &Response{ErrorCode: "BadOAuthToken"}
				}
			default:
				return &Response{ErrorCode: "NeedOAuthType"}
			}
			filterName = "Contacts.OAuthID="
			filterValue = contact.OAuthID
		}
		if emailPattern.MatchString(req.User.UserName) {
			filterName = "Contacts.Email="
			filterValue = strings.ToLower(req.User.UserName)
		}
		if phonePattern.MatchString(req.User.UserName) {
			filterName = "Contacts.Phone="
			phone, _, err := normalizePhone(req.User.UserName, "")
			if err != nil {
				return errResponse(err)
			}
			filterValue = phone
		}
		var users []*User
		keys, err := getAllUsers(map[string]interface{}{filterName: filterValue}, &users)
		if err != nil {
			return errResponse(err)
		}
		if req.User.TOTP == "" && req.User.PasswordHash == "" && req.User.Contacts == nil {
			if len(users) == 0 {
				return accessDenied()
			}
			return &Response{ErrorCode: "NeedPasswordHash"}
		}
		// see how many match password or meet TOTPSent requirement, too
		matchingUsers := []*User{}
		for index, user := range users {
			user.ID = keys[index].ID
			if req.User.TOTP == "SEND" {
				// if TOTPSent is within last minute, tell client to wait
				if user.TOTPSent != nil && now().Sub(*user.TOTPSent).Seconds() < 60 {
					return &Response{ErrorCode: "MustWaitToResendCode"}
				}
				// send email or SMS and save user with random TOTP and now in TOTPSent
				code, err := verifyCode(99999999, req.User.UserName)
				if err != nil {
					return errResponse(err)
				}
				user.TOTP = code
				user.TOTPSent = now()
				if _, err = putUser(user); err != nil {
					return errResponse(err)
				}
				return &Response{}
			} else if req.User.TOTP != "" {
				if now().Sub(*user.TOTPSent).Seconds() > 300 {
					return &Response{ErrorCode: "TOTPExpired"}
				}
				user.TOTP = ""
				user.TOTPSent = nil
				if _, err = putUser(user); err != nil {
					return errResponse(err)
				}
				matchingUsers = append(matchingUsers, user)
			} else if oAuthType != "" {
				matchingUsers = append(matchingUsers, user)
			} else if user.PasswordHashCrypt != "" {
				err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHashCrypt), []byte(req.User.PasswordHash))
				if err == nil {
					matchingUsers = append(matchingUsers, user)
				} else if err != bcrypt.ErrMismatchedHashAndPassword {
					return errResponse(err)
				}
			}
		}
		if len(matchingUsers) == 0 {
			return accessDenied()
		}
		if len(matchingUsers) > 1 {
			sessionLog(req, "Error", "Ambiguous sign-in for %s %s found %d users", filterName, req.User.UserName, len(matchingUsers))
		}
		user := matchingUsers[0]
		session.UserID = user.ID
		session.OrgID = user.OrgID
		if user.OrgID != 0 {
			org, err := getOrg(user.OrgID)
			if err != nil {
				return errResponse(err)
			}
			session.OrgTypes = org.Types
		}
		session.OrgAccess = user.OrgAccess
		session.Verified = isUserVerified(user)
		// expireSeconds = 24 * 3600
	}
	if session.ID != 0 {
		sessionLog(req, "Info", "now user %d IP %s on %q", session.UserID, req.Session.IP, req.Session.UserAgent)
		sseSink <- &Publication{SetLevel: 0}
		return &Response{ID: session.UserID}
	}
	// finish building new session and register it
	session.Started = now()
	session.Subscriptions = map[int64]*subscription{}
	sessionsMutex.Lock()
	lastSessionID++
	session.ID = lastSessionID
	sessions[session.ID] = session
	sessionsMutex.Unlock()
	sessionLog(req, "Info", "user %d IP %s on %q", session.UserID, req.Session.IP, req.Session.UserAgent)
	// TODO: write to datastore
	// build JSON Web Token
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = now().Add(time.Second * time.Duration(expireSeconds)).Unix()
	claims["sid"] = session.ID
	t, err := token.SignedString([]byte(Config.Env.JWTKey))
	if err != nil {
		return errResponse(err)
	}
	return &Response{Bearer: t, ExpiresIn: expireSeconds, ID: session.UserID}
}

var testJSONFromURL = ""

func getJSONFromURL(url string, dst interface{}) error {
	if testJSONFromURL != "" {
		return json.Unmarshal([]byte(testJSONFromURL), dst)
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}

// SignOut signs out
func SignOut(req *Request, pub *Publication) *Response {
	// TODO: write to datastore
	sessionsMutex.Lock()
	delete(sessions, req.Session.ID)
	sessionsMutex.Unlock()
	return &Response{}
}

func isUserVerified(user *User) bool {
	verifiedEmail := false
	verifiedPhone := false
	if user.Contacts != nil {
		for _, contact := range user.Contacts {
			if contact.Type == "Email" && contact.Verified != nil {
				verifiedEmail = true
			}
			if contact.Type == "Phone" && contact.Verified != nil {
				verifiedPhone = true
			}
		}
	}
	return verifiedEmail && verifiedPhone
}

func isVerifiedUser(req *Request) bool {
	return req.Session.IsGod || req.Session.Verified
}

func mustVerifyResp() *Response {
	return &Response{ErrorCode: "MustVerify"}
}

func isStaff(req *Request) bool {
	return req.Session.IsGod || req.Session.OrgTypes != nil && StringInArray("Marketplace", req.Session.OrgTypes)
}

func accessDenied() *Response {
	return &Response{ErrorCode: "AccessDenied"}
}

func staffOnly() *Response {
	return &Response{ErrorCode: "StaffOnly"}
}

func throttle(reason string, req *Request) {
	// TODO: slow down future calls by 1 sec, 2 sec, 4 sec, 8 sec, and then 8 sec until unthrottle(reason, req)
}

func unthrottle(reason string, req *Request) {
	// TODO: reset slow down to 0 sec
}
