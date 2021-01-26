package api

import (
	"log"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User is a person who uses this system, whether alone or part of a Org
type User struct {
	ID                int64          `json:",omitempty" datastore:"-"`
	OrgID             int64          `json:",omitempty" datastore:",omitempty"`
	Org               *Org           `json:",omitempty" datastore:"-"`
	OrgAccess         []string       `json:",omitempty" datastore:",omitempty,noindex" enum:"SetOrg, SetUser, SetBoat, SetDeal, SetEvent"`
	URLs              []string       `json:",omitempty" datastore:",omitempty"`
	ReferredByUserID  int64          `json:",omitempty" datastore:",omitempty"`
	ReferredByOrgID   int64          `json:",omitempty" datastore:",omitempty"`
	UserName          string         `json:",omitempty" datastore:",omitempty"`
	PasswordHash      string         `json:",omitempty" datastore:"-"`
	PasswordHashCrypt string         `json:",omitempty" datastore:",omitempty,noindex"`
	TOTP              string         `json:",omitempty" datastore:",omitempty"`
	TOTPSent          *time.Time     `json:",omitempty" datastore:",omitempty"`
	Birthdate         *time.Time     `json:",omitempty" datastore:",omitempty,noindex"`
	NameOrder         string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Given/Family, Family/Given, Other"`
	Prefix            string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Mr, Miss, Ms, Mrs, Dr"`
	GivenName         string         `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	FamilyName        string         `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Suffix            string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Sr, Jr, II, III, IV"`
	Description       string         `json:",omitempty" datastore:",omitempty,noindex,noindex" qa:"-"`
	Gender            string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Male, Female, Other"`
	MaritalStatus     string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Single, Engaged, Married, Civil Union, Domestic Partnership, Separated, Divorced, Widowed"`
	Relationship      string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Child, Guardian, Parent, Spouse, Sibling, Step-Sibling, Aunt/Uncle, Niece/Nephew, Cousin, Grandchild, Grandparent"`
	Relatives         []User         `json:",omitempty" datastore:",omitempty,noindex"`
	Jobs              []UserJob      `json:",omitempty" datastore:",omitempty,noindex"`
	Languages         []string       `json:",omitempty" datastore:",omitempty,noindex"`
	Images            []Image        `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Contacts          []Contact      `json:",omitempty" datastore:",omitempty"`
	UserApprovals     []UserApproval `json:",omitempty" datastore:",omitempty"`
	Favorites         []int64        `json:",omitempty" datastore:",omitempty,noindex"`
	Notifications     []string       `json:",omitempty" datastore:",omitempty,noindex" enum:"Rental Start / End, Message Received, Special Offers, News, Tips, Upcoming Rentals, User Reviews, Review Reminder, Booking Expired"`
	RewardPoints      int            `json:",omitempty" datastore:",omitempty,noindex"`
	Currency          string         `json:",omitempty" datastore:",omitempty,noindex" enum:"CAD, EUR, USD"`
	BankAccounts      []BankAccount  `json:",omitempty" datastore:",omitempty,noindex"`
	CreditCards       []CreditCard   `json:",omitempty" datastore:",omitempty,noindex"`
	W9s               []W9           `json:",omitempty" datastore:",omitempty,noindex"`
	RequestCount      int            `json:",omitempty" datastore:",omitempty,noindex"`
	ResponseCount     int            `json:",omitempty" datastore:",omitempty,noindex"`
	ResponseSecSum    int            `json:",omitempty" datastore:",omitempty,noindex"`
	Audit             *Audit         `json:",omitempty" datastore:",omitempty"`
}

// BankAccount is a user's bank account
type BankAccount struct {
	Name    string   `json:",omitempty" datastore:",omitempty,noindex"`
	Type    string   `json:",omitempty" datastore:",omitempty,noindex" enum:"Checking, Savings"`
	Routing string   `json:",omitempty" datastore:",omitempty,noindex"`
	Account string   `json:",omitempty" datastore:",omitempty,noindex"`
	Address *Contact `json:",omitempty" datastore:",omitempty,noindex"`
	Token   string   `json:",omitempty" datastore:",omitempty,noindex"`
}

// CreditCard is a user's credit or debit card
type CreditCard struct {
	NickName       string     `json:",omitempty" datastore:",omitempty,noindex"`
	Last4          string     `json:",omitempty" datastore:",omitempty,noindex"`
	ExpirationDate *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Token          string     `json:",omitempty" datastore:",omitempty,noindex"`
	// above are the only fields stored on the server; below are only used in the browser, and are only stored in the credit card processor's server
	// NameOnCard     string     `json:"-"`
	// CardNumber     string     `json:"-"`
	// SecurityCode   string     `json:"-"`
	// BillingZip     string     `json:"-"`
}

// W9 is a user's W-9 form
type W9 struct {
	FullLegalName string     `json:",omitempty" datastore:",omitempty,noindex"`
	BusinessName  string     `json:",omitempty" datastore:",omitempty,noindex"`
	TaxClass      string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Individual, Sole Proprietor, C Corp, S Corp, Partnership, Trust/Estate, LLC - Sole Member, LLC - C Corp, LLC - S Corp, LLC - Partnership"`
	TaxID         string     `json:",omitempty" datastore:",omitempty,noindex"`
	TaxIDType     string     `json:",omitempty" datastore:",omitempty,noindex" enum:"SSN, EIN"`
	Address       *Contact   `json:",omitempty" datastore:",omitempty,noindex"`
	Signature     string     `json:",omitempty" datastore:",omitempty,noindex"`
	Submitted     *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
}

// UserApproval is a user's application and subsequent approval or denial
type UserApproval struct {
	IDType            string           `json:",omitempty" datastore:",omitempty,noindex" enum:"None, Citizenship, PassportBook, PassportCard, DriverLicense, MerchantMarinerCredential, SIN, SSN"`
	Images            []Image          `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	IDCountry         string           `json:",omitempty" datastore:",omitempty,noindex"`
	IDState           string           `json:",omitempty" datastore:",omitempty,noindex"`
	IDNumber          string           `json:",omitempty" datastore:",omitempty,noindex"`
	FelonyConviction  string           `json:",omitempty" datastore:",omitempty,noindex" enum:"Unknown, No, Yes"`
	MovingViolations  []Violation      `json:",omitempty" datastore:",omitempty,noindex"`
	NoInsurance       string           `json:",omitempty" datastore:",omitempty,noindex" enum:"Unknown, No, Yes"`
	NoInsuranceReason []string         `json:",omitempty" datastore:",omitempty,noindex" enum:"Discontinued, Changed Guidelines, Other"`
	InsuranceClaims   []InsuranceClaim `json:",omitempty" datastore:",omitempty,noindex"`
	YearsExperience   int              `json:",omitempty" datastore:",omitempty,noindex"`
	LastSafetyClass   *time.Time       `json:",omitempty" datastore:",omitempty,noindex"`
	Submitted         *time.Time       `json:",omitempty" datastore:",omitempty"`
	Verification      UserVerification `json:",omitempty" datastore:",omitempty"`
}

// UserJob is a job a user holds/held
type UserJob struct {
	Status   string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Employed, Self-Employed, Retired"`
	Employer *Org       `json:",omitempty" datastore:",omitempty,noindex"`
	Position string     `json:",omitempty" datastore:",omitempty,noindex"`
	Monthly  float32    `json:",omitempty" datastore:",omitempty,noindex"`
	Since    *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	Until    *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
}

// UserVerification is a response to a UserApproval
type UserVerification struct {
	Name        string     `json:",omitempty" datastore:",omitempty,noindex"`
	Citizenship string     `json:",omitempty" datastore:",omitempty,noindex"`
	IssueDate   *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
	ExpirDate   *time.Time `json:",omitempty" datastore:",omitempty"`
	Details     string     `json:",omitempty" datastore:",omitempty,noindex"`
}

// InsuranceClaim is a claim made in the past that affects a user's approval
type InsuranceClaim struct {
	Type      string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Hurricane Or Storm, Towing Only, Lightning Strike, Dismasting, Hit Something Or Went Aground, Flooding, Theft Of Equipment, Collision With Another Boat, Theft Of Boat, Injury Or Fatality, Other"`
	Repaired  string     `json:",omitempty" datastore:",omitempty,noindex" enum:"Unknown, No, Yes"`
	ClaimDate *time.Time `json:",omitempty" datastore:",omitempty,noindex"`
}

// Violation is a moving violation in the past that affects a user's approval
type Violation struct {
	Type string `json:",omitempty" datastore:",omitempty,noindex" enum:"Suspended License, Speeding Over 20, Speeding Under 20, DUI, Reckless Driving, At Fault Accident"`
}

// Visit tracks when a user signs in and out
type Visit struct {
	IP        string     `json:",omitempty" datastore:",omitempty"`
	UserAgent string     `json:",omitempty" datastore:",omitempty"`
	SignedIn  *time.Time `json:",omitempty" datastore:",omitempty"`
	SignedOut *time.Time `json:",omitempty" datastore:",omitempty"`
}

func init() {
	addEnumsFor(User{})
	addEnumsFor(BankAccount{})
	addEnumsFor(W9{})
	addEnumsFor(UserApproval{})
	addEnumsFor(InsuranceClaim{})
	addEnumsFor(Violation{})
	apiHandlers["GetUsers"] = GetUsers
	apiHandlers["SetUser"] = SetUser
}

func getPublicUser(userID int64) *User {
	if userID == 0 {
		return nil
	}
	user, err := getUser(userID)
	if err != nil {
		log.Printf("getUser(%d) => %s", userID, err.Error())
		return nil
	}
	return &User{
		GivenName:      user.GivenName,
		Description:    user.Description,
		Images:         user.Images,
		RequestCount:   user.RequestCount,
		ResponseCount:  user.ResponseCount,
		ResponseSecSum: user.ResponseSecSum,
		Audit: &Audit{
			Created: user.Audit.Created,
		},
	}
}

// GetUsers gets users
func GetUsers(req *Request, pub *Publication) *Response {
	resp := &Response{Users: map[int64]*User{}}
	staff := isStaff(req)
	var orgsResp *Response
	if req.QA {
		if !staff {
			return staffOnly()
		}
		var users []*User
		keys, err := getAllUsers(qaFilter(), &users)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			users[index].ID = key.ID
			resp.Users[key.ID] = users[index]
		}
	} else if req.OrgID != 0 || req.OrgTypes != nil {
		orgsResp = GetOrgs(req, nil)
		if orgsResp.ErrorCode != "" {
			return orgsResp
		}
		for orgID := range orgsResp.Orgs {
			var users []*User
			keys, err := getAllUsers(map[string]interface{}{"OrgID=": orgID}, &users)
			if err != nil {
				return errResponse(err)
			}
			for index, key := range keys {
				users[index].ID = key.ID
				resp.Users[key.ID] = users[index]
			}
		}
	} else {
		userID := req.UserID
		if userID == 0 {
			userID = req.Session.UserID
		}
		user, err := getUser(userID)
		if err != nil {
			return errResponse(err)
		}
		if userID != 0 {
			resp.Users[userID] = user
		}
	}
	// sanitize before returning
	for _, user := range resp.Users {
		user.PasswordHashCrypt = ""
		user.TOTP = ""
		getAudit(req, user)
		getContacts(user.Contacts)
		if user.OrgID != 0 {
			// add Org field
			if orgsResp == nil {
				orgsReq := &Request{Session: req.Session, OrgID: user.OrgID}
				orgsResp = GetOrgs(orgsReq, nil)
				if orgsResp.ErrorCode != "" {
					return orgsResp
				}
			}
			if org, ok := orgsResp.Orgs[user.OrgID]; ok {
				getAudit(req, org)
				user.Org = org
			} else {
				return &Response{ErrorCode: "BadOrgID", ErrorDetails: map[string]string{"OrgID": strconv.FormatInt(user.OrgID, 10)}}
			}
		}
		if !staff && user.ID != req.Session.UserID {
			// if it's not me and I'm not staff, clear most fields
			user.OrgAccess = nil
			user.ReferredByUserID = 0
			user.ReferredByOrgID = 0
			user.UserName = ""
			user.TOTPSent = nil
			user.Birthdate = nil
			user.NameOrder = ""
			user.Prefix = ""
			user.FamilyName = ""
			user.Suffix = ""
			user.Description = ""
			user.Gender = ""
			user.Languages = nil
			user.Contacts = nil
			user.UserApprovals = nil
			user.Favorites = nil
			user.Notifications = nil
			user.RewardPoints = 0
			user.Currency = ""
			user.BankAccounts = nil
			user.CreditCards = nil
			user.W9s = nil
		}
	}
	resp.SubscriptionID = -1
	return resp
}

