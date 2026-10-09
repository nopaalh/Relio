package routes

import (
	"net/http"

	"github.com/nopaalh/Relio/backend/controllers"
)

func NewHandler(health *controllers.HealthController, deals *controllers.DealController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Get)
	mux.HandleFunc("/api/deals", deals.List)
	mux.HandleFunc("/api/deals/{deal_id}", deals.Get)
	return mux
}
