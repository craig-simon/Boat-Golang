package api

import (
	"errors"
	"reflect"
	"strings"
	"time"
)

// Audit used for Org, User, Boat, Deal, and Event to have audit details and changes not yet QA'ed by marketplace staff
type Audit struct {
	Created   *time.Time `json:",omitempty" datastore:",omitempty"`
	Updated   *time.Time `json:",omitempty" datastore:",omitempty"`
	Deleted   *time.Time `json:",omitempty" datastore:",omitempty"`
	QANeeded  *time.Time `json:",omitempty" datastore:",omitempty"`
	QAStarted *time.Time `json:",omitempty" datastore:",omitempty"`
	QAUserID  int64      `json:",omitempty" datastore:",omitempty"`
	QAFields  []string   `json:",omitempty" datastore:",omitempty,noindex"`
	Org       *Org       `json:",omitempty" datastore:",omitempty,noindex"`
	User      *User      `json:",omitempty" datastore:",omitempty,noindex"`
	Boat      *Boat      `json:",omitempty" datastore:",omitempty,noindex"`
	Deal      *Deal      `json:",omitempty" datastore:",omitempty,noindex"`
	Event     *Event     `json:",omitempty" datastore:",omitempty,noindex"`
}

func qaFilter() map[string]interface{} {
	return map[string]interface{}{"Audit.QANeeded>": new(time.Time)}
}

var possibleQAFieldsPerEntity = map[string][]string{}

// i.e. qaFieldsPerEntity["Boat"] = []string{"Boat.Images", "Boat.Rental.ListingTitle", ...}
func possibleQAFields(ptr interface{}) []string {
	typ := reflectStruct(ptr).Type()
	entityName := typ.Name()
	if fields, ok := possibleQAFieldsPerEntity[entityName]; ok {
		return fields
	}
	fields := recurseDeep(typ, entityName, []string{})
	possibleQAFieldsPerEntity[entityName] = fields
	return fields
}

func skipBaseQAFieldName(field string) string {
	// i.e., change "Boat.Rental.ListingTitle" to "Rental.ListingTitle"
	return field[strings.Index(field, ".")+1:]
}

func recurseDeep(typ reflect.Type, prefix string, fields []string) []string {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for fieldNum := 0; fieldNum < typ.NumField(); fieldNum++ {
		field := typ.Field(fieldNum)
		fieldName := field.Name // i.e., "Rental"
		if field.Type.Kind() == reflect.Ptr && fieldName != "Audit" {
			_, hasAudit := field.Type.Elem().FieldByName("Audit")
			if !hasAudit {
				// recurse into child object; i.e., "Boat.Rental"
				fields = recurseDeep(field.Type, prefix+"."+fieldName, fields)
			}
		}
		if qa := field.Tag.Get("qa"); qa != "" {
			// add to list of fields; i.e., "Boat.Rental.ListingTitle"
			fields = append(fields, prefix+"."+fieldName)
		}
	}
	return fields
}

func reflectStruct(ptr interface{}) reflect.Value {
	reflectPtr := reflect.ValueOf(ptr)
	if reflectPtr.Kind() != reflect.Ptr || reflectPtr.Elem().Kind() != reflect.Struct {
		panic(errors.New("NeedStructPtr"))
	}
	return reflectPtr.Elem()
}

func reflectAudit(ptr interface{}) reflect.Value {
	return reflectStruct(ptr).FieldByName("Audit")
}

func reflectID(ptr interface{}) reflect.Value {
	return reflectStruct(ptr).FieldByName("ID")
}

func getDeepField(fieldName string, ptr interface{}) interface{} {
	// i.e., fieldName = "Boat.Rental.ListingTitle", ptr is *Boat, return string value of ListingTitle
	names := strings.Split(fieldName, ".")
	for index, name := range names {
		field := reflectStruct(ptr).FieldByName(name)
		if index+1 == len(names) {
			return field.Interface()
		}
		if field.IsNil() {
			return nil
		}
		ptr = field.Interface()
		if reflect.ValueOf(ptr).IsNil() {
			return nil
		}
	}
	return ptr
}

