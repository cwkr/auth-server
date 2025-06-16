package otpauth

import (
	"github.com/cwkr/auth-server/internal/people"
	"github.com/pquerna/otp"
	"strings"
)

type embeddedStore struct {
	users map[string]people.AuthenticPerson
}

func NewEmbeddedStore(users map[string]people.AuthenticPerson) Store {
	return &embeddedStore{
		users: users,
	}
}

func (e embeddedStore) Lookup(userID string) (*KeyWrapper, error) {
	var authenticPerson, found = e.users[strings.ToLower(userID)]
	if !found || !strings.HasPrefix(authenticPerson.OTPKeyURI, PrefixOTPAuth) {
		return nil, ErrNotFound
	}
	if k, err := otp.NewKeyFromURL(authenticPerson.OTPKeyURI); err != nil {
		return nil, err
	} else {
		return &KeyWrapper{key: k}, nil
	}
}

func (e embeddedStore) Ping() error {
	return nil
}

func (e embeddedStore) ReadOnly() bool {
	return true
}
