package people

import (
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strings"
	"time"
)

type AuthenticPerson struct {
	Person
	PasswordHash string `json:"password_hash"`
}

type embeddedStore struct {
	sessionStore sessions.Store
	users        map[string]AuthenticPerson
	sessionTTL   int64
}

func NewEmbeddedStore(sessionStore sessions.Store, users map[string]AuthenticPerson, sessionTTL int64) Store {
	return &embeddedStore{
		sessionStore: sessionStore,
		users:        users,
		sessionTTL:   sessionTTL,
	}
}

func (e embeddedStore) Authenticate(userID, password string) (string, error) {
	var lowercaseUserID = strings.ToLower(userID)
	var authenticPerson, foundUser = e.users[strings.ToLower(lowercaseUserID)]

	if foundUser {
		if err := bcrypt.CompareHashAndPassword([]byte(authenticPerson.PasswordHash), []byte(password)); err != nil {
			log.Printf("!!! password comparison failed: %v", err)
		} else {
			return lowercaseUserID, nil
		}
	}

	return "", ErrAuthenticationFailed
}

func (e embeddedStore) IsSessionActive(r *http.Request, sessionName string) (string, bool, bool) {
	var session, _ = e.sessionStore.Get(r, sessionName)

	var (
		uid, sct, tfa, vfd = session.Values["uid"], session.Values["sct"], session.Values["2fa"], session.Values["vfd"]
		valid              = uid != nil && sct != nil && time.Unix(sct.(int64), 0).
			Add(time.Duration(e.sessionTTL)*time.Second).
			After(time.Now())
		verified = tfa != nil && vfd != nil && (tfa.(bool) == false || vfd.(bool))
	)

	if valid {
		return uid.(string), true, verified
	}

	return "", false, false
}

func (e embeddedStore) SaveSession(r *http.Request, w http.ResponseWriter, authTime time.Time, userID, sessionName string) error {
	var session, _ = e.sessionStore.Get(r, sessionName)
	session.Values["uid"] = userID
	session.Values["sct"] = authTime.Unix()
	session.Values["2fa"] = false
	session.Values["vfd"] = false
	if err := session.Save(r, w); err != nil {
		return err
	}
	return nil
}

func (e embeddedStore) Lookup(userID string) (*Person, error) {
	var authenticPerson, found = e.users[strings.ToLower(userID)]

	if found {
		return &authenticPerson.Person, nil
	}

	return nil, ErrPersonNotFound
}

func (e embeddedStore) Ping() error {
	return nil
}

func (e embeddedStore) ReadOnly() bool {
	return true
}

func (e embeddedStore) Put(userID string, person *Person) error {
	return ErrReadOnly
}

func (e embeddedStore) SetPassword(userID, password string) error {
	return ErrReadOnly
}
