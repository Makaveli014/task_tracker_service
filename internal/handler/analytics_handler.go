package handler

import (
	"net/http"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
)

type AnalyticsHandler struct {
	analyticsRepo *repository.AnalyticsRepository
}

func NewAnalyticsHandler(analyticsRepo *repository.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo}
}

func (h *AnalyticsHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.analyticsRepo.TeamStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if stats == nil {
		stats = []models.TeamStats{}
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *AnalyticsHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	creators, err := h.analyticsRepo.TopCreators(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if creators == nil {
		creators = []models.TopCreator{}
	}
	writeJSON(w, http.StatusOK, creators)
}

func (h *AnalyticsHandler) IntegrityCheck(w http.ResponseWriter, r *http.Request) {
	violations, err := h.analyticsRepo.IntegrityViolations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if violations == nil {
		violations = []models.IntegrityViolation{}
	}
	writeJSON(w, http.StatusOK, violations)
}
