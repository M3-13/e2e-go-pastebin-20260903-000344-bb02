package main

import (
	"net/http"
	"os"
	"strings"

	"pastebin/handler"
	"pastebin/store"
)

// appConfig carries the runtime configuration read from the environment.
type appConfig struct {
	certFile    string
	keyFile     string
	corsOrigins []string
}

// loadConfig reads the process configuration from the environment.
func loadConfig() appConfig {
	return appConfig{
		certFile:    os.Getenv("CERT_FILE"),
		keyFile:     os.Getenv("KEY_FILE"),
		corsOrigins: parseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}
}

// parseOrigins splits a comma-separated origin allowlist into a trimmed,
// non-empty list. An empty input yields nil (deny by default).
func parseOrigins(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	cfg := loadConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if cfg.certFile != "" && cfg.keyFile != "" {
		http.ListenAndServeTLS(":"+port, cfg.certFile, cfg.keyFile, newApp(cfg))
		return
	}
	http.ListenAndServe(":"+port, newApp(cfg))
}

func newApp(cfg appConfig) http.Handler {
	s := store.New()
	h := handler.NewHandler(s)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/pastes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreatePaste(w, r)
		case http.MethodGet:
			h.ListPastes(w, r)
		default:
			methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
		}
	})

	mux.HandleFunc("/pastes/{id}", func(w http.ResponseWriter, r *http.Request) {
		r = handler.WithPasteID(r, r.PathValue("id"))
		switch r.Method {
		case http.MethodGet:
			h.GetPaste(w, r)
		case http.MethodDelete:
			h.DeletePaste(w, r)
		default:
			methodNotAllowed(w, http.MethodGet+", "+http.MethodDelete)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteError(w, http.StatusNotFound, "not found")
	})

	var app http.Handler = mux
	app = corsMiddleware(app, cfg.corsOrigins)
	app = hstsMiddleware(app, cfg.certFile != "" && cfg.keyFile != "")

	return app
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// corsMiddleware reflects the request Origin as Access-Control-Allow-Origin only
// when it is present in the allowlist. An empty allowlist denies every origin by
// not emitting the header at all. '*' is never emitted.
func corsMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

// hstsMiddleware emits the Strict-Transport-Security header only when the server
// is serving over TLS.
func hstsMiddleware(next http.Handler, tlsEnabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tlsEnabled {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func contains(origins []string, target string) bool {
	for _, o := range origins {
		if o == target {
			return true
		}
	}
	return false
}
