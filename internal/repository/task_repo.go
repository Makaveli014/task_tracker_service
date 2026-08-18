package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"TestTask_Bazis/internal/models"
)

type TaskRepository struct{ db *sql.DB }

func NewTaskRepository(db *sql.DB) *TaskRepository { return &TaskRepository{db: db} }

const taskSelect = `SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version, priority FROM tasks`

func scanTask(scanner interface{ Scan(...any) error }) (*models.Task, error) {
	t := &models.Task{}
	err := scanner.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.CreatedBy, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt, &t.Version, &t.Priority)
	return t, err
}

func (r *TaskRepository) Create(ctx context.Context, req *models.CreateTaskRequest, createdBy uint64) (*models.Task, error) {
	status := req.Status
	if status == "" {
		status = "todo"
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, priority) VALUES (?, ?, ?, ?, ?, ?, ?)`, req.TeamID, req.Title, req.Description, status, createdBy, req.AssigneeID, priority)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	changes, err := json.Marshal(map[string]any{"created": map[string]any{"new": map[string]any{"title": req.Title, "description": req.Description, "status": status, "assignee_id": req.AssigneeID}}})
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO task_history (task_id, changed_by, changes) VALUES (?, ?, ?)", id, createdBy, changes); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, uint64(id))
}

func (r *TaskRepository) GetByID(ctx context.Context, id uint64) (*models.Task, error) {
	row := r.db.QueryRowContext(ctx, taskSelect+" WHERE id = ?", id)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) List(ctx context.Context, teamID uint64, status string, assigneeID *uint64, limit, offset int) ([]models.Task, int, error) {
	where := " WHERE team_id = ?"
	args := []any{teamID}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if assigneeID != nil {
		where += " AND assignee_id = ?"
		args = append(args, *assigneeID)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, taskSelect+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, total, rows.Err()
}

func (r *TaskRepository) UpdateWithHistory(ctx context.Context, id, actorID, expectedVersion uint64, updates map[string]any, changes map[string]map[string]string) (*models.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	set := make([]string, 0, len(updates)+2)
	args := make([]any, 0, len(updates)+2)
	for field, value := range updates {
		set = append(set, field+" = ?")
		args = append(args, value)
	}
	if _, ok := updates["status"]; ok {
		set = append(set, "closed_at = CASE WHEN status = 'done' THEN COALESCE(closed_at, CURRENT_TIMESTAMP) ELSE NULL END")
	}
	set = append(set, "version = version + 1")
	args = append(args, id, expectedVersion)
	result, err := tx.ExecContext(ctx, "UPDATE tasks SET "+strings.Join(set, ", ")+" WHERE id = ? AND version = ?", args...)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrVersionConflict
	}

	payload, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO task_history (task_id, changed_by, changes) VALUES (?, ?, ?)", id, actorID, payload); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

var ErrVersionConflict = fmt.Errorf("task version conflict")

func (r *TaskRepository) GetHistory(ctx context.Context, taskID uint64) ([]models.TaskHistory, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, task_id, changed_by, changes, created_at FROM task_history WHERE task_id = ? ORDER BY created_at DESC", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.TaskHistory
	for rows.Next() {
		var h models.TaskHistory
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &h.Changes, &h.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

func (r *TaskRepository) AddComment(ctx context.Context, taskID, userID uint64, content string) (*models.TaskComment, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)", taskID, userID, content)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	c := &models.TaskComment{}
	err = r.db.QueryRowContext(ctx, "SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE id = ?", id).Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt)
	return c, err
}

func (r *TaskRepository) GetComments(ctx context.Context, taskID uint64) ([]models.TaskComment, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE task_id = ? ORDER BY created_at ASC", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.TaskComment
	for rows.Next() {
		var c models.TaskComment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
