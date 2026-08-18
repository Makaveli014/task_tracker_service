package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
)

func setupTaskTest(t *testing.T) (*TaskService, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	taskRepo := repository.NewTaskRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	svc := NewTaskService(taskRepo, teamRepo)
	return svc, mock
}

var taskCols = []string{"id", "title", "description", "status", "priority", "assignee_id", "team_id", "created_by", "created_at", "updated_at"}

func TestCreateTask_Success(t *testing.T) {
	svc, mock := setupTaskTest(t)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectExec("INSERT INTO tasks").
		WithArgs("Test Task", "Description", "todo", "medium", nil, uint64(1), uint64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(1, "Test Task", "Description", "todo", "medium", nil, 1, 1, testTime, testTime))

	task, err := svc.Create(context.Background(), &models.CreateTaskRequest{
		Title:       "Test Task",
		Description: "Description",
		TeamID:      1,
	}, 1)

	require.NoError(t, err)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "todo", task.Status)
}

func TestCreateTask_NotMember(t *testing.T) {
	svc, mock := setupTaskTest(t)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	_, err := svc.Create(context.Background(), &models.CreateTaskRequest{
		Title:  "Task",
		TeamID: 1,
	}, 99)

	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestList_Pagination(t *testing.T) {
	svc, mock := setupTaskTest(t)

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	mock.ExpectQuery("SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at").
		WithArgs(uint64(1), 10, 0).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(1, "Task 1", "Desc", "todo", "medium", nil, 1, 1, testTime, testTime))

	tasks, total, err := svc.List(context.Background(), 1, "", nil, 1, 10)

	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, tasks, 1)
}

func TestList_DefaultPagination(t *testing.T) {
	page := 0
	perPage := -1
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, perPage)
}

func TestUpdateTask_RecordsHistory(t *testing.T) {
	svc, mock := setupTaskTest(t)

	mock.ExpectQuery("SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(1, "Old Title", "Desc", "todo", "medium", nil, 1, 1, testTime, testTime))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("owner"))

	mock.ExpectExec("INSERT INTO task_history").
		WithArgs(uint64(1), uint64(1), "title", "Old Title", "New Title").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE tasks SET").
		WithArgs("New Title", uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT id, title, description, status, priority, assignee_id, team_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows(taskCols).
			AddRow(1, "New Title", "Desc", "todo", "medium", nil, 1, 1, testTime, testTime))

	newTitle := "New Title"
	task, err := svc.Update(context.Background(), 1, &models.UpdateTaskRequest{
		Title: &newTitle,
	}, 1)

	require.NoError(t, err)
	assert.Equal(t, "New Title", task.Title)
}

func TestGetFieldValue(t *testing.T) {
	assigneeID := uint64(5)
	task := &models.Task{
		Title:      "Test",
		Status:     "todo",
		Priority:   "high",
		AssigneeID: &assigneeID,
	}

	assert.Equal(t, "Test", getFieldValue(task, "title"))
	assert.Equal(t, "todo", getFieldValue(task, "status"))
	assert.Equal(t, "high", getFieldValue(task, "priority"))
	assert.Equal(t, "5", getFieldValue(task, "assignee_id"))
	assert.Equal(t, "", getFieldValue(task, "unknown"))

	task.AssigneeID = nil
	assert.Equal(t, "", getFieldValue(task, "assignee_id"))
}
