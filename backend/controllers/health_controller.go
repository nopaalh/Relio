package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/nopaalh/Relio/backend/models"
)

type HealthStatusProvider interface {
	Status() models.HealthResponse
}

type HealthController struct {
	health HealthStatusProvider
}

func NewHealthController(health HealthStatusProvider) *HealthController {
	return &HealthController{health: health}
}

func (c *HealthController) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(c.health.Status()); err != nil {
		return
	}
}
