package oauth2

import (
	"encoding/json"
	"fmt"
	"github.com/cwkr/auth-server/internal/httputil"
	"github.com/cwkr/auth-server/internal/people"
	"log"
	"net/http"
)

type userInfoHandler struct {
	peopleStore   people.Store
	tokenVerifier TokenVerifier
	extraClaims   map[string]string
	sessionName   string
}

func (u *userInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	httputil.AllowCORS(w, r, []string{http.MethodGet, http.MethodOptions}, true)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var accessToken = httputil.ExtractAccessToken(r)
	if accessToken == "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\"", u.sessionName))
		Error(w, "unauthorized", "authentication required", http.StatusUnauthorized)
		return
	}

	var userID, authError = u.tokenVerifier.VerifyAccessToken(accessToken)
	if authError != nil {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\", error=\"invalid_token\", error_description=\"%s\"", u.sessionName, authError.Error()))
		Error(w, "invalid_token", authError.Error(), http.StatusUnauthorized)
		return
	}

	if person, err := u.peopleStore.Lookup(userID); err == nil {

		var user = User{*person, userID}

		var claims = map[string]any{
			ClaimSubject: userID,
		}

		AddProfileClaims(claims, user)
		AddEmailClaims(claims, user)
		AddPhoneClaims(claims, user)
		AddAddressClaims(claims, user)
		AddExtraClaims(claims, u.extraClaims, user)

		var bytes, err = json.Marshal(claims)
		if err != nil {
			Error(w, ErrorInternal, err.Error(), http.StatusInternalServerError)
			return
		}

		httputil.NoCache(w)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	} else {
		Error(w, ErrorInternal, err.Error(), http.StatusInternalServerError)
	}
}

func UserInfoHandler(peopleStore people.Store, tokenVerifier TokenVerifier, extraClaims map[string]string, sessionName string) http.Handler {
	return &userInfoHandler{
		peopleStore:   peopleStore,
		tokenVerifier: tokenVerifier,
		extraClaims:   extraClaims,
		sessionName:   sessionName,
	}
}
