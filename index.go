package main

import (
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
	"github.com/cwkr/auth-server/htmlutil"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

//go:embed templates/index.gohtml
var indexTpl string

func Index(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)
	var t, _ = template.New("index").Parse(indexTpl)

	var pubASN1 = x509.MarshalPKCS1PublicKey(&rsaPrivKey.PublicKey)

	var pubBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	var err = t.ExecuteTemplate(w, "index", map[string]interface{}{
		"issuer":     strings.TrimRight(cfg.Issuer, "/"),
		"public_key": string(pubBytes),
		"state":      fmt.Sprint(rand.Int()),
		"clients":    cfg.Clients,
	})
	if err != nil {
		htmlutil.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
