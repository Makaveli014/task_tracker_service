package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/service"
)

type TeamHandler struct {
	teamSvc *service.TeamService
}

func NewTeamHandler(teamSvc *service.TeamService) *TeamHandler {
	return &TeamHandler{teamSvc: teamSvc}
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req models.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	team, err := h.teamSvc.Create(r.Context(), &req, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, team)
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	teams, err := h.teamSvc.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if teams == nil {
		teams = []models.Team{}
	}

	writeJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	teamID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}

	var req models.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == 0 {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if err := h.teamSvc.Invite(r.Context(), teamID, userID, &req); err != nil {
		switch {
		case errors.Is(err, service.ErrNotAuthorized):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrTeamNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "invited"})
}
