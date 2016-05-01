package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/go-recaptcha/recaptcha"
	"github.com/gorilla/handlers"
	"github.com/nlopes/slack"
	"github.com/oxtoacart/bpool"
)

var captcha *recaptcha.Recaptcha
var api *slack.Client
var bufpool *bpool.BufferPool
var indexTemplate = template.Must(template.New("index.tmpl").ParseFiles("templates/index.tmpl"))

var captchaSitekey = flag.String("captchaSitekey", "REPLACEME", "reCaptcha Sitekey")
var captchaSecret = flag.String("captchaSecret", "REPLACEME", "reCaptcha Secret")
var slackToken = flag.String("slackToken", "REPLACEME", "Slack API Token")
var listenAddr = flag.String("listenAddr", ":8887", "Address to listen on")

// Slack statistics
var userCount int
var activeUserCount int
var statsMutex sync.RWMutex // Guards slack statistics variables

func init() {
	flag.Parse()
	// Init stuff
	captcha = recaptcha.New(*captchaSecret)
	api = slack.New(*slackToken)
	bufpool = bpool.NewBufferPool(64)
	go pollSlack()
}
func pollSlack() {
	for {
		users, err := api.GetUsers()
		if err != nil {
			log.Println("error polling slack for users:", err)
			continue
		}
		uCount := 0 // users
		aCount := 0 // active users
		for _, u := range users {
			if u.ID != "USLACKBOT" && !u.IsBot && !u.Deleted {
				uCount += 1
				if u.Presence == "active" {
					aCount += 1
				}
			}
		}
		statsMutex.Lock()
		userCount = uCount
		activeUserCount = aCount
		statsMutex.Unlock()
		time.Sleep(10 * time.Minute)
	}
}

// Homepage renders the homepage
func homepage(w http.ResponseWriter, r *http.Request) {
	statsMutex.RLock()
	data := map[string]interface{}{"SiteKey": *captchaSitekey, "UserCount": userCount, "ActiveCount": activeUserCount}
	statsMutex.RUnlock()
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	err := indexTemplate.Execute(buf, data)
	if err != nil {
		log.Println("error rendering template:", err)
		http.Error(w, "error rendering template :-(", http.StatusInternalServerError)
		return
	}
	// Set the header and write the buffer to the http.ResponseWriter
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// ShowPost renders a single post
func handleInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	captchaResponse := r.FormValue("g-recaptcha-response")
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	valid, err := captcha.Verify(captchaResponse, remoteIP)
	if err != nil {
		http.Error(w, "Error validating recaptcha.. Did you click it?", http.StatusPreconditionFailed)
		return
	}
	if !valid {
		http.Error(w, "Invalid recaptcha", http.StatusInternalServerError)
		return

	}
	fname := r.FormValue("fname")
	lname := r.FormValue("lname")
	email := r.FormValue("email")
	coc := r.FormValue("coc")
	if fname == "" {
		http.Error(w, "Missing first name", http.StatusPreconditionFailed)
		return
	}
	if lname == "" {
		http.Error(w, "Missing last name", http.StatusPreconditionFailed)
		return
	}
	if email == "" {
		http.Error(w, "Missing email", http.StatusPreconditionFailed)
		return
	}
	if coc != "1" {
		http.Error(w, "You need to accept the code of conduct", http.StatusPreconditionFailed)
		return
	}
	err = api.InviteToTeam("Gophers", fname, lname, email)
	if err != nil {
		log.Println("InviteToTeam error:", err)
		http.Error(w, "Error inviting you :-(", http.StatusInternalServerError)
		return
	}
}

func main() {
	// Check if we are missing vital config values
	if *captchaSitekey == "REPLACEME" || *captchaSecret == "REPLACEME" || *slackToken == "REPLACEME" {
		log.Fatalln("Missing required input values")
	}
	log.SetPrefix("slackinvite:")
	mux := http.NewServeMux()
	mux.HandleFunc("/invite/", handleInvite)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", homepage)
	log.Println("listening on port", *listenAddr)
	err := http.ListenAndServe(":"+*listenAddr, handlers.CombinedLoggingHandler(os.Stdout, mux))
	if err != nil {
		panic(err)
	}
}
