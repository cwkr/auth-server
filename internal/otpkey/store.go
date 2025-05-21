package otpkey

import (
	"errors"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"image/png"
	"io"
	"log"
	"time"
)

var ErrNotFound = errors.New("no otp key found")

type OTPKey struct {
	key *otp.Key
}

func (k OTPKey) Verify(code string) bool {
	if valid, err := totp.ValidateCustom(code, k.key.Secret(), time.Now(), totp.ValidateOpts{
		Period:    uint(k.key.Period()),
		Digits:    k.key.Digits(),
		Algorithm: k.key.Algorithm(),
	}); err != nil {
		log.Printf("!!! %s", err)
		return false
	} else {
		return valid
	}
}

func (k OTPKey) PNG(w io.Writer) error {
	var img, err = k.key.Image(512, 512)
	if err != nil {
		return err
	} else {
		if err := png.Encode(w, img); err != nil {
			return err
		}
	}
	return nil
}

type Store interface {
	Lookup(userID string) (*OTPKey, error)
	Ping() error
}
