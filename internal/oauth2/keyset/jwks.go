package keyset

import (
	"encoding/json"
	"github.com/go-jose/go-jose/v3"
)

func ToPublicKeys(jwks []jose.JSONWebKey) map[string]any {
	var publicKeys = make(map[string]any, len(jwks))
	for _, jwk := range jwks {
		publicKeys[jwk.KeyID] = jwk.Key
	}
	return publicKeys
}

func UnmarshalJWKS(bytes []byte) ([]jose.JSONWebKey, error) {
	var rawJwks map[string][]map[string]any

	if err := json.Unmarshal(bytes, &rawJwks); err != nil {
		return nil, err
	}

	var jwks []jose.JSONWebKey

	for _, rawJwk := range rawJwks["keys"] {
		var jwkBytes, _ = json.Marshal(rawJwk)
		var jwk jose.JSONWebKey
		if err := jwk.UnmarshalJSON(jwkBytes); err != nil {
			return nil, err
		}
		jwks = append(jwks, jwk)
	}
	return jwks, nil
}
