package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

// a request may have a subscription on it if it's long-lived and will use Server-Sent Events (SSE) to send inserts, updates, and deletes
type subscription struct {
	ID          int64
	Req         *Request
	Handler     func(req *Request, pub *Publication) *Response
	LastResp    *Response
	Started     *time.Time
	LastEventID int64
}

// publication is sent to sseSink when the API detects an org, user, boat, or event has changed
// Server-Sent Event (SSE) channels (assumes only one SSE per Session)
var sseSink = make(chan *Publication, 1)
var sseOpening = make(chan *Session)
var sseClosing = make(chan *Session)
var sseActive = make(map[chan *Publication]*Session)

// Publication is what may or may not trigger any changes in subscription data that is then sent via Server-Sent Events (SSE)
type Publication struct {
	// TODO
	Ping     bool
	SetLevel int // 1=Org, 2=User, 3=Boat, 4=Deal, 5=Event
}

// subscribe is called by DispatchToAPIHandler to register a new subscription in a Session
func subscribe(req *Request, handler func(req *Request, pub *Publication) *Response, resp *Response) {
	req.Session.LastSubscriptionID++
	subscriptionID := req.Session.LastSubscriptionID
	resp.SubscriptionID = subscriptionID
	subscription := &subscription{
		ID:       subscriptionID,
		Req:      req,
		Handler:  handler,
		LastResp: resp,
		Started:  now(),
	}
	req.Session.SubscriptionsMutex.Lock()
	req.Session.Subscriptions[subscriptionID] = subscription
	req.Session.SubscriptionsMutex.Unlock()
	req.Subscription = subscription
	reqName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	reqJSON, _ := json.Marshal(req)
	sessionLog(req, "Info", "subscribe %d %s %s", subscription.ID, reqName, reqJSON)
}

func init() {
	apiHandlers["Unsubscribe"] = unsubscribe
	go func() {
		for {
			select {
			case session := <-sseOpening:
				sseActive[session.SSEConnection] = session
				sessionLog(&Request{Session: session}, "Info", "opened connection")
			case session := <-sseClosing:
				sessionLog(&Request{Session: session}, "Info", "closed connection")
				delete(sseActive, session.SSEConnection)
				session.SSEConnection = nil
			case pub := <-sseSink:
				for _, session := range sseActive {
					select {
					case session.SSEConnection <- pub:
					case <-time.After(10 * time.Second):
						sessionLog(&Request{Session: session}, "Info", "lost connection")
						delete(sseActive, session.SSEConnection)
						session.SSEConnection = nil
					}
				}
			}
		}
	}()
	// we ping every minute to clean up disconnected sessions
	go func() {
		for {
			time.Sleep(time.Minute)
			sseSink <- &Publication{Ping: true}
		}
	}()
}

func handleSSE(w http.ResponseWriter, r *http.Request, session *Session) {
	// the writer must support flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusUnsupportedMediaType)
		return
	}
	if session.SSEConnection != nil {
		http.Error(w, "SSE called more than once", http.StatusTooManyRequests)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	session.SSEConnection = make(chan *Publication) // if there was a previous /api/SSE, it will no longer receive any publications
	// tell the event loop in init() that we're opening a new SSE on a session
	sseOpening <- session
	// when we end this response, tell the event loop in the init()
	defer func() {
		sseClosing <- session
	}()
	// listen to connection close
	// done := w.(http.CloseNotifier).CloseNotify()
	done := r.Context().Done()
	// send initial {} just so SSE knows it's working
	json.NewEncoder(w).Encode(Response{})
	flusher.Flush()
	// process any publications, possibly outputting JSON, until connection is done
	for {
		select {
		case <-done:
			sessionLog(&Request{Session: session}, "Info", "SSE done")
			return
		default:
			pub := <-session.SSEConnection
			if !pub.Ping {
				// get snapshot of subscriptions to avoid contention
				session.SubscriptionsMutex.RLock()
				subs := make([]*subscription, 0, len(session.Subscriptions))
				for _, sub := range session.Subscriptions {
					subs = append(subs, sub)
				}
				session.SubscriptionsMutex.RUnlock()
				// handle each subcription
				for _, sub := range subs {
					resp := sub.Handler(sub.Req, pub)
					deltaResp := delta(resp, sub.LastResp)
					sub.LastResp = resp
					if deltaResp != nil {
						deltaResp.SubscriptionID = sub.ID
						deltaRespJSON, _ := json.Marshal(deltaResp)
						sub.LastEventID++
						fmt.Fprintf(w, "id:%d\ndata: %s\n\n", sub.LastEventID, deltaRespJSON)
						flusher.Flush()
					}
				}
			}
		}
	}
}

