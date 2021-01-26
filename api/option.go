package api

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

// these will be filled in by addEnumsFor so that options["Org_Types"]["TaxAuthority"] = "Tax Authority"
// but if we simply make it a map of a map, we can't control the key order of the inside map
//var options = map[string]map[string]string{}
// therefore, we make it a map of an interface, and create a dynamic struct instead of a map[string]string
var options = map[string]interface{}{}

func init() {
	apiHandlers["GetOptions"] = GetOptions
}

// add any enumerated lists to the Marketplace.Options so the website will know what to put in <select> options
func addEnumsFor(entity interface{}) {
	v := reflect.ValueOf(entity)
	if v.Kind() != reflect.Struct {
		panic(errors.New("addEnumsFor arg should be struct but instead was " + v.String()))
	}
	structName := v.Type().Name()
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if enum := f.Tag.Get("enum"); enum != "" {
			//_, options[structName+"_"+f.Name] = Enums(entity, f.Name)
			keys, keyValues := Enums(entity, f.Name)
			structFields := make([]reflect.StructField, len(keys))
			for i, key := range keys {
				structFields[i] = reflect.StructField{Name: key, Type: reflect.TypeOf(string(0))}
			}
			structType := reflect.StructOf(structFields)
			structValue := reflect.New(structType).Elem()
			for i, key := range keys {
				structValue.Field(i).SetString(keyValues[key])
			}
			options[structName+"_"+f.Name] = structValue.Addr().Interface()
		}
	}
}

var enumCleaner = regexp.MustCompile(`[- /&]`)

// Enums gets map of enumerations, just so it's easy to validate values
func Enums(i interface{}, fieldName string) ([]string, map[string]string) {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(errors.New("enums arg should be struct but instead was " + v.String()))
	}
	f, _ := v.Type().FieldByName(fieldName)
	if enum := f.Tag.Get("enum"); enum != "" {
		var a []string
		m := map[string]string{}
		for _, label := range strings.Split(enum, ", ") {
			code := enumCleaner.ReplaceAllLiteralString(label, "")
			a = append(a, code)
			m[code] = label
		}
		return a, m
	}
	return nil, nil
}

// check struct that all properties with enum must be one of the enumerated values
func validate(entity interface{}) error {
	if reflect.TypeOf(entity).String() == "*time.Time" {
		return nil
	}
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return Err("NeedStruct", map[string]string{"Value": v.String()})
	}
	for fi := 0; fi < v.NumField(); fi++ {
		f := v.Type().Field(fi)
		fv := v.Field(fi)
		if enum := f.Tag.Get("enum"); enum != "" {
			_, codeLabels := Enums(entity, f.Name)
			switch fv.Kind() {
			case reflect.String:
				val := fv.String()
				if val != "" {
					if _, ok := codeLabels[val]; !ok {
						return Err("BadEnum", map[string]string{"Field": f.Name, "Value": val})
					}
				}
			case reflect.Slice:
				for fvi := 0; fvi < fv.Len(); fvi++ {
					val := fv.Index(fvi).String()
					if _, ok := codeLabels[val]; !ok {
						return Err("BadEnum", map[string]string{"Field": f.Name, "Value": val})
					}
				}
			}
		}
		// drill down into other objects?
		switch fv.Kind() {
		case reflect.Array:
			panic("TODO!")
		case reflect.Map:
			if !fv.IsNil() {
				panic("TODO!")
			}
		case reflect.Ptr, reflect.Interface:
			if !fv.IsNil() {
				validate(fv.Interface())
			}
		case reflect.Slice:
			if !fv.IsNil() {
				t := fv.Type().Elem().Kind()
				if t == reflect.Struct {
					for fvi := 0; fvi < fv.Len(); fvi++ {
						validate(fv.Index(fvi).Interface())
					}
				}
			}
		}
	}
	return nil
}

// GetOptions gets options
func GetOptions(req *Request, pub *Publication) *Response {
	if req.Language == "" {
		return &Response{ErrorCode: "NeedLanguage"}
	}
	if req.Language != "en-us" {
		return &Response{ErrorCode: "BadLanguage"}
	}
	return &Response{
		Options: options,
	}
}

func singleField(entity interface{}, names []string) (string, error) {
	// for example, if entity is a Deal and names is []string{"Rental", "Sale"}, then one and only one of those fields should be defined, and return the name of it
	found := []string{}
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", Err("NeedStruct", map[string]string{"Value": v.String()})
	}
	for _, name := range names {
		fv := v.FieldByName(name)
		if !fv.IsNil() {
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return "", Err("NeedField", map[string]string{"Fields": strings.Join(names, ",")})
	}
	if len(found) > 1 {
		return "", Err("ExtraFields", map[string]string{"Fields": strings.Join(found, ",")})
	}
	return found[0], nil
}
