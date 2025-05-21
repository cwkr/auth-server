package server

import (
	"github.com/cwkr/auth-server/internal/htmlutil"
	"github.com/cwkr/auth-server/internal/httputil"
	"github.com/cwkr/auth-server/internal/oauth2/clients"
	"github.com/cwkr/auth-server/internal/otpkey"
	"github.com/cwkr/auth-server/internal/people"
	"github.com/cwkr/auth-server/internal/stringutil"
	"log"
	"net/http"
	"strings"
)

type otpHandler struct {
	peopleStore people.Store
	clientStore clients.Store
	otpKeyStore otpkey.Store
	basePath    string
	sessionName string
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
		if otpKey, err := o.otpKeyStore.Lookup(uid); err == nil {
			w.Header().Set("Content-Type", "image/png")
			httputil.NoCache(w)
			if err := otpKey.PNG(w); err != nil {
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

func OTPHandler(peopleStore people.Store, clientStore clients.Store, otpKeyStore otpkey.Store, basePath, sessionName string) http.Handler {
	return &otpHandler{
		peopleStore: peopleStore,
		clientStore: clientStore,
		otpKeyStore: otpKeyStore,
		basePath:    basePath,
		sessionName: sessionName,
	}
}
