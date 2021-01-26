package api

import (
	"log"
)

// Org is a manufacturer or other organization type
type Org struct {
	ID          int64     `json:",omitempty" datastore:"-"`
	Types       []string  `json:",omitempty" datastore:",omitempty" enum:"Marketplace, Club, Crew, Dealer, Financer, Insurer, Manufacturer, Rideshare, Servicer, Tax Authority, Transporter"`
	Name        string    `json:",omitempty" datastore:",omitempty" qa:"-"`
	Description string    `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Contacts    []Contact `json:",omitempty" datastore:",omitempty"`
	EIN         string    `json:",omitempty" datastore:",omitempty,noindex"`
	Images      []Image   `json:",omitempty" datastore:",omitempty,noindex" qa:"-"`
	Audit       *Audit    `json:",omitempty" datastore:",omitempty"`
}

func init() {
	addEnumsFor(Org{})
	apiHandlers["GetOrgs"] = GetOrgs
	apiHandlers["SetOrg"] = SetOrg
}

func getPublicOrg(orgID int64) *Org {
	if orgID == 0 {
		return nil
	}
	org, err := getOrg(orgID)
	if err != nil {
		log.Printf("getOrg(%d) => %s", orgID, err.Error())
		return nil
	}
	return &Org{
		Name:  org.Name,
		Types: org.Types,
	}
}

// GetOrgs gets orgs
func GetOrgs(req *Request, pub *Publication) *Response {
	resp := &Response{Orgs: map[int64]*Org{}}
	staff := isStaff(req)
	var filter map[string]interface{}
	if req.QA {
		if !staff {
			return staffOnly()
		}
		filter = qaFilter()
	} else if req.OrgTypes != nil {
		if len(req.OrgTypes) != 1 {
			// TODO: support 0 or 2+ OrgTypes
			return &Response{ErrorCode: "Need1OrgType"}
		}
		filter = map[string]interface{}{"Types=": req.OrgTypes[0]}
		// TODO: if req.Location, append that
	}
	if filter != nil {
		var orgs []*Org
		keys, err := getAllOrgs(filter, &orgs)
		if err != nil {
			return errResponse(err)
		}
		for index, key := range keys {
			orgs[index].ID = key.ID
			resp.Orgs[key.ID] = orgs[index]
		}
	} else {
		orgID := req.OrgID
		if orgID == 0 {
			orgID = req.Session.OrgID
		}
		if orgID != 0 {
			org, err := getOrg(orgID)
			if err != nil {
				return errResponse(err)
			}
			resp.Orgs[orgID] = org
		}
	}
	// sanitize before returning
	for _, org := range resp.Orgs {
		getAudit(req, org)
		getContacts(org.Contacts)
	}
	resp.SubscriptionID = -1
	return resp
}

// SetOrg sets an org
func SetOrg(req *Request, pub *Publication) *Response {
	if !isVerifiedUser(req) {
		return mustVerifyResp()
	}
	if req.Org == nil {
		return &Response{ErrorCode: "NeedOrg"}
	}
	if req.Org.Types == nil || len(req.Org.Types) == 0 {
		return &Response{ErrorCode: "NeedOrgTypes"}
	}
	if err := validate(req.Org); err != nil {
		return errResponse(err)
	}
	staff := isStaff(req)
	// OrgTypes can only contain "Marketplace" if I'm God, and can only be other than "Crew" if I'm staff
	if !req.Session.IsGod && StringInArray("Marketplace", req.Org.Types) ||
		!(staff || len(req.Org.Types) == 1 && req.Org.Types[0] == "Crew") {
		return &Response{ErrorCode: "BadOrgTypes"}
	}
	// can only add new Org if I'm staff or I have no Org yet; can only edit Org if I'm staff or it's my Org
	addMyNewOrg := req.Org.ID == 0 && req.Session.OrgID == 0
	if !(staff || req.Org.ID == req.Session.OrgID || addMyNewOrg) {
		return accessDenied()
	}
	oldOrg, err := getOrg(req.Org.ID)
	if err != nil {
		return errResponse(err)
	}
	// finalize and save
	setAudit(staff, req.Org, oldOrg)
	if err := setContacts(req.Org.Contacts, oldOrg.Contacts, req); err != nil {
		return errResponse(err)
	}
	key, err := putOrg(req.Org)
	if err != nil {
		return errResponse(err)
	}
	if addMyNewOrg {
		req.Session.OrgID = key.ID
		req.Session.OrgTypes = req.Org.Types
	}
	return &Response{
		ID: key.ID,
	}
}
