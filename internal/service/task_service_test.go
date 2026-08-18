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
	return NewTaskService(repository.NewTaskRepository(db), repository.NewTeamRepository(db)), mock
}

func TestCreateTaskNotMember(t *testing.T) {
	svc, mock := setupTaskTest(t)
	mock.ExpectQuery("SELECT role FROM team_members").WithArgs(uint64(1), uint64(99)).WillReturnRows(sqlmock.NewRows([]string{"role"}))
	_, err := svc.Create(context.Background(), &models.CreateTaskRequest{Title: "Task", TeamID: 1}, 99)
	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestCreateTaskRejectsExternalAssignee(t *testing.T) {
	svc, mock := setupTaskTest(t)
	assigneeID := uint64(2)
	mock.ExpectQuery("SELECT role FROM team_members").WithArgs(uint64(1), uint64(1)).WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))
	mock.ExpectQuery("SELECT role FROM team_members").WithArgs(uint64(1), assigneeID).WillReturnRows(sqlmock.NewRows([]string{"role"}))
	_, err := svc.Create(context.Background(), &models.CreateTaskRequest{Title: "Task", TeamID: 1, AssigneeID: &assigneeID}, 1)
	assert.EqualError(t, err, "assignee must be a team member")
}

func TestListTasksRequiresMembership(t *testing.T) {
	svc, mock := setupTaskTest(t)
	mock.ExpectQuery("SELECT role FROM team_members").WithArgs(uint64(1), uint64(9)).WillReturnRows(sqlmock.NewRows([]string{"role"}))
	_, _, err := svc.List(context.Background(), 1, 9, "", nil, 20, 0)
	assert.ErrorIs(t, err, ErrNotAuthorized)
}
