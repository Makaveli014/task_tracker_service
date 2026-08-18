package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/service"
)

type TaskHandler struct {
	taskSvc   *service.TaskService
	redis     *redis.Client
	cacheTTL  time.Duration
}

func NewTaskHandler(taskSvc *service.TaskService, redisClient *redis.Client) *TaskHandler {
	return &TaskHandler{
		taskSvc:  taskSvc,
		redis:    redisClient,
		cacheTTL: 5 * time.Minute,
	}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" || req.TeamID == 0 {
		writeError(w, http.StatusBadRequest, "title and team_id are required")
		return
	}

	task, err := h.taskSvc.Create(r.Context(), &req, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotAuthorized) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.invalidateCache(r.Context(), req.TeamID)
	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	teamIDStr := r.URL.Query().Get("team_id")
	if teamIDStr == "" {
		writeError(w, http.StatusBadRequest, "team_id is required")
		return
	}
	teamID, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team_id")
		return
	}

	status := r.URL.Query().Get("status")
	var assigneeID *uint64
	if a := r.URL.Query().Get("assignee_id"); a != "" {
		if v, err := strconv.ParseUint(a, 10, 64); err == nil {
			assigneeID = &v
		}
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	// Try cache first (only for default filters)
	if status == "" && assigneeID == nil && page == 1 && perPage == 20 {
		cacheKey := fmt.Sprintf("tasks:team:%d", teamID)
		if cached, err := h.redis.Get(r.Context(), cacheKey).Result(); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(cached))
			return
		}
	}

	tasks, total, err := h.taskSvc.List(r.Context(), teamID, status, assigneeID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}

	resp := models.PaginatedResponse{
		Data:       tasks,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}

	// Cache the response for default filters
	if status == "" && assigneeID == nil && page == 1 && perPage == 20 {
		cacheKey := fmt.Sprintf("tasks:team:%d", teamID)
		if data, err := json.Marshal(resp); err == nil {
			h.redis.Set(r.Context(), cacheKey, data, h.cacheTTL)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, resp)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.taskSvc.Update(r.Context(), id, &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotAuthorized):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	teamID, _ := h.taskSvc.GetTeamID(r.Context(), id)
	h.invalidateCache(r.Context(), teamID)
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	history, err := h.taskSvc.GetHistory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if history == nil {
		history = []models.TaskHistory{}
	}

	writeJSON(w, http.StatusOK, history)
}

func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var req models.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	comment, err := h.taskSvc.AddComment(r.Context(), id, userID, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

func (h *TaskHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	comments, err := h.taskSvc.GetComments(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if comments == nil {
		comments = []models.TaskComment{}
	}

	writeJSON(w, http.StatusOK, comments)
}

func (h *TaskHandler) invalidateCache(ctx context.Context, teamID uint64) {
	if teamID == 0 {
		return
	}
	cacheKey := fmt.Sprintf("tasks:team:%d", teamID)
	h.redis.Del(ctx, cacheKey)
}
