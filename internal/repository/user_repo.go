package repository

import (
	"context"
	"database/sql"

	"TestTask_Bazis/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO users (email, name, password_hash) VALUES (?, ?, ?)",
		email, name, passwordHash,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, uint64(id))
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, email, name, password_hash, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil {
		u.Username = u.Name
	}
	return u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRowContext(ctx,
		"SELECT id, email, name, password_hash, created_at FROM users WHERE email = ?", email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err == nil {
		u.Username = u.Name
	}
	return u, err
}