// SetUser sets a user
func SetUser(req *Request, pub *Publication) *Response {
	if req.User == nil {
		return &Response{ErrorCode: "NeedUser"}
	}
	if err := validate(req.User); err != nil {
		return errResponse(err)
	}
	staff := isStaff(req)
	// can only add new User if I'm staff or I have no User yet; can only edit User if I'm staff or it's my User
	// TODO: handle OrgAccess and editing other users in same org
	addMyNewUser := req.User.ID == 0 && req.Session.UserID == 0
	if !(staff || req.User.ID == req.Session.UserID || addMyNewUser) {
		return accessDenied()
	}
	// OrgID can only be 0 or my OrgID, unless I'm staff
	if !(staff || req.User.OrgID == req.Session.OrgID || req.User.OrgID == 0) {
		return &Response{ErrorCode: "BadOrgID"}
	}
	oldUser, err := getUser(req.User.ID)
	if err != nil {
		return errResponse(err)
	}
	req.User.Org = nil
	// TODO: check OrgAccess, ReferredByUser, and ReferredByOrg
	// if password changing, bcrypt it, since we don't even want to store the original MD5 hash of the password
	if req.User.PasswordHash != "" {
		bytes, err := bcrypt.GenerateFromPassword([]byte(req.User.PasswordHash), 13)
		if err != nil {
			return errResponse(err)
		}
		req.User.PasswordHashCrypt = string(bytes)
		req.User.PasswordHash = ""
	} else if oldUser != nil {
		req.User.PasswordHashCrypt = oldUser.PasswordHashCrypt
	}
	// finalize and save
	setAudit(staff, req.User, oldUser)
	if err := setContacts(req.User.Contacts, oldUser.Contacts, req); err != nil {
		return errResponse(err)
	}
	key, err := putUser(req.User)
	if err != nil {
		return errResponse(err)
	}
	if addMyNewUser {
		req.Session.UserID = key.ID
	}
	if req.Session.UserID == key.ID {
		req.Session.OrgAccess = req.User.OrgAccess
		req.Session.Verified = isUserVerified(req.User)
	}
	return &Response{
		ID: key.ID,
	}
}
