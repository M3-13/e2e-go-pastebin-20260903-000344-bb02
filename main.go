package main

import (
	"net/http"
	"os"

	"pastebin/handler"
	"pastebin/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, newApp())
}

func newApp() http.Handler {
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
	app = corsMiddleware(app)

	return app
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
