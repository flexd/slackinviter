package main

import (
	"expvar"
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
	"github.com/paulbellamy/ratecounter"
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

var counter *ratecounter.RateCounter
var hitsPerMinute = expvar.NewInt("hits_per_minute")
var requests = expvar.NewInt("requests")
var inviteErrors = expvar.NewInt("invite_errors")
var missingFirstName = expvar.NewInt("missing_first_name")
var missingLastName = expvar.NewInt("missing_last_name")
var missingEmail = expvar.NewInt("missing_email")
var missingCoC = expvar.NewInt("missing_code_of_conduct")
var successfulCaptcha = expvar.NewInt("successful_captchas")
var failedCaptcha = expvar.NewInt("failed_captchas")
var invalidCaptcha = expvar.NewInt("invalid_captchas")

func init() {
	counter = ratecounter.NewRateCounter(1 * time.Minute)
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
	counter.Incr(1)
	hitsPerMinute.Set(counter.Rate())
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
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		failedCaptcha.Add(1)
		return
	}

	valid, err := captcha.Verify(captchaResponse, remoteIP)
	if err != nil {
		http.Error(w, "Error validating recaptcha.. Did you click it?", http.StatusPreconditionFailed)
		failedCaptcha.Add(1)
		return
	}
	if !valid {
		http.Error(w, "Invalid recaptcha", http.StatusInternalServerError)
		invalidCaptcha.Add(1)
		return

	}
	fname := r.FormValue("fname")
	lname := r.FormValue("lname")
	email := r.FormValue("email")
	coc := r.FormValue("coc")
	if fname == "" {
		http.Error(w, "Missing first name", http.StatusPreconditionFailed)
		missingFirstName.Add(1)
		return
	}
	if lname == "" {
		http.Error(w, "Missing last name", http.StatusPreconditionFailed)
		missingLastName.Add(1)
		return
	}
	if email == "" {
		http.Error(w, "Missing email", http.StatusPreconditionFailed)
		missingEmail.Add(1)
		return
	}
	if coc != "1" {
		http.Error(w, "You need to accept the code of conduct", http.StatusPreconditionFailed)
		missingCoC.Add(1)
		return
	}
	err = api.InviteToTeam("Gophers", fname, lname, email)
	if err != nil {
		log.Println("InviteToTeam error:", err)
		inviteErrors.Add(1)
		http.Error(w, "Error inviting you :-(", http.StatusInternalServerError)
		return
	}
}
func onlyLocalhost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if host == "127.0.0.1" {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(404), 404)
		}
	})
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
	mux.Handle("/debug/vars", onlyLocalhost(http.DefaultServeMux))
	log.Println("listening on port", *listenAddr)
	err := http.ListenAndServe(":"+*listenAddr, handlers.CombinedLoggingHandler(os.Stdout, mux))
	if err != nil {
		panic(err)
	}
}
