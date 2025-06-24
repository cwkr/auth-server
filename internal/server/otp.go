package server

import (
	_ "embed"
	"github.com/cwkr/authd/internal/htmlutil"
	"github.com/cwkr/authd/internal/httputil"
	"github.com/cwkr/authd/internal/oauth2/clients"
	"github.com/cwkr/authd/internal/otpauth"
	"github.com/cwkr/authd/internal/people"
	"github.com/cwkr/authd/internal/stringutil"
	"html/template"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/otp.gohtml
var otpTpl string

type otpHandler struct {
	peopleStore  people.Store
	clientStore  clients.Store
	otpAuthStore otpauth.Store
	tpl          *template.Template
	basePath     string
	sessionName  string
	version      string
}

func (o *otpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	var (
		clientID    = strings.TrimSpace(r.FormValue("client_id"))
		sessionName = o.sessionName
	)

	if stringutil.IsAnyEmpty(clientID) {
		htmlutil.Error(w, o.basePath, "client_id parameter is required", http.StatusBadRequest)
		return
	}

	var client clients.Client
	if c, err := o.clientStore.Lookup(clientID); err != nil {
		htmlutil.Error(w, o.basePath, "client not found", http.StatusForbidden)
		return
	} else {
		client = *c
	}
	if client.SessionName != "" {
		sessionName = client.SessionName
	}

	if uid, valid, _ := o.peopleStore.IsSessionActive(r, sessionName); valid {
		keyWrapper, _ := o.otpAuthStore.Lookup(uid)
		httputil.NoCache(w)
		var imageURL string
		if !client.DisableTOTP && keyWrapper != nil {
			if dataURL, err := keyWrapper.PNG(); err != nil {
				htmlutil.Error(w, o.basePath, err.Error(), http.StatusInternalServerError)
				return
			} else {
				imageURL = dataURL
			}
		}
		if err := o.tpl.ExecuteTemplate(w, "otp", map[string]any{
			"base_path":           o.basePath,
			"qrcode":              template.URL(imageURL),
			"client_totp_enabled": !client.DisableTOTP,
			"user_totp_enabled":   keyWrapper != nil,
			"version":             o.version,
		}); err != nil {
			htmlutil.Error(w, o.basePath, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		htmlutil.Error(w, o.basePath, "not logged in", http.StatusUnauthorized)
		return
	}
}

func OTPHandler(peopleStore people.Store, clientStore clients.Store, otpAuthStore otpauth.Store, basePath, sessionName, version string) http.Handler {
	return &otpHandler{
		peopleStore:  peopleStore,
		clientStore:  clientStore,
		otpAuthStore: otpAuthStore,
		tpl:          template.Must(template.New("otp").Parse(otpTpl)),
		basePath:     basePath,
		sessionName:  sessionName,
		version:      version,
	}
}
