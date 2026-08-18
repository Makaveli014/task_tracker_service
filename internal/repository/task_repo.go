package repository

import (
	"context"
	"database/sql"
	"strings"

	"TestTask_Bazis/internal/models"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
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

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (title, description, status, priority, assignee_id, team_id, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Description, status, priority, req.AssigneeID, req.TeamID, createdBy,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, uint64(id))
}

func (r *TaskRepository) GetByID(ctx context.Context, id uint64) (*models.Task, error) {
	t := &models.Task{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at
		FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.AssigneeID, &t.TeamID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TaskRepository) Update(ctx context.Context, id uint64, updates map[string]any) (*models.Task, error) {
	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+1)
	for field, val := range updates {
		setClauses = append(setClauses, field+" = ?")
		args = append(args, val)
	}
	args = append(args, id)

	query := "UPDATE tasks SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *TaskRepository) List(ctx context.Context, teamID uint64, status string, assigneeID *uint64, limit, offset int) ([]models.Task, int, error) {
	where := "WHERE team_id = ?"
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
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at
		FROM tasks ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.AssigneeID, &t.TeamID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

func (r *TaskRepository) AddHistory(ctx context.Context, taskID, changedBy uint64, fieldName, oldValue, newValue string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO task_history (task_id, changed_by, field_name, old_value, new_value)
		VALUES (?, ?, ?, ?, ?)`,
		taskID, changedBy, fieldName, oldValue, newValue,
	)
	return err
}

func (r *TaskRepository) GetHistory(ctx context.Context, taskID uint64) ([]models.TaskHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, changed_by, field_name, old_value, new_value, created_at
		FROM task_history WHERE task_id = ? ORDER BY created_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.TaskHistory
	for rows.Next() {
		var h models.TaskHistory
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &h.FieldName, &h.OldValue, &h.NewValue, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (r *TaskRepository) AddComment(ctx context.Context, taskID, userID uint64, content string) (*models.TaskComment, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)",
		taskID, userID, content,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	c := &models.TaskComment{}
	err = r.db.QueryRowContext(ctx, `
		SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE id = ?`, id,
	).Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt)
	return c, err
}

func (r *TaskRepository) GetComments(ctx context.Context, taskID uint64) ([]models.TaskComment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments WHERE task_id = ? ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.TaskComment
	for rows.Next() {
		var c models.TaskComment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}
