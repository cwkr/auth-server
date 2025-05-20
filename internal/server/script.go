package server

import (
	_ "embed"
	"fmt"
	"github.com/cwkr/auth-server/internal/httputil"
	"log"
	"net/http"
	"time"
)

var (
	//go:embed scripts/main.js
	mainScriptContent string
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
