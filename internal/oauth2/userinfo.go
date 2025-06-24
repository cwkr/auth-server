package oauth2

import (
	"encoding/json"
	"github.com/cwkr/authd/internal/httputil"
	"github.com/cwkr/authd/internal/people"
	"log"
	"net/http"
)

type userInfoHandler struct {
	peopleStore people.Store
	extraClaims map[string]string
}

func (u *userInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	httputil.AllowCORS(w, r, []string{http.MethodGet, http.MethodOptions}, true)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var userID = r.Context().Value("user_id").(string)

	if person, err := u.peopleStore.Lookup(userID); err == nil {

		var user = User{*person, userID}

		var claims = map[string]any{
			ClaimSubject: userID,
		}

		AddProfileClaims(claims, user)
		AddEmailClaims(claims, user)
		AddPhoneClaims(claims, user)
		AddAddressClaims(claims, user)
		AddExtraClaims(claims, u.extraClaims, user, "")

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

func UserInfoHandler(peopleStore people.Store, extraClaims map[string]string) http.Handler {
	return &userInfoHandler{
		peopleStore: peopleStore,
		extraClaims: extraClaims,
	}
}
