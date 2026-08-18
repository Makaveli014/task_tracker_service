package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"TestTask_Bazis/internal/models"
	"TestTask_Bazis/internal/repository"
)

var testTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func setupAuthTest(t *testing.T) (*AuthService, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	userRepo := repository.NewUserRepository(db)
	svc := NewAuthService(userRepo, "test-secret", 24)
	return svc, mock
}

func TestRegister_Success(t *testing.T) {
	svc, mock := setupAuthTest(t)

	// GetByEmail: no user found
	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE email").
		WithArgs("test@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}))

	mock.ExpectExec("INSERT INTO users").
		WithArgs("test@test.com", "testuser", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE id").
		WithArgs(uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}).
			AddRow(1, "test@test.com", "testuser", "hash", testTime))

	user, err := svc.Register(context.Background(), &models.RegisterRequest{
		Email:    "test@test.com",
		Username: "testuser",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "test@test.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
}

func TestRegister_EmailTaken(t *testing.T) {
	svc, mock := setupAuthTest(t)

	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE email").
		WithArgs("existing@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}).
			AddRow(1, "existing@test.com", "user1", "hash", testTime))

	_, err := svc.Register(context.Background(), &models.RegisterRequest{
		Email:    "existing@test.com",
		Username: "user2",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrEmailTaken)
}

func TestLogin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	svc := NewAuthService(userRepo, "test-secret", 24)

	hashBytes, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE email").
		WithArgs("test@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}).
			AddRow(1, "test@test.com", "testuser", string(hashBytes), testTime))

	token, user, err := svc.Login(context.Background(), &models.LoginRequest{
		Email: "test@test.com", Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "test@test.com", user.Email)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mock := setupAuthTest(t)

	mock.ExpectQuery("SELECT id, email, username, password_hash, created_at FROM users WHERE email").
		WithArgs("notfound@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "created_at"}))

	_, _, err := svc.Login(context.Background(), &models.LoginRequest{
		Email:    "notfound@test.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCreds)
}

func TestValidateToken_Valid(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	svc := NewAuthService(userRepo, "test-secret", 24)

	token, err := svc.generateToken(42)
	require.NoError(t, err)

	userID, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), userID)
}

func TestValidateToken_Invalid(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	svc := NewAuthService(userRepo, "test-secret", 24)

	_, err = svc.ValidateToken("invalid.token.here")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	svc1 := NewAuthService(userRepo, "secret-1", 24)
	svc2 := NewAuthService(userRepo, "secret-2", 24)

	token, err := svc1.generateToken(1)
	require.NoError(t, err)

	_, err = svc2.ValidateToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
