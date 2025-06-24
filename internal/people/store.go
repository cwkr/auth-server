package people

import (
	"net/http"
	"time"
)

type Store interface {
	Authenticate(userID, password string) (string, error)
	IsSessionActive(r *http.Request, sessionName string) (string, bool, bool)
	SaveSession(r *http.Request, w http.ResponseWriter, authTime time.Time, userID, sessionName string) error
	Lookup(userID string) (*Person, error)
	Ping() error
	ReadOnly() bool
	Put(userID string, person *Person) error
	SetPassword(userID, password string) error
}
