package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const ServiceName = "assessment-plan"

func NewRouter(check func(context.Context) error) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", Live())
	router.Get("/ready", Ready(check))
	return router
}

func Live() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeOK(w)
	}
}

func Ready(check func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := check(ctx); err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
		}
		writeOK(w)
	}
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": ServiceName,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
