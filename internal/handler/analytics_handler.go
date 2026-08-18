package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"TestTask_Bazis/internal/repository"
)

type AnalyticsHandler struct {
	analyticsRepo *repository.AnalyticsRepository
	teamRepo      *repository.TeamRepository
}

func NewAnalyticsHandler(analyticsRepo *repository.AnalyticsRepository, teamRepo *repository.TeamRepository) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo, teamRepo: teamRepo}
}

func (h *AnalyticsHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseUint(chi.URLParam(r, "team_id"), 10, 64)
	if err != nil || teamID == 0 {
		writeError(w, http.StatusBadRequest, "invalid team_id")
		return
	}
	role, err := h.teamRepo.GetMemberRole(r.Context(), teamID, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if role != "owner" && role != "admin" {
		writeError(w, http.StatusForbidden, "analytics requires owner or admin role")
		return
	}
	stats, err := h.analyticsRepo.TeamStats(r.Context(), teamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
