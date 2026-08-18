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

type TeamHandler struct{ teamSvc *service.TeamService }

func NewTeamHandler(teamSvc *service.TeamService) *TeamHandler { return &TeamHandler{teamSvc: teamSvc} }

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTeamRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	team, err := h.teamSvc.Create(r.Context(), &req, getUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, team)
}
func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	teams, err := h.teamSvc.ListByUser(r.Context(), getUserID(r))
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
	teamID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	var req models.InviteRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.UserID == 0 {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if err := h.teamSvc.Invite(r.Context(), teamID, getUserID(r), &req); err != nil {
		switch {
		case errors.Is(err, service.ErrNotAuthorized):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrTeamNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrAlreadyMember):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "invited"})
}
func (h *TeamHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	userID, err := strconv.ParseUint(chi.URLParam(r, "user_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req models.UpdateMemberRoleRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.teamSvc.UpdateMemberRole(r.Context(), teamID, getUserID(r), userID, req.Role); err != nil {
		if errors.Is(err, service.ErrNotAuthorized) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
