package keyset

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/go-jose/go-jose/v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Provider interface {
	Get() (map[string]any, error)
}

type provider struct {
	basePath         string
	keys             []string
	mu               sync.RWMutex
	cacheDuration    time.Duration
	cachedPublicKeys map[string]any
	cachedAt         time.Time
}

func NewProvider(basePath string, keys []string, cacheDuration time.Duration) Provider {
	return &provider{basePath: basePath, keys: keys, cacheDuration: cacheDuration}
}

func (p *provider) Get() (map[string]any, error) {
	if p.cachedPublicKeys != nil && time.Now().Sub(p.cachedAt) < p.cacheDuration {
		log.Print("Getting public keys from cache")
		p.mu.RLock()
		defer p.mu.RUnlock()
		return p.cachedPublicKeys, nil
	} else {
		log.Print("Retrieving public keys")
		p.mu.Lock()
		defer p.mu.Unlock()
		var publicKeys = make(map[string]any)

		for index, rawKey := range p.keys {
			var (
				block *pem.Block
				kid   string
				key   = strings.TrimSpace(rawKey)
			)
			if strings.HasPrefix(key, "-----BEGIN ") {
				block, _ = pem.Decode([]byte(key))
				if len(block.Headers) > 0 {
					kid = block.Headers[HeaderKeyID]
				}
				if kid == "" {
					kid = fmt.Sprintf("key%d", index+1)
				}
			} else if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
				log.Printf("GET %s", key)
				if resp, err := http.Get(key); err == nil && resp.StatusCode == http.StatusOK {
					var jwksBytes []byte
					jwksBytes, err = io.ReadAll(resp.Body)
					if err != nil {
						return nil, err
					}
					var jwks []jose.JSONWebKey
					jwks, err = UnmarshalJWKS(jwksBytes)
					if err != nil {
						return nil, err
					}
					jwksKeys := ToPublicKeys(jwks)
					for jwkid, jwkey := range jwksKeys {
						publicKeys[jwkid] = jwkey
					}
					continue
				} else {
					if err != nil {
						return nil, err
					} else {
						return nil, fmt.Errorf("%s", resp.Status)
					}
				}
			} else {
				var filename string
				if strings.HasPrefix(key, "@") {
					filename = filepath.Join(p.basePath, key[1:])
				} else {
					filename = filepath.Join(p.basePath, key)
				}
				bytes, err := os.ReadFile(filename)
				if err != nil {
					return nil, err
				}

				if strings.HasSuffix(strings.ToLower(filename), ".json") {
					var jwks []jose.JSONWebKey
					jwks, err = UnmarshalJWKS(bytes)
					if err != nil {
						return nil, err
					}
					jwksKeys := ToPublicKeys(jwks)
					for jwkid, jwkey := range jwksKeys {
						publicKeys[jwkid] = jwkey
					}
					continue
				}

				block, _ = pem.Decode(bytes)
				kid = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			}

			var publicKey any

			if kidHeaderValue, hasKidHeader := block.Headers[HeaderKeyID]; hasKidHeader && kidHeaderValue != "" {
				kid = kidHeaderValue
			}

			switch strings.TrimSpace(strings.ToLower(block.Type)) {
			case "rsa private key":
				rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
				if err != nil {
					return nil, err
				}
				publicKey = &rsaPrivateKey.PublicKey
			case "ec private key":
				ecPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
				if err != nil {
					return nil, err
				}
				publicKey = &ecPrivateKey.PublicKey
			case "rsa public key":
				var err error
				publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
				if err != nil {
					return nil, err
				}
			case "public key":
				var err error
				publicKey, err = x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("unsupported key type: " + block.Type)
			}

			publicKeys[kid] = publicKey
		}

		p.cachedPublicKeys = publicKeys
		p.cachedAt = time.Now()

		return publicKeys, nil
	}
}
