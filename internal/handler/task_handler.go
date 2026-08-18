package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
	"TestTask_Bazis/internal/service"
)

type TaskHandler struct {
	taskSvc  *service.TaskService
	redis    *redis.Client
	cacheTTL time.Duration
}

func NewTaskHandler(taskSvc *service.TaskService, redisClient *redis.Client) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc, redis: redisClient, cacheTTL: 5 * time.Minute}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" || req.TeamID == 0 {
		writeError(w, http.StatusBadRequest, "title and team_id are required")
		return
	}
	task, err := h.taskSvc.Create(r.Context(), &req, getUserID(r))
	if err != nil {
		h.writeTaskError(w, err)
		return
	}
	h.invalidateCache(r.Context(), req.TeamID)
	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseUint(r.URL.Query().Get("team_id"), 10, 64)
	if err != nil || teamID == 0 {
		writeError(w, http.StatusBadRequest, "valid team_id is required")
		return
	}
	userID := getUserID(r)
	if err := h.taskSvc.CheckTeamAccess(r.Context(), teamID, userID); err != nil {
		h.writeTaskError(w, err)
		return
	}
	status := r.URL.Query().Get("status")
	var assigneeID *uint64
	if raw := r.URL.Query().Get("assignee_id"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid assignee_id")
			return
		}
		assigneeID = &value
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	assigneeKey := ""
	if assigneeID != nil {
		assigneeKey = strconv.FormatUint(*assigneeID, 10)
	}
	cacheKey := fmt.Sprintf("tasks:team:%d:status:%s:assignee:%s:limit:%d:offset:%d", teamID, status, assigneeKey, limit, offset)
	if cached, err := h.redis.Get(r.Context(), cacheKey).Result(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}
	tasks, total, err := h.taskSvc.List(r.Context(), teamID, userID, status, assigneeID, limit, offset)
	if err != nil {
		h.writeTaskError(w, err)
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	resp := models.PaginatedResponse{Data: tasks, Total: total, Limit: limit, Offset: offset}
	if data, err := json.Marshal(resp); err == nil {
		_ = h.redis.Set(r.Context(), cacheKey, data, h.cacheTTL).Err()
	}
	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, resp)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var req models.UpdateTaskRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Version == 0 {
		writeError(w, http.StatusBadRequest, "valid version is required")
		return
	}
	task, err := h.taskSvc.Update(r.Context(), id, getUserID(r), &req)
	if err != nil {
		h.writeTaskError(w, err)
		return
	}
	h.invalidateCache(r.Context(), task.TeamID)
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	history, err := h.taskSvc.GetHistory(r.Context(), id, getUserID(r))
	if err != nil {
		h.writeTaskError(w, err)
		return
	}
	if history == nil {
		history = []models.TaskHistory{}
	}
	writeJSON(w, http.StatusOK, history)
}
func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var req models.CreateCommentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	comment, err := h.taskSvc.AddComment(r.Context(), id, getUserID(r), req.Content)
	if err != nil {
		h.writeTaskError(w, err)
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
	comments, err := h.taskSvc.GetComments(r.Context(), id, getUserID(r))
	if err != nil {
		h.writeTaskError(w, err)
		return
	}
	if comments == nil {
		comments = []models.TaskComment{}
	}
	writeJSON(w, http.StatusOK, comments)
}
func (h *TaskHandler) writeTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotAuthorized):
		writeError(w, http.StatusForbidden, "not authorized")
	case errors.Is(err, repository.ErrVersionConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}
func (h *TaskHandler) invalidateCache(ctx context.Context, teamID uint64) {
	if teamID == 0 {
		return
	}
	keys, err := h.redis.Keys(ctx, fmt.Sprintf("tasks:team:%d:*", teamID)).Result()
	if err == nil && len(keys) > 0 {
		_ = h.redis.Del(ctx, keys...).Err()
	}
}
