package api

import (
	"strings"
)

// Site is a third party site where we can harvest boat and user data and contact users
type Site interface {
	Harvest(string) error
}

// Sites is a registry of all sites by their URL; i.e., "https://www.boatsetter.com/"
var Sites = make(map[string]Site)

func init() {
	apiHandlers["Harvest"] = Harvest
}

// Harvest harvests users, boats, etc. from competitor sites
func Harvest(req *Request, pub *Publication) *Response {
	// there are 4 ways of searching boats and users:
	// 1. if QA is true, read from harvest directory instead of fetching from website
	// 2. if QA is false but User defined, harvest single user and all his/her boats
	// 3. if QA is false but Boat defined, harvest single boat and user
	// 4. otherwise, harvest all boats and users of those boats
	if !isStaff(req) {
		return accessDenied()
	}
	if req.QA {
		for _, site := range Sites {
			if err := site.Harvest(""); err != nil {
				return errResponse(err)
			}
		}
		return &Response{}
	}
	handleURLs := func(urls []string) *Response {
		for _, url := range urls {
			handled := false
			for baseURL, site := range Sites {
				if strings.HasPrefix(url, baseURL) {
					if err := site.Harvest(url); err != nil {
						return errResponse(err)
					}
					handled = true
					break
				}
			}
			if !handled {
				return &Response{ErrorCode: "BadURL", ErrorDetails: map[string]string{"URL": url}}
			}
		}
		return &Response{}
	}
	if req.User != nil && req.User.URLs != nil {
		return handleURLs(req.User.URLs)
	}
	if req.Boat != nil && req.Boat.URLs != nil {
		return handleURLs(req.Boat.URLs)
	}
	for baseURL, site := range Sites {
		if err := site.Harvest(baseURL); err != nil {
			return errResponse(err)
		}
	}
	return &Response{}
}
