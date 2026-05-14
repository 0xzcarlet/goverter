package web

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.repo.Ping(ctx); err != nil {
		s.writeJSONError(w, http.StatusServiceUnavailable, fmt.Sprintf("database not ready: %v", err))
		return
	}
	if err := s.converter.Check(); err != nil {
		s.writeJSONError(w, http.StatusServiceUnavailable, fmt.Sprintf("ebook-convert not ready: %v", err))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
