package recaptcha

//go:generate ffjson -noencoder $GOFILE

import (
	"encoding/json"
	"net/http"
	"net/url"
)

const endpoint = `https://www.google.com/recaptcha/api/siteverify`

// Recaptcha must be created by the function New.
type Recaptcha struct {
	// secret is stored as slice of strings to avoid new alloc on each verification
	secret []string
}

type response struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// New creates a new recaptcha instance with given secret.
func New(secret string) *Recaptcha {
	return &Recaptcha{
		secret: []string{secret},
	}
}

// Verify validates a captcha response with the google servers. When returns true: the catpcha was valid.
// The returned error might be of type Error, indicating that the API request had invalid information in it.
// See https://developers.google.com/recaptcha/docs/verify for a list of error codes.
func (r *Recaptcha) Verify(captchaResponse string, remoteIP string) (bool, error) {
	values := url.Values{"secret": r.secret, "response": []string{captchaResponse}}
	if remoteIP != "" {
		values["remoteip"] = []string{remoteIP}
	}

	resp, err := http.PostForm(endpoint, values)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	verification := response{}
	err = json.NewDecoder(resp.Body).Decode(&verification)
	if err != nil {
		return false, err
	}

	if len(verification.ErrorCodes) != 0 {
		return false, &Error{Codes: verification.ErrorCodes}
	}

	return verification.Success, nil
}
