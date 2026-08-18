package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mysqlcontainer "github.com/testcontainers/testcontainers-go/modules/mysql"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"

	"TestTask_Bazis/internal/circuitbreaker"
	"TestTask_Bazis/internal/handler"
	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
	"TestTask_Bazis/internal/service"
)

type testEnv struct {
	db     *sql.DB
	rdb    *redis.Client
	server *httptest.Server
}

func setupIntegration(t *testing.T) *testEnv {
	ctx := context.Background()

	// MySQL container
	mysqlC, err := mysqlcontainer.Run(ctx,
		"mysql:8.0",
		mysqlcontainer.WithDatabase("taskmanager"),
		mysqlcontainer.WithUsername("root"),
		mysqlcontainer.WithPassword("password"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { mysqlC.Terminate(ctx) })

	dsn, err := mysqlC.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4")
	require.NoError(t, err)

	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)

	// Wait for MySQL to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, db.Ping())

	// Run migrations manually (split multi-statement SQL)
	migrationSQL, err := os.ReadFile(filepath.Join(migrationsDir(t), "001_init.sql"))
	require.NoError(t, err)
	statements := splitStatements(string(migrationSQL))
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err = db.Exec(stmt)
		require.NoError(t, err, "Failed to run migration: %s", truncate(stmt, 80))
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Redis container
	redisC, err := rediscontainer.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { redisC.Terminate(ctx) })

	redisAddr, err := redisC.ConnectionString(ctx)
	require.NoError(t, err)

	// Remove redis:// prefix if present
	redisAddr = strings.TrimPrefix(redisAddr, "redis://")

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	require.NoError(t, rdb.Ping(ctx).Err())

	// Build app
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)

	breaker := circuitbreaker.New(5, 30*time.Second)
	emailSvc := service.NewEmailService(breaker)
	authSvc := service.NewAuthService(userRepo, "test-jwt-secret", 24)
	teamSvc := service.NewTeamService(teamRepo, userRepo, emailSvc)
	taskSvc := service.NewTaskService(taskRepo, teamRepo)

	router := handler.SetupRouter(authSvc, teamSvc, taskSvc, analyticsRepo, rdb, 100)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &testEnv{db: db, rdb: rdb, server: server}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func splitStatements(sql string) []string {
	// Split by semicolons, keeping CREATE INDEX as separate statements
	var result []string
	for _, stmt := range strings.Split(sql, ";") {
		s := strings.TrimSpace(stmt)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../migrations")
	require.NoError(t, err)

	// Verify migrations directory exists
	_, err = os.Stat(dir)
	require.NoError(t, err, "migrations directory not found at %s", dir)

	return dir
}

func doRequest(t *testing.T, method, url string, body any, token string) *http.Response {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func parseJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(t, err)
}

func TestIntegration_RegisterAndLogin(t *testing.T) {
	env := setupIntegration(t)

	// Register
	resp := doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{
		Email:    "user@test.com",
		Name:     "testuser",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var user models.User
	parseJSON(t, resp, &user)
	assert.Equal(t, "user@test.com", user.Email)

	// Register duplicate
	resp2 := doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{
		Email:    "user@test.com",
		Name:     "testuser2",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
	resp2.Body.Close()

	// Login
	resp3 := doRequest(t, "POST", env.server.URL+"/api/v1/login", models.LoginRequest{
		Email:    "user@test.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusOK, resp3.StatusCode)

	var authResp models.AuthResponse
	parseJSON(t, resp3, &authResp)
	assert.NotEmpty(t, authResp.Token)
}

func TestIntegration_FullWorkflow(t *testing.T) {
	env := setupIntegration(t)

	// Register user1
	resp := doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{
		Email: "user1@test.com", Name: "user1", Password: "pass123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Login user1
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/login", models.LoginRequest{
		Email: "user1@test.com", Password: "pass123",
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var auth1 models.AuthResponse
	parseJSON(t, resp, &auth1)

	// Register user2
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{
		Email: "user2@test.com", Name: "user2", Password: "pass123",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = doRequest(t, "POST", env.server.URL+"/api/v1/login", models.LoginRequest{
		Email: "user2@test.com", Password: "pass123",
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var auth2 models.AuthResponse
	parseJSON(t, resp, &auth2)

	token1 := auth1.Token
	token2 := auth2.Token

	resp = doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{Email: "outsider@test.com", Name: "outsider", Password: "pass123"}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/login", models.LoginRequest{Email: "outsider@test.com", Password: "pass123"}, "")
	var outsider models.AuthResponse
	parseJSON(t, resp, &outsider)

	// Create team
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/teams", models.CreateTeamRequest{
		Name: "Alpha Team", Description: "Test team",
	}, token1)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var team models.Team
	parseJSON(t, resp, &team)
	assert.Equal(t, "Alpha Team", team.Name)
	teamID := team.ID

	// List teams
	resp = doRequest(t, "GET", env.server.URL+"/api/v1/teams", nil, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var teams []models.Team
	parseJSON(t, resp, &teams)
	assert.Len(t, teams, 1)

	// Invite user2
	resp = doRequest(t, "POST", env.server.URL+fmt.Sprintf("/api/v1/teams/%d/invite", teamID), models.InviteRequest{
		UserID: auth2.User.ID, Role: "member",
	}, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Create task
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/tasks", models.CreateTaskRequest{
		Title: "Build feature", Description: "Important task", TeamID: teamID, Priority: "high",
	}, token1)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var task models.Task
	parseJSON(t, resp, &task)
	taskID := task.ID

	// List tasks with filters
	tasksURL := fmt.Sprintf("%s/api/v1/tasks?team_id=%d&status=todo", env.server.URL, teamID)
	resp = doRequest(t, "GET", tasksURL, nil, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var paginated models.PaginatedResponse
	parseJSON(t, resp, &paginated)
	assert.Equal(t, 1, paginated.Total)

	// Update task
	newStatus := "in_progress"
	resp = doRequest(t, "PUT", fmt.Sprintf("%s/api/v1/tasks/%d", env.server.URL, taskID), models.UpdateTaskRequest{
		Status: &newStatus, Version: task.Version,
	}, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var updatedTask models.Task
	parseJSON(t, resp, &updatedTask)
	assert.Equal(t, "in_progress", updatedTask.Status)
	assert.Equal(t, task.Version+1, updatedTask.Version)

	staleTitle := "stale update"
	resp = doRequest(t, "PUT", fmt.Sprintf("%s/api/v1/tasks/%d", env.server.URL, taskID), models.UpdateTaskRequest{Title: &staleTitle, Version: task.Version}, token1)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// Get task history
	resp = doRequest(t, "GET", fmt.Sprintf("%s/api/v1/tasks/%d/history", env.server.URL, taskID), nil, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var history []models.TaskHistory
	parseJSON(t, resp, &history)
	assert.NotEmpty(t, history)

	// Add comment
	resp = doRequest(t, "POST", fmt.Sprintf("%s/api/v1/tasks/%d/comments", env.server.URL, taskID), models.CreateCommentRequest{
		Content: "Working on it!",
	}, token2)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Get comments
	resp = doRequest(t, "GET", fmt.Sprintf("%s/api/v1/tasks/%d/comments", env.server.URL, taskID), nil, token1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var comments []models.TaskComment
	parseJSON(t, resp, &comments)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Working on it!", comments[0].Content)

	for _, url := range []string{
		fmt.Sprintf("%s/api/v1/tasks?team_id=%d", env.server.URL, teamID),
		fmt.Sprintf("%s/api/v1/tasks/%d/history", env.server.URL, taskID),
		fmt.Sprintf("%s/api/v1/tasks/%d/comments", env.server.URL, taskID),
	} {
		resp = doRequest(t, "GET", url, nil, outsider.Token)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()
	}
}

func TestIntegration_UnauthorizedAccess(t *testing.T) {
	env := setupIntegration(t)

	// Access without token
	resp := doRequest(t, "GET", env.server.URL+"/api/v1/teams", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Access with invalid token
	resp = doRequest(t, "GET", env.server.URL+"/api/v1/teams", nil, "invalid-token")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegration_TeamStats(t *testing.T) {
	env := setupIntegration(t)

	resp := doRequest(t, "POST", env.server.URL+"/api/v1/register", models.RegisterRequest{Email: "owner@test.com", Name: "owner", Password: "pass123"}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var owner models.User
	parseJSON(t, resp, &owner)
	resp = doRequest(t, "POST", env.server.URL+"/api/v1/login", models.LoginRequest{Email: owner.Email, Password: "pass123"}, "")
	var auth models.AuthResponse
	parseJSON(t, resp, &auth)

	resp = doRequest(t, "POST", env.server.URL+"/api/v1/teams", models.CreateTeamRequest{Name: "Stats Team"}, auth.Token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var team models.Team
	parseJSON(t, resp, &team)

	_, err := env.db.Exec(`INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, closed_at) VALUES
		(?, 'done-1', 'done', ?, ?, DATE_SUB(NOW(), INTERVAL 2 HOUR), DATE_SUB(NOW(), INTERVAL 1 HOUR)),
		(?, 'done-2', 'done', ?, ?, DATE_SUB(NOW(), INTERVAL 4 HOUR), DATE_SUB(NOW(), INTERVAL 2 HOUR)),
		(?, 'todo-1', 'todo', ?, ?, NOW(), NULL)`,
		team.ID, owner.ID, owner.ID, team.ID, owner.ID, owner.ID, team.ID, owner.ID, owner.ID)
	require.NoError(t, err)
	_, err = env.db.Exec(`INSERT INTO task_comments (task_id, user_id, content)
		SELECT id, ?, 'comment' FROM tasks WHERE team_id = ? LIMIT 2`, owner.ID, team.ID)
	require.NoError(t, err)

	resp = doRequest(t, "GET", fmt.Sprintf("%s/api/v1/teams/%d/stats", env.server.URL, team.ID), nil, auth.Token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var stats models.TeamStats
	parseJSON(t, resp, &stats)
	assert.Equal(t, team.ID, stats.TeamID)
	assert.Equal(t, 2, stats.TasksByStatus["done"])
	assert.Equal(t, 1, stats.TasksByStatus["todo"])
	assert.Equal(t, 2, stats.CommentCount)
	require.Len(t, stats.TopAssignees, 1)
	assert.Equal(t, owner.ID, stats.TopAssignees[0].UserID)
	assert.Equal(t, 2, stats.TopAssignees[0].ClosedTasks)
	require.NotNil(t, stats.AverageClosingTimeSeconds)
	assert.InDelta(t, 5400, *stats.AverageClosingTimeSeconds, 5)
}
