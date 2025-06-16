package server

import (
	_ "embed"
	"github.com/cwkr/auth-server/internal/htmlutil"
	"github.com/cwkr/auth-server/internal/httputil"
	"github.com/cwkr/auth-server/internal/oauth2/clients"
	"github.com/cwkr/auth-server/internal/otpauth"
	"github.com/cwkr/auth-server/internal/people"
	"github.com/cwkr/auth-server/internal/stringutil"
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
		if otpKey, err := o.otpAuthStore.Lookup(uid); err == nil {
			httputil.NoCache(w)
			var imageURL string
			if !client.DisableTOTP && otpKey != nil {
				if dataURL, err := otpKey.PNG(); err != nil {
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
				"user_totp_enabled":   otpKey != nil,
				"version":             o.version,
			}); err != nil {
				htmlutil.Error(w, o.basePath, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
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
