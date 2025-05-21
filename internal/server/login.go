package server

import (
	_ "embed"
	"fmt"
	"github.com/cwkr/authd/internal/htmlutil"
	"github.com/cwkr/authd/internal/httputil"
	"github.com/cwkr/authd/internal/oauth2/clients"
	"github.com/cwkr/authd/internal/otpkey"
	"github.com/cwkr/authd/internal/people"
	"github.com/cwkr/authd/internal/stringutil"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/login.gohtml
var loginTpl string

func LoadLoginTemplate(filename string) error {
	if bytes, err := os.ReadFile(filename); err == nil {
		loginTpl = string(bytes)
		return nil
	} else {
		return err
	}
}

type loginHandler struct {
	basePath    string
	peopleStore people.Store
	clientStore clients.Store
	otpKeyStore otpkey.Store
	issuer      string
	sessionName string
	tpl         *template.Template
}

func (j *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)
	var (
		message, userID, password, clientID, sessionName, code string
		activeSession, verifiedSession                         bool
		otpKey                                                 *otpkey.OTPKey
	)

	clientID = strings.ToLower(r.FormValue("client_id"))
	if clientID == "" {
		htmlutil.Error(w, j.basePath, "client_id parameter is required", http.StatusBadRequest)
		return
	}

	sessionName = j.sessionName
	if client, err := j.clientStore.Lookup(clientID); err == nil {
		if client.SessionName != "" {
			sessionName = client.SessionName
		}
	} else {
		htmlutil.Error(w, j.basePath, "invalid_client", http.StatusForbidden)
		return
	}

	userID, activeSession, verifiedSession = j.peopleStore.IsSessionActive(r, sessionName)

	if r.Method == http.MethodPost {
		if !activeSession {
			userID = strings.TrimSpace(r.PostFormValue("user_id"))
			if userID == "" {
				userID = strings.TrimSpace(r.PostFormValue("username"))
			}
		}

		if k, err := j.otpKeyStore.Lookup(userID); err == nil {
			otpKey = k
		}

		if !activeSession {
			password = r.PostFormValue("password")
			if stringutil.IsAnyEmpty(userID, password) {
				message = "username and password must not be empty"
			} else {
				if realUserID, err := j.peopleStore.Authenticate(userID, password); err == nil {
					var codeRequired = otpKey != nil
					if err := j.peopleStore.SaveSession(r, w, time.Now(), realUserID, sessionName, codeRequired); err != nil {
						htmlutil.Error(w, j.basePath, err.Error(), http.StatusInternalServerError)
						return
					}
					log.Printf("user_id=%s", realUserID)
					if codeRequired {
						activeSession = true
						verifiedSession = false
					} else {
						httputil.RedirectQuery(w, r, strings.TrimRight(j.issuer, "/")+"/authorize", r.URL.Query())
						return
					}
				} else {
					message = err.Error()
				}
			}
		} else {
			if !verifiedSession {
				code = strings.TrimSpace(r.PostFormValue("code"))
				if stringutil.IsAnyEmpty(code) {
					message = "code must not be empty"
				} else {
					if otpKey == nil {
						message = "OTP Key not missing"
					} else {
						if otpKey.Verify(code) == true {
							if err := j.peopleStore.VerifySession(r, w, userID, sessionName); err != nil {
								message = err.Error()
							} else {
								httputil.RedirectQuery(w, r, strings.TrimRight(j.issuer, "/")+"/authorize", r.URL.Query())
								return
							}
						} else {
							message = fmt.Sprintf("code %s is invalid", code)
						}
					}
				}
			}
		}
	} else if r.Method == http.MethodGet {
		if activeSession && verifiedSession {
			message = "current active session for " + userID
		}
		httputil.NoCache(w)
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	var err = j.tpl.ExecuteTemplate(w, "login", map[string]any{
		"base_path":      j.basePath,
		"issuer":         strings.TrimRight(j.issuer, "/"),
		"query":          template.HTML("?" + r.URL.RawQuery),
		"message":        message,
		"user_id":        userID,
		"password_empty": password == "",
		"code_required":  activeSession && !verifiedSession,
	})
	if err != nil {
		htmlutil.Error(w, j.basePath, err.Error(), http.StatusInternalServerError)
	}
}

func LoginHandler(basePath string, peopleStore people.Store, clientStore clients.Store, otpKeyStore otpkey.Store, issuer, sessionName string) http.Handler {
	return &loginHandler{
		basePath:    basePath,
		peopleStore: peopleStore,
		clientStore: clientStore,
		otpKeyStore: otpKeyStore,
		issuer:      issuer,
		sessionName: sessionName,
		tpl:         template.Must(template.New("login").Parse(loginTpl)),
	}
}
