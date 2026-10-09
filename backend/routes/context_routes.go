package routes

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/nopaalh/Relio/backend/controllers"
	"github.com/nopaalh/Relio/backend/models"
)

// NewContextHandler opts into the canonical context API; NewHandler stays legacy.
func NewContextHandler(health *controllers.HealthController, context *controllers.ContextController) http.Handler {
	mux := http.NewServeMux()
	liveness := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			contextRouteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "request method is not supported")
			return
		}
		health.Get(w, r)
	}
	mux.HandleFunc("/healthz", liveness)
	mux.HandleFunc("/api/health", liveness)
	mux.HandleFunc("/api/deals", context.List)
	mux.HandleFunc("/api/deals/{deal_id}", context.Get)
	mux.HandleFunc("/api/deals/{deal_id}/graph", context.Graph)
	mux.HandleFunc("/api/deals/{deal_id}/timeline", context.Timeline)
	mux.HandleFunc("/api/evidence/{evidence_id}", context.Evidence)
	mux.HandleFunc("/api/deals/{deal_id}/assessment", context.Assessment)
	mux.HandleFunc("/api/deals/{deal_id}/action-candidates", context.ActionCandidates)
	mux.HandleFunc("/api/deals/{deal_id}/actions/compare", context.Compare)
	mux.HandleFunc("/api/copilot/ask", context.Ask)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contextRouteError(w, http.StatusNotFound, "not_found", "requested endpoint was not found")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		// Reject noncanonical paths rather than ServeMux's plaintext redirects.
		if r.URL.Path != path.Clean(r.URL.Path) {
			contextRouteError(w, http.StatusNotFound, "not_found", "requested endpoint was not found")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func contextRouteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Code: code, Message: message})
}
