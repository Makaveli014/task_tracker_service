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
	return &TaskService{
		taskRepo: taskRepo,
		teamRepo: teamRepo,
	}
}

func (s *TaskService) Create(ctx context.Context, req *models.CreateTaskRequest, userID uint64) (*models.Task, error) {
	isMember, err := s.teamRepo.IsMember(ctx, req.TeamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotAuthorized
	}

	task, err := s.taskRepo.Create(ctx, req, userID)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

func (s *TaskService) GetByID(ctx context.Context, id uint64) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (s *TaskService) Update(ctx context.Context, id uint64, req *models.UpdateTaskRequest, userID uint64) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	// Проверка прав: создатель задачи или admin/owner команды
	isMember, err := s.teamRepo.IsMember(ctx, task.TeamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotAuthorized
	}

	role, _ := s.teamRepo.GetMemberRole(ctx, task.TeamID, userID)
	if task.CreatedBy != userID && role != "admin" && role != "owner" {
		return nil, ErrNotAuthorized
	}

	updates := make(map[string]any)
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.AssigneeID != nil {
		updates["assignee_id"] = *req.AssigneeID
	}

	// Записать историю изменений
	for field, newVal := range updates {
		oldVal := getFieldValue(task, field)
		newValStr := fmt.Sprintf("%v", newVal)
		if oldVal != newValStr {
			_ = s.taskRepo.AddHistory(ctx, id, userID, field, oldVal, newValStr)
		}
	}

	updated, err := s.taskRepo.Update(ctx, id, updates)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return updated, nil
}

func (s *TaskService) List(ctx context.Context, teamID uint64, status string, assigneeID *uint64, page, perPage int) ([]models.Task, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.taskRepo.List(ctx, teamID, status, assigneeID, perPage, offset)
}

func (s *TaskService) GetHistory(ctx context.Context, taskID uint64) ([]models.TaskHistory, error) {
	return s.taskRepo.GetHistory(ctx, taskID)
}

func (s *TaskService) AddComment(ctx context.Context, taskID, userID uint64, content string) (*models.TaskComment, error) {
	return s.taskRepo.AddComment(ctx, taskID, userID, content)
}

func (s *TaskService) GetComments(ctx context.Context, taskID uint64) ([]models.TaskComment, error) {
	return s.taskRepo.GetComments(ctx, taskID)
}

func (s *TaskService) GetTeamID(ctx context.Context, taskID uint64) (uint64, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, errors.New("task not found")
	}
	return task.TeamID, nil
}

func getFieldValue(task *models.Task, field string) string {
	switch field {
	case "title":
		return task.Title
	case "description":
		return task.Description
	case "status":
		return task.Status
	case "priority":
		return task.Priority
	case "assignee_id":
		if task.AssigneeID != nil {
			return fmt.Sprintf("%d", *task.AssigneeID)
		}
		return ""
	}
	return ""
}

