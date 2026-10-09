package routes

import (
	"net/http"

	"github.com/nopaalh/Relio/backend/controllers"
)

func NewHandler(health *controllers.HealthController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Get)
	return mux
}
