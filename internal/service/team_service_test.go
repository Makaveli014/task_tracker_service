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

func setupTeamTest(t *testing.T) (*TeamService, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	teamRepo := repository.NewTeamRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := NewTeamService(teamRepo, userRepo, nil)
	return svc, mock
}

func TestCreateTeam_Success(t *testing.T) {
	svc, mock := setupTeamTest(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO teams").
		WithArgs("Test Team", "A test team", uint64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// 'owner' is hardcoded in SQL, so only 2 args
	mock.ExpectExec("INSERT INTO team_members").
		WithArgs(int64(1), uint64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery("SELECT id, name, description, created_by, created_at FROM teams WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_by", "created_at"}).
			AddRow(1, "Test Team", "A test team", 1, testTime))

	team, err := svc.Create(context.Background(), &models.CreateTeamRequest{
		Name:        "Test Team",
		Description: "A test team",
	}, 1)

	require.NoError(t, err)
	assert.Equal(t, "Test Team", team.Name)
	assert.Equal(t, uint64(1), team.CreatedBy)
}

func TestListByUser(t *testing.T) {
	svc, mock := setupTeamTest(t)

	mock.ExpectQuery("SELECT t.id, t.name, t.description, t.created_by, t.created_at").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_by", "created_at"}).
			AddRow(1, "Team A", "Desc A", 1, testTime).
			AddRow(2, "Team B", "Desc B", 2, testTime))

	teams, err := svc.ListByUser(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Equal(t, "Team A", teams[0].Name)
}

func TestInvite_NotAuthorized(t *testing.T) {
	svc, mock := setupTeamTest(t)

	mock.ExpectQuery("SELECT role FROM team_members WHERE team_id").
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	err := svc.Invite(context.Background(), 1, 2, &models.InviteRequest{
		UserID: 3,
		Role:   "member",
	})

	assert.ErrorIs(t, err, ErrNotAuthorized)
}

func TestInvite_Success(t *testing.T) {
	svc, mock := setupTeamTest(t)

	mock.ExpectQuery("SELECT role FROM team_members WHERE team_id").
		WithArgs(uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("owner"))

	mock.ExpectQuery("SELECT id, name, description, created_by, created_at FROM teams WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_by", "created_at"}).
			AddRow(1, "Team A", "Desc", 1, testTime))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE id").
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}).
			AddRow(3, "user3@test.com", "user3", "hash", testTime))

	mock.ExpectExec("INSERT INTO team_members").
		WithArgs(uint64(1), uint64(3), "member").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := svc.Invite(context.Background(), 1, 1, &models.InviteRequest{
		UserID: 3,
		Role:   "member",
	})

	require.NoError(t, err)
}

func TestInvite_AlreadyMember(t *testing.T) {
	svc, mock := setupTeamTest(t)

	mock.ExpectQuery("SELECT role FROM team_members WHERE team_id").
		WithArgs(uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("admin"))

	mock.ExpectQuery("SELECT id, name, description, created_by, created_at FROM teams WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_by", "created_at"}).
			AddRow(1, "Team A", "Desc", 1, testTime))

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := svc.Invite(context.Background(), 1, 1, &models.InviteRequest{
		UserID: 3,
		Role:   "member",
	})

	assert.ErrorIs(t, err, ErrAlreadyMember)
}