func setDeepField(fieldName string, ptr, value interface{}) {
	names := strings.Split(fieldName, ".")
	for index, name := range names {
		field := reflectStruct(ptr).FieldByName(name)
		if index+1 == len(names) {
			if value == nil {
				field.Set(reflect.Zero(field.Type()))
			} else {
				field.Set(reflect.ValueOf(value))
			}
			return
		}
		// create new object if missing before, so we can drill down further
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		ptr = field.Interface()
	}
}

func isMine(req *Request, ptr interface{}) bool {
	// find out if this entity is owned by this user or org
	entity := reflectStruct(ptr)
	var orgID int64
	var userID int64
	if entity.Type().Name() == "Org" {
		orgID = entity.FieldByName("ID").Int()
	} else if entity.Type().Name() == "User" {
		userID = entity.FieldByName("ID").Int()
		orgID = entity.FieldByName("OrgID").Int()
	} else {
		userID = entity.FieldByName("UserID").Int()
		orgID = entity.FieldByName("OrgID").Int()
	}
	return orgID != 0 && req.Session.OrgID == orgID || userID != 0 && req.Session.UserID == userID
}

func getAudit(req *Request, ptr interface{}) {
	// prepare org, user, boat, deal, or event before responding to Get API
	if req.QA {
		return // don't omit or flatten down Audit.{EntityName}, since staff member must see both to review
	}
	// clear QA fields leaving only Created and updated, since non-staff users shouldn't see them
	auditField := reflectAudit(ptr)
	var audit *Audit
	if auditField.IsNil() {
		audit = &Audit{}
	} else {
		audit = auditField.Interface().(*Audit)
		// if this entity is owned by this user or org, copy the audit QA fields down to the direct entity (the user can see data pending review)
		if isMine(req, ptr) {
			for _, field := range audit.QAFields {
				setDeepField(skipBaseQAFieldName(field), ptr, getDeepField(field, audit))
			}
		}
		audit = &Audit{Created: audit.Created, Updated: audit.Updated}
	}
	auditField.Set(reflect.ValueOf(audit))
}

func setAudit(staff bool, newPtr, oldPtr interface{}) {
	if staff {
		// ignore oldPtr, and only add in Audit.Created or Audit.Updated
		newAuditField := reflectAudit(newPtr)
		if newAuditField.IsNil() {
			reflectAudit(newPtr).Set(reflect.ValueOf(&Audit{Created: now()}))
		} else {
			newAudit := newAuditField.Interface().(*Audit)
			newAudit.Updated = now()
		}
		return
	}
	// newPtr is what is being set now, and oldPtr is what was in datastore before (or nil if new)
	newAudit := &Audit{Created: now()}
	if oldPtr == nil {
		oldPtr = reflect.New(reflectStruct(newPtr).Type())
	} else {
		// copy over old Audit and timestamp Updated
		oldAuditField := reflectAudit(oldPtr)
		if !oldAuditField.IsNil() {
			oldAudit := oldAuditField.Interface().(*Audit)
			newAudit = &Audit{
				Created:   oldAudit.Created,
				Updated:   now(),
				QANeeded:  oldAudit.QANeeded,
				QAStarted: oldAudit.QAStarted,
				QAUserID:  oldAudit.QAUserID,
			}
		}
	}
	reflectAudit(newPtr).Set(reflect.ValueOf(newAudit))
	// copy over ID from old
	reflectID(newPtr).Set(reflectID(oldPtr))
	// if user works for marketplace (staff), new fields will be at root level
	// otherwise, compare old and new and build new Audit.{EntityName}
	fields := []string{}
	for _, fieldName := range possibleQAFields(newPtr) {
		fieldNameWithoutBase := skipBaseQAFieldName(fieldName)
		oldValue := getDeepField(fieldNameWithoutBase, oldPtr)
		newValue := getDeepField(fieldNameWithoutBase, newPtr)
		if !reflect.DeepEqual(oldValue, newValue) {
			fields = append(fields, fieldName)
			setDeepField(fieldName, newAudit, newValue)
			setDeepField(fieldNameWithoutBase, newPtr, oldValue)
		}
	}
	if len(fields) > 0 {
		if newAudit.QANeeded == nil {
			newAudit.QANeeded = now()
		}
		newAudit.QAFields = fields
	}
}
