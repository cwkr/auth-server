package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/cwkr/auth-server/htmlutil"
	"github.com/cwkr/auth-server/oauth2"
	"github.com/cwkr/auth-server/people"
	"github.com/cwkr/auth-server/server"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	settings     *server.Settings
	tokenService oauth2.TokenCreator
)

func main() {
	var err error
	var settingsFilename string
	var saveSettings bool

	log.SetOutput(os.Stdout)

	flag.StringVar(&settingsFilename, "config", "auth-server.json", "config file name")
	flag.BoolVar(&saveSettings, "save", false, "save config and exit")
	flag.Parse()

	// Set defaults
	settings = server.NewDefaultSettings()

	configBytes, err := os.ReadFile(settingsFilename)
	if err == nil {
		err = json.Unmarshal(configBytes, settings)
		if err != nil {
			panic(err)
		}
	}

	err = settings.LoadKeys(saveSettings)
	if err != nil {
		panic(err)
	}

	if saveSettings {
		log.Printf("Saving config file %s", settingsFilename)
		configJson, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(settingsFilename, configJson, 0644); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	tokenService, err = oauth2.NewTokenService(
		settings.PrivateKey(),
		settings.KeyID(),
		settings.Issuer,
		settings.Scope,
		int64(settings.AccessTokenLifetime),
		int64(settings.RefreshTokenLifetime),
		settings.Claims,
	)
	if err != nil {
		panic(err)
	}

	var sessionStore = sessions.NewCookieStore([]byte(settings.SessionSecret))
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.MaxAge = 0

	var peopleStore people.Store
	if settings.PeopleStore != nil {
		if strings.HasPrefix(settings.PeopleStore.URI, "postgresql:") {
			if peopleStore, err = people.NewDatabaseStore(sessionStore, settings.Users, settings.SessionName, settings.SessionLifetime, settings.PeopleStore); err != nil {
				panic(err)
			}
		} else if strings.HasPrefix(settings.PeopleStore.URI, "ldap:") || strings.HasPrefix(settings.PeopleStore.URI, "ldaps:") {
			if peopleStore, err = people.NewLdapStore(sessionStore, settings.Users, settings.SessionName, settings.SessionLifetime, settings.PeopleStore); err != nil {
				panic(err)
			}
		} else {
			panic(errors.New("unsupported or empty store uri: " + settings.PeopleStore.URI))
		}
	} else {
		peopleStore = people.NewEmbeddedStore(sessionStore, settings.Users, settings.SessionName, settings.SessionLifetime)
	}

	var router = mux.NewRouter()

	router.NotFoundHandler = htmlutil.NotFoundHandler()
	router.Handle("/", server.IndexHandler(settings, peopleStore, !settings.DisablePKCE)).
		Methods(http.MethodGet)
	router.Handle("/style", server.StyleHandler()).
		Methods(http.MethodGet)
	router.Handle("/jwks", oauth2.JwksHandler(settings.AllKeys())).
		Methods(http.MethodGet, http.MethodOptions)
	router.Handle("/token", oauth2.TokenHandler(tokenService, peopleStore, settings.Clients, settings.DisablePKCE)).
		Methods(http.MethodOptions, http.MethodPost)
	router.Handle("/authorize", oauth2.AuthorizeHandler(tokenService, peopleStore, settings.Clients, settings.Scope, settings.DisablePKCE)).
		Methods(http.MethodGet)
	router.Handle("/login", server.LoginHandler(settings, peopleStore, sessionStore)).
		Methods(http.MethodGet, http.MethodPost)
	router.Handle("/logout", server.LogoutHandler(settings, sessionStore))
	router.Handle("/.well-known/openid-configuration", oauth2.DiscoveryDocumentHandler(settings.Issuer, settings.Scope, settings.DisablePKCE)).
		Methods(http.MethodGet, http.MethodOptions)
	router.Handle("/userinfo", oauth2.UserInfoHandler(peopleStore, tokenService, settings.Claims, settings.SessionName)).
		Methods(http.MethodGet, http.MethodOptions)
	if !settings.DisablePeopleAPI {
		router.Handle("/people/{user_id}", server.PeopleAPIHandler(peopleStore)).
			Methods(http.MethodGet, http.MethodOptions)
	}

	log.Printf("Listening on http://localhost:%d/", settings.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", settings.Port), router)
	if err != nil {
		panic(err)
	}
}
