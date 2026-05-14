package web

import (
	"encoding/json"
	"net/http"

	"github.com/a-h/templ"
)

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	templ.Handler(
		component,
		templ.WithStatus(status),
		templ.WithContentType("text/html; charset=utf-8"),
	).ServeHTTP(w, r)
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("internal error", "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (s *Server) writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
