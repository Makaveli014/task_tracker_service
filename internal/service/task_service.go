package service

import (
	"context"
	"errors"
	"fmt"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
)

type TaskService struct {
	taskRepo *repository.TaskRepository
	teamRepo *repository.TeamRepository
}

func NewTaskService(taskRepo *repository.TaskRepository, teamRepo *repository.TeamRepository) *TaskService {
	return &TaskService{taskRepo: taskRepo, teamRepo: teamRepo}
}

func (s *TaskService) member(ctx context.Context, teamID, userID uint64) (string, error) {
	role, err := s.teamRepo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", ErrNotAuthorized
	}
	return role, nil
}
func (s *TaskService) CheckTeamAccess(ctx context.Context, teamID, userID uint64) error {
	_, err := s.member(ctx, teamID, userID)
	return err
}
func (s *TaskService) Create(ctx context.Context, req *models.CreateTaskRequest, userID uint64) (*models.Task, error) {
	if _, err := s.member(ctx, req.TeamID, userID); err != nil {
		return nil, err
	}
	if req.AssigneeID != nil {
		if _, err := s.member(ctx, req.TeamID, *req.AssigneeID); err != nil {
			return nil, errors.New("assignee must be a team member")
		}
	}
	return s.taskRepo.Create(ctx, req, userID)
}
func (s *TaskService) GetByID(ctx context.Context, id uint64) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (s *TaskService) Update(ctx context.Context, id, userID uint64, req *models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	role, err := s.member(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	isPrivileged := role == "owner" || role == "admin"
	isCreator := task.CreatedBy == userID
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == userID
	if !isPrivileged && !isCreator && !isAssignee {
		return nil, ErrNotAuthorized
	}
	updates := map[string]any{}
	changes := map[string]map[string]string{}
	add := func(field string, value any, old string) {
		newValue := fmt.Sprintf("%v", value)
		if old != newValue {
			updates[field] = value
			changes[field] = map[string]string{"old": old, "new": newValue}
		}
	}
	if req.Title != nil {
		if !isPrivileged && !isCreator {
			return nil, ErrNotAuthorized
		}
		add("title", *req.Title, task.Title)
	}
	if req.Description != nil {
		if !isPrivileged && !isCreator {
			return nil, ErrNotAuthorized
		}
		add("description", *req.Description, task.Description)
	}
	if req.Priority != nil {
		if !isPrivileged {
			return nil, ErrNotAuthorized
		}
		add("priority", *req.Priority, task.Priority)
	}
	if req.AssigneeID != nil {
		if !isPrivileged && !isCreator {
			return nil, ErrNotAuthorized
		}
		if _, err := s.member(ctx, task.TeamID, *req.AssigneeID); err != nil {
			return nil, errors.New("assignee must be a team member")
		}
		add("assignee_id", *req.AssigneeID, fmt.Sprintf("%v", task.AssigneeID))
	}
	if req.Status != nil {
		if !isPrivileged && !isCreator && !isAssignee {
			return nil, ErrNotAuthorized
		}
		add("status", *req.Status, task.Status)
	}
	if len(updates) == 0 {
		return task, nil
	}
	return s.taskRepo.UpdateWithHistory(ctx, id, userID, req.Version, updates, changes)
}
func (s *TaskService) List(ctx context.Context, teamID, userID uint64, status string, assigneeID *uint64, limit, offset int) ([]models.Task, int, error) {
	if _, err := s.member(ctx, teamID, userID); err != nil {
		return nil, 0, err
	}
	return s.taskRepo.List(ctx, teamID, status, assigneeID, limit, offset)
}
func (s *TaskService) GetHistory(ctx context.Context, taskID, userID uint64) ([]models.TaskHistory, error) {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err = s.member(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return s.taskRepo.GetHistory(ctx, taskID)
}
func (s *TaskService) AddComment(ctx context.Context, taskID, userID uint64, content string) (*models.TaskComment, error) {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err = s.member(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return s.taskRepo.AddComment(ctx, taskID, userID, content)
}
func (s *TaskService) GetComments(ctx context.Context, taskID, userID uint64) ([]models.TaskComment, error) {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err = s.member(ctx, task.TeamID, userID); err != nil {
		return nil, err
	}
	return s.taskRepo.GetComments(ctx, taskID)
}
func (s *TaskService) GetTeamID(ctx context.Context, taskID uint64) (uint64, error) {
	task, err := s.GetByID(ctx, taskID)
	if err != nil {
		return 0, err
	}
	return task.TeamID, nil
}
