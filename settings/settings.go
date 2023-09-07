package settings

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/cwkr/auth-server/internal/oauth2"
	"github.com/cwkr/auth-server/internal/oauth2/clients"
	"github.com/cwkr/auth-server/internal/oauth2/trl"
	"github.com/cwkr/auth-server/internal/people"
	"github.com/cwkr/auth-server/internal/stringutil"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	Issuer                  string                            `json:"issuer"`
	Port                    int                               `json:"port"`
	Title                   string                            `json:"title,omitempty"`
	Users                   map[string]people.AuthenticPerson `json:"users,omitempty"`
	Key                     string                            `json:"key"`
	AdditionalKeys          []string                          `json:"additional_keys,omitempty"`
	Clients                 map[string]clients.Client         `json:"clients,omitempty"`
	ClientStore             *clients.StoreSettings            `json:"client_store,omitempty"`
	ExtraScope              string                            `json:"extra_scope,omitempty"`
	AccessTokenExtraClaims  map[string]string                 `json:"access_token_extra_claims,omitempty"`
	AccessTokenTTL          int                               `json:"access_token_ttl"`
	RefreshTokenTTL         int                               `json:"refresh_token_ttl"`
	IDTokenTTL              int                               `json:"id_token_ttl"`
	IDTokenExtraClaims      map[string]string                 `json:"id_token_extra_claims,omitempty"`
	SessionSecret           string                            `json:"session_secret"`
	SessionName             string                            `json:"session_name"`
	SessionTTL              int                               `json:"session_ttl"`
	PeopleStore             *people.StoreSettings             `json:"people_store,omitempty"`
	DisableAPI              bool                              `json:"disable_api,omitempty"`
	PeopleAPICustomVersions map[string]map[string]string      `json:"people_api_custom_versions,omitempty"`
	PeopleAPIRequireAuthN   bool                              `json:"people_api_require_authn,omitempty"`
	LoginTemplate           string                            `json:"login_template,omitempty"`
	TRLStore                *trl.StoreSettings                `json:"trl_store,omitempty"`
	rsaSigningKey           *rsa.PrivateKey
	rsaSigningKeyID         string
	additionalPublicKeys    map[string]any
}

func NewDefault() *Server {
	return &Server{
		Issuer:          "http://localhost:6080/",
		Port:            6080,
		AccessTokenTTL:  3_600,
		RefreshTokenTTL: 28_800,
		IDTokenTTL:      28_800,
		SessionName:     "ASESSION",
		SessionSecret:   stringutil.RandomBytesString(32),
		SessionTTL:      28_800,
	}
}

func (s *Server) LoadKeys(basePath string, genNew bool) error {
	var err error
	s.rsaSigningKeyID = "sigkey"
	if strings.HasPrefix(s.Key, "-----BEGIN RSA PRIVATE KEY-----") {
		block, _ := pem.Decode([]byte(s.Key))
		s.rsaSigningKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
	} else if s.Key == "" || !strings.HasPrefix(s.Key, "@") {
		if !genNew && s.Key != "" {
			return errors.New("missing key")
		}
		var keyBytes []byte
		s.rsaSigningKey, keyBytes, err = oauth2.GeneratePrivateKey(2048)
		if err != nil {
			return err
		}

		if s.Key == "" {
			s.Key = string(keyBytes)
		} else {
			err := os.WriteFile(s.Key, keyBytes, 0600)
			if err != nil {
				return err
			}
		}
	} else {
		var filename = filepath.Join(basePath, s.Key[1:])
		pemBytes, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(pemBytes)
		s.rsaSigningKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		s.rsaSigningKeyID = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}

	s.additionalPublicKeys, err = oauth2.LoadPublicKeys(basePath, s.AdditionalKeys)
	return err
}

func (s Server) PrivateKey() *rsa.PrivateKey {
	return s.rsaSigningKey
}

func (s Server) PublicKey() *rsa.PublicKey {
	return &s.rsaSigningKey.PublicKey
}

func (s Server) KeyID() string {
	return s.rsaSigningKeyID
}

func (s Server) AllKeys() map[string]any {
	var allKeys = make(map[string]any)
	allKeys[s.rsaSigningKeyID] = s.PublicKey()
	for kid, publicKey := range s.additionalPublicKeys {
		allKeys[kid] = publicKey
	}
	return allKeys
}