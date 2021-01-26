package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"boatfuji.com/api"
	"boatfuji.com/sites"
)

func main() {
	defer func() {
		if recover() != nil {
			notifyAdmin("Panic")
		}
	}()
	// notify admin on shutdown
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch)
		_ = <-ch
		notifyAdmin("Stopped")
	}()
	api.Start()
	sites.Start()
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("www"))))
	http.HandleFunc("/api/", api.DispatchToAPIHandler)
	http.HandleFunc("/help/", helpHandler)
	// addr.txt will have content like boatfuji.com:8168,66.226.72.106:443
	if addrFile, err := ioutil.ReadFile("addr.txt"); err != nil {
		panic(err)
	} else {
		addrs := strings.Split(string(addrFile), ",")
		for i, addr := range addrs {
			// wait only on the last addr, since we don't want to block, nor do we want to exit
			if i == len(addrs)-1 {
				listenAndServe(addr)
			} else {
				go listenAndServe(addr)
			}
		}
	}
}

func notifyAdmin(body string) {
	api.SendEmail("BoatFuji", "support@boatfuji.com", []string{"4078348834@txt.att.net"}, body)
}

func helpHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/help/"):] // i.e., "terms"
	if !strings.Contains(name, "..") && name != "_template" {
		if bytes, err := ioutil.ReadFile("www/help/" + name + ".html"); err == nil {
			html := string(bytes)
			title := "Boat Fuji " + name
			h1 := strings.Split(html, "<h1>")
			if len(h1) > 1 {
				title = strings.Split(h1[1], "</h1>")[0]
			}
			bytes, _ = ioutil.ReadFile("www/help/_template.html")
			year := strconv.Itoa(time.Now().Year())
			html = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(bytes), "{{Title}}", title), "{{HTML}}", html), "{{Year}}", year)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func listenAndServe(addr string) {
	log.Println("Listening at " + addr)
	if strings.HasSuffix(addr, ":443") {
		log.Fatal(http.ListenAndServeTLS(addr, "certs/www_boatfuji_com.crt", "certs/server.key", nil))
	} else {
		log.Fatal(http.ListenAndServe(addr, nil))
	}
}
