package main

import (
	"expvar"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/go-recaptcha/recaptcha"
	"github.com/gorilla/handlers"
	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
	"github.com/oxtoacart/bpool"
	"github.com/paulbellamy/ratecounter"
)

var captcha *recaptcha.Recaptcha
var api *slack.Client
var bufpool *bpool.BufferPool
var indexTemplate = template.Must(template.New("index.tmpl").ParseFiles("templates/index.tmpl"))

// Slack statistics
var userCount int
var activeUserCount int
var statsMutex sync.RWMutex // Guards slack statistics variables

var m = expvar.NewMap("metrics")
var counter *ratecounter.RateCounter
var hitsPerMinute expvar.Int
var requests expvar.Int
var inviteErrors expvar.Int
var missingFirstName expvar.Int
var missingLastName expvar.Int
var missingEmail expvar.Int
var missingCoC expvar.Int
var successfulCaptcha expvar.Int
var failedCaptcha expvar.Int
var invalidCaptcha expvar.Int

// config
var c Specification

type Specification struct {
	Port           string `required:"true"`
	CaptchaSitekey string `required:"true"`
	CaptchaSecret  string `required:"true"`
	SlackToken     string `required:"true"`
}

func init() {
	err := envconfig.Process("slackinviter", &c)
	if err != nil {
		log.Fatal(err.Error())
	}
	counter = ratecounter.NewRateCounter(1 * time.Minute)
	m.Set("hits_per_minute", &hitsPerMinute)
	m.Set("requests", &requests)
	m.Set("invite_errors", &inviteErrors)
	m.Set("missing_first_name", &missingFirstName)
	m.Set("missing_last_name", &missingLastName)
	m.Set("missing_email", &missingEmail)
	m.Set("missing_coc", &missingCoC)
	m.Set("failed_captcha", &failedCaptcha)
	m.Set("invalid_captcha", &invalidCaptcha)
	m.Set("successful_captcha", &successfulCaptcha)
	// Init stuff
	captcha = recaptcha.New(c.CaptchaSecret)
	api = slack.New(c.SlackToken)
	bufpool = bpool.NewBufferPool(64)
}
func main() {
	go pollSlack()
	mux := http.NewServeMux()
	mux.HandleFunc("/invite/", handleInvite)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", homepage)
	mux.Handle("/debug/vars", http.DefaultServeMux)
	err := http.ListenAndServe(":"+c.Port, handlers.CombinedLoggingHandler(os.Stdout, mux))
	if err != nil {
		log.Fatal(err.Error())
	}
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
	requests.Add(1)
	statsMutex.RLock()
	data := map[string]interface{}{"SiteKey": c.CaptchaSitekey, "UserCount": userCount, "ActiveCount": activeUserCount}
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
		failedCaptcha.Add(1)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	valid, err := captcha.Verify(captchaResponse, remoteIP)
	if err != nil {
		failedCaptcha.Add(1)
		http.Error(w, "Error validating recaptcha.. Did you click it?", http.StatusPreconditionFailed)
		return
	}
	if !valid {
		invalidCaptcha.Add(1)
		http.Error(w, "Invalid recaptcha", http.StatusInternalServerError)
		return

	}
	successfulCaptcha.Add(1)
	fname := r.FormValue("fname")
	lname := r.FormValue("lname")
	email := r.FormValue("email")
	coc := r.FormValue("coc")
	if email == "" {
		missingEmail.Add(1)
		http.Error(w, "Missing email", http.StatusPreconditionFailed)
		return
	}
	if fname == "" {
		missingFirstName.Add(1)
		http.Error(w, "Missing first name", http.StatusPreconditionFailed)
		return
	}
	if lname == "" {
		missingLastName.Add(1)
		http.Error(w, "Missing last name", http.StatusPreconditionFailed)
		return
	}
	if coc != "1" {
		missingCoC.Add(1)
		http.Error(w, "You need to accept the code of conduct", http.StatusPreconditionFailed)
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
