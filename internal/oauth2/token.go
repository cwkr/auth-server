package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cwkr/auth-server/internal/httputil"
	"github.com/cwkr/auth-server/internal/oauth2/pkce"
	"github.com/cwkr/auth-server/internal/people"
	"github.com/cwkr/auth-server/internal/stringutil"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

type tokenHandler struct {
	tokenService         TokenCreator
	peopleStore          people.Store
	clients              Clients
	scope                string
	refreshTokenRotation bool
}

func (t *tokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	httputil.AllowCORS(w, r, []string{http.MethodOptions, http.MethodPost}, true)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var (
		timing                            = httputil.NewTiming()
		clientID, clientSecret, basicAuth = r.BasicAuth()
		grantType                         = strings.ToLower(strings.TrimSpace(r.PostFormValue("grant_type")))
		code                              = strings.TrimSpace(r.PostFormValue("code"))
		refreshToken                      = strings.TrimSpace(r.PostFormValue("refresh_token"))
		codeVerifier                      = strings.TrimSpace(r.PostFormValue("code_verifier"))
		accessToken                       string
		idToken                           string
	)
	// when not using basic auth load client_id and client_secret parameters
	if !basicAuth {
		clientID = strings.TrimSpace(r.PostFormValue("client_id"))
		clientSecret = r.PostFormValue("client_secret")
	}

	// debug output of parameters
	log.Printf("grant_type=%q client_id=%q client_secret=%q code=%q code_verifier=%q refresh_token=%q",
		grantType, clientID, strings.Repeat("*", utf8.RuneCountInString(clientSecret)), code, codeVerifier, refreshToken)

	if client, clientExists := t.clients[strings.ToLower(clientID)]; clientExists {
		if grantType == GrantTypeClientCredentials && client.Secret == "" {
			Error(w, ErrorUnauthorizedClient, "client has no secret", http.StatusBadRequest)
			return
		}
		if clientSecret != "" || grantType == "client_credentials" {
			timing.Start("secret")
			if strings.HasPrefix(client.Secret, "$2") {
				if err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret)); err != nil {
					Error(w, ErrorInvalidClient, "client authentication failed: "+err.Error(), http.StatusUnauthorized)
					return
				}
			} else {
				if clientSecret != client.Secret {
					Error(w, ErrorInvalidClient, "client authentication failed", http.StatusUnauthorized)
					return
				}
			}
			timing.Stop("secret")
		}
	} else {
		Error(w, ErrorInvalidClient, "client not found", http.StatusUnauthorized)
		return
	}

	switch grantType {
	case GrantTypePassword:
		var (
			userID   = strings.TrimSpace(r.PostFormValue("username"))
			password = strings.TrimSpace(r.PostFormValue("password"))
			scope    = strings.TrimSpace(r.PostFormValue("scope"))
		)

		// debug output of parameters
		log.Printf("username=%q password=%q scope=%q", userID, strings.Repeat("*", utf8.RuneCountInString(password)), scope)

		if stringutil.IsAnyEmpty(clientID, userID, password) {
			Error(w, ErrorInvalidRequest, "client_id, username and password parameters are required", http.StatusBadRequest)
			return
		}

		var err error
		userID, err = t.peopleStore.Authenticate(userID, password)
		if err != nil {
			Error(w, ErrorInvalidGrant, "invalid username and password combination", http.StatusBadRequest)
			return
		}

		timing.Start("store")
		var person *people.Person
		person, err = t.peopleStore.Lookup(userID)
		if err != nil {
			Error(w, ErrorInternal, "person not found", http.StatusInternalServerError)
			return
		}
		timing.Stop("store")
		var user = User{Person: *person, UserID: userID}
		timing.Start("jwtgen")
		accessToken, _ = t.tokenService.GenerateAccessToken(user, userID, clientID, IntersectScope(t.scope, scope))
		timing.Stop("jwtgen")
	case GrantTypeAuthorizationCode:
		var userID, scope, challenge, nonce, valid = t.tokenService.VerifyAuthCode(code)
		if !valid {
			Error(w, ErrorInvalidGrant, "invalid auth code", http.StatusBadRequest)
			return
		}

		// verify parameters and pkce
		if challenge == "" {
			if stringutil.IsAnyEmpty(clientID, code) {
				Error(w, ErrorInvalidRequest, "client_id and code parameters are required", http.StatusBadRequest)
				return
			}
		} else {
			if stringutil.IsAnyEmpty(clientID, code, codeVerifier) {
				Error(w, ErrorInvalidRequest, "client_id, code and code_verifier parameters are required", http.StatusBadRequest)
				return
			}

			if !pkce.Verify(challenge, codeVerifier) {
				Error(w, ErrorInvalidGrant, "invalid challenge", http.StatusBadRequest)
				return
			}
		}

		timing.Start("store")
		var person, err = t.peopleStore.Lookup(userID)
		if err != nil {
			Error(w, ErrorInternal, "person not found", http.StatusInternalServerError)
			return
		}
		timing.Stop("store")
		var user = User{Person: *person, UserID: userID}
		timing.Start("jwtgen")
		accessToken, _ = t.tokenService.GenerateAccessToken(user, userID, clientID, scope)
		if strings.Contains(scope, "offline_access") {
			refreshToken, _ = t.tokenService.GenerateRefreshToken(userID, clientID, scope, nonce)
		}
		if strings.Contains(scope, "openid") {
			var hash = sha256.Sum256([]byte(accessToken))
			idToken, _ = t.tokenService.GenerateIDToken(user, clientID, scope, base64.RawURLEncoding.EncodeToString(hash[:16]), nonce)
		}
		timing.Stop("jwtgen")
	case GrantTypeRefreshToken:
		if stringutil.IsAnyEmpty(clientID, refreshToken) {
			Error(w, ErrorInvalidRequest, "client_id and refresh_token parameters are required", http.StatusBadRequest)
			return
		}
		var userID, scope, nonce, valid = t.tokenService.VerifyRefreshToken(refreshToken)
		if !valid {
			Error(w, ErrorInvalidGrant, "invalid refresh_token", http.StatusBadRequest)
			return
		}
		timing.Start("store")
		var person, err = t.peopleStore.Lookup(userID)
		if err != nil {
			Error(w, ErrorInternal, "person not found", http.StatusInternalServerError)
			return
		}
		timing.Stop("store")
		var user = User{Person: *person, UserID: userID}
		timing.Start("jwtgen")
		accessToken, _ = t.tokenService.GenerateAccessToken(user, userID, clientID, scope)
		if t.refreshTokenRotation && strings.Contains(scope, "offline_access") {
			refreshToken, _ = t.tokenService.GenerateRefreshToken(userID, clientID, scope, nonce)
		} else {
			refreshToken = ""
		}
		if strings.Contains(scope, "openid") {
			var hash = sha256.Sum256([]byte(accessToken))
			idToken, _ = t.tokenService.GenerateIDToken(user, clientID, scope, base64.RawURLEncoding.EncodeToString(hash[:16]), nonce)
		}
		timing.Stop("jwtgen")
	case GrantTypeClientCredentials:
		var scope = strings.TrimSpace(r.PostFormValue("scope"))
		log.Printf("scope=%q", scope)
		timing.Start("jwtgen")
		accessToken, _ = t.tokenService.GenerateAccessToken(User{}, clientID, clientID, IntersectScope(t.scope, scope))
		timing.Stop("jwtgen")
	default:
		Error(w, ErrorUnsupportedGrantType, "only grant types 'authorization_code', 'client_credentials', 'password' and 'refresh_token' are supported", http.StatusBadRequest)
		return
	}

	var bytes, err = json.Marshal(TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    t.tokenService.AccessTokenTTL(),
		RefreshToken: refreshToken,
		IDToken:      idToken,
	})
	if err != nil {
		Error(w, ErrorInternal, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	timing.Report(w)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(bytes))) // no "Transfer-Encoding: chunked" please
	w.Write(bytes)
}

func TokenHandler(tokenService TokenCreator, peopleStore people.Store, clients Clients, scope string, refreshTokenRotation bool) http.Handler {
	return &tokenHandler{
		tokenService:         tokenService,
		peopleStore:          peopleStore,
		clients:              clients,
		scope:                scope,
		refreshTokenRotation: refreshTokenRotation,
	}
}
