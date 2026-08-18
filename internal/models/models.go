package models

import "time"

// --- Domain models ---

type User struct {
	ID           uint64    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Team struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   uint64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type TeamMember struct {
	ID       uint64    `json:"id"`
	TeamID   uint64    `json:"team_id"`
	UserID   uint64    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type Task struct {
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	AssigneeID  *uint64   `json:"assignee_id"`
	TeamID      uint64    `json:"team_id"`
	CreatedBy   uint64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TaskHistory struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	ChangedBy uint64    `json:"changed_by"`
	FieldName string    `json:"field_name"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskComment struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	UserID    uint64    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Request DTOs ---

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InviteRequest struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"`
}

type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	AssigneeID  *uint64 `json:"assignee_id"`
	TeamID      uint64  `json:"team_id"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *uint64 `json:"assignee_id"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

// --- Response DTOs ---

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PaginatedResponse struct {
	Data       any  `json:"data"`
	Total      int  `json:"total"`
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalPages int  `json:"total_pages"`
}

type TeamStats struct {
	TeamName      string `json:"team_name"`
	MemberCount   int    `json:"member_count"`
	DoneTasksWeek int    `json:"done_tasks_week"`
}

type TopCreator struct {
	TeamName  string `json:"team_name"`
	Username  string `json:"username"`
	TaskCount int    `json:"task_count"`
}

type IntegrityViolation struct {
	TaskID       uint64 `json:"task_id"`
	Title        string `json:"title"`
	AssigneeID   uint64 `json:"assignee_id"`
	TeamID       uint64 `json:"team_id"`
	AssigneeName string `json:"assignee_name"`
}
