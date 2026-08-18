package models

import "time"

type User struct {
	ID           uint64    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Username     string    `json:"-"`
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
	ID          uint64     `json:"id"`
	TeamID      uint64     `json:"team_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	CreatedBy   uint64     `json:"created_by"`
	AssigneeID  *uint64    `json:"assignee_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	Version     uint64     `json:"version"`
}

type TaskHistory struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	ChangedBy uint64    `json:"changed_by"`
	Changes   string    `json:"changes"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskComment struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	UserID    uint64    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
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

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type CreateTaskRequest struct {
	TeamID      uint64  `json:"team_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	AssigneeID  *uint64 `json:"assignee_id"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *uint64 `json:"assignee_id"`
	Version     uint64  `json:"version"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PaginatedResponse struct {
	Data   any `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type AssigneeStats struct {
	UserID      uint64 `json:"user_id"`
	Name        string `json:"name"`
	ClosedTasks int    `json:"closed_tasks"`
}

type TeamStats struct {
	TeamID                    uint64          `json:"team_id"`
	TasksByStatus             map[string]int  `json:"tasks_by_status"`
	TopAssignees              []AssigneeStats `json:"top_assignees"`
	AverageClosingTimeSeconds *float64        `json:"average_closing_time_seconds"`
	CommentCount              int             `json:"comment_count"`
}