func delta(nextResp *Response, lastResp *Response) *Response {
	deltaResp := Response{}
	hasDelta := false
	// check delta of Orgs
	if nextResp.Orgs != nil && lastResp.Orgs != nil {
		deltaResp.Orgs = map[int64]*Org{}
		for key, lastValue := range lastResp.Orgs {
			if nextValue, ok := nextResp.Orgs[key]; ok {
				// a lastResp item was also in nextResp, so if they differ, update!
				if !reflect.DeepEqual(nextValue, lastValue) {
					deltaResp.Orgs[key] = nextValue
					hasDelta = true
				}
			} else {
				// a lastResp item wasn't in nextResp, so delete!
				deltaResp.Orgs[key] = nil
				hasDelta = true
			}
		}
		for key, nextValue := range nextResp.Orgs {
			if _, ok := lastResp.Orgs[key]; !ok {
				// a nextResp item wasn't in lastResp, so insert!
				deltaResp.Orgs[key] = nextValue
				hasDelta = true
			}
		}
	}
	// check delta of Users
	if nextResp.Users != nil && lastResp.Users != nil {
		deltaResp.Users = map[int64]*User{}
		for key, lastValue := range lastResp.Users {
			if nextValue, ok := nextResp.Users[key]; ok {
				// a lastResp item was also in nextResp, so if they differ, update!
				if !reflect.DeepEqual(nextValue, lastValue) {
					deltaResp.Users[key] = nextValue
					hasDelta = true
				}
			} else {
				// a lastResp item wasn't in nextResp, so delete!
				deltaResp.Users[key] = nil
				hasDelta = true
			}
		}
		for key, nextValue := range nextResp.Users {
			if _, ok := lastResp.Users[key]; !ok {
				// a nextResp item wasn't in lastResp, so insert!
				deltaResp.Users[key] = nextValue
				hasDelta = true
			}
		}
	}
	// check delta of Boats
	if nextResp.Boats != nil && lastResp.Boats != nil {
		deltaResp.Boats = map[int64]*Boat{}
		for key, lastValue := range lastResp.Boats {
			if nextValue, ok := nextResp.Boats[key]; ok {
				// a lastResp item was also in nextResp, so if they differ, update!
				if !reflect.DeepEqual(nextValue, lastValue) {
					deltaResp.Boats[key] = nextValue
					hasDelta = true
				}
			} else {
				// a lastResp item wasn't in nextResp, so delete!
				deltaResp.Boats[key] = nil
				hasDelta = true
			}
		}
		for key, nextValue := range nextResp.Boats {
			if _, ok := lastResp.Boats[key]; !ok {
				// a nextResp item wasn't in lastResp, so insert!
				deltaResp.Boats[key] = nextValue
				hasDelta = true
			}
		}
	}
	// check delta of Deals
	if nextResp.Deals != nil && lastResp.Deals != nil {
		deltaResp.Deals = map[int64]*Deal{}
		for key, lastValue := range lastResp.Deals {
			if nextValue, ok := nextResp.Deals[key]; ok {
				// a lastResp item was also in nextResp, so if they differ, update!
				if !reflect.DeepEqual(nextValue, lastValue) {
					deltaResp.Deals[key] = nextValue
					hasDelta = true
				}
			} else {
				// a lastResp item wasn't in nextResp, so delete!
				deltaResp.Deals[key] = nil
				hasDelta = true
			}
		}
		for key, nextValue := range nextResp.Deals {
			if _, ok := lastResp.Deals[key]; !ok {
				// a nextResp item wasn't in lastResp, so insert!
				deltaResp.Deals[key] = nextValue
				hasDelta = true
			}
		}
	}
	// check delta of Events
	if nextResp.Events != nil && lastResp.Events != nil {
		deltaResp.Events = map[int64]*Event{}
		for key, lastValue := range lastResp.Events {
			if nextValue, ok := nextResp.Events[key]; ok {
				// a lastResp item was also in nextResp, so if they differ, update!
				if !reflect.DeepEqual(nextValue, lastValue) {
					deltaResp.Events[key] = nextValue
					hasDelta = true
				}
			} else {
				// a lastResp item wasn't in nextResp, so delete!
				deltaResp.Events[key] = nil
				hasDelta = true
			}
		}
		for key, nextValue := range nextResp.Events {
			if _, ok := lastResp.Events[key]; !ok {
				// a nextResp item wasn't in lastResp, so insert!
				deltaResp.Events[key] = nextValue
				hasDelta = true
			}
		}
	}
	if hasDelta {
		return &deltaResp
	}
	return nil
}

func unsubscribe(req *Request, pub *Publication) *Response {
	if req.Session == nil {
		return &Response{ErrorCode: "MustSignIn"}
	}
	if req.SubscriptionID == 0 {
		return &Response{ErrorCode: "NeedSubscriptionID"}
	}
	sessionLog(req, "Info", "unsubscribe %d", req.SubscriptionID)
	req.Session.SubscriptionsMutex.Lock()
	if _, ok := req.Session.Subscriptions[req.SubscriptionID]; ok {
		delete(req.Session.Subscriptions, req.SubscriptionID)
		req.Session.SubscriptionsMutex.Unlock()
		return &Response{}
	}
	req.Session.SubscriptionsMutex.Unlock()
	return &Response{ErrorCode: "BadSubscriptionID"}
}
