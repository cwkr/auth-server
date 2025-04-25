package server

import (
	_ "embed"
	"fmt"
	"github.com/cwkr/authd/internal/htmlutil"
	"github.com/cwkr/authd/internal/httputil"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed scripts/main.js
	mainScriptContent string
	//go:embed scripts/user-details-tag.js
	userDetailsTagScriptStr      string
	userDetailsTagScriptTemplate = template.Must(template.New("user-details-tag.js").Parse(userDetailsTagScriptStr))
)

func MainScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Content-Length", fmt.Sprint(len(mainScriptContent)))
		httputil.Cache(w, 120*time.Hour)
		fmt.Fprint(w, mainScriptContent)
	})
}

func UserDetailsTagScriptHandler(issuer, basePath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL)
		w.Header().Set("Content-Type", "text/javascript")
		httputil.Cache(w, 120*time.Hour)
		var data = map[string]any{
			"API_URL": strings.TrimSuffix(issuer, "/") + "/api/v1/people/",
		}
		if err := userDetailsTagScriptTemplate.Execute(w, data); err != nil {
			htmlutil.Error(w, basePath, err.Error(), http.StatusInternalServerError)
		}
	})
}
