package services

import "github.com/nopaalh/Relio/backend/models"

type HealthService struct{}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) Status() models.HealthResponse {
	return models.HealthResponse{
		Status:  "ok",
		Service: "relio-api",
	}
}
