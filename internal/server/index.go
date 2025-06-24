package server

import (
	"crypto/rsa"
	_ "embed"
	"fmt"
	"github.com/cwkr/authd/internal/htmlutil"
	"github.com/cwkr/authd/internal/httputil"
	"github.com/cwkr/authd/internal/oauth2/clients"
	"github.com/cwkr/authd/internal/oauth2/pkce"
	"github.com/cwkr/authd/internal/people"
	"github.com/cwkr/authd/internal/stringutil"
	"github.com/cwkr/authd/settings"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

//go:embed templates/index.gohtml
var indexTpl string

type indexHandler struct {
	basePath       string
	serverSettings *settings.Server
	publicKey      *rsa.PublicKey
	peopleStore    people.Store
	clientStore    clients.Store
	scope          string
	tpl            *template.Template
	version        string
}

type activeSession struct {
	ClientID string
	UserID   string
	Verified bool
}

func (i *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	var clientIDs []string
	var activeSessions []activeSession

	if clientsPerSessionName, err := i.clientStore.PerSessionNameMap(i.serverSettings.SessionName); err == nil {
		for sessionName, sessionClients := range clientsPerSessionName {
			clientIDs = append(clientIDs, sessionClients...)
			if uid, valid, verified := i.peopleStore.IsSessionActive(r, sessionName); valid == true {
				for _, cid := range sessionClients {
					activeSessions = append(activeSessions, activeSession{cid, uid, verified})
				}
			}
		}
	} else {
		htmlutil.Error(w, i.basePath, err.Error(), http.StatusInternalServerError)
	}

	httputil.NoCache(w)

	var title = strings.TrimSpace(i.serverSettings.Title)
	if title == "" {
		title = "Auth Server"
	}
	var codeVerifier = stringutil.RandomAlphanumericString(10)
	var err = i.tpl.ExecuteTemplate(w, "index", map[string]any{
		"base_path":       i.basePath,
		"issuer":          strings.TrimRight(i.serverSettings.Issuer, "/"),
		"title":           title,
		"public_key":      i.serverSettings.PublicKeyPEM(),
		"state":           fmt.Sprint(rand.Int()),
		"nonce":           stringutil.RandomAlphanumericString(10),
		"scopes":          strings.Fields(i.scope),
		"code_verifier":   codeVerifier,
		"code_challenge":  pkce.CodeChallange(codeVerifier),
		"version":         i.version,
		"client_ids":      clientIDs,
		"active_sessions": activeSessions,
	})
	if err != nil {
		htmlutil.Error(w, i.basePath, err.Error(), http.StatusInternalServerError)
	}
}

func IndexHandler(basePath string, serverSettings *settings.Server, peopleStore people.Store, clientStore clients.Store, scope, version string) http.Handler {
	return &indexHandler{
		basePath:       basePath,
		serverSettings: serverSettings,
		peopleStore:    peopleStore,
		clientStore:    clientStore,
		scope:          scope,
		tpl:            template.Must(template.New("index").Parse(indexTpl)),
		version:        version,
	}
}
